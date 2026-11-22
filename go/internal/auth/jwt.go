// Package auth проверяет access-токены, выпущенные djangorestframework-simplejwt.
//
// simplejwt подписывает токены HS256 ключом SECRET_KEY и кладёт в payload
// token_type, exp, iat, jti и user_id (UUID строкой). HS256 — это HMAC-SHA256,
// поэтому проверка сделана на стандартной библиотеке без внешних зависимостей.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	TokenType string `json:"token_type"`
	Exp       int64  `json:"exp"`
	UserID    string `json:"user_id"`
}

var b64 = base64.RawURLEncoding

// ParseAccessToken проверяет подпись, алгоритм, срок и тип токена.
func ParseAccessToken(token, secret string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := decodePart(parts[0], &header); err != nil || header.Alg != "HS256" {
		// Принимаем только HS256: защита от alg=none и подмены алгоритма
		return Claims{}, ErrInvalidToken
	}

	sig, err := b64.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, sign(parts[0]+"."+parts[1], secret)) {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := decodePart(parts[1], &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.TokenType != "access" || claims.UserID == "" || claims.Exp == 0 || now.Unix() >= claims.Exp {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

// SignToken выпускает токен в формате simplejwt. Нужен тестам.
func SignToken(claims map[string]any, secret string) string {
	header := b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	unsigned := header + "." + b64.EncodeToString(payload)
	return unsigned + "." + b64.EncodeToString(sign(unsigned, secret))
}

func sign(data, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func decodePart(part string, v any) error {
	raw, err := b64.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}
