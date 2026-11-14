package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type AddCartItemInput struct {
	ProductReference string `json:"product_reference"`
	Quantity         int    `json:"quantity"`
}

func (in AddCartItemInput) Validate() error {
	if in.ProductReference == "" {
		return errors.New("ProductReference est essentiel")
	}
	if in.Quantity < 1 {
		return errors.New("Quantity doit être supérieur à 0")
	}
	return nil
}

type CartItem struct {
	ProductReference  string `json:"product_reference"`
	Quantity          int    `json:"quantity"`
	UnitPriceTTCCents int64  `json:"unit_price_ttc_cents"`
}

func (item CartItem) SubtotalCents() int64 {
	return item.UnitPriceTTCCents * int64(item.Quantity)
}

type Cart struct {
	Reference string     `json:"reference"`
	Items     []CartItem `json:"items"`
}

func (cart Cart) TotalTTCCents() int64 {
	var total int64 = 0
	for _, item := range cart.Items {
		total += item.SubtotalCents()
	}
	return total
}

func (s *Store) GetOpenCartReference(
	ctx context.Context,
	userID int64,
) (string, error) {
	var reference string
	err := s.db.QueryRowContext(ctx,
		`SELECT reference FROM carts WHERE user_id = $1 AND status = 'open'`, userID,
	).Scan(&reference)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}

	if err != nil {
		return "", err
	}
	return reference, nil
}

// Crée un panier vide. ErrConflict si l'utilisateur a déjà un panier ouvert.
func (s *Store) CreateCart(ctx context.Context, userID int64) (string, error) {
	reference := GenerateUniqueReferenceID(PrefixCart)
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO carts (reference, user_id) VALUES ($1, $2)
		ON CONFLICT (user_id) WHERE status = 'open' DO NOTHING`,
		reference, userID,
	)
	if err != nil {
		return "", fmt.Errorf("create cart: %w", err)
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("count inserted carts: %w", err)
	}
	if inserted == 0 {
		return "", ErrConflict
	}
	return reference, nil
}

// Réutilise le panier ouvert, ou en crée un si l'utilisateur n'en a pas encore.
func (s *Store) GetOrCreateOpenCartReference(ctx context.Context, userID int64) (string, error) {
	reference, err := s.GetOpenCartReference(ctx, userID)
	if !errors.Is(err, ErrNotFound) {
		return reference, err
	}

	reference, err = s.CreateCart(ctx, userID)
	if errors.Is(err, ErrConflict) {
		// Une autre requête a pu créer le panier depuis notre première lecture.
		return s.GetOpenCartReference(ctx, userID)
	}
	return reference, err
}
