package utils

import (
	"crypto/tls"
	"net/http"
	"time"
)

func NewHTTPClient(timeout time.Duration, maxIdleConns, maxIdlePerHost int, idleTimeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
			MaxIdleConns:          maxIdleConns,
			MaxIdleConnsPerHost:   maxIdlePerHost,
			IdleConnTimeout:       idleTimeout,
			ResponseHeaderTimeout: 2 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}
}
