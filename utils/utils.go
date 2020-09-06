package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetIpAddress(r *http.Request, trustedProxyIPHeader []string) string {
	address := ""

	for _, proxyHeader := range trustedProxyIPHeader {
		header := r.Header.Get(proxyHeader)
		if len(header) > 0 {
			addresses := strings.Fields(header)
			if len(addresses) > 0 {
				address = strings.TrimRight(addresses[0], ",")
			}
		}
	}

	if len(address) == 0 {
		address, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	return address
}

func StringSliceDiff(a, b []string) []string {
	m := make(map[string]bool)
	result := []string{}

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if !m[item] {
			result = append(result, item)
		}
	}
	return result
}

func StringArrayIntersection(arr1, arr2 []string) []string {
	arrMap := map[string]bool{}
	result := []string{}

	for _, value := range arr1 {
		arrMap[value] = true
	}

	for _, value := range arr2 {
		if arrMap[value] {
			result = append(result, value)
		}
	}

	return result
}
