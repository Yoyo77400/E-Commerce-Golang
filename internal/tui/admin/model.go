package admin

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

// Écrans de l'application
const (
	screenForm = "form"
	screenMenu = "menu"
)

const choiceLogout = "Se déconnecter"

var menuChoices = []string{choiceLogout}

type Model struct {
	api    *apiclient.Client
	screen string
	cursor int

	// Écran formulaire : le formulaire affiché et l'action lancée à sa validation
	form      *huh.Form
	formTitle string
	formBack  string // écran affiché si on quitte le formulaire avec échap ("" = quitter l'application)
	submit    func() tea.Cmd

	// Valeurs saisies dans les formulaires (huh écrit directement dedans)
	email    string
	password string

	user    apiclient.User
	loading bool
	spinner spinner.Model
	err     error
	info    string
}

func New(api *apiclient.Client) *Model {
	return &Model{
		api:     api,
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(selectedStyle)),
	}
}

// L'admin arrive directement sur le formulaire de connexion
func (m *Model) Init() tea.Cmd {
	return m.openLogin()
}

// Passe en chargement et lance l'appel HTTP en même temps que l'animation du spinner
func (m *Model) startLoading(cmd tea.Cmd) tea.Cmd {
	m.loading = true
	m.err = nil
	return tea.Batch(cmd, m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Messages valables quel que soit l'écran : quitter, et résultats des appels HTTP
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case loginResultMsg:
		return m.handleLogin(msg)
	case logoutResultMsg:
		m.user = apiclient.User{}
		m.info = "Déconnecté."
		return m, m.openLogin()
	}

	switch m.screen {
	case screenForm:
		return m.updateForm(msg)
	case screenMenu:
		return m.updateMenu(msg)
	}
	return m, nil
}

func (m *Model) View() tea.View {
	var content string
	switch m.screen {
	case screenForm:
		content = m.viewForm()
	case screenMenu:
		content = m.viewMenu()
	}

	view := tea.NewView(appStyle.Render(content))
	view.AltScreen = true
	return view
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
		if m.formBack == "" {
			return m, tea.Quit
		}
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

// ---------- Menu admin ----------

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
		case choiceLogout:
			return m, m.logoutCmd()
		}
	}
	return m, nil
}

func (m *Model) viewMenu() string {
	s := titleStyle.Render(fmt.Sprintf("Administration — %s", m.user.Email)) + "\n"
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
