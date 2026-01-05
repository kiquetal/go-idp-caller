# Caching Strategy & How It Works

## Overview

The IDP Caller service implements a **two-level caching strategy**:

1. **Server-Side Cache** (in-memory) - The service periodically fetches and stores JWKS
2. **Client-Side Cache** (HTTP headers) - Clients cache responses using `Cache-Control` headers

This document explains how caching works at each level and how the system uses `max-age` from IDP responses.

---

## Architecture: How Goroutines and Caching Work Together

### System Flow

```
┌─────────────────────────────────────────────────────────────┐
│                  IDP Caller Service                         │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Goroutine per IDP (Background Workers)              │  │
│  │                                                        │  │
│  │  ┌─────────────┐   ┌─────────────┐   ┌────────────┐ │  │
│  │  │  Auth0      │   │  Okta       │   │  Keycloak  │ │  │
│  │  │  Goroutine  │   │  Goroutine  │   │  Goroutine │ │  │
│  │  │             │   │             │   │            │ │  │
│  │  │ Every 3600s │   │ Every 3600s │   │ Every 3600s│ │  │
│  │  │      ↓      │   │      ↓      │   │      ↓     │ │  │
│  │  │   Fetch     │   │   Fetch     │   │   Fetch    │ │  │
│  │  │   JWKS      │   │   JWKS      │   │   JWKS     │ │  │
│  │  └──────┬──────┘   └──────┬──────┘   └──────┬─────┘ │  │
│  │         │                 │                  │        │  │
│  │         └─────────────────┼──────────────────┘        │  │
│  │                           ↓                           │  │
│  │                  ┌────────────────┐                   │  │
│  │                  │  JWKS Manager  │                   │  │
│  │                  │  (Thread-Safe) │                   │  │
│  │                  │                │                   │  │
│  │                  │  In-Memory     │                   │  │
│  │                  │  Cache         │                   │  │
│  │                  └────────┬───────┘                   │  │
│  └─────────────────────────┬─┬─────────────────────────┘  │
│                            │ │                              │
│  ┌─────────────────────────┘ └──────────────────────────┐  │
│  │                                                        │  │
│  │  HTTP API Endpoints                                   │  │
│  │  ┌──────────────────────────────────────────────┐    │  │
│  │  │ GET /.well-known/jwks.json                   │    │  │
│  │  │ → Returns merged keys from memory            │    │  │
│  │  │ → Cache-Control: max-age=900 (15 min)       │    │  │
│  │  │                                               │    │  │
│  │  │ GET /jwks/{idp}                              │    │  │
│  │  │ → Returns single IDP keys from memory        │    │  │
│  │  │ → Cache-Control: max-age=900 (per IDP)      │    │  │
│  │  └──────────────────────────────────────────────┘    │  │
│  └───────────────────────────┬────────────────────────┘  │
└────────────────────────────┬─┬──────────────────────────┘
                             │ │
                ┌────────────┘ └────────────┐
                ↓                            ↓
         ┌─────────────┐             ┌─────────────┐
         │  KrakenD    │             │  Client App │
         │             │             │             │
         │  Caches for │             │  Caches for │
         │  900 seconds│             │  900 seconds│
         └─────────────┘             └─────────────┘
```

---

## Level 1: Server-Side Caching (In-Memory)

### How It Works

Each IDP has its own **independent goroutine** that runs in the background:

```go
// From internal/jwks/updater.go
func (u *Updater) Start(ctx context.Context) {
    // Perform initial fetch immediately
    u.fetchAndUpdate()

    // Setup ticker for periodic updates
    ticker := time.NewTicker(time.Duration(u.config.RefreshInterval) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            u.fetchAndUpdate()  // Fetch every refresh_interval
        }
    }
}
```

### Configuration Per IDP

```yaml
idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600      # Server fetches every 3600 seconds (1 hour)
    max_keys: 10                # Keep max 10 keys in memory
    cache_duration: 900         # Tell clients to cache for 900 seconds (15 min)
```

### Key Points

| Parameter | Purpose | Effect |
|-----------|---------|--------|
| **refresh_interval** | How often the **service** fetches from IDP | Controls server-side data freshness |
| **cache_duration** | How long **clients** should cache | Controls client-side caching via HTTP headers |
| **max_keys** | Maximum keys to store | Memory protection |

**Important:** `refresh_interval` and `cache_duration` are **independent**!

---

## Level 2: Client-Side Caching (HTTP Headers)

### How Clients Cache

When clients (KrakenD, apps, browsers) call the API, they receive cache headers:

#### Example: Merged JWKS Endpoint

```bash
$ curl -I http://localhost:8080/.well-known/jwks.json

HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: public, max-age=900
X-Total-Keys: 9
X-IDP-Count: 3
```

**What this means:**
- `Cache-Control: public, max-age=900` → Clients should cache for **900 seconds** (15 minutes)
- Clients won't request again for 15 minutes
- Service serves data from in-memory cache (instant response)

#### Example: Individual IDP Endpoint

```bash
$ curl -I http://localhost:8080/jwks/auth0

HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: public, max-age=900
X-Key-Count: 3
X-Max-Keys: 10
X-Last-Updated: 2026-01-05T10:30:00Z
```

---

## How max-age is Determined

### For Merged Endpoint (`/.well-known/jwks.json`)

The service uses the **minimum cache_duration** across all IDPs (after each IDP's cache_duration has been determined):

```go
// From internal/server/server.go
func (s *Server) handleGetMergedJWKS(w http.ResponseWriter, r *http.Request) {
    all := s.manager.GetAll()
    
    minCacheDuration := 900 // Default 15 minutes
    
    for _, data := range all {
        if data.JWKS != nil && len(data.JWKS.Keys) > 0 {
            // Use minimum to ensure freshness
            if data.CacheDuration > 0 && data.CacheDuration < minCacheDuration {
                minCacheDuration = data.CacheDuration
            }
        }
    }
    
    w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", minCacheDuration))
}
```

**How it works:**

1. **Each IDP determines its own cache_duration:**
   ```
   Auth0:     min(config: 900, idp_max_age: 86400) = 900
   Okta:      min(config: 1800, idp_max_age: 3600) = 1800  
   Keycloak:  min(config: 900, idp_max_age: 300) = 300
   ```

2. **Merged endpoint uses minimum of these:**
   ```
   min(900, 1800, 300) = 300
   ```

3. **Result:** Merged endpoint returns `Cache-Control: max-age=300`

**Why?** 
- Each IDP has **different** cache_duration based on its rotation schedule
- Keycloak rotates every 5 minutes (300s), so it uses 300
- Auth0 is stable (24h), so it uses its config (900)
- Merged endpoint must use the **shortest** (300) to ensure clients get fresh Keycloak keys

**Example with different IDP behaviors:**
```yaml
idps:
  - name: "auth0"
    cache_duration: 900     # Config: 15 min
    # IDP returns: max-age=86400 (24h)
    # Uses: min(900, 86400) = 900 ✅

  - name: "okta"
    cache_duration: 1800    # Config: 30 min
    # IDP returns: max-age=3600 (1h)
    # Uses: min(1800, 3600) = 1800 ✅
    
  - name: "keycloak"
    cache_duration: 900     # Config: 15 min
    # IDP returns: max-age=300 (5 min!)
    # Uses: min(900, 300) = 300 ✅
```

**Individual endpoints:**
- `GET /jwks/auth0` → `Cache-Control: max-age=900`
- `GET /jwks/okta` → `Cache-Control: max-age=1800`
- `GET /jwks/keycloak` → `Cache-Control: max-age=300`

**Merged endpoint:**
- `GET /.well-known/jwks.json` → `Cache-Control: max-age=300` (uses minimum!)

### For Individual IDP Endpoints (`/jwks/{idp}`)

Each IDP endpoint uses **its own cache_duration**:

```go
// From internal/server/server.go
func (s *Server) handleGetIDPJWKS(w http.ResponseWriter, r *http.Request) {
    data, exists := s.manager.Get(idpName)
    
    w.Header().Set("Cache-Control", 
        fmt.Sprintf("public, max-age=%d", data.CacheDuration))
}
```

---

## Cache Timing Examples

### Scenario 1: Standard Configuration

```yaml
idps:
  - name: "auth0"
    refresh_interval: 3600    # Fetch every 1 hour
    cache_duration: 900       # Clients cache 15 minutes
```

**Timeline:**
```
T=0:00    Service starts, fetches Auth0 JWKS, stores in memory
T=0:01    Client requests → Gets cached data, caches for 15 min
T=0:05    Client requests → Uses client cache (no network call)
T=0:16    Client requests → Cache expired, fetches from service
          → Service returns from memory (still fresh)
T=1:00    Service fetches fresh JWKS from Auth0
T=1:01    Client requests → Gets updated data, caches for 15 min
```

**Result:**
- Service fetches: **24 times/day** (every hour)
- Client fetches: **96 times/day** (every 15 min) max per client

### Scenario 2: Fast Rotation

```yaml
idps:
  - name: "custom-idp"
    refresh_interval: 600     # Fetch every 10 minutes
    cache_duration: 300       # Clients cache 5 minutes
```

**Timeline:**
```
T=0:00    Service fetches from IDP
T=0:01    Client gets data, caches for 5 min
T=0:06    Client cache expired, fetches from service
T=0:10    Service fetches fresh data from IDP
```

**Result:**
- Service fetches: **144 times/day** (every 10 min)
- Client fetches: **288 times/day** (every 5 min) max per client

---

## Reading IDP Cache Headers (Future Enhancement)

Currently, the service **does NOT** read `Cache-Control` headers from IDP responses. It uses the configured `cache_duration` from `config.yaml`.

### How IDPs Provide Cache Information

When you fetch JWKS from an IDP, they may return:

```bash
$ curl -I https://tenant.auth0.com/.well-known/jwks.json

HTTP/1.1 200 OK
Cache-Control: max-age=86400
```

**This means:** Auth0 says their JWKS is fresh for 86400 seconds (24 hours).

### Current Behavior

The service **ignores** this and uses `cache_duration` from config:

```yaml
idps:
  - name: "auth0"
    cache_duration: 900  # We tell clients 15 minutes
```

### Potential Enhancement (NOW IMPLEMENTED!)

We now **read and use** the IDP's `Cache-Control` headers dynamically:

```go
// In updater.go - IMPLEMENTED
func (u *Updater) fetch() (*JWKS, int, error) {
    resp, err := u.client.Do(req)
    
    // Parse Cache-Control header
    cacheControl := resp.Header.Get("Cache-Control")
    idpMaxAge := parseCacheControl(cacheControl)  // Extract max-age
    
    return &jwks, idpMaxAge, nil
}

// Determine best cache duration
func (u *Updater) determineCacheDuration(idpMaxAge int) int {
    configDuration := u.config.GetCacheDuration()
    
    // No IDP suggestion? Use config
    if idpMaxAge <= 0 {
        return configDuration
    }
    
    // Use MINIMUM to ensure freshness
    // If IDP says 5 min but config says 15 min → use 5 min (IDP rotates fast!)
    // If IDP says 24 hours but config says 15 min → use 15 min (we want fresher data)
    if idpMaxAge < configDuration {
        return idpMaxAge  // IDP's shorter duration
    }
    return configDuration  // Config's shorter duration
}
```

**Benefits:** 
- Automatically respect IDP's key rotation schedule
- If IDP rotates keys every 5 minutes, we use 5 minute cache
- If IDP is stable (24 hours), we still use config's fresher setting
- Always ensures clients get fresh enough keys

**See:** [CACHE_DECISION_LOGIC.md](CACHE_DECISION_LOGIC.md) for complete details

---

## Configuration Best Practices

### Relationship: refresh_interval vs cache_duration

**Rule of Thumb:** `cache_duration` should be **less than** `refresh_interval`

```yaml
# ✅ GOOD: Clients get fresh data before service updates
idps:
  - name: "auth0"
    refresh_interval: 3600    # Service fetches every hour
    cache_duration: 900       # Clients cache 15 min

# ⚠️ SUBOPTIMAL: Clients cache longer than service updates
idps:
  - name: "auth0"
    refresh_interval: 1800    # Service fetches every 30 min
    cache_duration: 3600      # Clients cache 1 hour (stale data possible)

# ✅ GOOD: Fast rotation IDP
idps:
  - name: "custom"
    refresh_interval: 600     # Service fetches every 10 min
    cache_duration: 300       # Clients cache 5 min
```

### Recommended Configurations

#### Standard Production Setup
```yaml
idps:
  - name: "auth0"
    refresh_interval: 3600    # 1 hour
    cache_duration: 900       # 15 minutes
    max_keys: 10

  - name: "okta"
    refresh_interval: 3600
    cache_duration: 900
    max_keys: 10
```

**Good for:** Most production environments, stable keys

#### High Security Setup
```yaml
idps:
  - name: "auth0"
    refresh_interval: 1800    # 30 minutes
    cache_duration: 600       # 10 minutes
    max_keys: 10
```

**Good for:** Environments requiring very fresh keys

#### High Traffic Setup
```yaml
idps:
  - name: "auth0"
    refresh_interval: 7200    # 2 hours
    cache_duration: 1800      # 30 minutes
    max_keys: 10
```

**Good for:** Stable IDPs, reduce load on IDPs

---

## KrakenD Caching Integration

### KrakenD Configuration

```json
{
  "endpoints": [
    {
      "endpoint": "/api/protected",
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller/.well-known/jwks.json",
          "cache": true,
          "cache_duration": 900  // KrakenD's cache (should match service)
        }
      }
    }
  ]
}
```

### Three-Level Caching

```
┌──────────────────────────────────────────────────┐
│ Client → JWT with kid="abc123"                  │
└────────────────┬─────────────────────────────────┘
                 ↓
┌──────────────────────────────────────────────────┐
│ KrakenD (Level 1 Cache)                         │
│ - Caches JWKS for 900 seconds                   │
│ - Validates JWT using cached keys               │
│ - Only fetches if cache expired                 │
└────────────────┬─────────────────────────────────┘
                 ↓ (if cache expired)
┌──────────────────────────────────────────────────┐
│ IDP Caller Service (Level 2 Cache)              │
│ - Returns merged JWKS from memory               │
│ - Sets Cache-Control: max-age=900               │
│ - Instant response (no IDP call)                │
└────────────────┬─────────────────────────────────┘
                 ↓ (background goroutines)
┌──────────────────────────────────────────────────┐
│ IDPs (Auth0, Okta, etc.)                        │
│ - Fetched every refresh_interval (3600s)        │
│ - Independent goroutine per IDP                 │
└──────────────────────────────────────────────────┘
```

---

## Monitoring Cache Effectiveness

### Check Last Update Times

```bash
# View when each IDP was last updated
curl http://localhost:8080/status | jq '.[] | {
  name,
  last_updated,
  update_count,
  cache_until
}'
```

**Output:**
```json
{
  "name": "auth0",
  "last_updated": "2026-01-05T10:30:00Z",
  "update_count": 24,
  "cache_until": "2026-01-05T10:45:00Z"
}
```

### Calculate Cache Hit Ratio

```bash
# In your application logs or monitoring
cache_hits = requests_served_from_cache
cache_misses = requests_to_idp_caller_service
cache_hit_ratio = cache_hits / (cache_hits + cache_misses)
```

**Good ratio:** > 80% cache hits

### View Cache Headers

```bash
# Check what cache headers are being sent
curl -I http://localhost:8080/.well-known/jwks.json | grep -i cache
```

---

## Troubleshooting

### Issue: Clients getting stale keys

**Check:**
```bash
# When was data last updated?
curl http://localhost:8080/status/auth0 | jq '.last_updated'

# Is refresh_interval too high?
# Reduce it in config.yaml
```

**Solution:** Reduce `refresh_interval` or `cache_duration`

### Issue: Too many requests to IDP

**Check logs:**
```bash
kubectl logs deployment/idp-caller | grep "Successfully updated JWKS"
```

**Solution:** Increase `refresh_interval` if updates are too frequent

### Issue: Service memory growing

**Check:**
```bash
# How many keys per IDP?
curl http://localhost:8080/status | jq '.[] | {name, key_count, max_keys}'
```

**Solution:** Reduce `max_keys` if IDPs return too many keys

---

## Summary

### Server-Side Caching (In-Memory)
- ✅ Each IDP has independent goroutine
- ✅ Fetches every `refresh_interval` seconds
- ✅ Stores in thread-safe memory cache
- ✅ Instant API responses (no IDP calls)

### Client-Side Caching (HTTP)
- ✅ `Cache-Control: max-age={cache_duration}` header
- ✅ Clients cache based on this value
- ✅ Merged endpoint uses minimum across all IDPs
- ✅ Individual endpoints use per-IDP cache_duration

### Best Practice
```yaml
# Standard production configuration
refresh_interval: 3600      # Service updates hourly
cache_duration: 900         # Clients cache 15 min
max_keys: 10               # Memory protection
```

This provides a good balance between freshness and performance! 🚀

