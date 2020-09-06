package httpservice

import (
	"net/http"
)

type QATransport struct {
	Transport http.RoundTripper
}

func (t *QATransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", defaultUserAgent)
	return t.Transport.RoundTrip(req)
}
