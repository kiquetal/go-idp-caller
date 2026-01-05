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

	// Use the MAXIMUM of IDP's suggestion and config
	// This ensures we respect our minimum freshness requirements
	// If IDP says cache for 1 hour but we want 15 min freshness, use 15 min
	// If IDP says cache for 5 min but we configured 15 min, use 15 min (IDP knows best about rotation)

	if idpMaxAge < configDuration {
		// IDP suggests shorter cache - they rotate keys more frequently
		// Use IDP's suggestion to ensure we get fresh keys
		u.logger.Info("Using IDP's shorter cache duration (IDP rotates keys faster)",
			"idp", u.config.Name,
			"idp_max_age", idpMaxAge,
			"config_duration", configDuration,
			"using", idpMaxAge,
			"reason", "IDP rotates keys more frequently",
		)
		return idpMaxAge
	}

	// IDP suggests longer cache - but we want fresher data
	// Use our config duration to ensure clients get updates more frequently
	u.logger.Info("Using config cache duration (more conservative than IDP)",
		"idp", u.config.Name,
		"idp_max_age", idpMaxAge,
		"config_duration", configDuration,
		"using", configDuration,
		"reason", "config requires fresher data than IDP suggests",
	)
	return configDuration
}

// fetch retrieves JWKS from the IDP endpoint and returns the data plus cache duration from headers
func (u *Updater) fetch() (*JWKS, int, error) {
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
