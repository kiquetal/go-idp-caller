# Configuration Guide

Complete configuration reference for the IDP JWKS Caller service.

## Quick Start Configuration

```yaml
server:
  port: 8080
  host: "0.0.0.0"

idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600  # Fetch every 1 hour
    cache_duration: 900     # Clients cache 15 min  
    max_keys: 10            # Maximum keys to store

logging:
  level: "info"   # debug, info, warn, error
  format: "json"  # json or text
```

---

## Configuration Parameters

### Server Configuration

```yaml
server:
  port: 8080        # HTTP server port
  host: "0.0.0.0"   # Bind address (0.0.0.0 for all interfaces)
```

### IDP Configuration

Each IDP requires these parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `name` | string | ✅ | - | Unique identifier for the IDP |
| `url` | string | ✅ | - | JWKS endpoint URL (HTTPS recommended) |
| `refresh_interval` | int | ✅ | - | How often service fetches from IDP (seconds) |
| `cache_duration` | int | ❌ | 900 | Maximum client cache time (seconds) |
| `max_keys` | int | ❌ | 10 | Maximum keys to store per IDP |

---

## Understanding the Parameters

### `refresh_interval` - Service Fetching

**Controls:** How often YOUR SERVICE fetches JWKS from the IDP

```yaml
refresh_interval: 3600  # Service fetches every 1 hour
```

**Guidelines:**
- Match or beat IDP's key rotation schedule
- Lower = fresher data, higher load on IDP
- Higher = less load, potentially stale data

**Recommended values:**
- Fast rotation IDP: `600` (10 minutes)
- Standard IDP: `3600` (1 hour)
- Stable IDP: `7200` (2 hours)

### `cache_duration` - Client Caching

**Controls:** Maximum time clients should cache responses

```yaml
cache_duration: 900  # Clients cache up to 15 minutes
```

**Important:** System automatically uses **minimum** of (config, IDP's Cache-Control max-age)

**Example:**
```yaml
cache_duration: 900  # You configure 15 min

# If IDP returns: Cache-Control: max-age=300 (5 min)
# System uses: 300 (IDP knows it rotates fast!)

# If IDP returns: Cache-Control: max-age=86400 (24 hours)
# System uses: 900 (your config is fresher)
```

**Recommended values:**
- High security: `300-600` (5-10 minutes)
- Standard: `900` (15 minutes)
- Low traffic: `1800` (30 minutes)

### `max_keys` - Memory Protection

**Controls:** Maximum number of keys to store per IDP

```yaml
max_keys: 10  # Store up to 10 keys
```

**Why limit keys:**
- Most IDPs maintain 2-5 active keys
- Protects against memory exhaustion
- System logs warning if IDP returns more keys

**Recommended:** Keep at `10` (standard default)

---

## The Relationship: refresh_interval vs cache_duration

```
┌──────────────────────────────┐
│ IDP (Auth0, Okta, etc.)      │
│ Actual JWKS source           │
└──────────┬───────────────────┘
           │
           │ refresh_interval: 3600s
           │ Service fetches hourly
           ↓
┌──────────────────────────────┐
│ Your Service (Memory Cache)  │
│ Always available, instant    │
└──────────┬───────────────────┘
           │
           │ cache_duration: 900s
           │ Clients cache 15 min
           ↓
┌──────────────────────────────┐
│ Clients (KrakenD, Apps)      │
│ HTTP cache                   │
└──────────────────────────────┘
```

**Rule of Thumb:** `cache_duration ≤ refresh_interval`

**Why:** Clients shouldn't cache longer than service updates

---

## Configuration Examples

### Standard Production

```yaml
idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600  # 1 hour
    cache_duration: 900     # 15 min
    max_keys: 10

  - name: "okta"
    url: "https://domain.okta.com/oauth2/default/v1/keys"
    refresh_interval: 3600
    cache_duration: 900
    max_keys: 10
```

**Good for:** Most production environments

### High Security / Fast Rotation

```yaml
idps:
  - name: "dev-keycloak"
    url: "https://keycloak.dev/realms/dev/protocol/openid-connect/certs"
    refresh_interval: 600   # 10 minutes
    cache_duration: 300     # 5 minutes
    max_keys: 10
```

**Good for:** IDPs that rotate keys frequently

### Stable / Low Traffic

```yaml
idps:
  - name: "stable-auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 7200  # 2 hours
    cache_duration: 1800    # 30 minutes
    max_keys: 10
```

**Good for:** Stable keys, reduce load

---

## Caching Strategy

### Two-Level Caching

**Level 1: Server-Side (In-Memory)**
- Each IDP has independent goroutine
- Fetches every `refresh_interval` seconds
- Thread-safe memory cache
- Instant API responses

**Level 2: Client-Side (HTTP Headers)**
- `Cache-Control: max-age={cache_duration}` header
- Clients cache based on this value
- Reduces load on service

### How Cache Duration is Determined

```
1. Each IDP calculates:
   actual_cache = min(config.cache_duration, idp.max_age)

2. Individual endpoints use their own duration:
   GET /jwks/auth0 → Cache-Control: max-age=900
   GET /jwks/keycloak → Cache-Control: max-age=300

3. Merged endpoint uses minimum across ALL IDPs:
   GET /.well-known/jwks.json → Cache-Control: max-age=300
```

**Why minimum for merged?** To ensure ALL IDP keys stay fresh

---

## Monitoring Configuration

### Check Current Settings

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  refresh_interval,
  cache_duration,
  idp_suggested_cache,
  key_count,
  max_keys
}'
```

**Example output:**
```json
{
  "name": "auth0",
  "refresh_interval": 3600,
  "cache_duration": 900,
  "idp_suggested_cache": 86400,
  "key_count": 3,
  "max_keys": 10
}
```

**What to check:**
- `cache_duration` vs `idp_suggested_cache` - System uses minimum
- `key_count` vs `max_keys` - Watch for truncation warnings
- `refresh_interval` should match IDP's rotation schedule

---

## Common IDP URLs

### Auth0
```yaml
url: "https://{tenant}.auth0.com/.well-known/jwks.json"
refresh_interval: 3600
```

### Okta
```yaml
url: "https://{domain}.okta.com/oauth2/default/v1/keys"
refresh_interval: 3600
```

### Keycloak
```yaml
url: "https://{domain}/realms/{realm}/protocol/openid-connect/certs"
refresh_interval: 3600
```

### Azure AD
```yaml
url: "https://login.microsoftonline.com/common/discovery/v2.0/keys"
refresh_interval: 3600
```

### Google
```yaml
url: "https://www.googleapis.com/oauth2/v3/certs"
refresh_interval: 3600
```

### AWS Cognito
```yaml
url: "https://cognito-idp.{region}.amazonaws.com/{userPoolId}/.well-known/jwks.json"
refresh_interval: 3600
```

---

## Logging Configuration

```yaml
logging:
  level: "info"   # debug, info, warn, error
  format: "json"  # json or text
```

### Log Levels

- `debug`: Detailed information, fetches, cache decisions
- `info`: Normal operations, successful updates
- `warn`: Key truncation, unusual conditions  
- `error`: Failed fetches, errors

### Log Format

**JSON (recommended for production):**
```json
{
  "level": "INFO",
  "msg": "Successfully updated JWKS",
  "idp": "auth0",
  "key_count": 3,
  "cache_duration": 900
}
```

**Text (easier for local development):**
```
INFO Successfully updated JWKS idp=auth0 key_count=3 cache_duration=900
```

---

## Best Practices

### ✅ Do

- Use HTTPS URLs for all IDPs
- Set `refresh_interval` ≤ IDP's rotation period
- Start with standard values (3600/900)
- Monitor `idp_suggested_cache` in status endpoint
- Use JSON logging in production
- Set appropriate resource limits in Kubernetes

### ❌ Don't

- Set `cache_duration` > `refresh_interval` (wasteful)
- Use HTTP URLs in production
- Set `refresh_interval` too low (< 300) without good reason
- Ignore "Truncating keys" warnings in logs
- Set `max_keys` too low (< 5)

---

## Configuration Validation

### Check if config is valid

```bash
# Test locally
go run main.go

# Check for errors in logs
kubectl logs deployment/idp-caller | grep -i error
```

### Verify IDPs are accessible

```bash
# Test each IDP URL manually
curl -I https://tenant.auth0.com/.well-known/jwks.json
```

### Validate YAML syntax

```bash
cat config.yaml | python3 -c "import yaml, sys; yaml.safe_load(sys.stdin)"
```

---

## Summary

**Essential configuration:**
```yaml
idps:
  - name: "idp-name"              # Unique identifier
    url: "https://..."            # JWKS endpoint
    refresh_interval: 3600        # Service fetches hourly
    cache_duration: 900           # Clients cache 15 min
    max_keys: 10                  # Memory protection
```

**Key concepts:**
- `refresh_interval`: How often service fetches (server-side)
- `cache_duration`: Maximum client cache time (client-side)
- System uses `min(config, IDP's max-age)` for client caching
- Merged endpoint uses minimum across all IDPs

**Monitoring:**
```bash
curl http://localhost:8080/status
```

See [README.md](README.md) for complete usage guide.

