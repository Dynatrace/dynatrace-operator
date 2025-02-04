package version

import "context"

type versionStatusUpdater interface {
	Name() string
	RequiresReconcile() bool
	Update(ctx context.Context) error
}
