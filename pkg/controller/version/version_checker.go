package version

type VersionChecker interface {
	IsLatest() (bool, error)
}
