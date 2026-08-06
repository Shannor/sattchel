package core

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type CompareFlagsOptions struct {
	Query          string
	BaseProjectID  string
	FocusProjectID string
}

type FlagComparisonEntry struct {
	Flag          FeatureFlagDefinition            `json:"flag"`
	SourceProject Project                          `json:"sourceProject"`
	PresentIn     []Project                        `json:"presentIn"`
	MissingIn     []Project                        `json:"missingIn"`
	FlagByProject map[string]FeatureFlagDefinition `json:"flagByProject"`
}

type FlagComparisonReport struct {
	Projects             []Project                                   `json:"projects"`
	FlagCountByProject   map[string]int                              `json:"flagCountByProject"`
	SharedFlags          []FeatureFlagDefinition                     `json:"sharedFlags"`
	SharedFlagsByProject map[string]map[string]FeatureFlagDefinition `json:"sharedFlagsByProject"`
	MissingFlags         []FlagComparisonEntry                       `json:"missingFlags"`
}

type UniqueFlagEntry struct {
	Flag            FeatureFlagDefinition `json:"flag"`
	TargetProject   Project               `json:"targetProject"`
	ComparedAgainst []Project             `json:"comparedAgainst"`
}

type DormantFlagEntry struct {
	Flag      FeatureFlagDefinition `json:"flag"`
	PresentIn []Project             `json:"presentIn"`
}

type VariableDefinitionSpec struct {
	Key          string `json:"key"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue"`
	Description  string `json:"description,omitempty"`
}

type VariableDriftValue struct {
	Exists       bool   `json:"exists"`
	Type         string `json:"type,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Description  string `json:"description,omitempty"`
}

type VariableDriftEntry struct {
	Key             string                        `json:"key"`
	ValuesByProject map[string]VariableDriftValue `json:"valuesByProject"`
}

type FlagVariableDrift struct {
	FlagKey   string               `json:"flagKey"`
	FlagName  string               `json:"flagName"`
	PresentIn []Project            `json:"presentIn"`
	Variables []VariableDriftEntry `json:"variables"`
}

type FlagVariableSyncUpdate struct {
	FlagKey          string                            `json:"flagKey"`
	MissingVariables map[string]VariableDefinitionSpec `json:"missingVariables"`
}

type FlagSyncTargetCreate struct {
	Project Project                 `json:"project"`
	Flags   []FeatureFlagDefinition `json:"flags"`
}

type FlagSyncTargetVariableUpdate struct {
	Project Project                  `json:"project"`
	Updates []FlagVariableSyncUpdate `json:"updates"`
}

type FlagSyncPlan struct {
	SourceProjectID       string                         `json:"sourceProjectId,omitempty"`
	UnionSource           bool                           `json:"unionSource"`
	Projects              []Project                      `json:"projects"`
	TargetMissing         []FlagSyncTargetCreate         `json:"targetMissing"`
	TargetVariableUpdates []FlagSyncTargetVariableUpdate `json:"targetVariableUpdates"`
}

type FlagSyncOptions struct {
	SourceProjectID  string
	TargetProjectIDs []string
	FlagKeys         []string
	UpdateVariables  bool
	UnionSource      bool
	SyncVariations   bool
}

type FlagSyncResult struct {
	CreatedFlags    int `json:"createdFlags"`
	AddedVariables  int `json:"addedVariables"`
	TouchedProjects int `json:"touchedProjects"`
}

func (v Variables) Definitions() map[string]VariableDefinitionSpec {
	defs := make(map[string]VariableDefinitionSpec)
	for key, val := range v.BoolVariables {
		defs[key] = VariableDefinitionSpec{Key: key, Type: val.Type, DefaultValue: strconv.FormatBool(val.Value), Description: val.Description}
	}
	for key, val := range v.IntVariables {
		defs[key] = VariableDefinitionSpec{Key: key, Type: val.Type, DefaultValue: strconv.Itoa(val.Value), Description: val.Description}
	}
	for key, val := range v.FloatVariables {
		defs[key] = VariableDefinitionSpec{Key: key, Type: val.Type, DefaultValue: strconv.FormatFloat(val.Value, 'f', -1, 64), Description: val.Description}
	}
	for key, val := range v.StringVariables {
		defs[key] = VariableDefinitionSpec{Key: key, Type: val.Type, DefaultValue: val.Value, Description: val.Description}
	}
	for key, val := range v.JsonVariables {
		defs[key] = VariableDefinitionSpec{Key: key, Type: val.Type, DefaultValue: stringifyVariableValue(val.Value), Description: val.Description}
	}
	return defs
}

func VariablesFromDefinitions(defs map[string]VariableDefinitionSpec) Variables {
	result := Variables{}
	for key, def := range defs {
		switch def.Type {
		case "boolean":
			parsed, err := strconv.ParseBool(def.DefaultValue)
			if err != nil {
				continue
			}
			if result.BoolVariables == nil {
				result.BoolVariables = make(VariableMap[bool])
			}
			result.BoolVariables[key] = Variable[bool]{Key: key, Value: parsed, Type: def.Type, Description: def.Description}
		case "integer":
			parsed, err := strconv.Atoi(def.DefaultValue)
			if err != nil {
				continue
			}
			if result.IntVariables == nil {
				result.IntVariables = make(VariableMap[int])
			}
			result.IntVariables[key] = Variable[int]{Key: key, Value: parsed, Type: def.Type, Description: def.Description}
		case "double":
			parsed, err := strconv.ParseFloat(def.DefaultValue, 64)
			if err != nil {
				continue
			}
			if result.FloatVariables == nil {
				result.FloatVariables = make(VariableMap[float64])
			}
			result.FloatVariables[key] = Variable[float64]{Key: key, Value: parsed, Type: def.Type, Description: def.Description}
		case "json":
			if result.JsonVariables == nil {
				result.JsonVariables = make(VariableMap[any])
			}
			result.JsonVariables[key] = Variable[any]{Key: key, Value: def.DefaultValue, Type: def.Type, Description: def.Description}
		default:
			if result.StringVariables == nil {
				result.StringVariables = make(VariableMap[string])
			}
			result.StringVariables[key] = Variable[string]{Key: key, Value: def.DefaultValue, Type: "string", Description: def.Description}
		}
	}
	return result
}

func (s *Service) CompareFlagsDetailed(ctx context.Context, projectIDs []string, opts CompareFlagsOptions) (*FlagComparisonReport, error) {
	projectIDs = normalizeIDs(projectIDs)
	if opts.BaseProjectID != "" && opts.FocusProjectID != "" {
		return nil, fmt.Errorf("base and focus comparison modes are mutually exclusive")
	}
	if opts.BaseProjectID != "" && !slices.Contains(projectIDs, opts.BaseProjectID) {
		projectIDs = append([]string{opts.BaseProjectID}, projectIDs...)
		projectIDs = normalizeIDs(projectIDs)
	}
	if len(projectIDs) < 2 {
		return nil, fmt.Errorf("at least 2 project IDs must be provided for comparison")
	}
	if opts.FocusProjectID != "" && !slices.Contains(projectIDs, opts.FocusProjectID) {
		return nil, fmt.Errorf("focus project %q is not in the comparison set", opts.FocusProjectID)
	}

	projectFlags, projects, err := s.loadProjectFlags(ctx, projectIDs)
	if err != nil {
		return nil, err
	}

	allKeys := make(map[string]struct{})
	for _, pid := range projectIDs {
		for _, flag := range filterFlags(projectFlags[pid], opts.Query) {
			allKeys[flag.Key] = struct{}{}
		}
	}

	report := &FlagComparisonReport{
		Projects:             projects,
		FlagCountByProject:   make(map[string]int, len(projectIDs)),
		SharedFlags:          make([]FeatureFlagDefinition, 0),
		SharedFlagsByProject: make(map[string]map[string]FeatureFlagDefinition),
		MissingFlags:         make([]FlagComparisonEntry, 0),
	}

	for _, pid := range projectIDs {
		report.FlagCountByProject[pid] = len(filterFlags(projectFlags[pid], opts.Query))
	}

	for key := range allKeys {
		presentIn := make([]Project, 0, len(projectIDs))
		missingIn := make([]Project, 0, len(projectIDs))
		flagByProject := make(map[string]FeatureFlagDefinition)
		var sourceFlag FeatureFlagDefinition
		var sourceProject Project

		for _, pid := range projectIDs {
			flag, ok := findFlagByKey(filterFlags(projectFlags[pid], opts.Query), key)
			project := projectByID(projects, pid)
			if ok {
				presentIn = append(presentIn, project)
				flagByProject[pid] = flag
				if sourceFlag.Key == "" {
					sourceFlag = flag
					sourceProject = project
				}
			} else {
				missingIn = append(missingIn, project)
			}
		}

		if len(missingIn) == 0 {
			report.SharedFlags = append(report.SharedFlags, sourceFlag)
			report.SharedFlagsByProject[key] = flagByProject
			continue
		}

		entry := FlagComparisonEntry{
			Flag:          sourceFlag,
			SourceProject: sourceProject,
			PresentIn:     presentIn,
			MissingIn:     missingIn,
			FlagByProject: flagByProject,
		}

		if opts.BaseProjectID != "" && !projectListContains(missingIn, opts.BaseProjectID) {
			continue
		}
		if opts.FocusProjectID != "" && !projectListContains(presentIn, opts.FocusProjectID) {
			continue
		}

		report.MissingFlags = append(report.MissingFlags, entry)
	}

	sortFlagsNewestFirst(report.SharedFlags)
	slices.SortFunc(report.MissingFlags, func(a, b FlagComparisonEntry) int {
		return compareFlagNewestFirst(a.Flag, b.Flag)
	})

	return report, nil
}

func (s *Service) FindUniqueFlags(ctx context.Context, targetProjectID string, againstProjectIDs []string, query string) ([]UniqueFlagEntry, error) {
	againstProjectIDs = normalizeIDs(againstProjectIDs)
	if targetProjectID == "" {
		return nil, fmt.Errorf("target project ID is required")
	}
	comparisonIDs := normalizeIDs(append([]string{targetProjectID}, againstProjectIDs...))
	if len(comparisonIDs) < 2 {
		return nil, fmt.Errorf("at least 1 comparison project ID is required")
	}

	projectFlags, projects, err := s.loadProjectFlags(ctx, comparisonIDs)
	if err != nil {
		return nil, err
	}

	targetFlags := filterFlags(projectFlags[targetProjectID], query)
	results := make([]UniqueFlagEntry, 0)
	for _, flag := range targetFlags {
		isUnique := true
		for _, pid := range againstProjectIDs {
			if _, ok := findFlagByKey(projectFlags[pid], flag.Key); ok {
				isUnique = false
				break
			}
		}
		if isUnique {
			results = append(results, UniqueFlagEntry{
				Flag:            flag,
				TargetProject:   projectByID(projects, targetProjectID),
				ComparedAgainst: projectsWithout(projects, targetProjectID),
			})
		}
	}

	slices.SortFunc(results, func(a, b UniqueFlagEntry) int {
		return compareFlagNewestFirst(a.Flag, b.Flag)
	})
	return results, nil
}

func (s *Service) FindDormantFlags(ctx context.Context, projectIDs []string, query string) ([]DormantFlagEntry, error) {
	projectIDs = normalizeIDs(projectIDs)
	if len(projectIDs) == 0 {
		return nil, fmt.Errorf("at least 1 project ID is required")
	}

	projectFlags, projects, err := s.loadProjectFlags(ctx, projectIDs)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string][]struct {
		project Project
		flag    FeatureFlagDefinition
	})
	for _, pid := range projectIDs {
		for _, flag := range filterFlags(projectFlags[pid], query) {
			byKey[flag.Key] = append(byKey[flag.Key], struct {
				project Project
				flag    FeatureFlagDefinition
			}{project: projectByID(projects, pid), flag: flag})
		}
	}

	results := make([]DormantFlagEntry, 0)
	for _, entries := range byKey {
		if len(entries) == 0 {
			continue
		}
		anyEnabled := false
		presentIn := make([]Project, 0, len(entries))
		for _, entry := range entries {
			presentIn = append(presentIn, entry.project)
			if flagEnabledAnywhere(entry.flag) {
				anyEnabled = true
			}
		}
		if !anyEnabled {
			results = append(results, DormantFlagEntry{Flag: entries[0].flag, PresentIn: presentIn})
		}
	}

	slices.SortFunc(results, func(a, b DormantFlagEntry) int {
		return compareFlagNewestFirst(a.Flag, b.Flag)
	})
	return results, nil
}

func (s *Service) FindVariableDrift(ctx context.Context, projectIDs []string, query string) ([]FlagVariableDrift, error) {
	projectIDs = normalizeIDs(projectIDs)
	if len(projectIDs) < 2 {
		return nil, fmt.Errorf("at least 2 project IDs are required")
	}

	projectFlags, projects, err := s.loadProjectFlags(ctx, projectIDs)
	if err != nil {
		return nil, err
	}

	flagByKey := make(map[string]map[string]FeatureFlagDefinition)
	for _, pid := range projectIDs {
		for _, flag := range filterFlags(projectFlags[pid], query) {
			if _, ok := flagByKey[flag.Key]; !ok {
				flagByKey[flag.Key] = make(map[string]FeatureFlagDefinition)
			}
			flagByKey[flag.Key][pid] = flag
		}
	}

	results := make([]FlagVariableDrift, 0)
	for key, byProject := range flagByKey {
		if len(byProject) < 2 {
			continue
		}

		var name string
		presentIn := make([]Project, 0, len(byProject))
		unionVarKeys := make(map[string]struct{})
		for _, pid := range projectIDs {
			flag, ok := byProject[pid]
			if !ok {
				continue
			}
			if name == "" {
				name = flag.Name
			}
			presentIn = append(presentIn, projectByID(projects, pid))
			for varKey := range flag.DefaultVariables.Definitions() {
				unionVarKeys[varKey] = struct{}{}
			}
		}

		variableEntries := make([]VariableDriftEntry, 0)
		for varKey := range unionVarKeys {
			hasMissing := false
			hasPresent := false
			for _, pid := range projectIDs {
				flag, ok := byProject[pid]
				if !ok {
					continue
				}
				defs := flag.DefaultVariables.Definitions()
				_, exists := defs[varKey]
				if exists {
					hasPresent = true
				} else {
					hasMissing = true
				}
			}

			if hasMissing && hasPresent {
				valuesByProject := make(map[string]VariableDriftValue)
				for _, pid := range projectIDs {
					flag, ok := byProject[pid]
					if !ok {
						continue
					}
					defs := flag.DefaultVariables.Definitions()
					def, exists := defs[varKey]
					value := VariableDriftValue{Exists: exists}
					if exists {
						value.Type = def.Type
						value.DefaultValue = def.DefaultValue
						value.Description = def.Description
					}
					valuesByProject[pid] = value
				}
				variableEntries = append(variableEntries, VariableDriftEntry{Key: varKey, ValuesByProject: valuesByProject})
			}
		}

		if len(variableEntries) == 0 {
			continue
		}
		slices.SortFunc(variableEntries, func(a, b VariableDriftEntry) int {
			return strings.Compare(strings.ToLower(a.Key), strings.ToLower(b.Key))
		})
		results = append(results, FlagVariableDrift{FlagKey: key, FlagName: name, PresentIn: presentIn, Variables: variableEntries})
	}

	slices.SortFunc(results, func(a, b FlagVariableDrift) int {
		return strings.Compare(strings.ToLower(a.FlagKey), strings.ToLower(b.FlagKey))
	})
	return results, nil
}

func (s *Service) PlanFlagSync(ctx context.Context, opts FlagSyncOptions) (*FlagSyncPlan, error) {
	if opts.UnionSource {
		return s.planUnionFlagSync(ctx, opts)
	}
	if opts.SourceProjectID == "" {
		return nil, fmt.Errorf("source project ID is required")
	}
	if len(opts.TargetProjectIDs) == 0 {
		return nil, fmt.Errorf("at least 1 target project ID is required")
	}

	allProjectIDs := normalizeIDs(append([]string{opts.SourceProjectID}, opts.TargetProjectIDs...))
	projectFlags, projects, err := s.loadProjectFlags(ctx, allProjectIDs)
	if err != nil {
		return nil, err
	}

	sourceFlags := projectFlags[opts.SourceProjectID]
	if len(opts.FlagKeys) > 0 {
		sourceFlags = filterFlagsByKeys(sourceFlags, opts.FlagKeys)
	}

	plan := &FlagSyncPlan{SourceProjectID: opts.SourceProjectID, Projects: projects}
	for _, targetProjectID := range opts.TargetProjectIDs {
		targetFlags := projectFlags[targetProjectID]
		targetMap := make(map[string]FeatureFlagDefinition, len(targetFlags))
		for _, flag := range targetFlags {
			targetMap[flag.Key] = flag
		}

		missing := make([]FeatureFlagDefinition, 0)
		updates := make([]FlagVariableSyncUpdate, 0)
		for _, sourceFlag := range sourceFlags {
			targetFlag, ok := targetMap[sourceFlag.Key]
			if !ok {
				flagToCreate := sourceFlag
				if !opts.SyncVariations {
					flagToCreate.Overrides = nil
				}
				missing = append(missing, flagToCreate)
				continue
			}
			if !opts.UpdateVariables {
				continue
			}
			missingVars := missingVariableDefinitions(targetFlag.DefaultVariables.Definitions(), sourceFlag.DefaultVariables.Definitions())
			if len(missingVars) > 0 {
				updates = append(updates, FlagVariableSyncUpdate{FlagKey: targetFlag.Key, MissingVariables: missingVars})
			}
		}
		if len(missing) > 0 {
			plan.TargetMissing = append(plan.TargetMissing, FlagSyncTargetCreate{Project: projectByID(projects, targetProjectID), Flags: missing})
		}
		if len(updates) > 0 {
			plan.TargetVariableUpdates = append(plan.TargetVariableUpdates, FlagSyncTargetVariableUpdate{Project: projectByID(projects, targetProjectID), Updates: updates})
		}
	}
	return plan, nil
}

func (s *Service) ApplyFlagSyncPlan(ctx context.Context, plan FlagSyncPlan) (*FlagSyncResult, error) {
	result := &FlagSyncResult{}
	touchedProjects := make(map[string]struct{})

	for _, target := range plan.TargetMissing {
		repo, err := s.flagFactory.Create(ctx, target.Project.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create flags repository for project %s: %w", target.Project.ID, err)
		}
		for _, flag := range target.Flags {
			if _, err := repo.Create(ctx, flag); err != nil {
				return nil, fmt.Errorf("failed to create flag %s in project %s: %w", flag.Key, target.Project.ID, err)
			}
			result.CreatedFlags++
			touchedProjects[target.Project.ID] = struct{}{}
		}
	}

	for _, target := range plan.TargetVariableUpdates {
		repo, err := s.flagFactory.Create(ctx, target.Project.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create flags repository for project %s: %w", target.Project.ID, err)
		}
		for _, update := range target.Updates {
			if err := repo.AddVariables(ctx, update.FlagKey, VariablesFromDefinitions(update.MissingVariables)); err != nil {
				return nil, fmt.Errorf("failed to add variables to flag %s in project %s: %w", update.FlagKey, target.Project.ID, err)
			}
			result.AddedVariables += len(update.MissingVariables)
			touchedProjects[target.Project.ID] = struct{}{}
		}
	}

	result.TouchedProjects = len(touchedProjects)
	return result, nil
}

func (s *Service) planUnionFlagSync(ctx context.Context, opts FlagSyncOptions) (*FlagSyncPlan, error) {
	projectIDs := normalizeIDs(opts.TargetProjectIDs)
	if len(projectIDs) < 2 {
		return nil, fmt.Errorf("at least 2 project IDs are required for union sync")
	}

	projectFlags, projects, err := s.loadProjectFlags(ctx, projectIDs)
	if err != nil {
		return nil, err
	}

	flagMaps := make(map[string]map[string]FeatureFlagDefinition, len(projectIDs))
	allKeys := make(map[string]struct{})
	for _, pid := range projectIDs {
		flags := projectFlags[pid]
		if len(opts.FlagKeys) > 0 {
			flags = filterFlagsByKeys(flags, opts.FlagKeys)
		}
		flagMap := make(map[string]FeatureFlagDefinition, len(flags))
		for _, flag := range flags {
			flagMap[flag.Key] = flag
			allKeys[flag.Key] = struct{}{}
		}
		flagMaps[pid] = flagMap
	}

	plan := &FlagSyncPlan{UnionSource: true, Projects: projects}
	for flagKey := range allKeys {
		unionDefs := make(map[string]VariableDefinitionSpec)
		var templateFlag FeatureFlagDefinition
		for _, pid := range projectIDs {
			flag, ok := flagMaps[pid][flagKey]
			if !ok {
				continue
			}
			if templateFlag.Key == "" {
				templateFlag = flag
			}
			for key, def := range flag.DefaultVariables.Definitions() {
				if _, ok := unionDefs[key]; !ok {
					unionDefs[key] = def
				}
			}
		}

		for _, pid := range projectIDs {
			flag, ok := flagMaps[pid][flagKey]
			if !ok {
				flagToCreate := templateFlag
				if !opts.SyncVariations {
					flagToCreate.Overrides = nil
				}
				appendMissingFlag(plan, projectByID(projects, pid), flagToCreate)
				continue
			}
			missingVars := missingVariableDefinitions(flag.DefaultVariables.Definitions(), unionDefs)
			if len(missingVars) > 0 {
				appendVariableUpdate(plan, projectByID(projects, pid), FlagVariableSyncUpdate{FlagKey: flag.Key, MissingVariables: missingVars})
			}
		}
	}

	return plan, nil
}

func appendMissingFlag(plan *FlagSyncPlan, project Project, flag FeatureFlagDefinition) {
	for i := range plan.TargetMissing {
		if plan.TargetMissing[i].Project.ID == project.ID {
			plan.TargetMissing[i].Flags = append(plan.TargetMissing[i].Flags, flag)
			return
		}
	}
	plan.TargetMissing = append(plan.TargetMissing, FlagSyncTargetCreate{Project: project, Flags: []FeatureFlagDefinition{flag}})
}

func appendVariableUpdate(plan *FlagSyncPlan, project Project, update FlagVariableSyncUpdate) {
	for i := range plan.TargetVariableUpdates {
		if plan.TargetVariableUpdates[i].Project.ID == project.ID {
			plan.TargetVariableUpdates[i].Updates = append(plan.TargetVariableUpdates[i].Updates, update)
			return
		}
	}
	plan.TargetVariableUpdates = append(plan.TargetVariableUpdates, FlagSyncTargetVariableUpdate{Project: project, Updates: []FlagVariableSyncUpdate{update}})
}

func (s *Service) loadProjectFlags(ctx context.Context, projectIDs []string) (map[string][]FeatureFlagDefinition, []Project, error) {
	projectIDs = normalizeIDs(projectIDs)
	projectFlags, err := s.GetFlags(ctx, projectIDs)
	if err != nil {
		return nil, nil, err
	}
	projects, err := s.lookupProjects(ctx, projectIDs)
	if err != nil {
		return nil, nil, err
	}
	return projectFlags, projects, nil
}

func (s *Service) lookupProjects(ctx context.Context, projectIDs []string) ([]Project, error) {
	projectIDs = normalizeIDs(projectIDs)
	lookup := make(map[string]Project, len(projectIDs))
	for _, id := range projectIDs {
		lookup[id] = Project{ID: id, Name: id}
	}
	if s.projectRepo != nil {
		projects, err := s.projectRepo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get projects: %w", err)
		}
		for _, project := range projects {
			if _, ok := lookup[project.ID]; ok {
				lookup[project.ID] = project
			}
		}
	}
	ordered := make([]Project, 0, len(projectIDs))
	for _, id := range projectIDs {
		ordered = append(ordered, lookup[id])
	}
	return ordered, nil
}

func normalizeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func filterFlagsByKeys(flags []FeatureFlagDefinition, keys []string) []FeatureFlagDefinition {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keySet[trimmed] = struct{}{}
		}
	}
	if len(keySet) == 0 {
		return flags
	}
	filtered := make([]FeatureFlagDefinition, 0)
	for _, flag := range flags {
		if _, ok := keySet[flag.Key]; ok {
			filtered = append(filtered, flag)
		}
	}
	return filtered
}

func findFlagByKey(flags []FeatureFlagDefinition, key string) (FeatureFlagDefinition, bool) {
	for _, flag := range flags {
		if flag.Key == key {
			return flag, true
		}
	}
	return FeatureFlagDefinition{}, false
}

func projectByID(projects []Project, id string) Project {
	for _, project := range projects {
		if project.ID == id {
			return project
		}
	}
	return Project{ID: id, Name: id}
}

func projectsWithout(projects []Project, id string) []Project {
	result := make([]Project, 0, len(projects))
	for _, project := range projects {
		if project.ID != id {
			result = append(result, project)
		}
	}
	return result
}

func projectListContains(projects []Project, id string) bool {
	for _, project := range projects {
		if project.ID == id {
			return true
		}
	}
	return false
}

func sortFlagsNewestFirst(flags []FeatureFlagDefinition) {
	slices.SortFunc(flags, compareFlagNewestFirst)
}

func compareFlagNewestFirst(a, b FeatureFlagDefinition) int {
	switch {
	case a.CreatedAt != nil && b.CreatedAt != nil:
		if a.CreatedAt.Equal(*b.CreatedAt) {
			return strings.Compare(strings.ToLower(a.Key), strings.ToLower(b.Key))
		}
		if a.CreatedAt.After(*b.CreatedAt) {
			return -1
		}
		return 1
	case a.CreatedAt != nil:
		return -1
	case b.CreatedAt != nil:
		return 1
	default:
		return strings.Compare(strings.ToLower(a.Key), strings.ToLower(b.Key))
	}
}

func flagEnabledAnywhere(flag FeatureFlagDefinition) bool {
	for _, target := range flag.Targets {
		if target.IsEnabled {
			return true
		}
	}
	return false
}

func missingVariableDefinitions(existing map[string]VariableDefinitionSpec, desired map[string]VariableDefinitionSpec) map[string]VariableDefinitionSpec {
	missing := make(map[string]VariableDefinitionSpec)
	for key, def := range desired {
		if _, ok := existing[key]; !ok {
			missing[key] = def
		}
	}
	return missing
}

func variableDriftSignature(value VariableDriftValue) string {
	return fmt.Sprintf("%t|%s|%s|%s", value.Exists, value.Type, value.DefaultValue, value.Description)
}

func stringifyVariableValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	}
}
