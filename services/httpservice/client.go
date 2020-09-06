package httpservice

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/clear-ness/qa-discussion/model"
)

const (
	ConnectTimeout = 3 * time.Second
	RequestTimeout = 30 * time.Second
)

var reservedIPRanges []*net.IPNet

// for preventing SSRF attack
func IsReservedIP(ip net.IP) bool {
	for _, ipRange := range reservedIPRanges {
		if ipRange.Contains(ip) {
			return true
		}
	}
	return false
}

func IsInterfacesIP(ip net.IP) (bool, error) {
	// システムのネットワークインターフェース情報を取得
	interfaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}

	for _, interf := range interfaces {
		addresses, err := interf.Addrs()
		if err != nil {
			return false, err
		}

		for _, addr := range addresses {
			var selfIP net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				selfIP = v.IP
			case *net.IPAddr:
				selfIP = v.IP
			}

			if ip.Equal(selfIP) {
				return true, nil
			}
		}
	}

	return false, nil
}

var defaultUserAgent string

func init() {
	for _, cidr := range []string{
		// See https://tools.ietf.org/html/rfc6890
		"0.0.0.0/8",      // This host on this network
		"10.0.0.0/8",     // Private-Use
		"127.0.0.0/8",    // Loopback
		"169.254.0.0/16", // Link Local
		"172.16.0.0/12",  // Private-Use Networks
		"192.168.0.0/16", // Private-Use Networks
		"::/128",         // Unspecified Address
		"::1/128",        // Loopback Address
		"fc00::/7",       // Unique-Local
		"fe80::/10",      // Linked-Scoped Unicast
	} {
		_, parsed, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		reservedIPRanges = append(reservedIPRanges, parsed)
	}

	defaultUserAgent = "qa-" + model.CurrentVersion
}

type DialContextFunction func(ctx context.Context, network, addr string) (net.Conn, error)

func dialContextFilter(dial DialContextFunction, allowHost func(host string) bool, allowIP func(ip net.IP) bool) DialContextFunction {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		if allowHost != nil && allowHost(host) {
			return dial(ctx, network, addr)
		}

		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}

		var firstErr error
		for _, ip := range ips {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			if allowIP == nil || !allowIP(ip) {
				continue
			}

			conn, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			if firstErr == nil {
				firstErr = err
			}
		}
		if firstErr == nil {
			return nil, errors.New("forbidden")
		}

		return nil, firstErr
	}
}

func NewTransport(allowHost func(host string) bool, allowIP func(ip net.IP) bool) http.RoundTripper {
	dialContext := (&net.Dialer{
		Timeout:   ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}).DialContext

	if allowHost != nil || allowIP != nil {
		dialContext = dialContextFilter(dialContext, allowHost, allowIP)
	}

	return &QATransport{
		&http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   ConnectTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
