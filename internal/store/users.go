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

// Crée un utilisateur avec un rôle ("customer" ou "admin"), confirmé ou non.
// Si l'email existe déjà, ON CONFLICT n'insère rien, aucune ligne n'est renvoyée → ErrConflict.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string, confirmed bool) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, role, confirmed_at)
		VALUES ($1, $2, $3, CASE WHEN $4::boolean THEN now() END)
		ON CONFLICT (email) DO NOTHING
		RETURNING id, email, role, confirmed_at, created_at`, email, passwordHash, role, confirmed,
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

// Modifie l'email et le rôle. ErrConflict si l'email est déjà pris par un autre compte.
func (s *Store) UpdateUser(ctx context.Context, id int64, email, role string) (User, error) {
	var taken bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM users WHERE email = $1 AND id <> $2)", email, id,
	).Scan(&taken)
	if err != nil {
		return User{}, fmt.Errorf("check email: %w", err)
	}
	if taken {
		return User{}, ErrConflict
	}

	var u User
	err = s.db.QueryRowContext(ctx,
		`UPDATE users SET email = $2, role = $3 WHERE id = $1
		RETURNING id, email, role, confirmed_at, created_at`, id, email, role,
	).Scan(&u.ID, &u.Email, &u.Role, &u.ConfirmedAt, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update user %d: %w", id, err)
	}
	return u, nil
}

// Supprime un utilisateur (ses sessions, codes, paniers et commandes partent avec : ON DELETE CASCADE)
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
