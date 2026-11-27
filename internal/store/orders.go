package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	cardNumberPattern = regexp.MustCompile(`^[0-9]{16}$`)
	cvcPattern        = regexp.MustCompile(`^[0-9]{3}$`)
)

type Order struct {
	Reference    string     `json:"reference"`
	Status       string     `json:"status"`
	CancelReason *string    `json:"cancel_reason"`
	CreatedAt    time.Time  `json:"created_at"`
	Items        []CartItem `json:"items"`
}

func (o Order) TotalTTCCents() int64 {
	return Cart{Items: o.Items}.TotalTTCCents()
}

// Informations bancaires saisies pour payer le panier
type PaymentInput struct {
	CardNumber string `json:"card_number"`
	Expiry     string `json:"expiry"` // MM/AA
	CVC        string `json:"cvc"`
}

func (in PaymentInput) Validate() error {
	if !cardNumberPattern.MatchString(in.CardNumber) {
		return errors.New("le numéro de carte doit contenir 16 chiffres")
	}
	expiry, err := time.Parse("01/06", in.Expiry)
	if err != nil {
		return errors.New("la date d'expiration doit être au format MM/AA")
	}
	// La carte reste valable jusqu'à la fin de son mois d'expiration.
	if !time.Now().Before(expiry.AddDate(0, 1, 0)) {
		return errors.New("la carte est expirée")
	}
	if !cvcPattern.MatchString(in.CVC) {
		return errors.New("le CVC doit contenir 3 chiffres")
	}
	return nil
}

// Paie le panier ouvert de l'utilisateur : crée une commande payée contenant ses produits
// (prix et TVA figés au moment du paiement), puis clôture le panier. Renvoie la référence de la commande.
// ErrNotFound si l'utilisateur n'a pas de panier ouvert ou si son panier est vide.
func (s *Store) PayCart(ctx context.Context, userID int64, in PaymentInput) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin payment: %w", err)
	}
	defer tx.Rollback()

	// FOR UPDATE : un second paiement simultané du même panier attend la fin de celui-ci,
	// puis ne le trouve plus ouvert.
	var cartID int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM carts WHERE user_id = $1 AND status = 'open' FOR UPDATE`, userID,
	).Scan(&cartID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lock cart: %w", err)
	}

	reference := GenerateUniqueReferenceID(PrefixOrder)
	var orderID int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO orders (reference, user_id, cart_id, status, card_last4, paid_at)
		VALUES ($1, $2, $3, 'paid', $4, now())
		RETURNING id`,
		reference, userID, cartID, in.CardNumber[len(in.CardNumber)-4:],
	).Scan(&orderID)
	if err != nil {
		return "", fmt.Errorf("create order: %w", err)
	}

	result, err := tx.ExecContext(ctx,
		`INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, vat_rate_bp)
		SELECT $1, p.id, ci.quantity, p.price_cents, p.vat_rate_bp
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $2`,
		orderID, cartID,
	)
	if err != nil {
		return "", fmt.Errorf("copy cart items: %w", err)
	}

	copied, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("count copied items: %w", err)
	}
	if copied == 0 {
		return "", ErrNotFound
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE carts SET status = 'paid', updated_at = now() WHERE id = $1`, cartID,
	); err != nil {
		return "", fmt.Errorf("close cart: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit payment: %w", err)
	}
	return reference, nil
}

// Commandes de l'utilisateur, de la plus récente à la plus ancienne.
// Le prix TTC est recalculé à partir du prix et de la TVA figés au paiement.
func (s *Store) ListUserOrders(ctx context.Context, userID int64) ([]Order, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.reference, o.status, o.cancel_reason, o.created_at,
			p.reference, p.name, oi.quantity,
			(oi.unit_price_cents * (10000 + oi.vat_rate_bp) + 5000) / 10000
		FROM orders o
		JOIN order_items oi ON oi.order_id = o.id
		JOIN products p ON p.id = oi.product_id
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC, o.id DESC, p.reference`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		var item CartItem
		err := rows.Scan(&o.Reference, &o.Status, &o.CancelReason, &o.CreatedAt,
			&item.ProductReference, &item.Name, &item.Quantity, &item.UnitPriceTTCCents)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		// Les lignes d'une même commande se suivent : on les regroupe.
		if n := len(orders); n > 0 && orders[n-1].Reference == o.Reference {
			orders[n-1].Items = append(orders[n-1].Items, item)
			continue
		}
		o.Items = []CartItem{item}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}
