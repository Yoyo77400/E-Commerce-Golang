package client

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

// Résultat de toute action sur le panier : le serveur renvoie le panier complet à jour
type cartResultMsg struct {
	cart apiclient.Cart
	info string // message affiché en cas de succès
	err  error
}

func (m *Model) openCart() tea.Cmd {
	m.screen = screenCart
	m.cursor = 0
	m.info = ""
	api := m.api
	return m.startLoading(func() tea.Msg {
		cart, err := api.GetCart(context.Background())
		return cartResultMsg{cart: cart, err: err}
	})
}

func (m *Model) addToCartCmd(reference string) tea.Cmd {
	api := m.api
	return func() tea.Msg {
		cart, err := api.AddCartItem(context.Background(), reference, 1)
		return cartResultMsg{cart: cart, info: "Ajouté au panier.", err: err}
	}
}

func (m *Model) handleCart(msg cartResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.err = msg.err
	if msg.err == nil {
		m.cart = msg.cart
		m.info = msg.info
	}
	// Après un retrait, le curseur ne doit pas pointer après la dernière ligne
	if m.cursor >= len(m.cart.Items) && m.cursor > 0 {
		m.cursor = len(m.cart.Items) - 1
	}
	return m, nil
}

// Quantité 0 = retirer le produit du panier
func (m *Model) setQuantityCmd(reference string, quantity int) tea.Cmd {
	api := m.api
	return func() tea.Msg {
		var cart apiclient.Cart
		var err error
		if quantity == 0 {
			cart, err = api.RemoveCartItem(context.Background(), reference)
		} else {
			cart, err = api.UpdateCartItem(context.Background(), reference, quantity)
		}
		return cartResultMsg{cart: cart, err: err}
	}
}

func (m *Model) updateCart(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok || m.loading {
		return m, nil
	}
	if key.String() == "esc" {
		m.screen = screenMenu
		m.cursor = 0
		m.err, m.info = nil, ""
		return m, nil
	}
	if len(m.cart.Items) == 0 {
		return m, nil
	}

	item := m.cart.Items[m.cursor]
	quantity := item.Quantity
	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, len(m.cart.Items))
		return m, nil
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, len(m.cart.Items))
		return m, nil
	case "+":
		quantity++
	case "-":
		quantity--
	case "delete", "x":
		quantity = 0
	default:
		return m, nil
	}
	m.info = ""
	return m, m.startLoading(m.setQuantityCmd(item.ProductReference, quantity))
}

func (m *Model) viewCart() string {
	s := titleStyle.Render("Mon panier "+m.cart.Reference) + "\n"
	if m.loading {
		return s + m.spinner.View() + " Chargement…"
	}

	if len(m.cart.Items) == 0 {
		s += "Votre panier est vide.\n"
	}
	for i, item := range m.cart.Items {
		subtotal := item.UnitPriceTTCCents * int64(item.Quantity)
		line := fmt.Sprintf("%-26s x%-3d %12s", item.Name, item.Quantity, formatEuros(subtotal))
		if i == m.cursor {
			s += selectedStyle.Render("> "+line) + "\n"
		} else {
			s += "  " + line + "\n"
		}
	}
	s += "\n" + selectedStyle.Render("Total TTC : "+formatEuros(m.cart.TotalTTCCents)) + "\n"
	s += helpStyle.Render("Livraison gratuite")
	s += m.renderMessages()
	return s + helpStyle.Render("↑/↓ naviguer • +/- quantité • x retirer • échap menu")
}
