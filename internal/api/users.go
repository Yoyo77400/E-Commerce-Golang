package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"

	"github.com/yoyo77400/E-Commerce-Go/internal/auth"
	"github.com/yoyo77400/E-Commerce-Go/internal/store"
)

func (a *API) listUsers(w http.ResponseWriter, r *http.Request, _ store.User) {
	users, err := a.store.ListUsers(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request, _ store.User) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := a.store.GetUser(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Lit l'{id} de l'URL. Répond 400 et renvoie false s'il n'est pas un nombre.
func readUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return 0, false
	}
	return id, true
}

// Traduit les erreurs du store en réponses HTTP
func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "email already registered")
	default:
		writeInternalError(w, err)
	}
}

func isValidRole(role string) bool {
	return role == "customer" || role == "admin"
}

// POST /admin/users {"email", "password", "role", "confirmed"}
func (a *API) createUser(w http.ResponseWriter, r *http.Request, _ store.User) {
	var in struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		Role      string `json:"role"`
		Confirmed bool   `json:"confirmed"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Email = normalizeEmail(in.Email)
	if in.Role == "" {
		in.Role = "customer"
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(in.Password) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if !isValidRole(in.Role) {
		writeError(w, http.StatusBadRequest, "role must be customer or admin")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	user, err := a.store.CreateUser(r.Context(), in.Email, hash, in.Role, in.Confirmed)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// PATCH /admin/users/{id} {"email"?, "role"?, "password"?}
// Les champs sont des pointeurs : nil = champ absent du JSON = on ne le modifie pas.
func (a *API) updateUser(w http.ResponseWriter, r *http.Request, admin store.User) {
	id, ok := readUserID(w, r)
	if !ok {
		return
	}
	var in struct {
		Email    *string `json:"email"`
		Role     *string `json:"role"`
		Password *string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}

	// On part des valeurs actuelles et on remplace celles qui sont envoyées
	user, err := a.store.GetUser(r.Context(), id)
	if err != nil {
		writeUserError(w, err)
		return
	}
	email, role := user.Email, user.Role

	if in.Email != nil {
		email = normalizeEmail(*in.Email)
		if _, err := mail.ParseAddress(email); err != nil {
			writeError(w, http.StatusBadRequest, "invalid email")
			return
		}
	}
	if in.Role != nil {
		role = *in.Role
		if !isValidRole(role) {
			writeError(w, http.StatusBadRequest, "role must be customer or admin")
			return
		}
		if id == admin.ID && role != "admin" {
			writeError(w, http.StatusBadRequest, "you cannot remove your own admin role")
			return
		}
	}
	if in.Password != nil && len(*in.Password) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	user, err = a.store.UpdateUser(r.Context(), id, email, role)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if in.Password != nil {
		hash, err := auth.HashPassword(*in.Password)
		if err != nil {
			writeInternalError(w, err)
			return
		}
		if err := a.store.SetPassword(r.Context(), id, hash); err != nil {
			writeInternalError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, user)
}

// DELETE /admin/users/{id}
func (a *API) deleteUser(w http.ResponseWriter, r *http.Request, admin store.User) {
	id, ok := readUserID(w, r)
	if !ok {
		return
	}
	if id == admin.ID {
		writeError(w, http.StatusBadRequest, "you cannot delete your own account")
		return
	}
	if err := a.store.DeleteUser(r.Context(), id); err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "user deleted"})
}

// POST /admin/users/{id}/confirm
func (a *API) confirmUser(w http.ResponseWriter, r *http.Request, _ store.User) {
	id, ok := readUserID(w, r)
	if !ok {
		return
	}
	if _, err := a.store.GetUser(r.Context(), id); err != nil {
		writeUserError(w, err)
		return
	}
	if err := a.store.ConfirmUser(r.Context(), id); err != nil {
		writeInternalError(w, err)
		return
	}
	user, err := a.store.GetUser(r.Context(), id)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}
