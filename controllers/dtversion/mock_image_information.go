package dtversion

import "github.com/stretchr/testify/mock"

type MockImageInformation struct {
	mock.Mock
}

func (o *MockImageInformation) GetVersionLabel() (string, error) {
	args := o.Called()
	return args.Get(0).(string), args.Error(1)
}