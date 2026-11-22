package media

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func serveReq(t *testing.T, s *Server, target string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle("GET /media/{path...}", s)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", target, nil))
	return rec
}

func TestServeSignedMedia(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "places"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "places", "a.png"), []byte("png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "places", "x.html"), []byte("<script>"), 0o644); err != nil {
		t.Fatal(err)
	}
	signer := NewSigner("test-key-for-local-test-run-only-32b")
	local := &Server{Signer: signer, Root: root}
	accel := &Server{Signer: signer, Root: root, Accel: true}
	now := time.Now()

	t.Run("valid link serves the file", func(t *testing.T) {
		rec := serveReq(t, local, signer.Path("places/a.png", now))
		if rec.Code != 200 || rec.Body.String() != "png-bytes" || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("got %d %q %v", rec.Code, rec.Body.String(), rec.Header())
		}
	})
	t.Run("behind nginx answers with X-Accel-Redirect", func(t *testing.T) {
		rec := serveReq(t, accel, signer.Path("places/a.png", now))
		if rec.Code != 200 || rec.Header().Get("X-Accel-Redirect") != "/protected-media/places/a.png" || rec.Body.Len() != 0 {
			t.Fatalf("got %d %v", rec.Code, rec.Header())
		}
	})
	t.Run("html is a download, never inline", func(t *testing.T) {
		rec := serveReq(t, local, signer.Path("places/x.html", now))
		if rec.Header().Get("Content-Disposition") != "attachment" || rec.Header().Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("headers %v", rec.Header())
		}
	})
	t.Run("missing, tampered and expired links are 404", func(t *testing.T) {
		good := signer.Path("places/a.png", now)
		for _, target := range []string{
			"/media/places/a.png",
			strings.Replace(good, "sig=", "sig=x", 1),
			strings.Replace(good, "places/a.png", "places/x.html", 1),
			signer.Path("places/missing.png", now),
			"/media/places/a.png?exp=1&sig=" + signer.Signature("places/a.png", "1"),
		} {
			if rec := serveReq(t, local, target); rec.Code != 404 {
				t.Errorf("%s -> %d", target, rec.Code)
			}
		}
	})
}
