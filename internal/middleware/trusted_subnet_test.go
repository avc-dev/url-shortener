package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func testNextHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestTrustedSubnet_EmptyCIDR(t *testing.T) {
	mw := TrustedSubnet("", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.5")
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTrustedSubnet_InvalidCIDR(t *testing.T) {
	mw := TrustedSubnet("not-a-cidr", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.5")
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTrustedSubnet_IPInSubnet(t *testing.T) {
	mw := TrustedSubnet("192.168.1.0/24", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.42")
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTrustedSubnet_IPOutsideSubnet(t *testing.T) {
	mw := TrustedSubnet("192.168.1.0/24", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTrustedSubnet_MissingXRealIP(t *testing.T) {
	mw := TrustedSubnet("192.168.1.0/24", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTrustedSubnet_InvalidIPInHeader(t *testing.T) {
	mw := TrustedSubnet("192.168.1.0/24", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "not-an-ip")
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTrustedSubnet_IPv6InSubnet(t *testing.T) {
	mw := TrustedSubnet("2001:db8::/32", zap.NewNop())(testNextHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "2001:db8::1")
	w := httptest.NewRecorder()

	mw.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTrustedSubnet_SubnetBoundaryIPs(t *testing.T) {
	tests := []struct {
		name           string
		ip             string
		expectedStatus int
	}{
		{"network address", "192.168.1.0", http.StatusOK},
		{"broadcast address", "192.168.1.255", http.StatusOK},
		{"first host", "192.168.1.1", http.StatusOK},
		{"last host", "192.168.1.254", http.StatusOK},
		{"outside subnet", "192.168.2.1", http.StatusForbidden},
	}

	mw := TrustedSubnet("192.168.1.0/24", zap.NewNop())(testNextHandler())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Real-IP", tt.ip)
			w := httptest.NewRecorder()

			mw.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
