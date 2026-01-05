package jwks

import "time"

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid     string   `json:"kid,omitempty"`
	Kty     string   `json:"kty"`
	Alg     string   `json:"alg,omitempty"`
	Use     string   `json:"use,omitempty"`
	N       string   `json:"n,omitempty"`
	E       string   `json:"e,omitempty"`
	X5c     []string `json:"x5c,omitempty"`
	X5t     string   `json:"x5t,omitempty"`
	X5tS256 string   `json:"x5t#S256,omitempty"`
	Crv     string   `json:"crv,omitempty"`
	X       string   `json:"x,omitempty"`
	Y       string   `json:"y,omitempty"`
	D       string   `json:"d,omitempty"`
	P       string   `json:"p,omitempty"`
	Q       string   `json:"q,omitempty"`
	Dp      string   `json:"dp,omitempty"`
	Dq      string   `json:"dq,omitempty"`
	Qi      string   `json:"qi,omitempty"`
	K       string   `json:"k,omitempty"`
}

// IDPData holds the JWKS data and metadata for an IDP
type IDPData struct {
	Name              string    `json:"name"`
	JWKS              *JWKS     `json:"jwks"`
	LastUpdated       time.Time `json:"last_updated"`
	LastError         string    `json:"last_error,omitempty"`
	UpdateCount       int       `json:"update_count"`
	KeyCount          int       `json:"key_count"`           // current number of keys
	MaxKeys           int       `json:"max_keys"`            // maximum allowed keys
	CacheDuration     int       `json:"cache_duration"`      // cache duration in seconds (what we use)
	IDPSuggestedCache int       `json:"idp_suggested_cache"` // what IDP recommended via Cache-Control
	CacheUntil        time.Time `json:"cache_until"`         // cache valid until
	RefreshInterval   int       `json:"refresh_interval"`    // how often we fetch from IDP
}
