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

// Quantité maximale d'un produit par requête (évite de dépasser l'INT de PostgreSQL).
const MaxItemQuantity = 100

func (in AddCartItemInput) Validate() error {
	if in.ProductReference == "" {
		return errors.New("ProductReference est essentiel")
	}
	return ValidateQuantity(in.Quantity)
}

// Vérifie qu'une quantité est comprise entre 1 et MaxItemQuantity.
func ValidateQuantity(quantity int) error {
	if quantity < 1 || quantity > MaxItemQuantity {
		return fmt.Errorf("Quantity doit être entre 1 et %d", MaxItemQuantity)
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

// Charge le panier ouvert de l'utilisateur avec les prix TTC actuels des produits.
func (s *Store) GetOpenCart(ctx context.Context, userID int64) (Cart, error) {
	reference, err := s.GetOpenCartReference(ctx, userID)
	if err != nil {
		return Cart{}, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT p.reference, ci.quantity, p.price_ttc_cents
		FROM cart_items ci
		JOIN carts c ON c.id = ci.cart_id
		JOIN products p ON p.id = ci.product_id
		WHERE c.reference = $1 AND c.user_id = $2 AND c.status = 'open'
		ORDER BY p.reference`,
		reference, userID,
	)
	if err != nil {
		return Cart{}, fmt.Errorf("load cart items: %w", err)
	}
	defer rows.Close()

	cart := Cart{Reference: reference, Items: []CartItem{}}
	for rows.Next() {
		var item CartItem
		if err := rows.Scan(&item.ProductReference, &item.Quantity, &item.UnitPriceTTCCents); err != nil {
			return Cart{}, fmt.Errorf("scan cart item: %w", err)
		}
		cart.Items = append(cart.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Cart{}, fmt.Errorf("read cart items: %w", err)
	}
	return cart, nil
}

// Ajoute un produit au panier ouvert de l'utilisateur (le panier est créé si besoin).
// Si le produit est déjà dans le panier, on additionne les quantités.
// ErrNotFound si le produit n'existe pas.
func (s *Store) AddCartItem(ctx context.Context, userID int64, in AddCartItemInput) (Cart, error) {
	reference, err := s.GetOrCreateOpenCartReference(ctx, userID)
	if err != nil {
		return Cart{}, err
	}

	// INSERT ... SELECT : la ligne n'est insérée que si le produit existe (sinon 0 ligne).
	// ON CONFLICT : le produit est déjà dans ce panier, on augmente sa quantité.
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity)
		SELECT c.id, p.id, $3
		FROM carts c, products p
		WHERE c.reference = $1 AND p.reference = $2
		ON CONFLICT (cart_id, product_id)
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity`,
		reference, in.ProductReference, in.Quantity,
	)
	if err != nil {
		return Cart{}, fmt.Errorf("add cart item: %w", err)
	}

	added, err := result.RowsAffected()
	if err != nil {
		return Cart{}, fmt.Errorf("count added items: %w", err)
	}
	if added == 0 {
		return Cart{}, ErrNotFound
	}
	return s.GetOpenCart(ctx, userID)
}

// Remplace la quantité d'un produit du panier ouvert de l'utilisateur.
// ErrNotFound si l'utilisateur n'a pas de panier ouvert ou si le produit n'y est pas.
func (s *Store) SetCartItemQuantity(ctx context.Context, userID int64, productReference string, quantity int) (Cart, error) {
	// UPDATE ... FROM : on retrouve la ligne grâce aux références (panier de l'utilisateur + produit).
	result, err := s.db.ExecContext(ctx,
		`UPDATE cart_items ci SET quantity = $3
		FROM carts c, products p
		WHERE ci.cart_id = c.id AND ci.product_id = p.id
		AND c.user_id = $1 AND c.status = 'open' AND p.reference = $2`,
		userID, productReference, quantity,
	)
	if err != nil {
		return Cart{}, fmt.Errorf("update cart item: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return Cart{}, fmt.Errorf("count updated items: %w", err)
	}
	if updated == 0 {
		return Cart{}, ErrNotFound
	}
	return s.GetOpenCart(ctx, userID)
}

// Retire un produit du panier ouvert de l'utilisateur.
// ErrNotFound si l'utilisateur n'a pas de panier ouvert ou si le produit n'y est pas.
func (s *Store) RemoveCartItem(ctx context.Context, userID int64, productReference string) (Cart, error) {
	// DELETE ... USING : même principe que UPDATE ... FROM.
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM cart_items ci
		USING carts c, products p
		WHERE ci.cart_id = c.id AND ci.product_id = p.id
		AND c.user_id = $1 AND c.status = 'open' AND p.reference = $2`,
		userID, productReference,
	)
	if err != nil {
		return Cart{}, fmt.Errorf("remove cart item: %w", err)
	}

	removed, err := result.RowsAffected()
	if err != nil {
		return Cart{}, fmt.Errorf("count removed items: %w", err)
	}
	if removed == 0 {
		return Cart{}, ErrNotFound
	}
	return s.GetOpenCart(ctx, userID)
}
