package apiclient

import "time"

type User struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

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

// Réponse de toutes les routes /cart (api/carts.go, cartResponse)
type Cart struct {
	Reference     string     `json:"reference"`
	Items         []CartItem `json:"items"`
	TotalTTCCents int64      `json:"total_ttc_cents"`
}

type CartItem struct {
	ProductReference  string `json:"product_reference"`
	Name              string `json:"name"`
	Quantity          int    `json:"quantity"`
	UnitPriceTTCCents int64  `json:"unit_price_ttc_cents"`
}

// Réponse de POST /auth/login (api/auth.go:134)
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Corps de toute réponse d'erreur du serveur (api/api.go, writeError)
type errorResponse struct {
	Error string `json:"error"`
}
