package store

import (
	"context"
	"database/sql"
	"errors"
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