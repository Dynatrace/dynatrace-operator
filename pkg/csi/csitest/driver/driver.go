package driver

import (
	"net"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// https://github.com/kubernetes-csi/csi-test/blob/master/driver/driver.go
type CSIDriverServers struct {
	Identity csi.IdentityServer
}

type CSIDriver struct {
	listener net.Listener
	server   *grpc.Server
	servers  *CSIDriverServers
	wg       sync.WaitGroup
	lock     sync.Mutex
	running  bool
}

func (c *CSIDriver) goServe(started chan<- bool) {
	goServe(c.server, &c.wg, c.listener, started)
}

func (c *CSIDriver) Address() string {
	return c.listener.Addr().String()
}

func (c *CSIDriver) Start(l net.Listener) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Set listener
	c.listener = l

	// Create a new grpc server
	c.server = grpc.NewServer()

	if c.servers.Identity != nil {
		csi.RegisterIdentityServer(c.server, c.servers.Identity)
	}

	reflection.Register(c.server)

	// Start listening for requests
	waitForServer := make(chan bool)
	c.goServe(waitForServer)
	<-waitForServer

	c.running = true

	return nil
}

func (c *CSIDriver) Stop() {
	stop(&c.lock, &c.wg, c.server, c.running)
}

// goServe starts a grpc server.
func goServe(server *grpc.Server, wg *sync.WaitGroup, listener net.Listener, started chan<- bool) {
	wg.Go(func() {
		started <- true

		err := server.Serve(listener)
		if err != nil {
			panic(err)
		}
	})
}

// stop stops a grpc server.
func stop(lock *sync.Mutex, wg *sync.WaitGroup, server *grpc.Server, running bool) {
	lock.Lock()
	defer lock.Unlock()

	if !running {
		return
	}

	server.Stop()
	wg.Wait()
}
