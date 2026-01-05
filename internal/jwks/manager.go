package jwks

import (
	"log/slog"
	"sync"
	"time"
)

// Manager manages JWKS data for multiple IDPs
type Manager struct {
	mu     sync.RWMutex
	data   map[string]*IDPData
	logger *slog.Logger
}

// NewManager creates a new JWKS manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		data:   make(map[string]*IDPData),
		logger: logger,
	}
}

// Update stores or updates JWKS data for an IDP
func (m *Manager) Update(name string, jwks *JWKS, maxKeys int, cacheDuration int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, exists := m.data[name]
	if !exists {
		data = &IDPData{
			Name: name,
		}
		m.data[name] = data
	}

	data.LastUpdated = time.Now()
	data.UpdateCount++
	data.MaxKeys = maxKeys
	data.CacheDuration = cacheDuration

	if err != nil {
		data.LastError = err.Error()
		m.logger.Error("Failed to update JWKS",
			"idp", name,
			"error", err,
			"last_updated", data.LastUpdated,
			"update_count", data.UpdateCount,
		)
	} else {
		// Apply key limiting
		originalCount := len(jwks.Keys)
		if originalCount > maxKeys {
			m.logger.Warn("Truncating keys to max limit",
				"idp", name,
				"original_count", originalCount,
				"max_keys", maxKeys,
			)
			jwks.Keys = jwks.Keys[:maxKeys]
		}

		data.JWKS = jwks
		data.KeyCount = len(jwks.Keys)
		data.CacheUntil = time.Now().Add(time.Duration(cacheDuration) * time.Second)
		data.LastError = ""

		m.logger.Info("Successfully updated JWKS",
			"idp", name,
			"key_count", data.KeyCount,
			"max_keys", data.MaxKeys,
			"cache_duration", data.CacheDuration,
			"cache_until", data.CacheUntil.Format(time.RFC3339),
			"last_updated", data.LastUpdated.Format(time.RFC3339),
			"update_count", data.UpdateCount,
		)
	}
}

// UpdateWithIDPCache stores or updates JWKS data with IDP's suggested cache duration
func (m *Manager) UpdateWithIDPCache(name string, jwks *JWKS, maxKeys int, cacheDuration int, idpSuggestedCache int, refreshInterval int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, exists := m.data[name]
	if !exists {
		data = &IDPData{
			Name: name,
		}
		m.data[name] = data
	}

	data.LastUpdated = time.Now()
	data.UpdateCount++
	data.MaxKeys = maxKeys
	data.CacheDuration = cacheDuration
	data.IDPSuggestedCache = idpSuggestedCache
	data.RefreshInterval = refreshInterval

	if err != nil {
		data.LastError = err.Error()
		m.logger.Error("Failed to update JWKS",
			"idp", name,
			"error", err,
			"last_updated", data.LastUpdated,
			"update_count", data.UpdateCount,
		)
	} else {
		// Apply key limiting
		originalCount := len(jwks.Keys)
		if originalCount > maxKeys {
			m.logger.Warn("Truncating keys to max limit",
				"idp", name,
				"original_count", originalCount,
				"max_keys", maxKeys,
			)
			jwks.Keys = jwks.Keys[:maxKeys]
		}

		data.JWKS = jwks
		data.KeyCount = len(jwks.Keys)
		data.CacheUntil = time.Now().Add(time.Duration(cacheDuration) * time.Second)
		data.LastError = ""

		logFields := []interface{}{
			"idp", name,
			"key_count", data.KeyCount,
			"max_keys", data.MaxKeys,
			"cache_duration", data.CacheDuration,
			"refresh_interval", data.RefreshInterval,
			"cache_until", data.CacheUntil.Format(time.RFC3339),
			"last_updated", data.LastUpdated.Format(time.RFC3339),
			"update_count", data.UpdateCount,
		}

		if idpSuggestedCache > 0 {
			logFields = append(logFields, "idp_suggested_cache", idpSuggestedCache)
		}

		m.logger.Info("Successfully updated JWKS", logFields...)
	}
}

// Get retrieves JWKS data for a specific IDP
func (m *Manager) Get(name string) (*IDPData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.data[name]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	dataCopy := &IDPData{
		Name:        data.Name,
		JWKS:        data.JWKS,
		LastUpdated: data.LastUpdated,
		LastError:   data.LastError,
		UpdateCount: data.UpdateCount,
	}

	return dataCopy, true
}

// GetAll retrieves all IDP data
func (m *Manager) GetAll() map[string]*IDPData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*IDPData, len(m.data))
	for name, data := range m.data {
		result[name] = &IDPData{
			Name:          data.Name,
			JWKS:          data.JWKS,
			LastUpdated:   data.LastUpdated,
			LastError:     data.LastError,
			UpdateCount:   data.UpdateCount,
			KeyCount:      data.KeyCount,
			MaxKeys:       data.MaxKeys,
			CacheDuration: data.CacheDuration,
			CacheUntil:    data.CacheUntil,
		}
	}

	return result
}

// GetJWKS retrieves only the JWKS for a specific IDP
func (m *Manager) GetJWKS(name string) (*JWKS, bool) {
	data, exists := m.Get(name)
	if !exists || data.JWKS == nil {
		return nil, false
	}
	return data.JWKS, true
}
