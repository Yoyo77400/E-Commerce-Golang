package api

import (
	"errors"
	"net/http"
	"strconv"

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
