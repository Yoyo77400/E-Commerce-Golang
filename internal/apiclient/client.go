package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yoyo77400/E-Commerce-Golang/internal/api"
)

const requestTimeOut = 10 * time.Second

type Client struct {
	baseURL string
	http    *http.Client

	mu    sync.RWMutex
	token string
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: requestTimeOut},
	}
}

// Utilisation de la fonction Lock en écriture pour ne permettre qu'une écriture en simultané sur le token. Defer avec unlock assure le déblocage en fin de fonction.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// Utilisation de la fonction RLock en lecture. Le token est vérouillé le temps de la lecture, mais est peut être lu plusieurs fois en simultané
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("api: unexpected status %d", e.StatusCode)
	}
	return e.Message
}

func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("apiclient: encode body %w", err)
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return fmt.Errorf("apiclient: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := c.Token(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr api.ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return &APIError{StatusCode: resp.StatusCode, Message: apiErr.Error}
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("apiclient: decode response: %w", err)
	}
	return nil
}

// --- authentification ---

// Register crée un compte. Un code de confirmation est envoyé à l'adresse.
func (c *Client) Register(ctx context.Context, email, password string) (api.UserDTO, error) {
	var user api.UserDTO
	err := c.do(ctx, http.MethodPost, "/api/auth/register",
		api.RegisterRequest{Email: email, Password: password}, &user)
	return user, err
}

// Confirm valide le code reçu à l'inscription.
func (c *Client) Confirm(ctx context.Context, email, code string) error {
	return c.do(ctx, http.MethodPost, "/api/auth/confirm",
		api.ConfirmRequest{Email: email, Code: code}, nil)
}

// Login ouvre une session et enregistre le jeton dans le client.
func (c *Client) Login(ctx context.Context, email, password string) (api.LoginResponse, error) {
	var out api.LoginResponse
	err := c.do(ctx, http.MethodPost, "/api/auth/login",
		api.LoginRequest{Email: email, Password: password}, &out)
	if err != nil {
		return api.LoginResponse{}, err
	}
	c.SetToken(out.Token)
	return out, nil
}

// Logout révoque la session et oublie le jeton.
func (c *Client) Logout(ctx context.Context) error {
	err := c.do(ctx, http.MethodPost, "/api/auth/logout", nil, nil)
	c.SetToken("")
	return err
}

// ForgotPassword demande un code de réinitialisation.
func (c *Client) ForgotPassword(ctx context.Context, email string) error {
	return c.do(ctx, http.MethodPost, "/api/auth/password/forgot",
		api.PasswordResetRequest{Email: email}, nil)
}

// ResetPassword applique un nouveau mot de passe à partir d'un code de réinitialisation.
func (c *Client) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	return c.do(ctx, http.MethodPost, "/api/auth/password/reset",
		api.PasswordResetConfirm{Email: email, Code: code, NewPassword: newPassword}, nil)
}

// --- catalogue ---

// ProductFilter reprend les paramètres de requête acceptés par GET /api/products avec des prix en centimes.
type ProductFilter struct {
	Name        string
	Description string
	Category    string
	MaxPriceTTC int64
	Limit       int
}

// Query convertit le filtre en paramètres d'URL, en ignorant les critères vides.
func (f ProductFilter) Query() url.Values {
	query := url.Values{}
	if f.Name != "" {
		query.Set("name", f.Name)
	}
	if f.Description != "" {
		query.Set("description", f.Description)
	}
	if f.Category != "" {
		query.Set("category", f.Category)
	}
	if f.MaxPriceTTC > 0 {
		query.Set("max_price_ttc", fmt.Sprint(f.MaxPriceTTC))
	}
	if f.Limit > 0 {
		query.Set("limit", fmt.Sprint(f.Limit))
	}
	return query
}

func (c *Client) SearchProduct(ctx context.Context, filter ProductFilter) ([]api.ProductDTO, error) {
	var products []api.ProductDTO
	err := c.do(ctx, http.MethodGet, "/api/products"+filter.Query().Encode(), nil, &products)
	return products, err
}


