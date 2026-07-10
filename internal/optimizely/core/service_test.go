package core

import (
	"context"
	"errors"
	"testing"
)

type mockProjectRepo struct {
	projects []Project
}

func (m *mockProjectRepo) GetAll(ctx context.Context) ([]Project, error)        { return m.projects, nil }
func (m *mockProjectRepo) Get(ctx context.Context, ID string) (*Project, error) { return nil, nil }
func (m *mockProjectRepo) Update(ctx context.Context, updater func(*Project) error) (*Project, error) {
	return nil, nil
}
func (m *mockProjectRepo) Delete(ctx context.Context, ID string) (string, error) { return "", nil }
func (m *mockProjectRepo) Create(ctx context.Context, value Project) (*Project, error) {
	return nil, nil
}

type mockFlagsRepo struct {
	flags []FeatureFlagDefinition
}

func (m *mockFlagsRepo) Get(ctx context.Context, ID string) (*FeatureFlagDefinition, error) {
	for _, f := range m.flags {
		if f.ID == ID || f.Key == ID {
			return &f, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockFlagsRepo) GetAll(ctx context.Context) ([]FeatureFlagDefinition, error) {
	return m.flags, nil
}

func (m *mockFlagsRepo) Create(ctx context.Context, value FeatureFlagDefinition) (*FeatureFlagDefinition, error) {
	return nil, nil
}

func (m *mockFlagsRepo) Update(ctx context.Context, updater func(*FeatureFlagDefinition) error) (*FeatureFlagDefinition, error) {
	return nil, nil
}

func (m *mockFlagsRepo) Delete(ctx context.Context, ID string) (string, error) {
	return "", nil
}

type mockFlagsRepoFactory struct {
	repos map[string]*mockFlagsRepo
}

func (m *mockFlagsRepoFactory) Create(ctx context.Context, projectID string) (FlagsRepository, error) {
	repo, ok := m.repos[projectID]
	if !ok {
		return nil, errors.New("not found")
	}
	return repo, nil
}

func TestSearchFlags(t *testing.T) {
	project1Flags := []FeatureFlagDefinition{
		{ID: "flag1", Key: "flag-1", Name: "Alpha Flag", Description: "First flag"},
		{ID: "flag2", Key: "flag-2", Name: "Beta Flag", Description: "Second flag"},
	}
	project2Flags := []FeatureFlagDefinition{
		{ID: "flag3", Key: "flag-3", Name: "Gamma Flag", Description: "Third flag"},
	}

	factory := &mockFlagsRepoFactory{
		repos: map[string]*mockFlagsRepo{
			"p1": {flags: project1Flags},
			"p2": {flags: project2Flags},
		},
	}

	service := NewService(nil, factory, nil)

	// 1. Search with empty query should return all
	result, err := service.SearchFlags(context.Background(), []string{"p1", "p2"}, ListFlagsOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result["p1"]) != 2 || len(result["p2"]) != 1 {
		t.Errorf("expected p1 to have 2 and p2 to have 1, got p1=%d p2=%d", len(result["p1"]), len(result["p2"]))
	}

	// 2. Search with query matching only "Alpha"
	result, err = service.SearchFlags(context.Background(), []string{"p1", "p2"}, ListFlagsOptions{Query: "Alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result["p1"]) != 1 || result["p1"][0].ID != "flag1" {
		t.Errorf("expected flag1 in p1, got: %v", result["p1"])
	}
	if len(result["p2"]) != 0 {
		t.Errorf("expected 0 flags in p2, got: %v", result["p2"])
	}

	// 3. Search case-insensitive substring
	result, err = service.SearchFlags(context.Background(), []string{"p1", "p2"}, ListFlagsOptions{Query: "ta"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result["p1"]) != 1 || result["p1"][0].ID != "flag2" {
		t.Errorf("expected flag2 in p1, got: %v", result["p1"])
	}
}

func TestCompareFlags(t *testing.T) {
	project1Flags := []FeatureFlagDefinition{
		{ID: "flag1", Key: "flag-1", Name: "Alpha Flag"},
		{ID: "flag2", Key: "flag-2", Name: "Beta Flag"},
	}
	project2Flags := []FeatureFlagDefinition{
		{ID: "flag2", Key: "flag-2", Name: "Beta Flag"},
		{ID: "flag3", Key: "flag-3", Name: "Gamma Flag"},
	}

	factory := &mockFlagsRepoFactory{
		repos: map[string]*mockFlagsRepo{
			"p1": {flags: project1Flags},
			"p2": {flags: project2Flags},
		},
	}

	projRepo := &mockProjectRepo{
		projects: []Project{
			{ID: "p1", Name: "Project One"},
			{ID: "p2", Name: "Project Two"},
		},
	}

	service := NewService(projRepo, factory, nil)

	// 1. Error on fewer than 2 project IDs
	_, err := service.CompareFlags(context.Background(), []string{"p1"})
	if err == nil {
		t.Error("expected error for less than 2 projects, got nil")
	}

	// 2. Perform comparison between p1 and p2
	comparisons, err := service.CompareFlags(context.Background(), []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// flag-1 only in p1 (missing in p2)
	// flag-3 only in p2 (missing in p1)
	// flag-2 is in both, so it shouldn't be in comparisons
	if len(comparisons) != 2 {
		t.Fatalf("expected 2 comparison results, got %d", len(comparisons))
	}

	// Check flag-1
	if comparisons[0].Key != "flag-1" {
		t.Errorf("expected first mismatch to be flag-1, got %s", comparisons[0].Key)
	}
	if len(comparisons[0].ExistsIn) != 1 || comparisons[0].ExistsIn[0].ID != "p1" || comparisons[0].ExistsIn[0].Name != "Project One" {
		t.Errorf("expected flag-1 to exist in p1, got %v", comparisons[0].ExistsIn)
	}
	if len(comparisons[0].MissingIn) != 1 || comparisons[0].MissingIn[0].ID != "p2" || comparisons[0].MissingIn[0].Name != "Project Two" {
		t.Errorf("expected flag-1 to be missing in p2, got %v", comparisons[0].MissingIn)
	}

	// Check flag-3
	if comparisons[1].Key != "flag-3" {
		t.Errorf("expected second mismatch to be flag-3, got %s", comparisons[1].Key)
	}
	if len(comparisons[1].ExistsIn) != 1 || comparisons[1].ExistsIn[0].ID != "p2" || comparisons[1].ExistsIn[0].Name != "Project Two" {
		t.Errorf("expected flag-3 to exist in p2, got %v", comparisons[1].ExistsIn)
	}
	if len(comparisons[1].MissingIn) != 1 || comparisons[1].MissingIn[0].ID != "p1" || comparisons[1].MissingIn[0].Name != "Project One" {
		t.Errorf("expected flag-3 to be missing in p1, got %v", comparisons[1].MissingIn)
	}
}
