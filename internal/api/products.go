package api

import (
	"errors"
	"net/http"

	"github.com/yoyo77400/E-Commerce-Go/internal/store"
)

func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := a.store.ListProducts(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	product, err := a.store.GetProduct(r.Context(), r.PathValue("reference"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}
