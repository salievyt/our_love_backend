package media

import (
	"net/http/httptest"
	"testing"
	"time"
)

const testSecret = "test-key-for-local-test-run-only-32b"

// Эталонные значения получены из Django (config.media._signature, signed_media_url)
// с SECRET_KEY=testSecret:
//
//	python -c "import django; django.setup(); from config.media import _signature; print(_signature(name, exp))"
func TestSignatureMatchesDjango(t *testing.T) {
	tests := []struct {
		name, exp, want, quoted string
	}{
		{"places/abc.png", "1795339200", "NNS4iEB05PSavLwZ-PuY7dYsHNcqHt_7CCSxcWSgHk0", "places/abc.png"},
		{"places/0f8fe5d7a1b24c6e9d3a2b1c0e9f8a7b.jpg", "1795342800", "FvEiVRSXNxsE6hg9iIg1z7chYP-557rVelzANLTR9lE",
			"places/0f8fe5d7a1b24c6e9d3a2b1c0e9f8a7b.jpg"},
		{"albums/фото 1.jpeg", "1795339200", "7-MD0n5FnLcOQCZEOLoycAosYYCZEd10WCsz8YpLUkw",
			"albums/%D1%84%D0%BE%D1%82%D0%BE%201.jpeg"},
		{"diary/a+b&c=d?.png", "1700000000", "8HwKTfdieb77ynG9-TcJ-OSKa4KcITsxUTcaqRONrk0", "diary/a%2Bb%26c%3Dd%3F.png"},
		// serve_media подписывает exp как строку из запроса: ведущий ноль — другая подпись
		{"places/abc.png", "01795339200", "1S8dAnqP4qi-Qc2GM_1NONcOhQKY0UNY6Gmu0yx8Vr4", "places/abc.png"},
	}
	s := NewSigner(testSecret)
	for _, tt := range tests {
		t.Run(tt.name+"@"+tt.exp, func(t *testing.T) {
			if got := s.Signature(tt.name, tt.exp); got != tt.want {
				t.Errorf("Signature = %s, want %s", got, tt.want)
			}
			if got := Quote(tt.name); got != tt.quoted {
				t.Errorf("Quote = %s, want %s", got, tt.quoted)
			}
		})
	}
}

func TestPathMatchesDjango(t *testing.T) {
	// signed_media_url("albums/фото 1.jpeg") при time.time() == 1795336000.5
	const want = "/media/albums/%D1%84%D0%BE%D1%82%D0%BE%201.jpeg?exp=1795341600&sig=4uSgqBr6uPep9pGJbcROWO-ay4CQEYAMbs_XrQF9nYo"
	now := time.Unix(1795336000, 500_000_000)
	if got := NewSigner(testSecret).Path("albums/фото 1.jpeg", now); got != want {
		t.Fatalf("Path =\n%s\nwant\n%s", got, want)
	}
	// обратные слеши (Windows-пути в storage) Django заменяет на "/"
	if a, b := NewSigner(testSecret).Path(`places\a.png`, now), NewSigner(testSecret).Path("places/a.png", now); a != b {
		t.Fatalf("backslash: %s != %s", a, b)
	}
}

func TestExpiry(t *testing.T) {
	tests := []struct {
		now  int64
		want int64
	}{
		{1795334400, 1795341600}, // ровно начало часа: живёт 2 часа
		{1795337999, 1795341600}, // конец часа: живёт чуть больше часа
		{1795338000, 1795345200},
	}
	for _, tt := range tests {
		if got := Expiry(time.Unix(tt.now, 0)); got != tt.want {
			t.Errorf("Expiry(%d) = %d, want %d", tt.now, got, tt.want)
		}
	}
}

func TestVerify(t *testing.T) {
	s := NewSigner(testSecret)
	now := time.Unix(1795336000, 0)
	const name, exp = "places/abc.png", "1795339200"
	sig := s.Signature(name, exp)

	tests := []struct {
		desc           string
		path, exp, sig string
		now            time.Time
		want           bool
	}{
		{"valid", name, exp, sig, now, true},
		{"valid until the last second", name, exp, sig, time.Unix(1795339200, 0), true},
		{"expired", name, exp, sig, time.Unix(1795339201, 0), false},
		{"other file", "places/abd.png", exp, sig, now, false},
		{"exp tampered", name, "1795342800", sig, now, false},
		{"sig tampered", name, exp, sig[:len(sig)-1] + "A", now, false},
		{"other key", name, exp, NewSigner(testSecret+"x").Signature(name, exp), now, false},
		{"empty sig", name, exp, "", now, false},
		{"empty exp", name, "", s.Signature(name, ""), now, false},
		{"negative exp", name, "-1", s.Signature(name, "-1"), now, false},
		{"non-digit exp", name, "1795339200.0", s.Signature(name, "1795339200.0"), now, false},
		{"dot-dot segment", "places/../secret.png", exp, s.Signature("places/../secret.png", exp), now, false},
		{"leading dot-dot", "../etc/passwd", exp, s.Signature("../etc/passwd", exp), now, false},
		{"dots inside a name are fine", "places/a..b.png", exp, s.Signature("places/a..b.png", exp), now, true},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := s.Verify(tt.path, tt.exp, tt.sig, tt.now); got != tt.want {
				t.Fatalf("Verify = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURLIsAbsolute(t *testing.T) {
	s := NewSigner(testSecret)
	r := httptest.NewRequest("GET", "http://api-go:8080/api/v1/places/", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "dev.example.com")
	u := s.URL(r, "places/abc.png")
	const prefix = "https://dev.example.com/media/places/abc.png?exp="
	if u == nil || len(*u) <= len(prefix) || (*u)[:len(prefix)] != prefix {
		t.Fatalf("URL = %v", u)
	}
	if s.URL(r, "") != nil {
		t.Fatal("empty name must give null")
	}
}
