package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type User struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, email, role, confirmed_at, created_at FROM users ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.ConfirmedAt, &u.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (s *Store) GetUser(ctx context.Context, id int64) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		"SELECT id, email, role, confirmed_at, created_at FROM users WHERE id = $1", id,
	).Scan(&u.ID, &u.Email, &u.Role, &u.ConfirmedAt, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

// Renvoie l'utilisateur et le hash de son mot de passe
func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := s.db.QueryRowContext(ctx,
		"SELECT id, email, role, confirmed_at, created_at, password_hash FROM users WHERE email = $1", email,
	).Scan(&u.ID, &u.Email, &u.Role, &u.ConfirmedAt, &u.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	if err != nil {
		return User{}, "", fmt.Errorf("get user by email: %w", err)
	}
	return u, hash, nil
}

// Crée un client non confirmé. Si l'email existe déjà, ON CONFLICT n'insère rien et return error
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING
		RETURNING id, email, role, confirmed_at, created_at`, email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Role, &u.ConfirmedAt, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrConflict
	}
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// Crée le compte admin s'il n'existe pas encore
func (s *Store) EnsureAdmin(ctx context.Context, email, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, role, confirmed_at)
		VALUES ($1, $2, 'admin', now())
		ON CONFLICT (email) DO NOTHING`, email, passwordHash)
	if err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}
	return nil
}

// Marque le compte comme confirmé
func (s *Store) ConfirmUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET confirmed_at = COALESCE(confirmed_at, now()) WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("confirm user %d: %w", id, err)
	}
	return nil
}

// Met à jour le mot de passe d'un utilisateur
func (s *Store) SetPassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = $2 WHERE id = $1", id, passwordHash)
	if err != nil {
		return fmt.Errorf("set password %d: %w", id, err)
	}
	return nil
}
