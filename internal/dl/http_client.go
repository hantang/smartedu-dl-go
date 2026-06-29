package dl

import (
	"net/http"
	"time"
)

var defaultHTTPClient = createHTTPClient()

func createHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     32,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		DisableKeepAlives:   false,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}
