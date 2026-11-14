package main

import (
	"fmt"
	"os"
	"flag"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

)

func main() {
	defaultURL := os.Getenv("API_URL")
	if defaultURL == "" {
		defaultURL = "http://localhost:8080"
	}

	// Utilisé pour définir un paramètre au lancement du client (-api ou --api)
	// Definit ensuite sa valeur par défaut
	// Affiche le texte avec l'option -h
	// Le Parse() sert à lire les arguments
	apiURL := flag.String("api", defaultURL, "base URL of the API")
	flag.Parse()

}