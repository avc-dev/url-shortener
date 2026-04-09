package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// prepare создаёт HTTP/HTTPS и gRPC серверы, открывает слушателей и возвращает их.
// TLS используется для обоих протоколов, если EnableHTTPS=true.
func (a *App) prepare() (httpLn net.Listener, grpcLn net.Listener, err error) {
	router := newRouter(a.handler, a.logger, a.authService, a.config.TrustedSubnet)
	httpAddr := a.config.ServerAddress.String()
	grpcAddr := a.config.GRPCAddress.String()

	if a.config.EnableHTTPS {
		tlsCfg, tlsErr := buildSelfSignedTLSConfig()
		if tlsErr != nil {
			return nil, nil, tlsErr
		}

		httpLn, err = tls.Listen("tcp", httpAddr, tlsCfg)
		if err != nil {
			return nil, nil, err
		}
		a.httpServer = &http.Server{Handler: router}

		grpcLn, err = tls.Listen("tcp", grpcAddr, tlsCfg)
		if err != nil {
			httpLn.Close()
			return nil, nil, err
		}
		return httpLn, grpcLn, nil
	}

	a.httpServer = &http.Server{Addr: httpAddr, Handler: router}

	grpcLn, err = net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, nil, err
	}
	return nil, grpcLn, nil
}

// serveHTTP запускает HTTP-сервер и блокирует до его остановки.
func (a *App) serveHTTP(ln net.Listener) error {
	if ln != nil {
		a.logger.Info("Starting HTTPS server", zap.String("address", ln.Addr().String()))
		if err := a.httpServer.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTPS server: %w", err)
		}
		return nil
	}

	a.logger.Info("Starting HTTP server", zap.String("address", a.httpServer.Addr))
	if err := a.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server: %w", err)
	}
	return nil
}

// serveGRPC запускает gRPC-сервер и блокирует до его остановки.
func (a *App) serveGRPC(ln net.Listener) error {
	a.logger.Info("Starting gRPC server", zap.String("address", ln.Addr().String()))
	if err := a.grpcServer.Serve(ln); err != nil {
		return fmt.Errorf("gRPC server: %w", err)
	}
	return nil
}

// shutdown выполняет graceful shutdown обоих серверов параллельно.
func (a *App) shutdown(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		if a.httpServer != nil {
			errCh <- a.httpServer.Shutdown(ctx)
		} else {
			errCh <- nil
		}
	}()

	go func() {
		// stopCh закрывается только когда GracefulStop реально завершился.
		// При таймауте контекста вызываем Stop(), затем дожидаемся закрытия stopCh —
		// это исключает гонку между завершением shutdown() и работающей горутиной GracefulStop.
		stopCh := make(chan struct{})
		go func() {
			if a.grpcServer != nil {
				a.grpcServer.GracefulStop()
			}
			close(stopCh)
		}()
		select {
		case <-ctx.Done():
			if a.grpcServer != nil {
				a.grpcServer.Stop()
			}
			<-stopCh // ждём фактического завершения перед возвратом
			errCh <- ctx.Err()
		case <-stopCh:
			errCh <- nil
		}
	}()

	var firstErr error
	for range 2 {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// buildSelfSignedTLSConfig генерирует самоподписанный TLS-сертификат в памяти.
func buildSelfSignedTLSConfig() (*tls.Config, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"url-shortener"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, err
	}

	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}
