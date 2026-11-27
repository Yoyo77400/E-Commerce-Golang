package client

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

// Résultat de toute action sur le panier : le serveur renvoie le panier complet à jour
type cartResultMsg struct {
	cart apiclient.Cart
	info string // message affiché en cas de succès
	err  error
}

type payResultMsg struct {
	reference string
	err       error
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
	case "p":
		m.resetInputs()
		return m, m.openPayment()
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
		line := formatItemLine(item)
		if i == m.cursor {
			s += selectedStyle.Render("> "+line) + "\n"
		} else {
			s += "  " + line + "\n"
		}
	}
	s += "\n" + selectedStyle.Render("Total TTC : "+formatEuros(m.cart.TotalTTCCents)) + "\n"
	s += helpStyle.Render("Livraison gratuite")
	s += m.renderMessages()
	return s + helpStyle.Render("↑/↓ naviguer • +/- quantité • x retirer • p payer • échap menu")
}

func formatItemLine(item apiclient.CartItem) string {
	subtotal := item.UnitPriceTTCCents * int64(item.Quantity)
	return fmt.Sprintf("%-26s x%-3d %12s", item.Name, item.Quantity, formatEuros(subtotal))
}

// ---------- Paiement ----------

func (m *Model) openPayment() tea.Cmd {
	return m.openForm("Paiement — "+formatEuros(m.cart.TotalTTCCents), screenCart, m.payCmd,
		huh.NewInput().Title("Numéro de carte").Placeholder("16 chiffres").Value(&m.cardNumber),
		huh.NewInput().Title("Date d'expiration").Placeholder("MM/AA").Value(&m.cardExpiry),
		passwordField("CVC", &m.cardCVC),
	)
}

func (m *Model) payCmd() tea.Cmd {
	api, number, expiry, cvc := m.api, m.cardNumber, m.cardExpiry, m.cardCVC
	return func() tea.Msg {
		reference, err := api.PayCart(context.Background(), number, expiry, cvc)
		return payResultMsg{reference: reference, err: err}
	}
}

// En cas d'erreur, on rouvre le formulaire (les saisies sont conservées) avec le message.
// En cas de succès, le panier payé est clôturé côté serveur : on affiche un panier vide.
func (m *Model) handlePay(msg payResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, m.openPayment()
	}
	m.resetInputs()
	m.cart = apiclient.Cart{}
	m.cursor = 0
	m.screen = screenCart
	m.info = "Paiement accepté, commande " + msg.reference + "."
	return m, nil
}
