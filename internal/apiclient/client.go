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
		var apiErr errorResponse
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

func (c *Client) Register(ctx context.Context, email, password string) (User, error) {
	var user User
	err := c.do(ctx, http.MethodPost, "/auth/register",
		map[string]string{"email": email, "password": password}, &user)
	return user, err
}

func (c *Client) Confirm(ctx context.Context, code string) error {
	return c.do(ctx, http.MethodPost, "/auth/confirm",
		map[string]string{"code": code}, nil)
}

func (c *Client) Login(ctx context.Context, email, password string) (LoginResponse, error) {
	var out LoginResponse
	err := c.do(ctx, http.MethodPost, "/auth/login",
		map[string]string{"email": email, "password": password}, &out)
	if err != nil {
		return LoginResponse{}, err
	}
	c.SetToken(out.Token)
	return out, nil
}

func (c *Client) Logout(ctx context.Context) error {
	err := c.do(ctx, http.MethodPost, "/auth/logout", nil, nil)
	c.SetToken("")
	return err
}

func (c *Client) ForgotPassword(ctx context.Context, email string) error {
	return c.do(ctx, http.MethodPost, "/auth/password/forgot",
		map[string]string{"email": email}, nil)
}

func (c *Client) ResetPassword(ctx context.Context, code, password string) error {
	return c.do(ctx, http.MethodPost, "/auth/password/reset",
		map[string]string{"code": code, "password": password}, nil)
}

// --- catalogue ---

// Filtres de GET /products. Prix en centimes ; une valeur vide ou 0 est ignorée.
type ProductFilter struct {
	Query    string
	Category string
	MinPrice int64
	MaxPrice int64
	MinTTC   int64
	MaxTTC   int64
}

func (f ProductFilter) Values() url.Values {
	values := url.Values{}
	if f.Query != "" {
		values.Set("q", f.Query)
	}
	if f.Category != "" {
		values.Set("category", f.Category)
	}
	if f.MinPrice > 0 {
		values.Set("min_price", fmt.Sprint(f.MinPrice))
	}
	if f.MaxPrice > 0 {
		values.Set("max_price", fmt.Sprint(f.MaxPrice))
	}
	if f.MinTTC > 0 {
		values.Set("min_ttc", fmt.Sprint(f.MinTTC))
	}
	if f.MaxTTC > 0 {
		values.Set("max_ttc", fmt.Sprint(f.MaxTTC))
	}
	return values
}

func (c *Client) SearchProducts(ctx context.Context, filter ProductFilter) ([]Product, error) {
	var products []Product
	err := c.do(ctx, http.MethodGet, "/products?"+filter.Values().Encode(), nil, &products)
	return products, err
}