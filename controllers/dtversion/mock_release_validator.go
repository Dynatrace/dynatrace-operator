package dtversion

import "github.com/stretchr/testify/mock"

type MockReleaseValidator struct {
	mock.Mock
}

func (o *MockReleaseValidator) IsLatest() (bool, error) {
	args := o.Called()
	return args.Bool(0), args.Error(1)
}
