package jwks

import "time"

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string   `json:"kid"`
	Kty string   `json:"kty"`
	Alg string   `json:"alg"`
	Use string   `json:"use"`
	N   string   `json:"n,omitempty"`
	E   string   `json:"e,omitempty"`
	X5c []string `json:"x5c,omitempty"`
	X5t string   `json:"x5t,omitempty"`
	Crv string   `json:"crv,omitempty"`
	X   string   `json:"x,omitempty"`
	Y   string   `json:"y,omitempty"`
}

// IDPData holds the JWKS data and metadata for an IDP
type IDPData struct {
	Name        string    `json:"name"`
	JWKS        *JWKS     `json:"jwks"`
	LastUpdated time.Time `json:"last_updated"`
	LastError   string    `json:"last_error,omitempty"`
	UpdateCount int       `json:"update_count"`
}
