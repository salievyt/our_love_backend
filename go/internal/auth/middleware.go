package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/httpx"
)

// User — аутентифицированный пользователь и его пара (если есть).
type User struct {
	ID             string
	Username       string
	RelationshipID string // "" — пары нет
}

func (u User) HasPair() bool { return u.RelationshipID != "" }

type ctxKey struct{}

func FromContext(ctx context.Context) User {
	u, _ := ctx.Value(ctxKey{}).(User)
	return u
}

func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// Middleware повторяет поведение JWTAuthentication + IsAuthenticated из DRF.
func Middleware(db *sql.DB, secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)

		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			httpx.Error(w, http.StatusUnauthorized, "Учетные данные не были предоставлены.")
			return
		}
		claims, err := ParseAccessToken(token, secret, time.Now())
		if err != nil || !httpx.IsUUID(claims.UserID) {
			httpx.JSON(w, http.StatusUnauthorized, map[string]string{
				"detail": "Токен недействителен или просрочен",
				"code":   "token_not_valid",
			})
			return
		}

		user, active, err := loadUser(r.Context(), db, claims.UserID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			httpx.JSON(w, http.StatusUnauthorized, map[string]string{"detail": "Пользователь не найден", "code": "user_not_found"})
			return
		case err != nil:
			httpx.ServerError(w, r, err)
			return
		case !active:
			httpx.JSON(w, http.StatusUnauthorized, map[string]string{"detail": "Пользователь неактивен", "code": "user_inactive"})
			return
		}

		w.Header().Del("WWW-Authenticate")
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

// loadUser берёт пару так же, как Django: сначала relationship_as_partner1, потом partner2.
func loadUser(ctx context.Context, db *sql.DB, id string) (User, bool, error) {
	var (
		u      User
		active bool
		rel    sql.NullString
	)
	err := db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.is_active, r.id
		FROM accounts_user u
		LEFT JOIN accounts_relationship r ON r.partner1_id = u.id OR r.partner2_id = u.id
		WHERE u.id = $1
		ORDER BY (r.partner1_id = u.id) DESC NULLS LAST
		LIMIT 1`, id).Scan(&u.ID, &u.Username, &active, &rel)
	u.RelationshipID = rel.String
	return u, active, err
}
