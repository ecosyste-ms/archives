package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleReadmeAndChangelogSkipOversizedFiles(t *testing.T) {
	const limit = 10 * 1024 * 1024
	for _, tc := range []struct {
		endpoint string
		file     string
		handler  http.HandlerFunc
	}{
		{"readme", "package/README.md", HandleReadme},
		{"changelog", "package/CHANGELOG.md", HandleChangelog},
	} {
		for _, size := range []int{limit, limit + 1} {
			data := tarGzFixture(t, []archiveFixtureEntry{{name: tc.file, contents: strings.Repeat("\x00", size)}})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(data)
			}))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/archives/"+tc.endpoint+"?url="+url.QueryEscape(server.URL+"/package.tgz"), nil)
			w := httptest.NewRecorder()
			tc.handler(w, req)
			server.Close()

			want := http.StatusOK
			if size > limit {
				want = http.StatusNotFound
			}
			if w.Code != want {
				t.Errorf("%s with %d bytes: status = %d, want %d", tc.endpoint, size, w.Code, want)
			}
		}
	}
}
