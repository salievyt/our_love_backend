package auth

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"
)

const secret = "test-key-for-local-test-run-only-32b"

func claims(typ string, exp time.Time) map[string]any {
	return map[string]any{
		"token_type": typ, "exp": exp.Unix(), "iat": time.Now().Unix(),
		"jti": "x", "user_id": "245094f8-79ab-44e1-92f6-fd29450c1cbe",
	}
}

func TestParseAccessToken(t *testing.T) {
	now := time.Now()
	valid := SignToken(claims("access", now.Add(time.Hour)), secret)

	noneHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	parts := strings.Split(valid, ".")

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"valid access token", valid, false},
		{"expired", SignToken(claims("access", now.Add(-time.Second)), secret), true},
		{"refresh token is not accepted as access", SignToken(claims("refresh", now.Add(time.Hour)), secret), true},
		{"wrong secret", SignToken(claims("access", now.Add(time.Hour)), secret+"x"), true},
		{"alg none", noneHeader + "." + parts[1] + ".", true},
		{"tampered payload", parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"token_type":"access","exp":9999999999,"user_id":"other"}`)) + "." + parts[2], true},
		{"garbage", "abc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseAccessToken(tt.token, secret, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && c.UserID != "245094f8-79ab-44e1-92f6-fd29450c1cbe" {
				t.Fatalf("user_id = %q", c.UserID)
			}
		})
	}
}

// Токен, выпущенный настоящим simplejwt, должен приниматься Go-версией.
// DJANGO_ACCESS_TOKEN выпускается скриптом проверки совместимости.
func TestDjangoIssuedToken(t *testing.T) {
	token := os.Getenv("DJANGO_ACCESS_TOKEN")
	if token == "" {
		t.Skip("DJANGO_ACCESS_TOKEN не задан")
	}
	if _, err := ParseAccessToken(token, os.Getenv("SECRET_KEY"), time.Now()); err != nil {
		t.Fatalf("token from simplejwt rejected: %v", err)
	}
}
