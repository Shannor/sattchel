package driven

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"

	"github.com/google/uuid"

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

// NewFileStorage builds a FileStorage backed by path. The DB is loaded lazily
// on first use. Pass a non-nil db to seed an in-memory state without reading
// from disk.
func NewFileStorage(path string, db *DB) core.TrackerRepository {
	return &FileStorage{path: path, db: db}
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

// flush writes the in-memory DB atomically via tmp + rename. Caller must hold s.mu.
func (s *FileStorage) flush() error {
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

func (s *FileStorage) CreateProject(_ context.Context, project *core.Project) (*core.Project, error) {
	if project == nil {
		return nil, errors.New("nil project")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	project.ID = uuid.NewString()
	s.db.Projects[project.ID] = *project
	if err := s.flush(); err != nil {
		delete(s.db.Projects, project.ID)
		return nil, err
	}
	return new(s.db.Projects[project.ID]), nil
}

func (s *FileStorage) GetProjects(_ context.Context) ([]core.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var results []core.Project
	for _, project := range s.db.Projects {
		results = append(results, project)
	}
	return results, nil
}

func (s *FileStorage) GetProject(_ context.Context, projectID string) (*core.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if project, ok := s.db.Projects[projectID]; ok {
		return &project, nil
	}

	return nil, fmt.Errorf("project %w", ErrNotFound)

}
func (s *FileStorage) UpdateProject(_ context.Context, project *core.Project) (*core.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

	if err := s.flush(); err != nil {
		return nil, err
	}
	return &current, nil

}

func (s *FileStorage) GetGoals(ctx context.Context, projectID string) ([]core.Goal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	var results []core.Goal
	for _, goal := range s.db.Goals {
		results = append(results, goal)
	}
	return results, nil
}

func (s *FileStorage) GetGoal(ctx context.Context, goalID string) (*core.Goal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if goal, ok := s.db.Goals[goalID]; ok {
		return &goal, nil
	}
	return nil, fmt.Errorf("goal %s: %w", goalID, ErrNotFound)
}

func (s *FileStorage) UpdateGoal(_ context.Context, goal *core.Goal) (*core.Goal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

	if err := s.flush(); err != nil {
		return nil, err
	}
	return &current, nil
}

func (s *FileStorage) CreateGoal(_ context.Context, projectID string, goal *core.Goal) (*core.Goal, error) {
	if goal == nil {
		return nil, errors.New("nil goal")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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

	if err := s.flush(); err != nil {
		delete(s.db.Goals, goal.ID)
		return nil, err
	}
	return new(s.db.Goals[goal.ID]), nil
}

func (s *FileStorage) CreateMember(_ context.Context, member *core.Member) (*core.Member, error) {
	if member == nil {
		return nil, errors.New("nil member")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	if member.ID == "" {
		member.ID = uuid.NewString()
	} else if _, ok := s.db.Members[member.ID]; ok {
		return nil, fmt.Errorf("member %s: %w", member.ID, ErrAlreadyExists)
	}
	s.db.Members[member.ID] = *member
	if err := s.flush(); err != nil {
		delete(s.db.Members, member.ID)
		return nil, err
	}
	return new(s.db.Members[member.ID]), nil
}

func (s *FileStorage) GetMember(_ context.Context, memberID string) (*core.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	if member, ok := s.db.Members[memberID]; ok {
		return new(member), nil
	}
	return nil, fmt.Errorf("member %s: %w", memberID, ErrNotFound)
}

func (s *FileStorage) GetMembers(_ context.Context) ([]core.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	var members []core.Member
	for _, member := range s.db.Members {
		members = append(members, member)
	}
	return members, nil
}
