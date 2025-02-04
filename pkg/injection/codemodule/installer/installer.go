package installer

import "context"

type Installer interface {
	InstallAgent(ctx context.Context, targetDir string) (bool, error)
}
