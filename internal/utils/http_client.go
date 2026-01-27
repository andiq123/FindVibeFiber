package utils

import (
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/constants"
)

var (
	sharedHTTPClient *http.Client
	httpClientOnce   sync.Once
)

func GetHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		timeout := time.Duration(constants.DefaultHTTPTimeout) * time.Second
		sharedHTTPClient = createHTTPClient(timeout, constants.DefaultHTTPMaxIdleConns, constants.DefaultHTTPMaxIdlePerHost, time.Duration(constants.DefaultHTTPIdleTimeout)*time.Second)
	})
	return sharedHTTPClient
}

func createHTTPClient(timeout time.Duration, maxIdleConns, maxIdlePerHost int, idleTimeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdlePerHost,
			IdleConnTimeout:     idleTimeout,
		},
	}
}

func GetHTTPClientWithConfig(timeout time.Duration, maxIdleConns, maxIdlePerHost int, idleTimeout time.Duration) *http.Client {
	return createHTTPClient(timeout, maxIdleConns, maxIdlePerHost, idleTimeout)
}
