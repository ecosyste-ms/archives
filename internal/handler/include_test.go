package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleReadmeAndChangelogDoNotIncludeLocalFiles(t *testing.T) {
	const secret = "ARCHIVES_INCLUDE_SECRET"
	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte(secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	orgInclude := "#+INCLUDE: \"" + secretPath + "\" src text\n"
	adocInclude := "include::" + secretPath + "[]\n"
	_, asciidoctorErr := exec.LookPath("asciidoctor")

	for _, tc := range []struct {
		file     string
		contents string
		handler  http.HandlerFunc
		skip     bool
	}{
		{"package/README.org", orgInclude, HandleReadme, false},
		{"package/CHANGELOG.org", orgInclude, HandleChangelog, false},
		{"package/README.adoc", adocInclude, HandleReadme, asciidoctorErr != nil},
		{"package/CHANGELOG.adoc", adocInclude, HandleChangelog, asciidoctorErr != nil},
	} {
		t.Run(tc.file, func(t *testing.T) {
			if tc.skip {
				t.Skip("asciidoctor not installed")
			}
			data := tarGzFixture(t, []archiveFixtureEntry{{name: tc.file, contents: tc.contents}})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(data)
			}))
			defer server.Close()

			req := httptest.NewRequest(http.MethodGet, "/?url="+url.QueryEscape(server.URL+"/package.tgz"), nil)
			w := httptest.NewRecorder()
			tc.handler(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), secret) {
				t.Errorf("response includes local file contents: %s", w.Body.String())
			}
		})
	}
}
