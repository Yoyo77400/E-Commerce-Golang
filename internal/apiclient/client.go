package apiclient

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yoyo77400/E-Commerce-Golang/internal/api"
)

const requestTimeOut = 10 * time.Second

type Client struct {
	baseURL string
	http    *http.Client

	mu      sync.RWMutex
	token   string
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL,"/"),
		http:    &http.Client{Timeout: requestTimeOut},
	}
}

//Utilisation de la fonction Lock en écriture pour ne permettre qu'une écriture en simultané sur le token.
//Defer avec unlock assure le déblocage en fin de fonction. 
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

//Utilisation de la fonction RLock en lecture. Le token est vérouillé le temps de la lecture, mais est peut être lu plusieurs fois en simultané
func (c *Client) getToken() string {
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

