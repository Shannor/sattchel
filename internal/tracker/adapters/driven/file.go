package driven

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"

	"sattchel/internal/tracker/core"
)

const schemaVersion = 1

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrAlreadyAssigned = errors.New("already assigned")
)

type FileStorage struct {
	mu   sync.Mutex
	path string
	db   *DB
}

type DB struct {
	Version  int                     `json:"version"`
	Projects map[string]core.Project `json:"projects"`
	Goals    map[string]core.Goal    `json:"goals"`
	Members  map[string]core.Member  `json:"members"`

	// GoalsByMembers maps member IDs to goal IDs.
	// This an optimization to get member information faster.
	// It will need to be updated when members change in goals.
	// Create, Update, and Deletes will need to check against this
	GoalsByMembers map[string][]string `json:"goalsByMembers"`
}

func newDB() *DB {
	return &DB{
		Version:        schemaVersion,
		Projects:       map[string]core.Project{},
		Goals:          map[string]core.Goal{},
		Members:        map[string]core.Member{},
		GoalsByMembers: map[string][]string{},
	}
}

func (db *DB) clone() *DB {
	if db == nil {
		return nil
	}
	clone := &DB{
		Version:        db.Version,
		Projects:       make(map[string]core.Project, len(db.Projects)),
		Goals:          make(map[string]core.Goal, len(db.Goals)),
		Members:        make(map[string]core.Member, len(db.Members)),
		GoalsByMembers: make(map[string][]string, len(db.GoalsByMembers)),
	}

	// Going to try this copy way first
	maps.Copy(clone.GoalsByMembers, db.GoalsByMembers)
	maps.Copy(clone.Projects, db.Projects)
	maps.Copy(clone.Goals, db.Goals)
	maps.Copy(clone.Members, db.Members)
	for k, v := range db.GoalsByMembers {
		clone.GoalsByMembers[k] = slices.Clone(v)
	}
	return clone
}

// NewFileStorage builds a FileStorage backed by path. The DB is loaded lazily
// on first use. Pass a non-nil db to seed an in-memory state without reading
// from disk.
func NewFileStorage(path string, db *DB) core.TrackerRepository {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return &FileStorage{path: os.ExpandEnv(path), db: db}
}

// txKey is used to mark a context as being within a transaction.
type txKey struct{}

// txState placeholder value for txKey
type txState struct{}

func (s *FileStorage) isTx(ctx context.Context) bool {
	return ctx.Value(txKey{}) != nil
}

func (s *FileStorage) lock(ctx context.Context) func() {
	if s.isTx(ctx) {
		return func() {}
	}
	s.mu.Lock()
	return func() {
		s.mu.Unlock()
	}
}

func (s *FileStorage) flushMaybe(ctx context.Context) error {
	if s.isTx(ctx) {
		return nil
	}
	return s.flush()
}

func (s *FileStorage) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.isTx(ctx) {
		return fn(ctx)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return err
	}

	backup := s.db.clone()
	txCtx := context.WithValue(ctx, txKey{}, txState{})
	err := fn(txCtx)
	if err != nil {
		s.db = backup
		return err
	}

	if err := s.flush(); err != nil {
		s.db = backup
		return err
	}
	return nil
}

// ensureLoaded loads the DB from disk on first access. Caller must hold s.mu.
func (s *FileStorage) ensureLoaded() error {
	if s.db != nil {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.db = newDB()
		return nil
	}
	if err != nil {
		return fmt.Errorf("read db: %w", err)
	}
	if len(data) == 0 {
		s.db = newDB()
		return nil
	}
	db := newDB()
	if err := json.Unmarshal(data, &db); err != nil {
		return fmt.Errorf("decode db: %w", err)
	}
	ensureMaps(db)
	s.db = db
	return nil
}

func ensureMaps(db *DB) {
	if db.Projects == nil {
		db.Projects = map[string]core.Project{}
	}
	if db.Goals == nil {
		db.Goals = map[string]core.Goal{}
	}
	if db.Members == nil {
		db.Members = map[string]core.Member{}
	}
	if db.GoalsByMembers == nil {
		db.GoalsByMembers = map[string][]string{}
	}
	if db.Version == 0 {
		db.Version = schemaVersion
	}
}

// syncChildren rebuilds the children lists dynamically based on the parent links
// to guarantee that the nested children arrays are 100% consistent and up to date.
func (s *FileStorage) syncChildren() {
	if s.db == nil || s.db.Goals == nil {
		return
	}

	// 1. Build a map of child IDs for each parent ID
	parentToChildren := make(map[string][]string)
	for id, g := range s.db.Goals {
		if g.HasParent() {
			pID := g.Parent.TargetID
			parentToChildren[pID] = append(parentToChildren[pID], id)
		}
	}

	// 2. Define a recursive function to build the Goal tree with nested children
	var buildTree func(id string) core.Goal
	buildTree = func(id string) core.Goal {
		g := s.db.Goals[id]
		childIDs := parentToChildren[id]
		if len(childIDs) == 0 {
			g.Children = nil
			return g
		}

		g.Children = make([]core.Goal, 0, len(childIDs))
		for _, childID := range childIDs {
			g.Children = append(g.Children, buildTree(childID))
		}
		return g
	}

	// 3. Rebuild every goal so its Children slice is fully hydrated and consistent
	for id := range s.db.Goals {
		s.db.Goals[id] = buildTree(id)
	}
}

// flush writes the in-memory DB atomically via tmp + rename. Caller must hold s.mu.
func (s *FileStorage) flush() error {
	s.syncChildren()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	tmp := s.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.db); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("encode db: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename tmp: %w", err)
	}
	return nil
}

func (s *FileStorage) CreateProject(ctx context.Context, project *core.Project) (*core.Project, error) {
	if project == nil {
		return nil, errors.New("nil project")
	}
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	project.ID = uuid.NewString()
	s.db.Projects[project.ID] = *project
	if err := s.flushMaybe(ctx); err != nil {
		delete(s.db.Projects, project.ID)
		return nil, err
	}
	return new(s.db.Projects[project.ID]), nil
}

func (s *FileStorage) GetProjects(ctx context.Context) ([]core.Project, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var results []core.Project
	for _, project := range s.db.Projects {
		results = append(results, project)
	}
	return results, nil
}

func (s *FileStorage) GetProject(ctx context.Context, projectID string) (*core.Project, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if project, ok := s.db.Projects[projectID]; ok {
		return &project, nil
	}

	return nil, fmt.Errorf("project %w", ErrNotFound)

}
func (s *FileStorage) UpdateProject(ctx context.Context, project *core.Project) (*core.Project, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	current, ok := s.db.Projects[project.ID]
	if !ok {
		return nil, fmt.Errorf("project %w", ErrNotFound)
	}

	current.Label = project.Label
	current.Description = project.Description
	current.RootGoalID = project.RootGoalID
	s.db.Projects[project.ID] = current

	if err := s.flushMaybe(ctx); err != nil {
		return nil, err
	}
	return &current, nil

}

func (s *FileStorage) GetGoals(ctx context.Context, projectID string) ([]core.Goal, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var results []core.Goal
	for _, goal := range s.db.Goals {
		results = append(results, goal)
	}
	return results, nil
}

func (s *FileStorage) QueryGoals(ctx context.Context, projectID string, query *core.GoalQuery) ([]core.Goal, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var results []core.Goal
	for _, goal := range s.db.Goals {
		if goal.ProjectID != projectID {
			continue
		}
		if query != nil {
			if query.ParentID != "" {
				if goal.Parent == nil || goal.Parent.TargetID != query.ParentID {
					continue
				}
			}
			if len(query.MemberIDs) > 0 {
				if goal.Member == nil || !slices.Contains(query.MemberIDs, goal.Member.ID) {
					continue
				}
			}
			if len(query.Impacts) > 0 {
				if !slices.Contains(query.Impacts, goal.Impact) {
					continue
				}
			}
			if len(query.Efforts) > 0 {
				if !slices.Contains(query.Efforts, goal.Effort) {
					continue
				}
			}
			if len(query.Relationships) > 0 {
				if goal.Parent == nil || !slices.Contains(query.Relationships, goal.Parent.Relationship) {
					continue
				}
			}
			if len(query.Statuses) > 0 {
				if !slices.Contains(query.Statuses, goal.Status) {
					continue
				}
			}
			if len(query.MissingFields) > 0 {
				hasMissingField := false
				for _, field := range query.MissingFields {
					switch strings.ToLower(strings.TrimSpace(field)) {
					case "member":
						if goal.Member == nil || goal.Member.ID == "" {
							hasMissingField = true
						}
					case "impact":
						if goal.Impact == core.UnknownImpact || goal.Impact == "" {
							hasMissingField = true
						}
					case "effort":
						if goal.Effort == core.UnknownEffort || goal.Effort == "" {
							hasMissingField = true
						}
					}
				}
				if !hasMissingField {
					continue
				}
			}
		}
		results = append(results, goal)
	}
	return results, nil
}

func (s *FileStorage) GetGoal(ctx context.Context, goalID string) (*core.Goal, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if goal, ok := s.db.Goals[goalID]; ok {
		return &goal, nil
	}
	return nil, fmt.Errorf("goal %s: %w", goalID, ErrNotFound)
}

func (s *FileStorage) UpdateGoal(ctx context.Context, goal *core.Goal) (*core.Goal, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	current, ok := s.db.Goals[goal.ID]
	if !ok {
		return nil, fmt.Errorf("goal %s: %w", goal.ID, ErrNotFound)
	}

	if goal.HasMember() {
		_, ok := s.db.Members[goal.Member.ID]
		if !ok {
			return nil, fmt.Errorf("member %s: %w", goal.Member.ID, ErrNotFound)
		}
	}

	current.Name = goal.Name
	current.Description = goal.Description
	current.Status = goal.Status
	current.Effort = goal.Effort
	current.Impact = goal.Impact
	current.ProjectID = goal.ProjectID
	current.Children = goal.Children
	current.Parent = goal.Parent
	current.Member = goal.Member

	// Remove old member if applicable
	if current.HasMember() {
		s.db.GoalsByMembers[current.Member.ID] = slices.DeleteFunc(
			s.db.GoalsByMembers[current.Member.ID],
			func(id string) bool { return id == goal.ID },
		)
	}

	// Add new member if applicable
	if goal.HasMember() {
		s.db.GoalsByMembers[goal.Member.ID] = append(s.db.GoalsByMembers[goal.Member.ID], goal.ID)
	}
	s.db.Goals[current.ID] = current

	if err := s.flushMaybe(ctx); err != nil {
		return nil, err
	}
	return &current, nil
}

func (s *FileStorage) CreateGoal(ctx context.Context, projectID string, goal *core.Goal) (*core.Goal, error) {
	if goal == nil {
		return nil, errors.New("nil goal")
	}
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if _, ok := s.db.Projects[projectID]; !ok {
		return nil, fmt.Errorf("project %s: %w", projectID, ErrNotFound)
	}
	if goal.ID == "" {
		goal.ID = uuid.NewString()
	} else if _, ok := s.db.Goals[goal.ID]; ok {
		return nil, fmt.Errorf("goal %s: %w", goal.ID, ErrAlreadyExists)
	}

	s.db.Goals[goal.ID] = *goal
	if goal.HasMember() {
		s.db.GoalsByMembers[goal.Member.ID] = append(s.db.GoalsByMembers[goal.Member.ID], goal.ID)
	}

	if err := s.flushMaybe(ctx); err != nil {
		delete(s.db.Goals, goal.ID)
		return nil, err
	}
	return new(s.db.Goals[goal.ID]), nil
}

func (s *FileStorage) CreateMember(ctx context.Context, member *core.Member) (*core.Member, error) {
	if member == nil {
		return nil, errors.New("nil member")
	}
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if member.ID == "" {
		member.ID = uuid.NewString()
	} else if _, ok := s.db.Members[member.ID]; ok {
		return nil, fmt.Errorf("member %s: %w", member.ID, ErrAlreadyExists)
	}
	s.db.Members[member.ID] = *member
	if err := s.flushMaybe(ctx); err != nil {
		delete(s.db.Members, member.ID)
		return nil, err
	}
	return new(s.db.Members[member.ID]), nil
}

func (s *FileStorage) GetMember(ctx context.Context, memberID string) (*core.Member, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	if member, ok := s.db.Members[memberID]; ok {
		return new(member), nil
	}
	return nil, fmt.Errorf("member %s: %w", memberID, ErrNotFound)
}

func (s *FileStorage) GetMembers(ctx context.Context) ([]core.Member, error) {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	var members []core.Member
	for _, member := range s.db.Members {
		members = append(members, member)
	}
	return members, nil
}

func (s *FileStorage) UpdateMember(ctx context.Context, member *core.Member) (*core.Member, error) {
	if member == nil {
		return nil, errors.New("nil member")
	}
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	current, ok := s.db.Members[member.ID]
	if !ok {
		return nil, fmt.Errorf("member %s: %w", member.ID, ErrNotFound)
	}

	current.Name = member.Name
	current.Email = member.Email
	s.db.Members[member.ID] = current

	if err := s.flushMaybe(ctx); err != nil {
		return nil, err
	}
	return &current, nil
}

func (s *FileStorage) DeleteMember(ctx context.Context, memberID string) error {
	unlock := s.lock(ctx)
	defer unlock()
	if err := s.ensureLoaded(); err != nil {
		return err
	}

	if _, ok := s.db.Members[memberID]; !ok {
		return fmt.Errorf("member %s: %w", memberID, ErrNotFound)
	}

	delete(s.db.Members, memberID)

	// Clean up related data/associations
	delete(s.db.GoalsByMembers, memberID)
	for goalID, goal := range s.db.Goals {
		if goal.HasMember() && goal.Member.ID == memberID {
			goal.Member = nil
			s.db.Goals[goalID] = goal
		}
	}

	return s.flushMaybe(ctx)
}
