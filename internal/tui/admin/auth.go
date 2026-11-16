package admin

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

// Même valeur que le serveur (api/users.go)
const roleAdmin = "admin"

var errNotAdmin = errors.New("accès réservé aux administrateurs")

// Résultats des appels HTTP, renvoyés à Update sous forme de messages
type loginResultMsg struct {
	resp apiclient.LoginResponse
	err  error
}

type logoutResultMsg struct{}

// formBack vide : échap sur la connexion quitte l'application
func (m *Model) openLogin() tea.Cmd {
	return m.openForm("Connexion administrateur", "", m.loginCmd,
		huh.NewInput().Title("Email").Value(&m.email),
		huh.NewInput().Title("Mot de passe").EchoMode(huh.EchoModePassword).Value(&m.password),
	)
}

// Les valeurs sont copiées avant la closure : la goroutine ne lit pas le modèle.
func (m *Model) loginCmd() tea.Cmd {
	api, email, password := m.api, m.email, m.password
	return func() tea.Msg {
		resp, err := api.Login(context.Background(), email, password)
		if err == nil && resp.User.Role != roleAdmin {
			// Compte valide mais non admin : on referme la session
			api.Logout(context.Background())
			err = errNotAdmin
		}
		return loginResultMsg{resp: resp, err: err}
	}
}

func (m *Model) logoutCmd() tea.Cmd {
	api := m.api
	return func() tea.Msg {
		api.Logout(context.Background())
		return logoutResultMsg{}
	}
}

func (m *Model) handleLogin(msg loginResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.password = ""
	if msg.err != nil {
		m.err = msg.err
		return m, m.openLogin()
	}
	m.info = ""
	m.user = msg.resp.User
	m.screen = screenMenu
	m.cursor = 0
	return m, nil
}
