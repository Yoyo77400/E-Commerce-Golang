package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
	"github.com/yoyo77400/E-Commerce-Golang/internal/tui/client"
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

	program := tea.NewProgram(client.New(apiclient.New(*apiURL)))
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
