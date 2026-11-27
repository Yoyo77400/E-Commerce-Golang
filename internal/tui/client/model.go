package client

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

// Écrans de l'application
const (
	screenHome    = "home"
	screenForm    = "form"
	screenMenu    = "menu"
	screenCatalog = "catalog"
	screenProduct = "product"
	screenCart    = "cart"
)

const (
	choiceLogin    = "Se connecter"
	choiceRegister = "Créer un compte"
	choiceConfirm  = "Confirmer mon compte"
	choiceForgot   = "Mot de passe oublié"
	choiceQuit     = "Quitter"
	choiceCatalog  = "Catalogue"
	choiceCart     = "Mon panier"
	choiceLogout   = "Se déconnecter"
)

var homeChoices = []string{choiceLogin, choiceRegister, choiceConfirm, choiceForgot, choiceQuit}
var menuChoices = []string{choiceCatalog, choiceCart, choiceLogout}

type Model struct {
	api    *apiclient.Client
	screen string
	cursor int

	// Écran formulaire : le formulaire affiché et l'action lancée à sa validation
	form      *huh.Form
	formTitle string
	formBack  string // écran affiché si on quitte le formulaire avec échap
	submit    func() tea.Cmd

	// Valeurs saisies dans les formulaires (huh écrit directement dedans)
	email           string
	password        string
	passwordConfirm string
	code            string
	cardNumber      string
	cardExpiry      string
	cardCVC         string

	// Catalogue : filtres saisis (prix en euros, tels que tapés), résultats et produit ouvert
	search   searchInputs
	products []apiclient.Product
	table    table.Model
	selected apiclient.Product

	cart apiclient.Cart

	user    apiclient.User
	loading bool
	spinner spinner.Model
	err     error
	info    string
}

func New(api *apiclient.Client) *Model {
	return &Model{
		api:     api,
		screen:  screenHome,
		table:   newProductTable(),
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(selectedStyle)),
	}
}

// Passe en chargement et lance l'appel HTTP en même temps que l'animation du spinner
func (m *Model) startLoading(cmd tea.Cmd) tea.Cmd {
	m.loading = true
	m.err = nil
	return tea.Batch(cmd, m.spinner.Tick)
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Messages valables quel que soit l'écran : quitter, et résultats des appels HTTP
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		// Le spinner ne s'anime que pendant un chargement : sans nouveau tick, il s'arrête
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case productsResultMsg:
		return m.handleProducts(msg)
	case cartResultMsg:
		return m.handleCart(msg)
	case payResultMsg:
		return m.handlePay(msg)
	case loginResultMsg:
		return m.handleLogin(msg)
	case registerResultMsg:
		return m.handleRegister(msg)
	case confirmResultMsg:
		return m.handleConfirm(msg)
	case forgotResultMsg:
		return m.handleForgot(msg)
	case resetResultMsg:
		return m.handleReset(msg)
	case logoutResultMsg:
		m.user = apiclient.User{}
		m.goHome()
		return m, nil
	}

	switch m.screen {
	case screenHome:
		return m.updateHome(msg)
	case screenForm:
		return m.updateForm(msg)
	case screenMenu:
		return m.updateMenu(msg)
	case screenCatalog:
		return m.updateCatalog(msg)
	case screenProduct:
		return m.updateProduct(msg)
	case screenCart:
		return m.updateCart(msg)
	}
	return m, nil
}

func (m *Model) View() tea.View {
	var content string
	switch m.screen {
	case screenHome:
		content = m.viewHome()
	case screenForm:
		content = m.viewForm()
	case screenMenu:
		content = m.viewMenu()
	case screenCatalog:
		content = m.viewCatalog()
	case screenProduct:
		content = m.viewProduct()
	case screenCart:
		content = m.viewCart()
	}

	view := tea.NewView(appStyle.Render(content))
	view.AltScreen = true
	return view
}

func (m *Model) goHome() {
	m.screen = screenHome
	m.cursor = 0
}

// Vide les saisies et les messages avant d'ouvrir un nouveau parcours depuis un menu
func (m *Model) resetInputs() {
	m.email, m.password, m.passwordConfirm, m.code = "", "", "", ""
	m.cardNumber, m.cardExpiry, m.cardCVC = "", "", ""
	m.err = nil
	m.info = ""
}

// ---------- Accueil ----------

func (m *Model) updateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, len(homeChoices))
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, len(homeChoices))
	case "q":
		return m, tea.Quit
	case "enter":
		choice := homeChoices[m.cursor]
		if choice == choiceQuit {
			return m, tea.Quit
		}
		m.resetInputs()
		switch choice {
		case choiceLogin:
			return m, m.openLogin()
		case choiceRegister:
			return m, m.openRegister()
		case choiceConfirm:
			return m, m.openConfirm()
		case choiceForgot:
			return m, m.openForgot()
		}
	}
	return m, nil
}

func (m *Model) viewHome() string {
	s := titleStyle.Render("E-Commerce Go") + "\n"
	s += renderChoices(homeChoices, m.cursor)
	s += m.renderMessages()
	s += helpStyle.Render("↑/↓ naviguer • entrée valider • q quitter")
	return s
}

// ---------- Écran formulaire (commun à tous les formulaires) ----------

func (m *Model) openForm(title, back string, submit func() tea.Cmd, fields ...huh.Field) tea.Cmd {
	m.screen = screenForm
	m.formTitle = title
	m.formBack = back
	m.submit = submit
	m.form = huh.NewForm(huh.NewGroup(fields...))
	return m.form.Init()
}

func (m *Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "esc" {
		m.err = nil
		m.info = ""
		m.screen = m.formBack
		m.cursor = 0
		return m, nil
	}

	_, cmd := m.form.Update(msg)
	if m.form.State == huh.StateCompleted {
		return m, m.startLoading(m.submit())
	}
	return m, cmd
}

func (m *Model) viewForm() string {
	s := titleStyle.Render(m.formTitle) + "\n"
	if m.loading {
		return s + m.spinner.View() + " Chargement…"
	}
	s += m.form.View()
	s += m.renderMessages()
	s += helpStyle.Render("échap retour")
	return s
}

// ---------- Menu connecté ----------

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, -1, len(menuChoices))
	case "down", "j":
		m.cursor = moveCursor(m.cursor, 1, len(menuChoices))
	case "enter":
		switch menuChoices[m.cursor] {
		case choiceCatalog:
			return m, m.openCatalog()
		case choiceCart:
			return m, m.openCart()
		case choiceLogout:
			return m, m.logoutCmd()
		}
	}
	return m, nil
}

func (m *Model) viewMenu() string {
	s := titleStyle.Render(fmt.Sprintf("Bonjour %s", m.user.Email)) + "\n"
	s += renderChoices(menuChoices, m.cursor)
	s += m.renderMessages()
	s += helpStyle.Render("↑/↓ naviguer • entrée valider • ctrl+c quitter")
	return s
}

// ---------- Utilitaires d'affichage ----------

func moveCursor(cursor, step, total int) int {
	cursor += step
	if cursor < 0 {
		return total - 1
	}
	if cursor >= total {
		return 0
	}
	return cursor
}

func renderChoices(choices []string, cursor int) string {
	s := ""
	for i, choice := range choices {
		if i == cursor {
			s += selectedStyle.Render("> "+choice) + "\n"
		} else {
			s += "  " + choice + "\n"
		}
	}
	return s
}

func (m *Model) renderMessages() string {
	s := ""
	if m.info != "" {
		s += "\n" + infoStyle.Render(m.info)
	}
	if m.err != nil {
		s += "\n" + errorStyle.Render(m.err.Error())
	}
	return s
}
