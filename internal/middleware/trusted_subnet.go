package middleware

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

// TrustedSubnet возвращает middleware, разрешающий запросы только из указанной подсети CIDR.
// Адрес клиента берётся из заголовка X-Real-IP.
// При пустом cidr или IP вне подсети возвращает 403 Forbidden.
func TrustedSubnet(cidr string, logger *zap.Logger) func(http.Handler) http.Handler {
	ipNet := parseCIDR(cidr, logger)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ipNet == nil {
				// cidr пуст или невалиден — доступ запрещён для всех
				w.WriteHeader(http.StatusForbidden)
				return
			}

			realIP := r.Header.Get("X-Real-IP")
			ip := net.ParseIP(realIP)
			if ip == nil || !ipNet.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// parseCIDR разбирает CIDR-строку и логирует ошибку при невалидном значении.
// Возвращает nil при пустом или невалидном cidr.
func parseCIDR(cidr string, logger *zap.Logger) *net.IPNet {
	if cidr == "" {
		return nil
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		logger.Error("invalid trusted_subnet CIDR", zap.String("cidr", cidr), zap.Error(err))
		return nil
	}
	return ipNet
}
