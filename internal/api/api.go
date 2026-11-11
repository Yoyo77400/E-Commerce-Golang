package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/yoyo77400/E-Commerce-Go/internal/store"
)

type API struct {
	store *store.Store
}

func Register(mux *http.ServeMux, st *store.Store) {
	a := &API{store: st}

	// Authentification (sans être connecté)
	mux.HandleFunc("POST /auth/register", a.register)
	mux.HandleFunc("POST /auth/confirm", a.confirm)
	mux.HandleFunc("POST /auth/login", a.login)
	mux.HandleFunc("POST /auth/password/forgot", a.forgotPassword)
	mux.HandleFunc("POST /auth/password/reset", a.resetPassword)

	// Utilisateur connecté
	mux.HandleFunc("POST /auth/logout", a.requireUser(a.logout))
	mux.HandleFunc("GET /me", a.requireUser(a.me))

	mux.HandleFunc("GET /products", a.listProducts)
	mux.HandleFunc("GET /products/{reference}", a.getProduct)

	// Administration (admin connecté)
	mux.HandleFunc("POST /admin/products", a.requireAdmin(a.createProduct))
	mux.HandleFunc("GET /admin/users", a.requireAdmin(a.listUsers))
	mux.HandleFunc("GET /admin/users/{id}", a.requireAdmin(a.getUser))
}

// Handler qui reçoit en plus l'utilisateur connecté
type userHandler func(w http.ResponseWriter, r *http.Request, user store.User)

// Middleware : lit le jeton dans "Authorization: Bearer <jeton>", retrouve l'utilisateur,
// puis appelle le handler. Sinon répond 401.
func (a *API) requireUser(next userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		user, err := a.store.UserBySession(r.Context(), token)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid or expired session")
			return
		}
		if err != nil {
			writeInternalError(w, err)
			return
		}
		next(w, r, user)
	}
}

// Middleware : comme requireUser, mais refuse (403) si l'utilisateur n'est pas admin
func (a *API) requireAdmin(next userHandler) http.HandlerFunc {
	return a.requireUser(func(w http.ResponseWriter, r *http.Request, user store.User) {
		if user.Role != "admin" {
			writeError(w, http.StatusForbidden, "admin only")
			return
		}
		next(w, r, user)
	})
}

func bearerToken(r *http.Request) (string, bool) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return token, ok && token != ""
}

// Décode le corps JSON de la requête dans v. Répond 400 et renvoie false si invalide.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
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
