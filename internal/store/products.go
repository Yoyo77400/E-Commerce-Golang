package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

// Critères de recherche. Un champ vide (ou nil pour les prix) est ignoré.
type ProductFilter struct {
	Query    string // cherche dans le nom OU la description
	Category string
	MinPrice *int64 // prix HT en centimes
	MaxPrice *int64
	MinTTC   *int64 // prix TTC en centimes
	MaxTTC   *int64
}

// Recherche les produits correspondant au filtre (sans filtre : tout le catalogue).
func (s *Store) SearchProducts(ctx context.Context, f ProductFilter) ([]Product, error) {
	var conditions []string
	var args []any

	// Ajoute une condition au WHERE : la valeur va dans args, et "?" est remplacé
	// par son numéro de paramètre ($1, $2...). Jamais de valeur collée dans le SQL.
	addCondition := func(condition string, value any) {
		args = append(args, value)
		placeholder := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, strings.ReplaceAll(condition, "?", placeholder))
	}

	if f.Query != "" {
		addCondition("(name ILIKE ? OR description ILIKE ?)", "%"+f.Query+"%")
	}
	if f.Category != "" {
		addCondition("category ILIKE ?", f.Category)
	}
	if f.MinPrice != nil {
		addCondition("price_cents >= ?", *f.MinPrice)
	}
	if f.MaxPrice != nil {
		addCondition("price_cents <= ?", *f.MaxPrice)
	}
	if f.MinTTC != nil {
		addCondition("price_ttc_cents >= ?", *f.MinTTC)
	}
	if f.MaxTTC != nil {
		addCondition("price_ttc_cents <= ?", *f.MaxTTC)
	}

	query := `SELECT reference, name, description, category, price_cents, vat_rate_bp, price_ttc_cents, created_at
		FROM products`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
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
		return nil, fmt.Errorf("search products: %w", err)
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

// Champs saisis par l'admin pour créer un produit
type ProductInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	PriceCents  int64  `json:"price_cents"`
	VATRateBP   int    `json:"vat_rate_bp"` // TVA en points de base : 2000 = 20 %
}

// Crée un produit avec une référence PDT-XXXXXX générée.
// Si la référence existe déjà (très rare), ON CONFLICT n'insère rien et on réessaie.
func (s *Store) CreateProduct(ctx context.Context, in ProductInput) (Product, error) {
	for range 3 {
		var p Product
		err := s.db.QueryRowContext(ctx,
			`INSERT INTO products (reference, name, description, category, price_cents, vat_rate_bp)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (reference) DO NOTHING
			RETURNING reference, name, description, category, price_cents, vat_rate_bp, price_ttc_cents, created_at`,
			GenerateUniqueReferenceID(PrefixProduct), in.Name, in.Description, in.Category, in.PriceCents, in.VATRateBP,
		).Scan(&p.Reference, &p.Name, &p.Description, &p.Category,
			&p.PriceCents, &p.VATRateBP, &p.PriceTTCCents, &p.CreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			continue // référence déjà prise : on en génère une autre
		}
		if err != nil {
			return Product{}, fmt.Errorf("create product: %w", err)
		}
		return p, nil
	}
	return Product{}, errors.New("create product: could not generate a unique reference")
}
