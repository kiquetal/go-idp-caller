# Independent IDP Goroutines & Cache Optimization

## Overview

This document explains how **each IDP has its own independent goroutine** with different fetch intervals, and how the system now **uses IDP's Cache-Control headers** to optimize caching dynamically.

---

## How Each IDP Gets Its Own Goroutine

### Architecture

Each IDP configured in `config.yaml` gets its **own independent goroutine** that runs in parallel:

```go
// From main.go
for _, idp := range cfg.IDPs {
    logger.Info("Starting updater for IDP", 
        "name", idp.Name, 
        "url", idp.URL, 
        "interval", idp.RefreshInterval)
    
    updater := jwks.NewUpdater(idp, manager, logger)
    go updater.Start(ctx)  // ← Each IDP = separate goroutine
}
```

### Visual Representation

```
┌─────────────────────────────────────────────────────────────┐
│                  IDP Caller Service                         │
│                                                               │
│  Independent Goroutines:                                     │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Goroutine 1: Auth0                                   │  │
│  │ refresh_interval: 3600s (1 hour)                     │  │
│  │                                                        │  │
│  │ T=0:00 → Fetch                                        │  │
│  │ T=1:00 → Fetch                                        │  │
│  │ T=2:00 → Fetch                                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Goroutine 2: Okta                                    │  │
│  │ refresh_interval: 1800s (30 min)                     │  │
│  │                                                        │  │
│  │ T=0:00 → Fetch                                        │  │
│  │ T=0:30 → Fetch                                        │  │
│  │ T=1:00 → Fetch                                        │  │
│  │ T=1:30 → Fetch                                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Goroutine 3: Keycloak                                │  │
│  │ refresh_interval: 600s (10 min)                      │  │
│  │                                                        │  │
│  │ T=0:00 → Fetch                                        │  │
│  │ T=0:10 → Fetch                                        │  │
│  │ T=0:20 → Fetch                                        │  │
│  │ T=0:30 → Fetch                                        │  │
│  │ ... (every 10 minutes)                                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
│  All goroutines write to thread-safe Manager (RWMutex)      │
└─────────────────────────────────────────────────────────────┘
```

### Configuration Example

```yaml
idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600  # ← Goroutine fetches every 1 hour
    cache_duration: 900
    max_keys: 10

  - name: "okta"
    url: "https://okta.com/oauth2/v1/keys"
    refresh_interval: 1800  # ← Goroutine fetches every 30 minutes
    cache_duration: 600
    max_keys: 10

  - name: "keycloak"
    url: "https://keycloak.com/certs"
    refresh_interval: 600   # ← Goroutine fetches every 10 minutes
    cache_duration: 300
    max_keys: 10
```

**Key Point:** Each IDP's goroutine is **completely independent**. They don't wait for each other.

---

## Dynamic Cache Optimization from IDP Responses

### How It Works Now (Enhanced)

The service now **reads `Cache-Control` headers** from IDP responses and uses them to optimize caching:

```
┌─────────────────────────────────────────────────────────┐
│ 1. Service Fetches from IDP                            │
│    GET https://tenant.auth0.com/.well-known/jwks.json  │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ 2. IDP Responds with Cache-Control Header              │
│                                                          │
│    HTTP/1.1 200 OK                                      │
│    Cache-Control: public, max-age=86400                │
│    Content-Type: application/json                      │
│                                                          │
│    { "keys": [...] }                                    │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ 3. Service Parses max-age=86400                        │
│    "IDP suggests 86400 seconds (24 hours)"             │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ 4. Service Determines Best Cache Duration              │
│                                                          │
│    config.cache_duration: 900 (15 min)                 │
│    idp.max_age: 86400 (24 hours)                       │
│                                                          │
│    → Use minimum (more conservative): 900              │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ 5. Service Returns to Clients                          │
│    Cache-Control: public, max-age=900                  │
└─────────────────────────────────────────────────────────┘
```

### Implementation Details

#### 1. Parse Cache-Control Header

```go
// From internal/jwks/updater.go
func (u *Updater) fetch() (*JWKS, int, error) {
    resp, err := u.client.Do(req)
    
    // Parse Cache-Control header
    cacheControl := resp.Header.Get("Cache-Control")
    idpMaxAge := parseCacheControl(cacheControl)
    
    return &jwks, idpMaxAge, nil
}

func parseCacheControl(cacheControl string) int {
    // Extracts max-age=VALUE from header
    // Examples:
    //   "max-age=86400" → 86400
    //   "public, max-age=3600, must-revalidate" → 3600
    //   "no-cache" → 0
}
```

#### 2. Determine Best Cache Duration

```go
func (u *Updater) determineCacheDuration(idpMaxAge int) int {
    configDuration := u.config.GetCacheDuration()
    
    // If IDP didn't provide max-age, use config
    if idpMaxAge <= 0 {
        return configDuration
    }
    
    // Use MINIMUM to be safe (more conservative)
    if idpMaxAge < configDuration {
        return idpMaxAge  // IDP's suggestion is more strict
    }
    
    return configDuration  // Config is more strict
}
```

**Strategy:** Always use the **minimum** between IDP's suggestion and config. This ensures:
- We don't cache longer than IDP recommends
- We don't cache longer than our config allows
- We're conservative and safe

#### 3. Store Both Values

```go
type IDPData struct {
    CacheDuration     int  `json:"cache_duration"`      // What we're using
    IDPSuggestedCache int  `json:"idp_suggested_cache"` // What IDP suggested
    RefreshInterval   int  `json:"refresh_interval"`    // How often we fetch
}
```

This allows monitoring and comparison.

---

## Example Scenarios

### Scenario 1: IDP Suggests Longer Cache

**Configuration:**
```yaml
idps:
  - name: "auth0"
    cache_duration: 900  # 15 minutes
```

**IDP Response:**
```
Cache-Control: max-age=86400  # 24 hours
```

**Result:**
```json
{
  "cache_duration": 900,         // Using config (more conservative)
  "idp_suggested_cache": 86400   // IDP suggested 24h
}
```

**Clients receive:** `Cache-Control: max-age=900`

**Why?** Our config is more strict (15 min vs 24h), so we use it.

---

### Scenario 2: IDP Suggests Shorter Cache

**Configuration:**
```yaml
idps:
  - name: "custom-idp"
    cache_duration: 1800  # 30 minutes
```

**IDP Response:**
```
Cache-Control: max-age=300  # 5 minutes
```

**Result:**
```json
{
  "cache_duration": 300,          // Using IDP's suggestion (more conservative)
  "idp_suggested_cache": 300      // IDP suggested 5 min
}
```

**Clients receive:** `Cache-Control: max-age=300`

**Why?** IDP rotates keys frequently (5 min), so we respect that.

---

### Scenario 3: IDP Doesn't Provide Cache-Control

**Configuration:**
```yaml
idps:
  - name: "legacy-idp"
    cache_duration: 900
```

**IDP Response:**
```
(no Cache-Control header)
```

**Result:**
```json
{
  "cache_duration": 900,          // Using config
  "idp_suggested_cache": 0        // IDP didn't suggest
}
```

**Clients receive:** `Cache-Control: max-age=900`

**Why?** Fallback to config when IDP doesn't provide guidance.

---

## Different Refresh Intervals Per IDP

### Why Different Intervals?

Each IDP may have different key rotation policies:

| IDP | Typical Rotation | Recommended refresh_interval |
|-----|------------------|------------------------------|
| **Auth0** | Days to weeks | 3600s (1 hour) |
| **Okta** | Days | 3600s (1 hour) |
| **Keycloak** | Hours (configurable) | 1800s (30 min) |
| **Custom IDP** | Very frequent | 600s (10 min) |
| **Azure AD** | Days | 3600s (1 hour) |

### Configuration Strategy

```yaml
idps:
  # Stable IDP - infrequent rotation
  - name: "auth0"
    refresh_interval: 7200  # 2 hours (less load on IDP)
    cache_duration: 1800    # 30 min client cache
    
  # Standard IDP
  - name: "okta"
    refresh_interval: 3600  # 1 hour
    cache_duration: 900     # 15 min client cache
    
  # Fast rotation IDP
  - name: "dev-keycloak"
    refresh_interval: 600   # 10 minutes (need fresh keys)
    cache_duration: 300     # 5 min client cache
```

### Timeline Example

```
Time    Auth0           Okta            Keycloak
        (2 hour)        (1 hour)        (10 min)
─────────────────────────────────────────────────
0:00    FETCH ●         FETCH ●         FETCH ●
0:10    -               -               FETCH ●
0:20    -               -               FETCH ●
0:30    -               -               FETCH ●
0:40    -               -               FETCH ●
0:50    -               -               FETCH ●
1:00    -               FETCH ●         FETCH ●
1:10    -               -               FETCH ●
1:20    -               -               FETCH ●
1:30    -               -               FETCH ●
1:40    -               -               FETCH ●
1:50    -               -               FETCH ●
2:00    FETCH ●         FETCH ●         FETCH ●
```

**Notice:** Each IDP operates independently. Keycloak fetches much more frequently.

---

## Monitoring the System

### Check Status Endpoint

```bash
curl http://localhost:8080/status | jq
```

**Response:**
```json
{
  "auth0": {
    "name": "auth0",
    "last_updated": "2026-01-05T14:30:00Z",
    "key_count": 3,
    "max_keys": 10,
    "cache_duration": 900,
    "idp_suggested_cache": 86400,
    "refresh_interval": 3600,
    "cache_until": "2026-01-05T14:45:00Z",
    "update_count": 24
  },
  "keycloak": {
    "name": "keycloak",
    "last_updated": "2026-01-05T14:40:00Z",
    "key_count": 2,
    "max_keys": 10,
    "cache_duration": 300,
    "idp_suggested_cache": 300,
    "refresh_interval": 600,
    "cache_until": "2026-01-05T14:45:00Z",
    "update_count": 144
  }
}
```

**Key Insights:**
- `cache_duration`: What clients will cache
- `idp_suggested_cache`: What IDP recommended
- `refresh_interval`: How often we fetch from IDP
- `update_count`: Number of fetches (auth0: 24/day, keycloak: 144/day)

### View Logs

```bash
kubectl logs -f deployment/idp-caller | grep "Successfully updated"
```

**Output:**
```json
{
  "level": "INFO",
  "msg": "Successfully updated JWKS",
  "idp": "keycloak",
  "key_count": 2,
  "cache_duration": 300,
  "idp_suggested_cache": 300,
  "refresh_interval": 600,
  "update_count": 144
}
```

---

## Best Practices

### 1. Match refresh_interval to IDP's Rotation

```yaml
# ✅ GOOD: IDP rotates every 6 hours
idps:
  - name: "stable-idp"
    refresh_interval: 3600  # Fetch hourly (well before rotation)
    cache_duration: 900     # Clients cache 15 min
```

### 2. Use Shorter refresh_interval for Fast Rotation

```yaml
# ✅ GOOD: IDP rotates every hour
idps:
  - name: "fast-idp"
    refresh_interval: 600   # Fetch every 10 min (need fresh keys)
    cache_duration: 300     # Clients cache 5 min
```

### 3. Let IDP's Cache-Control Optimize

```yaml
# ✅ GOOD: Let system use IDP's suggestion if better
idps:
  - name: "smart-idp"
    refresh_interval: 3600
    cache_duration: 1800    # 30 min - but system will use IDP's if lower
```

### 4. Balance Load and Freshness

```yaml
# Relationship guide:
refresh_interval: 3600  # How often WE hit the IDP (their load)
cache_duration: 900     # How often CLIENTS hit us (our load)
                        # But system will use min(config, idp_max_age)
```

---

## Future Enhancements

### Dynamic refresh_interval

Currently, `refresh_interval` is static. Future version could adjust based on IDP's `max-age`:

```go
// Future: Adjust refresh_interval dynamically
if idpMaxAge > 0 && idpMaxAge < refreshInterval {
    // IDP rotates faster than we're fetching
    // Consider fetching more frequently
    newInterval := idpMaxAge / 2  // Fetch halfway through IDP's cache
}
```

### Exponential Backoff on Errors

```go
// Future: Retry with backoff on failures
if err != nil {
    backoff := calculateBackoff(consecutiveErrors)
    time.Sleep(backoff)
}
```

---

## Summary

### ✅ Independent Goroutines
- Each IDP = separate goroutine
- Each has its own `refresh_interval`
- Run completely independently
- All write to thread-safe Manager

### ✅ Dynamic Cache Optimization
- Service reads `Cache-Control` from IDP responses
- Extracts `max-age` value
- Uses **minimum** of (config, IDP's suggestion)
- Logs both values for monitoring

### ✅ Configuration
```yaml
idps:
  - name: "example"
    refresh_interval: 3600  # How often service fetches
    cache_duration: 900     # Fallback/max for client caching
                            # System uses min(this, idp_max_age)
```

### ✅ Monitoring
```bash
# Check current state
curl http://localhost:8080/status | jq '.[] | {
  name,
  cache_duration,
  idp_suggested_cache,
  refresh_interval,
  last_updated
}'
```

This architecture provides **optimal flexibility** while respecting both your configuration and each IDP's recommendations! 🚀

