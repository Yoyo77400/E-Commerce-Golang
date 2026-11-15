package client

import "testing"

func TestParseEuros(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"vide = filtre ignoré", "", 0, false},
		{"virgule", "19,99", 1999, false},
		{"point", "19.99", 1999, false},
		{"entier", "20", 2000, false},
		{"espaces", "  5,5 ", 550, false},
		{"arrondi au centime", "0,105", 11, false},
		{"négatif", "-3", 0, true},
		{"texte", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEuros(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseEuros(%q) erreur = %v, erreur attendue = %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseEuros(%q) = %d ; attendu %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatEuros(t *testing.T) {
	tests := []struct {
		cents int64
		want  string
	}{
		{1999, "19,99 €"},
		{2000, "20,00 €"},
		{5, "0,05 €"},
		{0, "0,00 €"},
	}
	for _, tt := range tests {
		if got := formatEuros(tt.cents); got != tt.want {
			t.Errorf("formatEuros(%d) = %q ; attendu %q", tt.cents, got, tt.want)
		}
	}
}

func TestFormatVAT(t *testing.T) {
	tests := []struct {
		basisPoints int
		want        string
	}{
		{2000, "20 %"},
		{1000, "10 %"},
		{550, "5.5 %"},
	}
	for _, tt := range tests {
		if got := formatVAT(tt.basisPoints); got != tt.want {
			t.Errorf("formatVAT(%d) = %q ; attendu %q", tt.basisPoints, got, tt.want)
		}
	}
}
