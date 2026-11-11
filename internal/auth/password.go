// Package auth gère le hachage des mots de passe (bibliothèque standard uniquement).
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

const (
	iterations = 600_000 // nombre de répétitions
	saltLength = 16      // taille du sel en octets
	hashLength = 32      // taille du hash en octets
)

// Renvoie "salt$hash" en hexa => stocker en base
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	rand.Read(salt)

	hash, err := pbkdf2.Key(sha256.New, password, salt, iterations, hashLength)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(salt) + "$" + hex.EncodeToString(hash), nil
}

// Recalcule le hash avec le salt stocké et compare avec le hash en base
func CheckPassword(password, stored string) bool {
	saltHex, hashHex, ok := strings.Cut(stored, "$")
	if !ok {
		return false
	}
	salt, err1 := hex.DecodeString(saltHex)
	want, err2 := hex.DecodeString(hashHex)
	if err1 != nil || err2 != nil {
		return false
	}

	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, hashLength)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
