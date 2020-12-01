package dtversion

import "github.com/stretchr/testify/mock"

type MockImageInformation struct {
	mock.Mock
}

func (o *MockImageInformation) GetVersionLabel() (string, error) {
	args := o.Called()
	return args.String(0), args.Error(1)
}
