package api

import (
	"net/http"
	"time"

	"github.com/yoyo77400/E-Commerce-Golang/internal/store"
)

type orderResponse struct {
	Reference     string           `json:"reference"`
	Status        string           `json:"status"`
	CancelReason  *string          `json:"cancel_reason"`
	CreatedAt     time.Time        `json:"created_at"`
	Items         []store.CartItem `json:"items"`
	TotalTTCCents int64            `json:"total_ttc_cents"`
}

// GET /orders : les commandes de l'utilisateur connecté.
func (a *API) listOrders(w http.ResponseWriter, r *http.Request, user store.User) {
	orders, err := a.store.ListUserOrders(r.Context(), user.ID)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	response := make([]orderResponse, len(orders))
	for i, o := range orders {
		response[i] = orderResponse{
			Reference:     o.Reference,
			Status:        o.Status,
			CancelReason:  o.CancelReason,
			CreatedAt:     o.CreatedAt,
			Items:         o.Items,
			TotalTTCCents: o.TotalTTCCents(),
		}
	}
	writeJSON(w, http.StatusOK, response)
}
