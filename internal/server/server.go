package server

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/aigateway/pkg/logger"
	"go.uber.org/zap"
)

type Server struct {
	server          *http.Server
	activeRequests  sync.WaitGroup
	shutdownTimeout time.Duration
	shuttingDown    atomic.Bool
}

func New(addr string, handler http.Handler, readTimeout, writeTimeout, idleTimeout time.Duration, maxHeaderBytes int, shutdownTimeout time.Duration) *Server {
	return &Server{
		server: &http.Server{
			Addr:           addr,
			Handler:        handler,
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			IdleTimeout:    idleTimeout,
			MaxHeaderBytes: maxHeaderBytes,
		},
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *Server) Start() error {
	logger.L.Info("server starting", zap.String("addr", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.shuttingDown.Store(true)
	logger.L.Info("server shutting down")

	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.activeRequests.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.L.Info("all active requests completed")
	case <-ctx.Done():
		logger.L.Warn("shutdown timeout, forcing close")
	}

	return s.server.Shutdown(ctx)
}

func (s *Server) SetHandler(handler http.Handler) {
	s.server.Handler = handler
}

func (s *Server) IsShuttingDown() bool {
	return s.shuttingDown.Load()
}

func (s *Server) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.shuttingDown.Load() {
			w.Header().Set("Connection", "close")
			http.Error(w, "server shutting down", http.StatusServiceUnavailable)
			return
		}

		s.activeRequests.Add(1)
		defer s.activeRequests.Done()

		next.ServeHTTP(w, r)
	})
}


