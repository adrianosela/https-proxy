package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// getHandler returns the http handler for the proxy server.
func getHandler(logger *zap.Logger, remoteDialTimeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			logger.Debug(
				"received non HTTP CONNECT request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("client_addr", r.RemoteAddr),
			)
			http.Error(w, fmt.Sprintf("only %s is allowed, but got %s", http.MethodConnect, r.Method), http.StatusMethodNotAllowed)
			return
		}

		logger.Debug(
			"received new HTTP CONNECT request",
			zap.String("client_addr", r.RemoteAddr),
			zap.String("target_addr", r.Host),
		)
		tunnelToDestination(logger, remoteDialTimeout, r.Host, w)
	})
}

// tunnelToDestination connects to a remote server and sets up a full
// duplex connection between the client and the remote server.
func tunnelToDestination(
	logger *zap.Logger,
	dialTimeout time.Duration,
	targetAddress string,
	w http.ResponseWriter,
) {
	toTarget, err := net.DialTimeout("tcp", targetAddress, dialTimeout)
	if err != nil {
		logger.Error("failed to dial tcp to target server", zap.String("target_address", targetAddress), zap.Error(err))
		http.Error(w, "an unknown error occurred... try again later", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		logger.Error("failed to type-assert http.ResponseWriter as http.Hijacker", zap.String("target_address", targetAddress))
		http.Error(w, "an unknown error occurred... try again later", http.StatusInternalServerError)
		return
	}

	toClient, _, err := hijacker.Hijack()
	if err != nil {
		logger.Error("failed to hijack connection to client", zap.Error(err))
		http.Error(w, "an unknown error occurred... try again later", http.StatusServiceUnavailable)
		return
	}

	if err = iocopy(toClient, toTarget); err != nil {
		logger.Error("failed to forward bytes between client and target server", zap.Error(err))
		http.Error(w, "an unknown error occurred... try again later", http.StatusServiceUnavailable)
		return
	}
}

func iocopy(src, dst net.Conn) error {
	defer src.Close()
	defer dst.Close()

	errC := make(chan error, 2)
	defer close(errC)

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// abort operations on dst when src has an error (incl. EOF)
		defer dst.SetDeadline(time.Now().Add(-1 * time.Hour))

		_, err := io.Copy(dst, src)
		errC <- err
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// abort operations on src when dst has an error (incl. EOF)
		defer src.SetDeadline(time.Now().Add(-1 * time.Hour))

		_, err := io.Copy(src, dst)
		errC <- err
	}()

	// returns the first error encountered
	return <-errC
}
