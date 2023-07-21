package installer

type Installer interface {
	InstallAgent(targetDir string) (bool, error)
	Cleanup() error
}
