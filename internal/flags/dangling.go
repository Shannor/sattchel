package flags

import (
	"context"
	"test-cli/internal/models"
)

// Dangling will return if a feature flag can be safely deleted.
func Dangling(ctx context.Context, flag models.FeatureFlag) bool {
	environmentSet := make(map[string]bool)
	for _, environment := range flag.Environments {
		if environment.Enabled {

		}
	}
}
