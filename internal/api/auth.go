package api

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/yoyo77400/E-Commerce-Go/internal/auth"
	"github.com/yoyo77400/E-Commerce-Go/internal/store"
)

const (
	sessionDuration     = 24 * time.Hour
	confirmCodeDuration = 24 * time.Hour
	resetCodeDuration   = 30 * time.Minute
	minPasswordLength   = 8
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Simule l'envoi d'un email : le message est écrit dans les logs du serveur
// (visible avec `docker compose logs server`).
func sendMail(to, subject, body string) {
	log.Printf("[MAIL] to=%s subject=%q body=%q", to, subject, body)
}

// Met l'email en minuscules et retire les espaces autour
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ---------- Inscription ----------

// POST /auth/register {"email", "password"}
func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !readJSON(w, r, &in) {
		return
	}
	in.Email = normalizeEmail(in.Email)
	if _, err := mail.ParseAddress(in.Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(in.Password) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	user, err := a.store.CreateUser(r.Context(), in.Email, hash)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}

	code, err := a.store.CreateCode(r.Context(), user.ID, store.CodeConfirm, confirmCodeDuration)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	sendMail(user.Email, "Confirmez votre inscription", "Votre code de confirmation : "+code)

	writeJSON(w, http.StatusCreated, user)
}

// POST /auth/confirm {"code"}
func (a *API) confirm(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	userID, err := a.store.UseCode(r.Context(), store.CodeConfirm, strings.TrimSpace(in.Code))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	if err := a.store.ConfirmUser(r.Context(), userID); err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "account confirmed"})
}

// ---------- Connexion ----------

// POST /auth/login {"email", "password"} -> {"token", "user"}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !readJSON(w, r, &in) {
		return
	}
	user, hash, err := a.store.GetUserByEmail(r.Context(), normalizeEmail(in.Email))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeInternalError(w, err)
		return
	}
	// Même message si l'email n'existe pas ou si le mot de passe est faux :
	// on ne révèle pas quels emails ont un compte.
	if errors.Is(err, store.ErrNotFound) || !auth.CheckPassword(in.Password, hash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if user.ConfirmedAt == nil {
		writeError(w, http.StatusForbidden, "account not confirmed")
		return
	}

	token, err := a.store.CreateSession(r.Context(), user.ID, sessionDuration)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

// POST /auth/logout (connecté)
func (a *API) logout(w http.ResponseWriter, r *http.Request, _ store.User) {
	token, _ := bearerToken(r)
	if err := a.store.DeleteSession(r.Context(), token); err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// GET /me (connecté) : renvoie l'utilisateur connecté
func (a *API) me(w http.ResponseWriter, _ *http.Request, user store.User) {
	writeJSON(w, http.StatusOK, user)
}

// ---------- Mot de passe oublié ----------

// POST /auth/password/forgot {"email"}
// Répond toujours la même chose, que l'email existe ou non.
func (a *API) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	user, _, err := a.store.GetUserByEmail(r.Context(), normalizeEmail(in.Email))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeInternalError(w, err)
		return
	}
	if err == nil {
		code, err := a.store.CreateCode(r.Context(), user.ID, store.CodeReset, resetCodeDuration)
		if err != nil {
			writeInternalError(w, err)
			return
		}
		sendMail(user.Email, "Réinitialisation du mot de passe", "Votre code : "+code)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "if this email exists, a code has been sent"})
}

// POST /auth/password/reset {"code", "password"}
func (a *API) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.Password) < minPasswordLength {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	userID, err := a.store.UseCode(r.Context(), store.CodeReset, strings.TrimSpace(in.Code))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	if err := a.store.SetPassword(r.Context(), userID, hash); err != nil {
		writeInternalError(w, err)
		return
	}
	// Les anciennes sessions ne doivent plus fonctionner avec l'ancien mot de passe
	if err := a.store.DeleteUserSessions(r.Context(), userID); err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "password updated"})
}
