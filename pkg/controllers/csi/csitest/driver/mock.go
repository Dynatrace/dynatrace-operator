package driver

import (
	"net"

	mocks "github.com/Dynatrace/dynatrace-operator/test/mocks/github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/container-storage-interface/spec/lib/go/csi"
)

type MockCSIDriverServers struct {
	Identity *mocks.IdentityServer
}

type MockCSIDriver struct {
	CSIDriver
}

func NewMockCSIDriver(servers *MockCSIDriverServers) *MockCSIDriver {
	return &MockCSIDriver{
		CSIDriver: CSIDriver{
			servers: &CSIDriverServers{
				Identity: struct {
					csi.UnsafeIdentityServer
					*mocks.IdentityServer
				}{IdentityServer: servers.Identity},
			},
		},
	}
}

// StartOnAddress starts a new gRPC server listening on given address.
func (m *MockCSIDriver) StartOnAddress(network, address string) error {
	l, err := net.Listen(network, address)
	if err != nil {
		return err
	}

	if err := m.Start(l); err != nil {
		l.Close()

		return err
	}

	return nil
}
