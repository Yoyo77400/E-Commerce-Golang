package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/yoyo77400/E-Commerce-Go/internal/store"
)

type API struct {
	store *store.Store
}

func Register(mux *http.ServeMux, st *store.Store) {
	a := &API{store: st}

	mux.HandleFunc("GET /products", a.listProducts)
	mux.HandleFunc("GET /products/{reference}", a.getProduct)

	mux.HandleFunc("GET /admin/users", a.listUsers)
	mux.HandleFunc("GET /admin/users/{id}", a.getUser)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeInternalError logge la cause réelle et ne renvoie au client qu'un message générique.
func writeInternalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
