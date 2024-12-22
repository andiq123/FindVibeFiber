package scrapper

import (
	"crypto/tls"
	"net/http"

	"github.com/gocolly/colly/v2"
)

func GetInstance() *colly.Collector {
	collector := colly.NewCollector(
		colly.CacheDir("./cache/music_cache"),
	)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	collector.WithTransport(transport)

	return collector
}
