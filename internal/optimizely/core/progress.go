package core

import "context"

type progressKey struct{}

// WithProgress stores a ProgressReporter in the context so it can be
// extracted by downstream layers.
func WithProgress(ctx context.Context, reporter ProgressReporter) context.Context {
	return context.WithValue(ctx, progressKey{}, reporter)
}

// ProgressFromContext retrieves the ProgressReporter from the context.
// Returns nil if no reporter was set (e.g. non-CLI usage).
func ProgressFromContext(ctx context.Context) ProgressReporter {
	if v := ctx.Value(progressKey{}); v != nil {
		return v.(ProgressReporter)
	}
	return nil
}
