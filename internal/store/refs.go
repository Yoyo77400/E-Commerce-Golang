package store

import "crypto/rand"

// Prefixes des identifiants métiers
const (
	PrefixProduct = "PDT"
	PrefixCart    = "BSK"
	PrefixOrder   = "CMD"
)

// Generer un identifiant métier unique
func GenerateUniqueReferenceID(prefix string) string { return prefix + "-" + rand.Text()[:6] }

// Code de confirmation ou de réinitialisation
func GenerateCodeForConfirmOrReset() string { return rand.Text()[:8] }

// Génère un jeton de session.
func GenerateNewSessionToken() string { return rand.Text() }
