package client

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

const (
	centsPerEuro          = 100
	basisPointsPerPercent = 100
	tableHeight           = 10
)

// Filtres tels que saisis dans le formulaire. Les prix sont en euros ("19,99") ; vide = ignoré.
type searchInputs struct {
	query    string
	category string
	minPrice string
	maxPrice string
	minTTC   string
	maxTTC   string
}

type productsResultMsg struct {
	products []apiclient.Product
	err      error
}

// ---------- Ouverture, recherche et résultats ----------

// Ouvre le catalogue sans filtre : tous les produits
func (m *Model) openCatalog() tea.Cmd {
	m.screen = screenCatalog
	m.search = searchInputs{}
	m.info = ""
	return m.startLoading(m.searchCmd())
}

func (m *Model) openFilters() tea.Cmd {
	return m.openForm("Rechercher un produit", screenCatalog, m.searchCmd,
		huh.NewInput().Title("Nom ou description").Value(&m.search.query),
		huh.NewInput().Title("Catégorie").Value(&m.search.category),
		euroField("Prix HT minimum (€)", &m.search.minPrice),
		euroField("Prix HT maximum (€)", &m.search.maxPrice),
		euroField("Prix TTC minimum (€)", &m.search.minTTC),
		euroField("Prix TTC maximum (€)", &m.search.maxTTC),
	)
}

func euroField(title string, value *string) huh.Field {
	return huh.NewInput().Title(title).Value(value).
		Validate(func(s string) error {
			_, err := parseEuros(s)
			return err
		})
}

func (m *Model) searchCmd() tea.Cmd {
	api, search := m.api, m.search
	return func() tea.Msg {
		filter, err := search.toFilter()
		if err != nil {
			return productsResultMsg{err: err}
		}
		products, err := api.SearchProducts(context.Background(), filter)
		return productsResultMsg{products: products, err: err}
	}
}

func (m *Model) handleProducts(msg productsResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.screen = screenCatalog
	m.err = msg.err
	m.products = msg.products

	rows := []table.Row{}
	for _, p := range m.products {
		rows = append(rows, table.Row{
			p.Reference, p.Name, p.Category, formatEuros(p.PriceCents), formatEuros(p.PriceTTCCents),
		})
	}
	m.table.SetRows(rows)
	m.table.SetCursor(0)
	return m, nil
}

// ---------- Écran liste ----------

func newProductTable() table.Model {
	columns := []table.Column{
		{Title: "Référence", Width: 11},
		{Title: "Nom", Width: 26},
		{Title: "Catégorie", Width: 14},
		{Title: "Prix HT", Width: 11},
		{Title: "Prix TTC", Width: 11},
	}
	styles := table.DefaultStyles()
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("#EE6FF8")).Bold(true)

	return table.New(
		table.WithColumns(columns),
		table.WithHeight(tableHeight),
		table.WithFocused(true),
		table.WithStyles(styles),
	)
}

func (m *Model) updateCatalog(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenMenu
			m.cursor = 0
			m.err = nil
			return m, nil
		case "f", "/":
			return m, m.openFilters()
		case "r":
			return m, m.openCatalog()
		case "enter":
			if len(m.products) > 0 {
				m.selected = m.products[m.table.Cursor()]
				m.screen = screenProduct
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) viewCatalog() string {
	s := titleStyle.Render("Catalogue") + "\n"
	if m.loading {
		return s + m.spinner.View() + " Recherche…"
	}

	if summary := m.search.summary(); summary != "" {
		s += helpStyle.Render("Filtres : "+summary) + "\n"
	}
	if len(m.products) == 0 && m.err == nil {
		s += "Aucun produit ne correspond à la recherche.\n"
	} else {
		s += m.table.View() + "\n"
		s += fmt.Sprintf("%d produit(s)\n", len(m.products))
	}
	s += m.renderMessages()
	s += helpStyle.Render("↑/↓ naviguer • entrée détail • f filtrer • r réinitialiser • échap menu")
	return s
}

// ---------- Écran détail ----------

func (m *Model) updateProduct(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok || m.loading {
		return m, nil
	}

	switch key.String() {
	case "esc":
		m.info = ""
		m.err = nil
		m.screen = screenCatalog
	case "a":
		m.info = ""
		return m, m.startLoading(m.addToCartCmd(m.selected.Reference))
	}
	return m, nil
}

func (m *Model) viewProduct() string {
	p := m.selected
	details := titleStyle.Render(p.Name) + "\n"
	details += fmt.Sprintf("Référence : %s\n", p.Reference)
	details += fmt.Sprintf("Catégorie : %s\n", p.Category)
	if p.Description != "" {
		details += "\n" + p.Description + "\n"
	}
	details += fmt.Sprintf("\nPrix HT   : %s\n", formatEuros(p.PriceCents))
	details += fmt.Sprintf("TVA       : %s\n", formatVAT(p.VATRateBP))
	details += selectedStyle.Render("Prix TTC  : "+formatEuros(p.PriceTTCCents)) + "\n"
	details += helpStyle.Render("Livraison gratuite")

	s := boxStyle.Render(details) + "\n"
	if m.loading {
		s += m.spinner.View() + " Ajout au panier…"
	}
	s += m.renderMessages()
	return s + helpStyle.Render("a ajouter au panier • échap retour au catalogue")
}

// ---------- Conversions ----------

// "19,99" ou "19.99" -> 1999 centimes. Une saisie vide vaut 0 (= filtre ignoré).
func parseEuros(s string) (int64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil || value < 0 {
		return 0, errors.New("montant invalide (exemple : 19,99)")
	}
	return int64(math.Round(value * centsPerEuro)), nil
}

// 1999 -> "19,99 €"
func formatEuros(cents int64) string {
	return fmt.Sprintf("%d,%02d €", cents/centsPerEuro, cents%centsPerEuro)
}

// TVA en points de base : 2000 -> "20 %", 550 -> "5.5 %"
func formatVAT(basisPoints int) string {
	return fmt.Sprintf("%g %%", float64(basisPoints)/basisPointsPerPercent)
}

func (s searchInputs) toFilter() (apiclient.ProductFilter, error) {
	minPrice, err := parseEuros(s.minPrice)
	if err != nil {
		return apiclient.ProductFilter{}, err
	}
	maxPrice, err := parseEuros(s.maxPrice)
	if err != nil {
		return apiclient.ProductFilter{}, err
	}
	minTTC, err := parseEuros(s.minTTC)
	if err != nil {
		return apiclient.ProductFilter{}, err
	}
	maxTTC, err := parseEuros(s.maxTTC)
	if err != nil {
		return apiclient.ProductFilter{}, err
	}
	return apiclient.ProductFilter{
		Query:    strings.TrimSpace(s.query),
		Category: strings.TrimSpace(s.category),
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		MinTTC:   minTTC,
		MaxTTC:   maxTTC,
	}, nil
}

// Résumé des filtres actifs, affiché au-dessus du tableau
func (s searchInputs) summary() string {
	var parts []string
	if s.query != "" {
		parts = append(parts, "« "+s.query+" »")
	}
	if s.category != "" {
		parts = append(parts, "catégorie "+s.category)
	}
	if s.minPrice != "" {
		parts = append(parts, "HT ≥ "+s.minPrice+" €")
	}
	if s.maxPrice != "" {
		parts = append(parts, "HT ≤ "+s.maxPrice+" €")
	}
	if s.minTTC != "" {
		parts = append(parts, "TTC ≥ "+s.minTTC+" €")
	}
	if s.maxTTC != "" {
		parts = append(parts, "TTC ≤ "+s.maxTTC+" €")
	}
	return strings.Join(parts, " • ")
}
