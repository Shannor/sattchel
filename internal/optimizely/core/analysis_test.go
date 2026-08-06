package core

import (
	"context"
	"testing"
)

type trackingFlagsRepo struct {
	flags          []FeatureFlagDefinition
	created        []FeatureFlagDefinition
	addVariableOps []struct {
		flagKey string
		vars    Variables
	}
}

func (m *trackingFlagsRepo) Get(ctx context.Context, ID string) (*FeatureFlagDefinition, error) {
	for _, f := range m.flags {
		if f.ID == ID || f.Key == ID {
			return &f, nil
		}
	}
	return nil, nil
}

func (m *trackingFlagsRepo) GetAll(ctx context.Context) ([]FeatureFlagDefinition, error) {
	return m.flags, nil
}

func (m *trackingFlagsRepo) Create(ctx context.Context, value FeatureFlagDefinition) (*FeatureFlagDefinition, error) {
	m.created = append(m.created, value)
	m.flags = append(m.flags, value)
	return &value, nil
}

func (m *trackingFlagsRepo) AddVariables(ctx context.Context, flagKey string, vars Variables) error {
	m.addVariableOps = append(m.addVariableOps, struct {
		flagKey string
		vars    Variables
	}{flagKey: flagKey, vars: vars})
	for i, flag := range m.flags {
		if flag.Key == flagKey || flag.ID == flagKey {
			m.flags[i].DefaultVariables.Merge(vars)
			break
		}
	}
	return nil
}

func (m *trackingFlagsRepo) Update(ctx context.Context, updater func(*FeatureFlagDefinition) error) (*FeatureFlagDefinition, error) {
	return nil, nil
}

func (m *trackingFlagsRepo) Delete(ctx context.Context, ID string) (string, error) {
	return ID, nil
}

type trackingFlagsRepoFactory struct {
	repos map[string]*trackingFlagsRepo
}

func (m *trackingFlagsRepoFactory) Create(ctx context.Context, projectID string) (FlagsRepository, error) {
	return m.repos[projectID], nil
}

func TestCompareFlagsDetailed_BaseAndFocusModes(t *testing.T) {
	factory := &mockFlagsRepoFactory{repos: map[string]*mockFlagsRepo{
		"p1": {flags: []FeatureFlagDefinition{{ID: "1", Key: "alpha", Name: "Alpha"}, {ID: "2", Key: "shared", Name: "Shared"}}},
		"p2": {flags: []FeatureFlagDefinition{{ID: "2", Key: "shared", Name: "Shared"}, {ID: "3", Key: "charlie", Name: "Charlie"}}},
		"p3": {flags: []FeatureFlagDefinition{{ID: "2", Key: "shared", Name: "Shared"}, {ID: "3", Key: "charlie", Name: "Charlie"}}},
	}}
	service := NewService(&mockProjectRepo{projects: []Project{{ID: "p1", Name: "One"}, {ID: "p2", Name: "Two"}, {ID: "p3", Name: "Three"}}}, factory, nil)

	baseReport, err := service.CompareFlagsDetailed(context.Background(), []string{"p2", "p3"}, CompareFlagsOptions{BaseProjectID: "p1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(baseReport.MissingFlags) != 1 || baseReport.MissingFlags[0].Flag.Key != "charlie" {
		t.Fatalf("expected only charlie to be missing from base project, got %+v", baseReport.MissingFlags)
	}

	focusReport, err := service.CompareFlagsDetailed(context.Background(), []string{"p1", "p2", "p3"}, CompareFlagsOptions{FocusProjectID: "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(focusReport.MissingFlags) != 1 || focusReport.MissingFlags[0].Flag.Key != "charlie" {
		t.Fatalf("expected only charlie to be owned by focus project and missing elsewhere, got %+v", focusReport.MissingFlags)
	}
	if len(focusReport.SharedFlags) != 1 || focusReport.SharedFlags[0].Key != "shared" {
		t.Fatalf("expected shared flag to remain in shared set, got %+v", focusReport.SharedFlags)
	}
}

func TestFindDormantFlags(t *testing.T) {
	factory := &mockFlagsRepoFactory{repos: map[string]*mockFlagsRepo{
		"p1": {flags: []FeatureFlagDefinition{
			{ID: "1", Key: "dormant", Targets: []Target{{EnvironmentID: "qa", IsEnabled: false}, {EnvironmentID: "production", IsEnabled: false}}},
			{ID: "2", Key: "active", Targets: []Target{{EnvironmentID: "qa", IsEnabled: true}}},
		}},
		"p2": {flags: []FeatureFlagDefinition{
			{ID: "1", Key: "dormant", Targets: []Target{{EnvironmentID: "production", IsEnabled: false}}},
			{ID: "2", Key: "active", Targets: []Target{{EnvironmentID: "production", IsEnabled: true}}},
		}},
	}}
	service := NewService(&mockProjectRepo{projects: []Project{{ID: "p1", Name: "One"}, {ID: "p2", Name: "Two"}}}, factory, nil)

	flags, err := service.FindDormantFlags(context.Background(), []string{"p1", "p2"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 1 || flags[0].Flag.Key != "dormant" {
		t.Fatalf("expected dormant flag only, got %+v", flags)
	}
}

func TestFindVariableDrift(t *testing.T) {
	factory := &mockFlagsRepoFactory{repos: map[string]*mockFlagsRepo{
		"p1": {flags: []FeatureFlagDefinition{{ID: "1", Key: "checkout", Name: "Checkout", DefaultVariables: VariablesFromDefinitions(map[string]VariableDefinitionSpec{
			"color":   {Key: "color", Type: "string", DefaultValue: "blue"},
			"enabled": {Key: "enabled", Type: "boolean", DefaultValue: "true"},
		})}}},
		"p2": {flags: []FeatureFlagDefinition{{ID: "1", Key: "checkout", Name: "Checkout", DefaultVariables: VariablesFromDefinitions(map[string]VariableDefinitionSpec{
			"color":   {Key: "color", Type: "string", DefaultValue: "green"},
			"enabled": {Key: "enabled", Type: "boolean", DefaultValue: "true"},
			"mode":    {Key: "mode", Type: "string", DefaultValue: "fast"},
		})}}},
	}}
	service := NewService(&mockProjectRepo{projects: []Project{{ID: "p1", Name: "One"}, {ID: "p2", Name: "Two"}}}, factory, nil)

	drift, err := service.FindVariableDrift(context.Background(), []string{"p1", "p2"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drift) != 1 {
		t.Fatalf("expected 1 drifted flag, got %d", len(drift))
	}
	if len(drift[0].Variables) != 1 {
		t.Fatalf("expected 1 drifted variable (missing in p1), got %+v", drift[0].Variables)
	}
	if drift[0].Variables[0].Key != "mode" {
		t.Fatalf("expected drifted variable key to be 'mode', got %q", drift[0].Variables[0].Key)
	}
}

func TestFindPromotionCandidates(t *testing.T) {
	factory := &mockFlagsRepoFactory{repos: map[string]*mockFlagsRepo{
		"p1": {flags: []FeatureFlagDefinition{
			{ID: "1", Key: "behind", Targets: []Target{{EnvironmentID: "qa", IsEnabled: true}}},
			{ID: "2", Key: "shared-lower", Targets: []Target{{EnvironmentID: "qa", IsEnabled: true}}},
			{ID: "3", Key: "brand-only", Targets: []Target{{EnvironmentID: "qa", IsEnabled: true}}},
		}},
		"p2": {flags: []FeatureFlagDefinition{
			{ID: "1", Key: "behind", Targets: []Target{{EnvironmentID: "production", IsEnabled: true}}},
			{ID: "2", Key: "shared-lower", Targets: []Target{{EnvironmentID: "development", IsEnabled: true}}},
		}},
	}}
	service := NewService(&mockProjectRepo{projects: []Project{{ID: "p1", Name: "One"}, {ID: "p2", Name: "Two"}}}, factory, nil)

	candidates, err := service.FindPromotionCandidates(context.Background(), "p1", []string{"p2"}, PromotionOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(candidates))
	}
	if candidates[0].Reason == "" || candidates[1].Reason == "" || candidates[2].Reason == "" {
		t.Fatalf("expected reasons to be populated, got %+v", candidates)
	}

	reasons := map[string]PromotionReason{}
	for _, candidate := range candidates {
		reasons[candidate.Flag.Key] = candidate.Reason
	}
	if reasons["behind"] != PromotionReasonBrandBehind {
		t.Fatalf("expected behind flag reason %q, got %q", PromotionReasonBrandBehind, reasons["behind"])
	}
	if reasons["shared-lower"] != PromotionReasonSharedLowerEnv {
		t.Fatalf("expected shared-lower flag reason %q, got %q", PromotionReasonSharedLowerEnv, reasons["shared-lower"])
	}
	if reasons["brand-only"] != PromotionReasonBrandOnlyLowerEnv {
		t.Fatalf("expected brand-only flag reason %q, got %q", PromotionReasonBrandOnlyLowerEnv, reasons["brand-only"])
	}
}

func TestPlanAndApplyFlagSync(t *testing.T) {
	sourceFlag := FeatureFlagDefinition{ID: "1", Key: "alpha", Name: "Alpha", DefaultVariables: VariablesFromDefinitions(map[string]VariableDefinitionSpec{
		"one": {Key: "one", Type: "string", DefaultValue: "1"},
		"two": {Key: "two", Type: "boolean", DefaultValue: "true"},
	})}
	targetFlag := FeatureFlagDefinition{ID: "1", Key: "alpha", Name: "Alpha", DefaultVariables: VariablesFromDefinitions(map[string]VariableDefinitionSpec{
		"one": {Key: "one", Type: "string", DefaultValue: "1"},
	})}
	missingFlag := FeatureFlagDefinition{ID: "2", Key: "beta", Name: "Beta"}

	factory := &trackingFlagsRepoFactory{repos: map[string]*trackingFlagsRepo{
		"source": {flags: []FeatureFlagDefinition{sourceFlag, missingFlag}},
		"target": {flags: []FeatureFlagDefinition{targetFlag}},
	}}
	service := NewService(&mockProjectRepo{projects: []Project{{ID: "source", Name: "Source"}, {ID: "target", Name: "Target"}}}, factory, nil)

	plan, err := service.PlanFlagSync(context.Background(), FlagSyncOptions{SourceProjectID: "source", TargetProjectIDs: []string{"target"}, UpdateVariables: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.TargetMissing) != 1 || len(plan.TargetMissing[0].Flags) != 1 || plan.TargetMissing[0].Flags[0].Key != "beta" {
		t.Fatalf("expected beta to be created, got %+v", plan.TargetMissing)
	}
	if len(plan.TargetVariableUpdates) != 1 || len(plan.TargetVariableUpdates[0].Updates) != 1 {
		t.Fatalf("expected one variable update, got %+v", plan.TargetVariableUpdates)
	}
	if _, ok := plan.TargetVariableUpdates[0].Updates[0].MissingVariables["two"]; !ok {
		t.Fatalf("expected variable 'two' to be missing, got %+v", plan.TargetVariableUpdates)
	}

	result, err := service.ApplyFlagSyncPlan(context.Background(), *plan)
	if err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}
	if result.CreatedFlags != 1 || result.AddedVariables != 1 || result.TouchedProjects != 1 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	if len(factory.repos["target"].created) != 1 || factory.repos["target"].created[0].Key != "beta" {
		t.Fatalf("expected beta to be created in target repo, got %+v", factory.repos["target"].created)
	}
	if len(factory.repos["target"].addVariableOps) != 1 || factory.repos["target"].addVariableOps[0].flagKey != "alpha" {
		t.Fatalf("expected variables to be added to alpha, got %+v", factory.repos["target"].addVariableOps)
	}
}

func TestPlanUnionFlagSync(t *testing.T) {
	factory := &mockFlagsRepoFactory{repos: map[string]*mockFlagsRepo{
		"p1": {flags: []FeatureFlagDefinition{{ID: "1", Key: "alpha", Name: "Alpha", DefaultVariables: VariablesFromDefinitions(map[string]VariableDefinitionSpec{
			"one": {Key: "one", Type: "string", DefaultValue: "1"},
		})}, {ID: "2", Key: "beta", Name: "Beta"}}},
		"p2": {flags: []FeatureFlagDefinition{{ID: "1", Key: "alpha", Name: "Alpha", DefaultVariables: VariablesFromDefinitions(map[string]VariableDefinitionSpec{
			"two": {Key: "two", Type: "boolean", DefaultValue: "true"},
		})}}},
		"p3": {flags: []FeatureFlagDefinition{}},
	}}
	service := NewService(&mockProjectRepo{projects: []Project{{ID: "p1", Name: "One"}, {ID: "p2", Name: "Two"}, {ID: "p3", Name: "Three"}}}, factory, nil)

	plan, err := service.PlanFlagSync(context.Background(), FlagSyncOptions{UnionSource: true, TargetProjectIDs: []string{"p1", "p2", "p3"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.TargetMissing) != 2 {
		t.Fatalf("expected 2 projects with missing flags, got %+v", plan.TargetMissing)
	}
	if len(plan.TargetVariableUpdates) != 2 {
		t.Fatalf("expected p1 and p2 to each need alpha variable updates, got %+v", plan.TargetVariableUpdates)
	}
}
