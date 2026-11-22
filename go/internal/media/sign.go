// Package media — медиафайлы, общие с Django: подписанные ссылки
// (порт config/media.py) и сохранение загрузок (порт config/uploads.py).
//
// Файлы лежат в MEDIA_ROOT — томе, общем с Django. Отдаёт их по-прежнему
// Django (/media/ → проверка подписи → X-Accel-Redirect), поэтому подпись
// обязана совпадать с django.core.signing.Signer(salt="media-link") бит в бит.
package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/httpx"
)

const (
	// LinkTTL — шаг округления срока жизни ссылки (LINK_TTL_SECONDS в Django).
	LinkTTL = 3600
	// URLPrefix совпадает с MEDIA_URL.
	URLPrefix = "/media/"
	// salt из Signer(salt="media-link"); Signer.signature добавляет "signer".
	keySalt = "media-link" + "signer"
)

// Signer подписывает и проверяет ссылки на медиа ключом SECRET_KEY.
type Signer struct {
	key []byte // производный ключ salted_hmac: sha256(key_salt + secret)
}

// NewSigner повторяет django.utils.crypto.salted_hmac(algorithm="sha256"):
// ключ HMAC — sha256(key_salt + SECRET_KEY), а не сам SECRET_KEY.
func NewSigner(secret string) *Signer {
	sum := sha256.Sum256([]byte(keySalt + secret))
	return &Signer{key: sum[:]}
}

// Signature — Signer(salt="media-link").signature(f"{name}:{exp}"):
// HMAC-SHA256, urlsafe base64 без "=".
func (s *Signer) Signature(name, exp string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(name + ":" + exp))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Expiry — срок действия ссылки, созданной в now: конец следующего часа.
// Ссылка живёт от 1 до 2 часов и не меняется в пределах часа (клиенты кэшируют).
func Expiry(now time.Time) int64 {
	return (now.Unix()/LinkTTL + 2) * LinkTTL
}

// Path — относительная подписанная ссылка /media/<quote(name)>?exp=…&sig=…
// (signed_media_url в Django).
func (s *Signer) Path(name string, now time.Time) string {
	name = strings.ReplaceAll(name, `\`, "/")
	exp := strconv.FormatInt(Expiry(now), 10)
	return URLPrefix + Quote(name) + "?exp=" + exp + "&sig=" + s.Signature(name, exp)
}

// URL — абсолютная ссылка (request.build_absolute_uri) или nil, если файла нет:
// так DRF отдаёт и ImageField, и photo_url.
func (s *Signer) URL(r *http.Request, name string) *string {
	if name == "" {
		return nil
	}
	u := httpx.BaseURL(r) + s.Path(name, time.Now())
	return &u
}

// Verify — проверки serve_media: exp из цифр и не в прошлом, нет сегментов "..",
// подпись совпадает (сравнение за постоянное время). path — уже декодированный путь
// после /media/.
func (s *Signer) Verify(path, exp, sig string, now time.Time) bool {
	if exp == "" || strings.Trim(exp, "0123456789") != "" {
		return false
	}
	n, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || n < now.Unix() {
		return false
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == ".." {
			return false
		}
	}
	return hmac.Equal([]byte(sig), []byte(s.Signature(path, exp)))
}

// Quote — urllib.parse.quote(name) с safe="/": не экранируются только
// ASCII-буквы, цифры, "_.-~" и "/"; всё остальное — %XX по байтам UTF-8.
func Quote(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' ||
			c == '_' || c == '.' || c == '-' || c == '~' || c == '/' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}
