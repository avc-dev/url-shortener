package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// start запускает HTTP или HTTPS сервер в зависимости от конфигурации.
func (a *App) start() error {
	router := newRouter(a.handler, a.logger, a.authService)
	addr := a.config.ServerAddress.String()

	if a.config.EnableHTTPS {
		return a.startTLS(addr, router)
	}

	a.logger.Info("Starting HTTP server", zap.String("address", addr))
	if err := http.ListenAndServe(addr, router); err != nil {
		a.logger.Fatal("Server failed", zap.Error(err))
		return err
	}
	return nil
}

// startTLS запускает HTTPS сервер с авто-сгенерированным самоподписанным сертификатом.
func (a *App) startTLS(addr string, handler http.Handler) error {
	tlsCfg, err := buildSelfSignedTLSConfig()
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}

	a.logger.Info("Starting HTTPS server", zap.String("address", addr))

	srv := &http.Server{Handler: handler}
	if err := srv.Serve(ln); err != nil {
		a.logger.Fatal("Server failed", zap.Error(err))
		return err
	}
	return nil
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
