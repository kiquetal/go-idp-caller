# Architecture Overview

## System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     IDP Caller Service                          │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    HTTP Server (Port 8080)                │  │
│  │                                                            │  │
│  │  /.well-known/jwks.json  → Merged JWKS (all IDPs)        │  │
│  │  /jwks/auth0             → Auth0 keys only               │  │
│  │  /status                 → All IDP status                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                            ↓                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │            JWKS Manager (Thread-Safe)                     │  │
│  │                                                            │  │
│  │  • RWMutex for concurrent access                          │  │
│  │  • Stores JWKS per IDP                                    │  │
│  │  • Enforces max_keys limit (default: 10)                  │  │
│  │  • Tracks cache metadata                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                            ↑                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                 Background Updaters                        │  │
│  │                                                            │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐               │  │
│  │  │ Goroutine│  │ Goroutine│  │ Goroutine│               │  │
│  │  │  Auth0   │  │   Okta   │  │ Keycloak │               │  │
│  │  │          │  │          │  │          │               │  │
│  │  │ Every    │  │ Every    │  │ Every    │               │  │
│  │  │ 3600s    │  │ 3600s    │  │ 3600s    │               │  │
│  │  └──────────┘  └──────────┘  └──────────┘               │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────────────────────────┐
        │         External IDPs                  │
        │                                        │
        │  • https://auth0.com/.well-known/...  │
        │  • https://okta.com/oauth2/.../keys   │
        │  • https://keycloak.com/realms/.../.. │
        └───────────────────────────────────────┘
```

## Request Flow

### 1. Client Requests Merged JWKS

```
Client → GET /.well-known/jwks.json
         │
         ↓
    HTTP Server
         │
         ↓
    JWKS Manager (Read Lock)
         │
         ├─→ Get Auth0 keys (3 keys)
         ├─→ Get Okta keys (2 keys)
         └─→ Get Keycloak keys (4 keys)
         │
         ↓
    Merge into single array (9 total keys)
         │
         ↓
    Add headers:
    • Cache-Control: public, max-age=900
    • X-Total-Keys: 9
    • X-IDP-Count: 3
         │
         ↓
    Return: { "keys": [...9 keys...] }
```

### 2. Background Update Process

```
Timer (every 3600s)
         │
         ↓
    Fetch from IDP
    https://auth0.com/.well-known/jwks.json
         │
         ↓
    Parse JSON
    Got 25 keys
         │
         ↓
    Check max_keys = 10
    Truncate to first 10 keys
         │
         ↓
    JWKS Manager (Write Lock)
         │
         ↓
    Update IDP data:
    • JWKS with 10 keys
    • key_count = 10
    • max_keys = 10
    • cache_duration = 900
    • cache_until = now + 900s
    • last_updated = now
         │
         ↓
    Log: "Successfully updated JWKS"
```

## Data Structure

### Manager Storage

```
Manager {
    mu: RWMutex
    data: map[string]*IDPData {
        "auth0": {
            Name: "auth0"
            JWKS: {
                Keys: [
                    {kid: "key1", kty: "RSA", ...},
                    {kid: "key2", kty: "RSA", ...},
                    {kid: "key3", kty: "RSA", ...}
                ]
            }
            LastUpdated: 2026-01-05T10:30:00Z
            KeyCount: 3                    // ← NEW
            MaxKeys: 10                    // ← NEW
            CacheDuration: 900             // ← NEW
            CacheUntil: 2026-01-05T10:45:00Z // ← NEW
            UpdateCount: 42
            LastError: ""
        },
        "okta": { ... },
        "keycloak": { ... }
    }
}
```

## Configuration Flow

```
config.yaml
    │
    ├─→ server:
    │     port: 8080
    │     host: "0.0.0.0"
    │
    ├─→ idps:
    │     - name: "auth0"
    │       url: "https://..."
    │       refresh_interval: 3600
    │       max_keys: 10          ← Standard
    │       cache_duration: 900   ← Standard
    │
    └─→ logging:
          level: "info"
          format: "json"
```

## Cache Strategy

### Per-IDP Caching

```
GET /jwks/auth0
    ↓
Response Headers:
    Cache-Control: public, max-age=900   ← IDP-specific
    
GET /jwks/okta
    ↓
Response Headers:
    Cache-Control: public, max-age=600   ← Different IDP
```

### Merged Endpoint Caching

```
Configured IDPs:
• auth0: cache_duration = 900
• okta: cache_duration = 600
• keycloak: cache_duration = 1200

GET /.well-known/jwks.json
    ↓
Use minimum: min(900, 600, 1200) = 600
    ↓
Response Headers:
    Cache-Control: public, max-age=600   ← Minimum
```

## Key Limiting Flow

```
IDP returns 25 keys
    ↓
max_keys = 10 configured
    ↓
25 > 10? YES
    ↓
Log WARNING:
    "Truncating keys to max limit"
    idp: "auth0"
    original_count: 25
    max_keys: 10
    ↓
Keep only first 10 keys:
    keys = keys[:10]
    ↓
Store in Manager:
    KeyCount = 10
    MaxKeys = 10
    ↓
Status shows:
    key_count: 10
    max_keys: 10  ← Hitting limit!
```

## Thread Safety

### Concurrent Reads (Multiple Clients)

```
Client 1 → GET /jwks/auth0 ─┐
Client 2 → GET /jwks/okta   ├─→ Manager.RLock()
Client 3 → GET /.well-known ┘    ↓
                                Read data (concurrent)
                                  ↓
                              Manager.RUnlock()
                                  ↓
                              All clients get data
```

### Read + Write (Client + Background Update)

```
Client → GET /jwks/auth0 → Manager.RLock() → Read
                              ↓
                          (waiting...)
                              ↓
Updater → Update JWKS → Manager.Lock() → Write (blocked)
                           ↓
                       Wait for RUnlock
                           ↓
                       RUnlock happens
                           ↓
                       Lock acquired
                           ↓
                       Update data
                           ↓
                       Unlock
```

## Monitoring Points

```
┌─────────────────────────────────────────┐
│         Observable Metrics              │
├─────────────────────────────────────────┤
│                                         │
│  Per IDP:                               │
│  • key_count / max_keys ratio           │
│  • last_updated timestamp               │
│  • update_count                         │
│  • cache_until                          │
│  • last_error (if any)                  │
│                                         │
│  Merged Endpoint:                       │
│  • X-Total-Keys header                  │
│  • X-IDP-Count header                   │
│  • Cache-Control max-age                │
│                                         │
│  Logs:                                  │
│  • "Successfully updated JWKS"          │
│  • "Truncating keys to max limit"       │
│  • "Failed to update JWKS"              │
│                                         │
└─────────────────────────────────────────┘
```

## Production Deployment

```
Kubernetes Cluster
    │
    ├─→ ConfigMap
    │     • config.yaml with IDP URLs
    │     • max_keys: 10 per IDP
    │     • cache_duration: 900
    │
    ├─→ Deployment (2 replicas)
    │     Pod 1                Pod 2
    │     ├─ Container         ├─ Container
    │     │  • Go app          │  • Go app
    │     │  • Port 8080       │  • Port 8080
    │     │  • CPU: 100m       │  • CPU: 100m
    │     │  • Mem: 64Mi       │  • Mem: 64Mi
    │     │                    │
    │     └─ Probes            └─ Probes
    │        • Liveness             • Liveness
    │        • Readiness            • Readiness
    │
    └─→ Service (ClusterIP)
          • Name: idp-caller
          • Port: 80 → 8080
          • Accessible as:
            http://idp-caller.default.svc.cluster.local
```

## Integration with KrakenD

```
┌──────────────────────────────────────────────────┐
│                   KrakenD                        │
│                                                  │
│  JWT Validator Config:                          │
│  {                                               │
│    "jwk_url": "http://idp-caller/                │
│                .well-known/jwks.json"            │
│    "cache": true,                                │
│    "cache_duration": 900                         │
│  }                                               │
└──────────────────────────────────────────────────┘
                    ↓
            (Fetch JWKS)
                    ↓
┌──────────────────────────────────────────────────┐
│              IDP Caller Service                  │
│                                                  │
│  Returns merged keys from all IDPs:              │
│  • Auth0 keys (3)                                │
│  • Okta keys (2)                                 │
│  • Keycloak keys (4)                             │
│                                                  │
│  Total: 9 keys in single array                  │
│  Cache-Control: public, max-age=900              │
└──────────────────────────────────────────────────┘
                    ↓
            (JWT arrives)
                    ↓
┌──────────────────────────────────────────────────┐
│         KrakenD JWT Verification                 │
│                                                  │
│  1. Extract kid from JWT header                  │
│  2. Find matching key in JWKS                    │
│  3. Verify signature                             │
│  4. Check expiry, issuer, audience              │
│  5. Allow/Deny request                           │
└──────────────────────────────────────────────────┘
```

## Summary

This architecture provides:
- ✅ **Scalability**: Independent goroutines per IDP
- ✅ **Safety**: Thread-safe concurrent access
- ✅ **Protection**: Per-IDP key limits
- ✅ **Performance**: Optimized caching per IDP
- ✅ **Reliability**: Automatic background updates
- ✅ **Observability**: Rich metadata and logging
- ✅ **Standards**: OIDC-compliant endpoints
- ✅ **Integration**: Works with standard JWT libraries

