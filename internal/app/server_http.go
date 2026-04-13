package app

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

// httpSrv реализует Server для HTTP/HTTPS.
type httpSrv struct {
	addr    string
	handler http.Handler
	srv     *http.Server
}

func newHTTPServer(addr string, handler http.Handler) *httpSrv {
	return &httpSrv{addr: addr, handler: handler}
}

func (h *httpSrv) Name() string { return "HTTP" }

// Prepare инициализирует http.Server и открывает listener.
// Если tlsCfg != nil — listener открывается с TLS.
func (h *httpSrv) Prepare(tlsCfg *tls.Config) (net.Listener, error) {
	h.srv = &http.Server{Handler: h.handler}
	if tlsCfg != nil {
		return tls.Listen("tcp", h.addr, tlsCfg)
	}
	return net.Listen("tcp", h.addr)
}

// Serve запускает HTTP-сервер; http.ErrServerClosed не считается ошибкой (штатное завершение).
func (h *httpSrv) Serve(ln net.Listener) error {
	if err := h.srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (h *httpSrv) Shutdown(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}
