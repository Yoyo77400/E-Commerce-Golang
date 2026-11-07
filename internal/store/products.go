package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Product struct {
	Reference     string    `json:"reference"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	PriceCents    int64     `json:"price_cents"`
	VATRateBP     int       `json:"vat_rate_bp"`
	PriceTTCCents int64     `json:"price_ttc_cents"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *Store) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT reference, name, description, category, price_cents, vat_rate_bp, price_ttc_cents, created_at
		FROM products ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.Reference, &p.Name, &p.Description, &p.Category,
			&p.PriceCents, &p.VATRateBP, &p.PriceTTCCents, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	return products, nil
}

func (s *Store) GetProduct(ctx context.Context, reference string) (Product, error) {
	var p Product
	err := s.db.QueryRowContext(ctx,
		`SELECT reference, name, description, category, price_cents, vat_rate_bp, price_ttc_cents, created_at
		FROM products WHERE reference = $1`, reference,
	).Scan(&p.Reference, &p.Name, &p.Description, &p.Category,
		&p.PriceCents, &p.VATRateBP, &p.PriceTTCCents, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product %s: %w", reference, err)
	}
	return p, nil
}
