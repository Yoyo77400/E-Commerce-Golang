package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/yoyo77400/E-Commerce-Golang/internal/store"
)

// Ce que le client reçoit : le panier et son total TTC (livraison toujours gratuite).
type cartResponse struct {
	Reference     string           `json:"reference"`
	Items         []store.CartItem `json:"items"`
	TotalTTCCents int64            `json:"total_ttc_cents"`
}

func writeCart(w http.ResponseWriter, status int, cart store.Cart) {
	writeJSON(w, status, cartResponse{
		Reference:     cart.Reference,
		Items:         cart.Items,
		TotalTTCCents: cart.TotalTTCCents(),
	})
}

// GET /cart : le panier ouvert de l'utilisateur.
// S'il n'en a pas encore, on renvoie un panier vide sans le créer (un GET ne modifie rien).
func (a *API) getCart(w http.ResponseWriter, r *http.Request, user store.User) {
	cart, err := a.store.GetOpenCart(r.Context(), user.ID)
	if errors.Is(err, store.ErrNotFound) {
		cart = store.Cart{Items: []store.CartItem{}}
	} else if err != nil {
		writeInternalError(w, err)
		return
	}
	writeCart(w, http.StatusOK, cart)
}

// POST /cart/items {"product_reference", "quantity"}
// Ajoute le produit (ou augmente sa quantité) et renvoie le panier à jour.
func (a *API) addCartItem(w http.ResponseWriter, r *http.Request, user store.User) {
	var in store.AddCartItemInput
	if !readJSON(w, r, &in) {
		return
	}
	in.ProductReference = strings.TrimSpace(in.ProductReference)
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cart, err := a.store.AddCartItem(r.Context(), user.ID, in)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeCart(w, http.StatusOK, cart)
}

// PATCH /cart/items/{reference} {"quantity"}
// Remplace la quantité du produit (pour le retirer : DELETE).
func (a *API) updateCartItem(w http.ResponseWriter, r *http.Request, user store.User) {
	var in struct {
		Quantity int `json:"quantity"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if err := store.ValidateQuantity(in.Quantity); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cart, err := a.store.SetCartItemQuantity(r.Context(), user.ID, r.PathValue("reference"), in.Quantity)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not in cart")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeCart(w, http.StatusOK, cart)
}

// DELETE /cart/items/{reference}
// Retire le produit du panier et renvoie le panier à jour.
func (a *API) removeCartItem(w http.ResponseWriter, r *http.Request, user store.User) {
	cart, err := a.store.RemoveCartItem(r.Context(), user.ID, r.PathValue("reference"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not in cart")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeCart(w, http.StatusOK, cart)
}
