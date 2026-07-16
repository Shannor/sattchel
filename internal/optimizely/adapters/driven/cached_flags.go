package driven

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sattchel/internal/optimizely/core"

	"charm.land/log/v2"
)

type ContextKey string

const BypassCacheKey ContextKey = "bypass_cache"

type cacheData struct {
	UpdatedAt time.Time                    `json:"updated_at"`
	Flags     []core.FeatureFlagDefinition `json:"flags"`
}

type cachedFlagsRepository struct {
	underlying core.FlagsRepository
	cachePath  string
	projectID  string
	ttl        time.Duration
	mu         sync.RWMutex
}

func NewCachedFlagsRepository(underlying core.FlagsRepository, cachePath string, projectID string, ttl time.Duration) core.FlagsRepository {
	return &cachedFlagsRepository{
		underlying: underlying,
		cachePath:  cachePath,
		projectID:  projectID,
		ttl:        ttl,
	}
}

func (r *cachedFlagsRepository) getProjectCachePath() string {
	ext := filepath.Ext(r.cachePath)
	base := strings.TrimSuffix(r.cachePath, ext)
	return fmt.Sprintf("%s_%s%s", base, r.projectID, ext)
}

func (r *cachedFlagsRepository) loadCache(ctx context.Context) ([]core.FeatureFlagDefinition, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bypassCacheVal, _ := ctx.Value(BypassCacheKey).(bool)
	if bypassCacheVal {
		return nil, false, nil
	}

	filePath := r.getProjectCachePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var cache cacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false, err
	}

	if time.Since(cache.UpdatedAt) > r.ttl {
		return nil, false, nil
	}

	return cache.Flags, true, nil
}

func (r *cachedFlagsRepository) saveCache(ctx context.Context, flags []core.FeatureFlagDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filePath := r.getProjectCachePath()
	cache := cacheData{
		UpdatedAt: time.Now(),
		Flags:     flags,
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (r *cachedFlagsRepository) updateCachedFlag(ctx context.Context, flags []core.FeatureFlagDefinition, updatedFlag core.FeatureFlagDefinition) {
	found := false
	for i, f := range flags {
		if f.ID == updatedFlag.ID || f.Key == updatedFlag.Key {
			flags[i] = updatedFlag
			found = true
			break
		}
	}
	if !found {
		flags = append(flags, updatedFlag)
	}
	if err := r.saveCache(ctx, flags); err != nil {
		log.Warn("failed to update cached flag", "projectID", r.projectID, "error", err.Error())
	}
}

func (r *cachedFlagsRepository) Get(ctx context.Context, ID string) (*core.FeatureFlagDefinition, error) {
	flags, ok, err := r.loadCache(ctx)
	if err == nil && ok {
		for _, f := range flags {
			if f.ID == ID || f.Key == ID {
				return &f, nil
			}
		}
	}

	flag, err := r.underlying.Get(ctx, ID)
	if err != nil {
		return nil, err
	}

	if ok {
		r.updateCachedFlag(ctx, flags, *flag)
	}

	return flag, nil
}

func (r *cachedFlagsRepository) GetAll(ctx context.Context) ([]core.FeatureFlagDefinition, error) {
	flags, ok, err := r.loadCache(ctx)
	if err != nil {
		log.Warn("failed to load cache", "projectID", r.projectID, "error", err.Error())
	}
	if ok {
		return flags, nil
	}

	freshFlags, err := r.underlying.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.saveCache(ctx, freshFlags); err != nil {
		log.Warn("failed to save cache", "projectID", r.projectID, "error", err.Error())
	}

	return freshFlags, nil
}

func (r *cachedFlagsRepository) Create(ctx context.Context, value core.FeatureFlagDefinition) (*core.FeatureFlagDefinition, error) {
	return r.underlying.Create(ctx, value)
}

func (r *cachedFlagsRepository) Update(ctx context.Context, updater func(value *core.FeatureFlagDefinition) error) (*core.FeatureFlagDefinition, error) {
	return r.underlying.Update(ctx, updater)
}

func (r *cachedFlagsRepository) Delete(ctx context.Context, ID string) (string, error) {
	return r.underlying.Delete(ctx, ID)
}

type cachedFlagsFactory struct {
	underlying core.FlagsRepositoryFactory
	cachePath  string
	ttl        time.Duration
}

func NewCachedFlagsFactory(underlying core.FlagsRepositoryFactory, cachePath string, ttl time.Duration) core.FlagsRepositoryFactory {
	return &cachedFlagsFactory{
		underlying: underlying,
		cachePath:  cachePath,
		ttl:        ttl,
	}
}

func (f *cachedFlagsFactory) Create(ctx context.Context, projectID string) (core.FlagsRepository, error) {
	underlyingRepo, err := f.underlying.Create(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return NewCachedFlagsRepository(underlyingRepo, f.cachePath, projectID, f.ttl), nil
}
