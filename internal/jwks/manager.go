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
func (m *Manager) Update(name string, jwks *JWKS, err error) {
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

	if err != nil {
		data.LastError = err.Error()
		m.logger.Error("Failed to update JWKS",
			"idp", name,
			"error", err,
			"last_updated", data.LastUpdated,
			"update_count", data.UpdateCount,
		)
	} else {
		data.JWKS = jwks
		data.LastError = ""
		m.logger.Info("Successfully updated JWKS",
			"idp", name,
			"keys_count", len(jwks.Keys),
			"last_updated", data.LastUpdated,
			"update_count", data.UpdateCount,
		)
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
			Name:        data.Name,
			JWKS:        data.JWKS,
			LastUpdated: data.LastUpdated,
			LastError:   data.LastError,
			UpdateCount: data.UpdateCount,
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
