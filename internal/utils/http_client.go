package utils

import (
	"crypto/tls"
	"net/http"
	"time"
)

var sharedHTTPClient *http.Client

func GetHTTPClient() *http.Client {
	if sharedHTTPClient == nil {
		sharedHTTPClient = createHTTPClient()
	}
	return sharedHTTPClient
}

func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}
}
