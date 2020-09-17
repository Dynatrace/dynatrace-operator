package version

type ReleaseValidator interface {
	IsLatest() (bool, error)
}
