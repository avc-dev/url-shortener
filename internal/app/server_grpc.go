package app

import (
	"context"
	"crypto/tls"
	"net"

	"google.golang.org/grpc"
)

// grpcSrv реализует Server для gRPC.
type grpcSrv struct {
	addr string
	srv  *grpc.Server
}

func newGRPCServer(addr string, srv *grpc.Server) *grpcSrv {
	return &grpcSrv{addr: addr, srv: srv}
}

func (g *grpcSrv) Name() string { return "gRPC" }

// Prepare открывает listener для gRPC-сервера.
// Если tlsCfg != nil — listener открывается с TLS.
func (g *grpcSrv) Prepare(tlsCfg *tls.Config) (net.Listener, error) {
	if tlsCfg != nil {
		return tls.Listen("tcp", g.addr, tlsCfg)
	}
	return net.Listen("tcp", g.addr)
}

func (g *grpcSrv) Serve(ln net.Listener) error {
	return g.srv.Serve(ln)
}

// Shutdown выполняет graceful shutdown gRPC-сервера.
// При таймауте контекста вызывает Stop(), затем ждёт фактического завершения GracefulStop,
// чтобы исключить гонку между завершением Shutdown() и работающей горутиной GracefulStop.
func (g *grpcSrv) Shutdown(ctx context.Context) error {
	stopCh := make(chan struct{})
	go func() {
		g.srv.GracefulStop()
		close(stopCh)
	}()
	select {
	case <-ctx.Done():
		g.srv.Stop()
		<-stopCh
		return ctx.Err()
	case <-stopCh:
		return nil
	}
}
