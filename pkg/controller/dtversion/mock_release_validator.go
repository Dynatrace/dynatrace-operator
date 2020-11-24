package dtversion

import "github.com/stretchr/testify/mock"

type MockReleaseValidator struct {
	mock.Mock
}

func (o *MockReleaseValidator) IsLatest() (bool, error) {
	args := o.Called()
	return args.Get(0).(bool), args.Error(1)
}
