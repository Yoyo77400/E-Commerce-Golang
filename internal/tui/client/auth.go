package client

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/yoyo77400/E-Commerce-Golang/internal/apiclient"
)

// Même règle que le serveur (api/auth.go)
const minPasswordLength = 8

// Résultats des appels HTTP, renvoyés à Update sous forme de messages
type loginResultMsg struct {
	resp apiclient.LoginResponse
	err  error
}

type registerResultMsg struct{ err error }
type confirmResultMsg struct{ err error }
type forgotResultMsg struct{ err error }
type resetResultMsg struct{ err error }
type logoutResultMsg struct{}

// ---------- Champs de formulaire ----------

func (m *Model) emailField() huh.Field {
	return huh.NewInput().Title("Email").Value(&m.email).
		Validate(func(s string) error {
			if s == "" {
				return errors.New("email obligatoire")
			}
			return nil
		})
}

func passwordField(title string, value *string) *huh.Input {
	return huh.NewInput().Title(title).EchoMode(huh.EchoModePassword).Value(value)
}

func (m *Model) newPasswordFields() []huh.Field {
	password := passwordField("Mot de passe", &m.password).
		Validate(func(s string) error {
			if len(s) < minPasswordLength {
				return errors.New("8 caractères minimum")
			}
			return nil
		})
	confirm := passwordField("Confirmer le mot de passe", &m.passwordConfirm).
		Validate(func(s string) error {
			if s != m.password {
				return errors.New("les mots de passe ne correspondent pas")
			}
			return nil
		})
	return []huh.Field{password, confirm}
}

func (m *Model) codeField(title string) huh.Field {
	return huh.NewInput().Title(title).Value(&m.code)
}

// ---------- Ouverture des formulaires ----------

func (m *Model) openLogin() tea.Cmd {
	return m.openForm("Connexion", m.loginCmd,
		m.emailField(),
		passwordField("Mot de passe", &m.password),
	)
}

func (m *Model) openRegister() tea.Cmd {
	fields := []huh.Field{m.emailField()}
	fields = append(fields, m.newPasswordFields()...)
	return m.openForm("Créer un compte", m.registerCmd, fields...)
}

func (m *Model) openConfirm() tea.Cmd {
	return m.openForm("Confirmer mon compte", m.confirmCmd,
		m.codeField("Code de confirmation"),
	)
}

func (m *Model) openForgot() tea.Cmd {
	return m.openForm("Mot de passe oublié", m.forgotCmd,
		m.emailField(),
	)
}

func (m *Model) openReset() tea.Cmd {
	fields := []huh.Field{m.codeField("Code reçu par email")}
	fields = append(fields, m.newPasswordFields()...)
	return m.openForm("Nouveau mot de passe", m.resetCmd, fields...)
}

// ---------- Appels HTTP (exécutés en arrière-plan par bubbletea) ----------
// Les valeurs sont copiées avant la closure : la goroutine ne lit jamais le modèle.

func (m *Model) loginCmd() tea.Cmd {
	api, email, password := m.api, m.email, m.password
	return func() tea.Msg {
		resp, err := api.Login(context.Background(), email, password)
		return loginResultMsg{resp: resp, err: err}
	}
}

func (m *Model) registerCmd() tea.Cmd {
	api, email, password := m.api, m.email, m.password
	return func() tea.Msg {
		_, err := api.Register(context.Background(), email, password)
		return registerResultMsg{err: err}
	}
}

func (m *Model) confirmCmd() tea.Cmd {
	api, code := m.api, m.code
	return func() tea.Msg {
		return confirmResultMsg{err: api.Confirm(context.Background(), code)}
	}
}

func (m *Model) forgotCmd() tea.Cmd {
	api, email := m.api, m.email
	return func() tea.Msg {
		return forgotResultMsg{err: api.ForgotPassword(context.Background(), email)}
	}
}

func (m *Model) resetCmd() tea.Cmd {
	api, code, password := m.api, m.code, m.password
	return func() tea.Msg {
		return resetResultMsg{err: api.ResetPassword(context.Background(), code, password)}
	}
}

func (m *Model) logoutCmd() tea.Cmd {
	api := m.api
	return func() tea.Msg {
		api.Logout(context.Background())
		return logoutResultMsg{}
	}
}

// ---------- Traitement des résultats ----------
// En cas d'erreur, on rouvre le même formulaire (les saisies sont conservées) avec le message.

func (m *Model) handleLogin(msg loginResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, m.openLogin()
	}
	m.resetInputs()
	m.user = msg.resp.User
	m.screen = screenMenu
	m.cursor = 0
	return m, nil
}

func (m *Model) handleRegister(msg registerResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, m.openRegister()
	}
	m.info = "Compte créé. Saisissez le code de confirmation envoyé à " + m.email + "."
	return m, m.openConfirm()
}

func (m *Model) handleConfirm(msg confirmResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		m.info = ""
		return m, m.openConfirm()
	}
	m.resetInputs()
	m.info = "Compte confirmé, vous pouvez vous connecter."
	m.goHome()
	return m, nil
}

func (m *Model) handleForgot(msg forgotResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		return m, m.openForgot()
	}
	m.info = "Si un compte existe pour " + m.email + ", un code vient d'être envoyé."
	return m, m.openReset()
}

func (m *Model) handleReset(msg resetResultMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.err = msg.err
		m.info = ""
		return m, m.openReset()
	}
	m.resetInputs()
	m.info = "Mot de passe modifié, vous pouvez vous connecter."
	m.goHome()
	return m, nil
}
