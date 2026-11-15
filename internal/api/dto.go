// Package api expose la couche HTTP, construite uniquement avec net/http.
//
// Les types ci-dessous forment le contrat échangé sur le réseau entre le
// serveur et les deux CLI ; internal/apiclient les réutilise, donc un
// changement casse la compilation au lieu de casser à l'exécution.
package api

import "time"

// ErrorResponse est le corps de toute réponse hors 2xx.
type ErrorResponse struct {
	Error string `json:"error"`
}

// --- authentification ---

// RegisterRequest crée un compte.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ConfirmRequest valide le code reçu après l'inscription.
type ConfirmRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// LoginRequest ouvre une session.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse transporte le jeton bearer utilisé par tous les appels
// suivants.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserDTO   `json:"user"`
}

// PasswordResetRequest demande un code de réinitialisation, sans être
// connecté.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirm applique le nouveau mot de passe.
type PasswordResetConfirm struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// --- utilisateurs ---

// UserDTO est la vue publique d'un utilisateur : le hash du mot de passe ne
// circule jamais sur le réseau.
type UserDTO struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Confirmed bool      `json:"confirmed"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest est le formulaire administrateur de création de compte.
type CreateUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	Confirmed bool   `json:"confirmed"`
}

// UpdateUserRequest contient les champs qu'un administrateur peut modifier.
// Un champ nil signifie « ne pas changer ».
type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
}

// --- produits ---

// ProductDTO est une fiche du catalogue. Les prix sont en centimes.
type ProductDTO struct {
	ID          int64  `json:"id"`
	Reference   string `json:"reference"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	PriceHT     int64  `json:"price_ht"`
	VATRate     int    `json:"vat_rate"`
	PriceTTC    int64  `json:"price_ttc"`
	Stock       int    `json:"stock"`
}

// CreateProductRequest est le formulaire administrateur d'ajout de produit.
// La référence métier est générée côté serveur.
type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	PriceHT     int64  `json:"price_ht"`
	VATRate     int    `json:"vat_rate"`
	Stock       int    `json:"stock"`
}

// --- panier ---

// CartItemDTO est une ligne du panier.
type CartItemDTO struct {
	ProductID   int64  `json:"product_id"`
	Reference   string `json:"reference"`
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	UnitPriceHT int64  `json:"unit_price_ht"`
	VATRate     int    `json:"vat_rate"`
	LineTTC     int64  `json:"line_ttc"`
}

// CartDTO est le panier de l'utilisateur authentifié. La livraison est
// toujours gratuite, donc TotalTTC est le montant réellement débité.
type CartDTO struct {
	Reference string        `json:"reference"`
	Items     []CartItemDTO `json:"items"`
	TotalTTC  int64         `json:"total_ttc"`
}

// AddCartItemRequest ajoute des unités d'un produit au panier.
type AddCartItemRequest struct {
	ProductReference string `json:"product_reference"`
	Quantity         int    `json:"quantity"`
}

// SetCartItemRequest remplace la quantité d'une ligne ; zéro la supprime.
type SetCartItemRequest struct {
	Quantity int `json:"quantity"`
}

// --- paiement et commandes ---

// CheckoutRequest paie le panier enregistré. Ces champs sont validés puis
// jetés : seuls les quatre derniers chiffres arrivent en base.
type CheckoutRequest struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
	CVC         string `json:"cvc"`
}

// OrderItemDTO est une ligne figée d'une commande.
type OrderItemDTO struct {
	Reference   string `json:"reference"`
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	UnitPriceHT int64  `json:"unit_price_ht"`
	VATRate     int    `json:"vat_rate"`
	LineTTC     int64  `json:"line_ttc"`
}

// OrderDTO est une commande telle qu'affichée par les CLI client et admin.
type OrderDTO struct {
	Reference      string         `json:"reference"`
	UserID         int64          `json:"user_id"`
	Status         string         `json:"status"`
	CancelReason   string         `json:"cancel_reason,omitempty"`
	TotalTTC       int64          `json:"total_ttc"`
	PaymentCardEnd string         `json:"payment_card_end,omitempty"`
	Items          []OrderItemDTO `json:"items"`
	CreatedAt      time.Time      `json:"created_at"`
}

// UpdateOrderStatusRequest est le changement de statut fait par un
// administrateur.
type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// AdminCreateOrderRequest permet à un administrateur de créer une commande
// pour un client.
type AdminCreateOrderRequest struct {
	UserID int64                 `json:"user_id"`
	Items  []AdminOrderLineInput `json:"items"`
	Status string                `json:"status"`
}

// AdminOrderLineInput est une ligne d'une commande créée par un
// administrateur.
type AdminOrderLineInput struct {
	ProductReference string `json:"product_reference"`
	Quantity         int    `json:"quantity"`
}
