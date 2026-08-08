package snapshotapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"
)

const DefaultAddress = "127.0.0.1:8080"

type ServerConfig struct {
	Address        string
	AllowedOrigins []string
}

type Server struct {
	http     *http.Server
	listener net.Listener
	done     chan error
}

func Listen(source CaptureSource, config ServerConfig) (*Server, error) {
	address := config.Address
	if address == "" {
		address = DefaultAddress
	}
	if !loopbackAddress(address) {
		return nil, errors.New("snapshot server requires an explicit loopback address")
	}
	handler, err := NewHandler(source, HandlerConfig{AllowedOrigins: config.AllowedOrigins})
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	server := &Server{listener: listener, done: make(chan error, 1)}
	server.http = &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		err := server.http.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		server.done <- err
		close(server.done)
	}()
	return server, nil
}

func (server *Server) Address() string {
	if server == nil || server.listener == nil {
		return ""
	}
	return server.listener.Addr().String()
}

func (server *Server) Done() <-chan error {
	if server == nil {
		closed := make(chan error)
		close(closed)
		return closed
	}
	return server.done
}

func (server *Server) Shutdown(ctx context.Context) error {
	if server == nil || server.http == nil || ctx == nil {
		return errors.New("invalid snapshot server shutdown")
	}
	err := server.http.Shutdown(ctx)
	if err != nil {
		_ = server.http.Close()
	}
	return err
}

func loopbackAddress(address string) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return false
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil || parsedPort > 65535 {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
