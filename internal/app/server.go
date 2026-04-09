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
	"math/big"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// prepare создаёт HTTP или HTTPS сервер и сохраняет его в App.
// Для HTTPS дополнительно открывает TLS-listener и возвращает его.
// Вызывается до запуска горутины, чтобы a.httpServer был доступен без гонки.
func (a *App) prepare() (net.Listener, error) {
	router := newRouter(a.handler, a.logger, a.authService, a.config.TrustedSubnet)
	addr := a.config.ServerAddress.String()

	if a.config.EnableHTTPS {
		tlsCfg, err := buildSelfSignedTLSConfig()
		if err != nil {
			return nil, err
		}
		ln, err := tls.Listen("tcp", addr, tlsCfg)
		if err != nil {
			return nil, err
		}
		a.httpServer = &http.Server{Handler: router}
		return ln, nil
	}

	a.httpServer = &http.Server{Addr: addr, Handler: router}
	return nil, nil
}

// serve запускает сервер и блокирует до его остановки.
// Возвращает nil при штатном завершении через Shutdown.
func (a *App) serve(ln net.Listener) error {
	if ln != nil {
		a.logger.Info("Starting HTTPS server", zap.String("address", ln.Addr().String()))
		if err := a.httpServer.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	a.logger.Info("Starting HTTP server", zap.String("address", a.httpServer.Addr))
	if err := a.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// shutdown выполняет graceful shutdown: прекращает приём новых запросов,
// ждёт завершения всех активных запросов, затем возвращает управление.
func (a *App) shutdown(ctx context.Context) error {
	if a.httpServer == nil {
		return nil
	}
	return a.httpServer.Shutdown(ctx)
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
