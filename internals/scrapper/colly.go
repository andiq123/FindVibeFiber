package scrapper

import (
	"crypto/tls"
	"net/http"

	"github.com/gocolly/colly/v2"
)

func GetInstance() *colly.Collector {
	collector := colly.NewCollector(
		colly.CacheDir("./cache/music_cache"),
		colly.AllowURLRevisit(),
		colly.Async(true),
	)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	collector.WithTransport(transport)

	return collector
}
