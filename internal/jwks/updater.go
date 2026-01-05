package jwks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-idp-caller/internal/config"
)

// Updater handles periodic updates of JWKS from an IDP
type Updater struct {
	config  config.IDPConfig
	manager *Manager
	logger  *slog.Logger
	client  *http.Client
}

// NewUpdater creates a new JWKS updater
func NewUpdater(cfg config.IDPConfig, manager *Manager, logger *slog.Logger) *Updater {
	return &Updater{
		config:  cfg,
		manager: manager,
		logger:  logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start begins the periodic update process
func (u *Updater) Start(ctx context.Context) {
	u.logger.Info("Starting JWKS updater", "idp", u.config.Name)

	// Perform initial fetch immediately
	u.fetchAndUpdate()

	// Setup ticker for periodic updates
	ticker := time.NewTicker(time.Duration(u.config.RefreshInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("Stopping JWKS updater", "idp", u.config.Name)
			return
		case <-ticker.C:
			u.fetchAndUpdate()
		}
	}
}

// fetchAndUpdate fetches JWKS from the IDP and updates the manager
func (u *Updater) fetchAndUpdate() {
	u.logger.Debug("Fetching JWKS", "idp", u.config.Name, "url", u.config.URL)

	jwks, err := u.fetch()
	u.manager.Update(u.config.Name, jwks, err)
}

// fetch retrieves JWKS from the IDP endpoint
func (u *Updater) fetch() (*JWKS, error) {
	req, err := http.NewRequest("GET", u.config.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	return &jwks, nil
}
