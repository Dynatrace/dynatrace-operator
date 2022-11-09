package graceful

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

type ShutdownManager struct {
	serverCtx          context.Context
	serverShutdownFunc context.CancelFunc
	livezCtx           context.Context
	livezShutdownFunc  context.CancelFunc
	currentlyRunning   int
	mutex              sync.Mutex
}

func NewShutdownManager() *ShutdownManager {
	serverCtx, serverCancel := context.WithCancel(context.Background())
	livezCtx, livezCancel := context.WithCancel(serverCtx)
	return &ShutdownManager{
		serverCtx:          serverCtx,
		serverShutdownFunc: serverCancel,
		livezCtx:           livezCtx,
		livezShutdownFunc:  livezCancel,
		mutex:              sync.Mutex{},
	}
}

func (sm *ShutdownManager) Start() {
	sm.SetCurrentlyRunning(0)
	sm.setupSignalHandler()
}

func (sm *ShutdownManager) GetServerContext() context.Context {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.serverCtx
}

func (sm *ShutdownManager) GetLivezContext() context.Context {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.livezCtx
}

func (sm *ShutdownManager) SetCurrentlyRunning(value int) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.currentlyRunning = 0
}

func (sm *ShutdownManager) IncCurrentlyRunning() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.currentlyRunning += 1
}

func (sm *ShutdownManager) DecCurrentlyRunning() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.currentlyRunning -= 1
}

func (sm *ShutdownManager) GetCurrentlyRunning() int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.currentlyRunning
}

func (sm *ShutdownManager) setupSignalHandler() {
	signal.Reset(shutdownSignals...)
	shutdownChannel := make(chan os.Signal, 2)
	signal.Notify(shutdownChannel, shutdownSignals...)

	go func() {
		<-shutdownChannel
		log.Info("graceful shutdown started")
		sm.livezShutdownFunc()
		log.Info("livez shutdown started")
		ctx, cancel := context.WithCancel(context.Background())
		log.Info("safe webhook server shutdown started")
		go sm.gracefulPeriodShutdown(ctx, cancel)
		go sm.noRequestsShutdown(ctx, cancel)
		<-shutdownChannel
		os.Exit(1)
	}()
}

func (sm *ShutdownManager) gracefulPeriodShutdown(ctx context.Context, cancel context.CancelFunc) {
	timer := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-timer.C:
			log.Info("shutting down after grace timer ended")
			cancel()
			sm.shutdown()
			return
		case <-ctx.Done():
			return
		}
	}
}

func (sm *ShutdownManager) noRequestsShutdown(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			currentlyRunning := sm.GetCurrentlyRunning()
			if currentlyRunning <= 0 {
				log.Info("shutting down after all requests finished")
				cancel()
				sm.shutdown()
				return
			}
			log.Info("requests are still being handled", "amount", currentlyRunning)
		case <-ctx.Done():
			return
		}
	}
}

func (sm *ShutdownManager) shutdown() {
	if sm.serverShutdownFunc == nil {
		return
	}
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.serverShutdownFunc()
}
