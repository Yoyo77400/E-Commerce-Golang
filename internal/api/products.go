package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/yoyo77400/E-Commerce-Go/internal/store"
)

// GET /products?q=&category=&min_price=&max_price=&min_ttc=&max_ttc=
// Tous les filtres sont optionnels. Les prix sont en centimes.
func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := store.ProductFilter{
		Query:    strings.TrimSpace(query.Get("q")),
		Category: strings.TrimSpace(query.Get("category")),
	}

	var ok bool
	if filter.MinPrice, ok = readPriceParam(w, r, "min_price"); !ok {
		return
	}
	if filter.MaxPrice, ok = readPriceParam(w, r, "max_price"); !ok {
		return
	}
	if filter.MinTTC, ok = readPriceParam(w, r, "min_ttc"); !ok {
		return
	}
	if filter.MaxTTC, ok = readPriceParam(w, r, "max_ttc"); !ok {
		return
	}

	products, err := a.store.SearchProducts(r.Context(), filter)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

// Lit un prix (en centimes) dans l'URL.
// Renvoie nil s'il est absent, et répond 400 (ok = false) s'il est invalide.
func readPriceParam(w http.ResponseWriter, r *http.Request, name string) (*int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		writeError(w, http.StatusBadRequest, name+" must be a positive number of cents")
		return nil, false
	}
	return &value, true
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

// POST /admin/products {"name", "description", "category", "price_cents", "vat_rate_bp"}
func (a *API) createProduct(w http.ResponseWriter, r *http.Request, _ store.User) {
	var in store.ProductInput
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Category = strings.TrimSpace(in.Category)

	if in.Name == "" || in.Category == "" {
		writeError(w, http.StatusBadRequest, "name and category are required")
		return
	}
	if in.PriceCents < 0 || in.VATRateBP < 0 {
		writeError(w, http.StatusBadRequest, "price_cents and vat_rate_bp must be positive")
		return
	}

	product, err := a.store.CreateProduct(r.Context(), in)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, product)
}
