package lifecycle

import (
	"context"
)

// Executor is a thing that can be executed. Rob is still wrong.
type Executor interface {
	// Execute runs the step
	Execute(ctx context.Context, lifecycle *Runner) error
	// String is.. it makes strings
	String() string
}
