package httpservice

import (
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/clear-ness/qa-discussion/services/configservice"
)

type HTTPService interface {
	MakeClient(trustURLs bool) *http.Client
	MakeTransport(trustURLs bool) http.RoundTripper
}

type HTTPServiceImpl struct {
	configService  configservice.ConfigService
	RequestTimeout time.Duration
}

func splitFields(c rune) bool {
	return unicode.IsSpace(c) || c == ','
}

func MakeHTTPService(configService configservice.ConfigService) HTTPService {
	return &HTTPServiceImpl{
		configService,
		RequestTimeout,
	}
}

func (h *HTTPServiceImpl) MakeClient(trustURLs bool) *http.Client {
	return &http.Client{
		Transport: h.MakeTransport(trustURLs),
		Timeout:   h.RequestTimeout,
	}
}

func (h *HTTPServiceImpl) MakeTransport(trustURLs bool) http.RoundTripper {
	if trustURLs {
		return NewTransport(nil, nil)
	}

	allowHost := func(host string) bool {
		// 設定により、ホワイトリストで内部ネットへの通信を許可できる仕様。
		if h.configService.Config().ServiceSettings.AllowedUntrustedInternalConnections == nil {
			return false
		}

		for _, allowed := range strings.FieldsFunc(*h.configService.Config().ServiceSettings.AllowedUntrustedInternalConnections, splitFields) {
			if host == allowed {
				return true
			}
		}
		return false
	}

	allowIP := func(ip net.IP) bool {
		reservedIP := IsReservedIP(ip)
		interfacesIP, err := IsInterfacesIP(ip)

		if err != nil {
			return false
		}

		if !reservedIP && !interfacesIP {
			return true
		}

		if h.configService.Config().ServiceSettings.AllowedUntrustedInternalConnections == nil {
			return false
		}

		// 設定により、ホワイトリストで内部ネットへの通信を許可できる仕様。
		for _, allowed := range strings.FieldsFunc(*h.configService.Config().ServiceSettings.AllowedUntrustedInternalConnections, splitFields) {
			if _, ipRange, err := net.ParseCIDR(allowed); err == nil && ipRange.Contains(ip) {
				return true
			}
		}
		return false
	}

	return NewTransport(allowHost, allowIP)
}
