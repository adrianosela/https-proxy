package main

import (
	"crypto/tls"
	"log"
	"path/filepath"

	"github.com/adrianosela/https-proxy/proxy"
	"go.uber.org/zap"
)

const listenAddr = "localhost:8443"

var (
	certPath = filepath.Join(".", "_test_certs_", "kubernetes-https-proxy-test.com.crt")
	keyPath  = filepath.Join(".", "_test_certs_", "kubernetes-https-proxy-test.com.key")
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		logger.Fatal("failed to load x509 key pair", zap.Error(err))
	}

	err = proxy.New(logger, listenAddr, cert).ListenAndServeTLS()
	if err != nil {
		logger.Fatal("failed to listen and serve TLS", zap.Error(err))
	}
}
