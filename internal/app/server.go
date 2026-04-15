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
	"fmt"
	"math/big"
	"net"
	"time"

	"go.uber.org/zap"
)

// Server описывает полный жизненный цикл протокольного сервера.
// Каждый протокол реализует этот интерфейс независимо.
type Server interface {
	Name() string
	Prepare(tlsCfg *tls.Config) (net.Listener, error)
	Serve(ln net.Listener) error
	Shutdown(ctx context.Context) error
}

// shutdowner используется внутри runParallel и не привязан к Server,
// чтобы оставить runParallel универсальным.
type shutdowner interface {
	Shutdown(ctx context.Context) error
}

// prepare открывает listeners для всех серверов приложения.
// TLS-конфиг генерируется один раз и передаётся каждому серверу, если EnableHTTPS=true.
func (a *App) prepare() ([]net.Listener, error) {
	var tlsCfg *tls.Config
	if a.config.EnableHTTPS {
		var err error
		tlsCfg, err = buildSelfSignedTLSConfig()
		if err != nil {
			return nil, err
		}
	}

	listeners := make([]net.Listener, 0, len(a.servers))
	for _, s := range a.servers {
		ln, err := s.Prepare(tlsCfg)
		if err != nil {
			for _, opened := range listeners {
				opened.Close()
			}
			return nil, err
		}
		listeners = append(listeners, ln)
	}
	return listeners, nil
}

// serve запускает сервер на listener и блокирует до его остановки.
func (a *App) serve(s Server, ln net.Listener) error {
	a.logger.Info("Starting server",
		zap.String("name", s.Name()),
		zap.String("address", ln.Addr().String()),
	)
	if err := s.Serve(ln); err != nil {
		return fmt.Errorf("%s server: %w", s.Name(), err)
	}
	return nil
}

// shutdown выполняет graceful shutdown всех серверов параллельно.
func (a *App) shutdown(ctx context.Context) error {
	servers := make([]shutdowner, len(a.servers))
	for i, s := range a.servers {
		servers[i] = s
	}
	return runParallel(ctx, servers...)
}

// runParallel останавливает серверы параллельно и возвращает первую встреченную ошибку.
func runParallel(ctx context.Context, servers ...shutdowner) error {
	errCh := make(chan error, len(servers))
	for _, s := range servers {
		s := s
		go func() { errCh <- s.Shutdown(ctx) }()
	}
	var firstErr error
	for range servers {
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
