package client

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

const dateLayout = "02/01/2006"

var statusLabels = map[string]string{
	"pending":   "En attente",
	"paid":      "Payée",
	"shipping":  "En cours de livraison",
	"delivered": "Livrée",
	"cancelled": "Annulée",
}

type ordersResultMsg struct {
	orders []apiclient.Order
	err    error
}

func (m *Model) openOrders() tea.Cmd {
	m.screen = screenOrders
	m.cursor = 0
	m.info = ""
	api := m.api
	return m.startLoading(func() tea.Msg {
		orders, err := api.ListOrders(context.Background())
		return ordersResultMsg{orders: orders, err: err}
	})
}

func (m *Model) handleOrders(msg ordersResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.err = msg.err
	m.orders = msg.orders
	return m, nil
}

func (m *Model) updateOrders(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	if len(m.orders) == 0 {
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, len(m.orders))
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, len(m.orders))
	}
	return m, nil
}

func (m *Model) viewOrders() string {
	s := titleStyle.Render("Mes commandes") + "\n"
	if m.loading {
		return s + m.spinner.View() + " Chargement…"
	}

	if len(m.orders) == 0 && m.err == nil {
		s += "Aucune commande.\n"
	}
	for i, o := range m.orders {
		line := fmt.Sprintf("%s  %s  %-22s %12s",
			o.Reference, o.CreatedAt.Local().Format(dateLayout), statusLabels[o.Status], formatEuros(o.TotalTTCCents))
		if i == m.cursor {
			s += selectedStyle.Render("> "+line) + "\n"
		} else {
			s += "  " + line + "\n"
		}
	}
	if len(m.orders) > 0 {
		s += "\n" + boxStyle.Render(orderDetails(m.orders[m.cursor])) + "\n"
	}
	s += m.renderMessages()
	return s + helpStyle.Render("↑/↓ naviguer • échap menu")
}

func orderDetails(o apiclient.Order) string {
	s := ""
	for _, item := range o.Items {
		s += formatItemLine(item) + "\n"
	}
	s += selectedStyle.Render("Total TTC : " + formatEuros(o.TotalTTCCents))
	if o.Status == "cancelled" && o.CancelReason != nil {
		s += "\n" + errorStyle.Render("Motif d'annulation : "+*o.CancelReason)
	}
	return s
}
