// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"fmt"
	"time"
)

// Header defines an interface for injecting metadata into outgoing requests.
// This allows Resile to remain agnostic of the underlying transport (HTTP, gRPC, etc.).
type Header interface {
	Set(key, value string)
}

// InjectDeadlineHeader calculates the remaining time in the context and injects
// it into the provided header. This is useful for distributed deadline propagation.
// If the key is "Grpc-Timeout", it uses the gRPC-specific timeout format.
// Otherwise, it injects the remaining milliseconds as a string.
func InjectDeadlineHeader(ctx context.Context, header Header, key string) {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return
		}

		if key == "Grpc-Timeout" {
			header.Set(key, formatGRPCTimeout(remaining))
		} else {
			header.Set(key, fmt.Sprintf("%d", remaining.Milliseconds()))
		}
	}
}

// formatGRPCTimeout formats a duration according to the gRPC timeout specification.
// See: https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests
func formatGRPCTimeout(d time.Duration) string {
	if d <= 0 {
		return "0n"
	}

	if d < time.Microsecond {
		return fmt.Sprintf("%dn", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%du", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dm", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%dS", int64(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dM", int64(d.Minutes()))
	}
	return fmt.Sprintf("%dH", int64(d.Hours()))
}
