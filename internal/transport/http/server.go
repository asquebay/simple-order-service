package http

import (
	"context"
	"net/http"
	"time"
)

// Server — это обёртка над стандартным http.Server
type Server struct {
	httpServer *http.Server
}

// NewServer создает и конфигурирует экземпляр Server
func NewServer(port string, handler http.Handler, timeout time.Duration) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         port,
			Handler:      handler,
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
		},
	}
}

// Run запускает HTTP-сервер
func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
