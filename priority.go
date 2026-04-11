// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import "context"

// Priority represents the importance of a request.
type Priority int

const (
	// PriorityLow is for non-critical, background, or asynchronous tasks.
	PriorityLow Priority = iota
	// PriorityStandard is the default priority for most requests.
	PriorityStandard
	// PriorityCritical is for high-priority, user-facing, or essential tasks.
	PriorityCritical
)

type priorityKey struct{}

// WithPriority attaches a Priority to the context.
func WithPriority(ctx context.Context, p Priority) context.Context {
	return context.WithValue(ctx, priorityKey{}, p)
}

// GetPriority retrieves the Priority from the context.
// Returns PriorityStandard if no priority is found.
func GetPriority(ctx context.Context) Priority {
	if p, ok := ctx.Value(priorityKey{}).(Priority); ok {
		return p
	}
	return PriorityStandard
}
