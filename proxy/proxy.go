package proxy

import (
	"crypto/tls"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	defaultRemoteDialTimeout = time.Second * 10
)

type Proxy struct {
	logger *zap.Logger
	server *http.Server
}

func New(logger *zap.Logger, addr string, cert tls.Certificate) *Proxy {
	return &Proxy{
		logger: logger,
		server: &http.Server{
			Addr:    addr,
			Handler: getHandler(logger, defaultRemoteDialTimeout),
			TLSConfig: &tls.Config{
				// Must force HTTP/1.1 to be used since HTTP/2.0
				// does not support the CONNECT HTTP method.
				NextProtos:   []string{"http/1.1"},
				Certificates: []tls.Certificate{cert},
			},
		},
	}
}

func (p *Proxy) ListenAndServeTLS() error {
	p.logger.Info("HTTPS proxy listening for requests", zap.String("address", p.server.Addr))
	return p.server.ListenAndServeTLS("", "")
}
