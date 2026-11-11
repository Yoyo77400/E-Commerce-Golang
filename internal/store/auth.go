package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Types de codes (colonne user_codes.kind)
const (
	CodeConfirm = "confirm"
	CodeReset   = "reset"
)

// ---------- Codes à usage unique ----------

// Génère un code, l'enregistre avec une date d'expiration et le renvoie
func (s *Store) CreateCode(ctx context.Context, userID int64, kind string, ttl time.Duration) (string, error) {
	code := GenerateCodeForConfirmOrReset()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO user_codes (user_id, kind, code, expires_at) VALUES ($1, $2, $3, $4)",
		userID, kind, code, time.Now().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("create code: %w", err)
	}
	return code, nil
}

// Consomme un code : il doit exister, ne pas être expiré ni déjà utilisé.
// L'UPDATE le marque comme utilisé dans la même requête, donc il ne peut servir qu'une fois.
func (s *Store) UseCode(ctx context.Context, kind, code string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx,
		`UPDATE user_codes SET used_at = now()
		WHERE kind = $1 AND code = $2 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id`, kind, code,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("use code: %w", err)
	}
	return userID, nil
}

// ---------- Sessions ----------

// Crée une session et renvoie son jeton
func (s *Store) CreateSession(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	token := GenerateNewSessionToken()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)",
		token, userID, time.Now().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

// Retrouve l'utilisateur à partir d'un jeton de session non expiré
func (s *Store) UserBySession(ctx context.Context, token string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.email, u.role, u.confirmed_at, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > now()`, token,
	).Scan(&u.ID, &u.Email, &u.Role, &u.ConfirmedAt, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user by session: %w", err)
	}
	return u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token = $1", token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// Déconnecte l'utilisateur partout (après un changement de mot de passe)
func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}
