package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMuzskyProvider_Search_LinkExtraction(t *testing.T) {
	// Simplified HTML based on user report
	htmlContent := `
<section><div class="container mt-5"> <div class="card mb-3"> <table class="table table-striped border-white tablestyle"> 
<thead> <tr> <td scope="col"><span class="tabletitle tablecolor">Morgenstern Bistro</span></td> <th scope=""></th> </tr> </thead> <tbody> 
<tr><td scope="row"><img class="lazy" src="/images/load.svg" data-src="https://lh3.googleusercontent.com/yvbKy9YE_hjgBVJyXxJScBMWTRfIYeDKBRVCRVnBVOU6EBiIAEgUpAu_D_5zcr0y33q0H22Kqlxue49ktQ=w200-h200-l90-rj" width="50"><span class="tablestyle tablecolor"><a href="/handler/morgenstern-bistro/0">Slava Marlow - Bystro Ft Morgenshtern</a></span></td><td class="align-middle"><div class="d-flex justify-content-end"><a href="/handler/morgenstern-bistro/0" class="icon me-2"> <i class="bi bi-download "></i></a><div data-id="https://cdn.muzsky.net/?h=JGraYpdVSDgXtbN7dTVRMsw9mol6gpvLQcg9TYDbzo5XXdZcnlh_Sh3CY4O5JSIQyxOauYptaq4Uf5B7bS_2kiIn" class="list-songs icon icon2"><div class="mp3data a playbtn"><i class="bi bi-play"></i></div></div></td></tr> 
</tbody></table></div></div></section>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	provider := NewMuzskyProvider(http.DefaultClient)
	// Override sourceURL to point to test server
	provider.sourceURL = server.URL + "/search/"

	results, err := provider.Search(context.Background(), "test")
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	if len(results) > 0 {
		song := results[0].Song
		// This is what we expect
		expectedLink := "https://cdn.muzsky.net/?h=JGraYpdVSDgXtbN7dTVRMsw9mol6gpvLQcg9TYDbzo5XXdZcnlh_Sh3CY4O5JSIQyxOauYptaq4Uf5B7bS_2kiIn"

		// The current implementation probably returns "/handler/morgenstern-bistro/0" or "https://muzsky.net/handler/morgenstern-bistro/0"
		// User requested to ensure the link ends with a backslash
		if !strings.HasSuffix(expectedLink, "\\") {
			expectedLink += "\\"
		}
		assert.Equal(t, expectedLink, song.Link, "Download URL should match data-id attribute with added backslash")
	}
}
