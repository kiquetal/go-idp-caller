package jwks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kiquetal/go-idp-caller/internal/config"
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

	jwks, idpCacheDuration, err := u.fetch()
	maxKeys := u.config.GetMaxKeys()

	// Use IDP's suggested cache duration if available and reasonable
	cacheDuration := u.determineCacheDuration(idpCacheDuration)
	refreshInterval := u.config.RefreshInterval

	u.manager.UpdateWithIDPCache(u.config.Name, jwks, maxKeys, cacheDuration, idpCacheDuration, refreshInterval, err)
}

// determineCacheDuration determines the best cache duration based on IDP response and config
func (u *Updater) determineCacheDuration(idpMaxAge int) int {
	configDuration := u.config.GetCacheDuration()

	// If IDP didn't provide max-age, use config
	if idpMaxAge <= 0 {
		u.logger.Debug("No cache control from IDP, using config",
			"idp", u.config.Name,
			"cache_duration", configDuration,
		)
		return configDuration
	}

	// Always use the MAXIMUM of IDP's suggestion and config to ensure minimum cache duration
	// This prevents IDPs with very short cache times from causing excessive refreshes
	// Our config represents our minimum desired cache duration

	if idpMaxAge < configDuration {
		// IDP suggests shorter cache - but we enforce our minimum
		u.logger.Info("IDP suggests shorter cache, using config minimum",
			"idp", u.config.Name,
			"idp_max_age", idpMaxAge,
			"config_duration", configDuration,
			"using", configDuration,
			"reason", "config sets minimum cache duration",
		)
		return configDuration
	}

	// IDP suggests longer cache - honor their suggestion as they know their rotation schedule
	u.logger.Info("Using IDP's longer cache duration",
		"idp", u.config.Name,
		"idp_max_age", idpMaxAge,
		"config_duration", configDuration,
		"using", idpMaxAge,
		"reason", "IDP suggests longer cache than config minimum",
	)
	return idpMaxAge
}

// fetch retrieves JWKS from the IDP endpoint and returns the data plus cache duration from headers
func (u *Updater) fetch() (*JWKS, int, error) {
	req, err := http.NewRequest("GET", u.config.URL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse Cache-Control header from IDP response
	cacheControl := resp.Header.Get("Cache-Control")
	idpMaxAge := parseCacheControl(cacheControl)

	if idpMaxAge > 0 {
		u.logger.Debug("IDP provided cache control",
			"idp", u.config.Name,
			"cache_control", cacheControl,
			"max_age", idpMaxAge,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, 0, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	return &jwks, idpMaxAge, nil
}

// parseCacheControl extracts max-age value from Cache-Control header
// Returns 0 if not found or invalid
func parseCacheControl(cacheControl string) int {
	if cacheControl == "" {
		return 0
	}

	// Parse directives (e.g., "public, max-age=86400, must-revalidate")
	directives := splitCacheControl(cacheControl)

	for _, directive := range directives {
		// Look for max-age=value
		if len(directive) > 8 && directive[:8] == "max-age=" {
			var maxAge int
			if _, err := fmt.Sscanf(directive[8:], "%d", &maxAge); err == nil {
				return maxAge
			}
		}
	}

	return 0
}

// splitCacheControl splits Cache-Control header by commas and trims spaces
func splitCacheControl(s string) []string {
	var result []string
	current := ""

	for _, char := range s {
		if char == ',' {
			trimmed := trimSpace(current)
			if trimmed != "" {
				result = append(result, trimmed)
			}
			current = ""
		} else {
			current += string(char)
		}
	}

	// Add last directive
	trimmed := trimSpace(current)
	if trimmed != "" {
		result = append(result, trimmed)
	}

	return result
}

// trimSpace removes leading and trailing spaces
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}
