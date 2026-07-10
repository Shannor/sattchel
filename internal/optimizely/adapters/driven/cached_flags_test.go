package driven

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sattchel/internal/optimizely/core"
)

type mockFlagsRepository struct {
	getCalls    int
	getAllCalls int
	searchCalls int
	flags       []core.FeatureFlagDefinition
	err         error
}

func (m *mockFlagsRepository) Get(ctx context.Context, ID string) (*core.FeatureFlagDefinition, error) {
	m.getCalls++
	if m.err != nil {
		return nil, m.err
	}
	for _, f := range m.flags {
		if f.ID == ID || f.Key == ID {
			return &f, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockFlagsRepository) GetAll(ctx context.Context) ([]core.FeatureFlagDefinition, error) {
	m.getAllCalls++
	if m.err != nil {
		return nil, m.err
	}
	return m.flags, nil
}

func (m *mockFlagsRepository) Create(ctx context.Context, value core.FeatureFlagDefinition) (*core.FeatureFlagDefinition, error) {
	return nil, nil
}

func (m *mockFlagsRepository) Update(ctx context.Context, updater func(value *core.FeatureFlagDefinition) error) (*core.FeatureFlagDefinition, error) {
	return nil, nil
}

func (m *mockFlagsRepository) Delete(ctx context.Context, ID string) (string, error) {
	return "", nil
}

func TestCachedFlagsRepository(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sattchel-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cachePath := filepath.Join(tempDir, "optimizely_cache.json")
	projectID := "12345"
	ttl := 100 * time.Millisecond

	mockFlags := []core.FeatureFlagDefinition{
		{ID: "flag1", Key: "flag-1", Name: "Flag One", Description: "This is flag one"},
		{ID: "flag2", Key: "flag-2", Name: "Flag Two", Description: "This is flag two"},
	}

	mockRepo := &mockFlagsRepository{flags: mockFlags}
	cachedRepo := NewCachedFlagsRepository(mockRepo, cachePath, projectID, ttl)

	// 1. Cold Cache: GetAll should fetch from mock repo and save cache
	flags, err := cachedRepo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
	if mockRepo.getAllCalls != 1 {
		t.Errorf("expected 1 call to mock repo, got %d", mockRepo.getAllCalls)
	}

	// 2. Warm Cache: GetAll should read from file, not calling mock repo
	flags, err = cachedRepo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
	if mockRepo.getAllCalls != 1 {
		t.Errorf("expected still 1 call to mock repo, got %d", mockRepo.getAllCalls)
	}

	// 3. Bypass Cache: passing BypassCacheKey context should bypass cache and refresh
	ctxClear := context.WithValue(context.Background(), BypassCacheKey, true)
	flags, err = cachedRepo.GetAll(ctxClear)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockRepo.getAllCalls != 2 {
		t.Errorf("expected 2 calls to mock repo after force clear, got %d", mockRepo.getAllCalls)
	}

	// 4. Expired Cache: sleep until TTL expires, next call should query API
	time.Sleep(150 * time.Millisecond)
	flags, err = cachedRepo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mockRepo.getAllCalls != 3 {
		t.Errorf("expected 3 calls to mock repo after expiration, got %d", mockRepo.getAllCalls)
	}
}

func TestCachedFlagsRepository_Get(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sattchel-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cachePath := filepath.Join(tempDir, "optimizely_cache.json")
	projectID := "12345"
	ttl := 10 * time.Second

	mockFlags := []core.FeatureFlagDefinition{
		{ID: "flag1", Key: "flag-1", Name: "Flag One", Description: "This is flag one"},
	}

	mockRepo := &mockFlagsRepository{flags: mockFlags}
	cachedRepo := NewCachedFlagsRepository(mockRepo, cachePath, projectID, ttl)

	// Call GetAll to populate cache
	_, _ = cachedRepo.GetAll(context.Background())
	mockRepo.getAllCalls = 0

	// 1. Warm Cache hit on Get
	f, err := cachedRepo.Get(context.Background(), "flag1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID != "flag1" {
		t.Errorf("expected flag1, got %s", f.ID)
	}
	if mockRepo.getCalls != 0 {
		t.Errorf("expected 0 mock Get calls, got %d", mockRepo.getCalls)
	}

	// 2. Cache miss, but valid cache: calls Get on mock repo, and updates/saves cache
	mockRepo.flags = append(mockRepo.flags, core.FeatureFlagDefinition{ID: "flag2", Key: "flag-2", Name: "Flag Two"})
	f, err = cachedRepo.Get(context.Background(), "flag2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID != "flag2" {
		t.Errorf("expected flag2, got %s", f.ID)
	}
	if mockRepo.getCalls != 1 {
		t.Errorf("expected 1 mock Get call, got %d", mockRepo.getCalls)
	}

	// 3. Confirm flag2 is now in the cache file (so another Get is a cache hit)
	f, err = cachedRepo.Get(context.Background(), "flag-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ID != "flag2" {
		t.Errorf("expected flag2, got %s", f.ID)
	}
	if mockRepo.getCalls != 1 {
		t.Errorf("expected still 1 mock Get call (cache hit), got %d", mockRepo.getCalls)
	}
}

func TestJSONSerialization(t *testing.T) {
	// Test that targets and overrides serialize/deserialize correctly with the custom json marshalers
	def := core.FeatureFlagDefinition{
		ID:   "flag1",
		Key:  "flag-1",
		Name: "Flag One",
		Targets: []core.Target{
			{EnvironmentID: "production", IsEnabled: true, OverrideID: "var1"},
		},
		Overrides: []core.Override{
			{ID: "var1", Key: "var-1", Name: "Override One"},
		},
	}

	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed core.FeatureFlagDefinition
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != "flag1" {
		t.Errorf("expected flag1, got %s", parsed.ID)
	}

	instances, err := parsed.AllInstances()
	if err != nil {
		t.Fatalf("failed to get instances: %v", err)
	}

	// Check if target and override were restored correctly
	foundProd := false
	for _, inst := range instances {
		if inst.EnvironmentID == "production" {
			foundProd = true
			if !inst.Enabled {
				t.Error("expected instance to be enabled")
			}
		}
	}
	if !foundProd {
		t.Error("expected to find production target instance")
	}
}
