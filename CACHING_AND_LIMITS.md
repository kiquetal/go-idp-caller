# Per-IDP Caching and Key Limits

## Overview

The IDP Caller service now includes **per-IDP caching** and **key count limits** to optimize performance and protect each Identity Provider independently.

## New Features

### 1. **Maximum Keys Per IDP** (`max_keys`)
- **Standard Default: 10 keys per IDP**
- Prevents memory bloat from IDPs returning excessive keys
- Protects against misconfigured or compromised IDPs
- Configurable per IDP for flexibility

### 2. **Per-IDP Cache Duration** (`cache_duration`)
- **Standard Default: 900 seconds (15 minutes)**
- Independent cache control for each IDP
- Allows shorter cache for frequently rotating keys
- Sent as `Cache-Control` header to clients

## Configuration

### config.yaml Example

```yaml
idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600  # Refresh every hour
    max_keys: 10            # Keep max 10 keys (standard)
    cache_duration: 900     # Cache for 15 minutes (standard)
    
  - name: "high-security-idp"
    url: "https://secure.example.com/.well-known/jwks.json"
    refresh_interval: 1800  # Refresh every 30 minutes
    max_keys: 5             # Stricter limit
    cache_duration: 300     # Cache for only 5 minutes
    
  - name: "dev-idp"
    url: "https://dev.example.com/.well-known/jwks.json"
    refresh_interval: 600   # Refresh every 10 minutes
    max_keys: 20            # Allow more keys for testing
    cache_duration: 600     # Cache for 10 minutes
```

## How It Works

### Key Limiting

When JWKS are fetched from an IDP:

1. Service retrieves all keys from IDP endpoint
2. If key count > `max_keys`, truncate to `max_keys`
3. Warning logged: `"Truncating keys to max limit"`
4. Only first N keys are kept (by order in JWKS response)

**Example:**
- IDP returns 25 keys
- `max_keys: 10` configured
- Service keeps only first 10 keys
- Logs: `"original_count": 25, "max_keys": 10`

### Independent Caching

Each IDP has its own cache duration:

**Single IDP Endpoint** (`/jwks/{idp}`):
```http
Cache-Control: public, max-age={idp_cache_duration}
```

**Merged Endpoint** (`/.well-known/jwks.json`):
```http
Cache-Control: public, max-age={minimum_cache_duration_across_all_idps}
```

The merged endpoint uses the **minimum** cache duration to ensure fresh keys for the most restrictive IDP.

## API Response Changes

### Status Endpoint Now Includes

```json
{
  "name": "auth0",
  "jwks": { "keys": [...] },
  "last_updated": "2026-01-05T10:30:00Z",
  "update_count": 42,
  "key_count": 8,              // ✨ NEW: Current number of keys
  "max_keys": 10,              // ✨ NEW: Maximum allowed keys
  "cache_duration": 900,       // ✨ NEW: Cache duration in seconds
  "cache_until": "2026-01-05T10:45:00Z",  // ✨ NEW: Cache valid until
  "last_error": ""
}
```

### HTTP Response Headers

**Single IDP:**
```http
GET /jwks/auth0

Cache-Control: public, max-age=900
Content-Type: application/json
```

**Merged JWKS:**
```http
GET /.well-known/jwks.json

Cache-Control: public, max-age=600
X-Total-Keys: 25
X-IDP-Count: 3
Content-Type: application/json
```

New headers:
- `X-Total-Keys`: Total number of keys across all IDPs
- `X-IDP-Count`: Number of IDPs configured

## Benefits

### 🛡️ Protection

**Per IDP:**
- Prevents one IDP from consuming excessive memory
- Limits impact of misconfigured IDPs
- Protects against DoS via excessive keys

**Example:** If one IDP is compromised and returns 1000 keys, only the first 10 are kept.

### ⚡ Performance

**Optimized Caching:**
- High-security IDPs: shorter cache = fresher keys
- Stable IDPs: longer cache = fewer requests
- Per-IDP tuning for optimal performance

**Example Scenarios:**

| IDP Type | Refresh Interval | Max Keys | Cache Duration | Use Case |
|----------|-----------------|----------|----------------|----------|
| Production Auth0 | 3600s (1h) | 10 | 900s (15m) | Standard |
| Dev Keycloak | 600s (10m) | 20 | 600s (10m) | Frequent changes |
| High Security | 1800s (30m) | 5 | 300s (5m) | Strict freshness |
| Legacy System | 7200s (2h) | 10 | 1800s (30m) | Rarely changes |

### 📊 Observability

**Logs Include:**
```json
{
  "msg": "Successfully updated JWKS",
  "idp": "auth0",
  "keys_count": 8,
  "max_keys": 10,
  "cache_duration": 900,
  "last_updated": "2026-01-05T10:30:00Z"
}
```

**Warning When Truncating:**
```json
{
  "level": "WARN",
  "msg": "Truncating keys to max limit",
  "idp": "problematic-idp",
  "original_count": 25,
  "max_keys": 10
}
```

## Standard Recommendations

### Recommended Defaults

```yaml
# Production IDPs
max_keys: 10
cache_duration: 900  # 15 minutes
refresh_interval: 3600  # 1 hour

# High-Security IDPs
max_keys: 5
cache_duration: 300  # 5 minutes
refresh_interval: 1800  # 30 minutes

# Development IDPs
max_keys: 20
cache_duration: 600  # 10 minutes
refresh_interval: 600  # 10 minutes
```

### Why 10 Keys Standard?

- Most IDPs maintain 2-5 active keys
- Allows for key rotation overlap (old + new keys)
- Provides buffer for multiple key algorithms
- Prevents memory abuse
- Industry standard for OIDC providers

**Typical IDP Key Counts:**
- Auth0: 1-3 keys
- Okta: 2-4 keys
- Keycloak: 2-5 keys
- Azure AD: 2-6 keys
- Google: 2-3 keys

## Testing

### Check Key Counts

```bash
# View key count for specific IDP
curl http://localhost:8080/status/auth0 | jq '.key_count, .max_keys'

# View all IDP key counts
curl http://localhost:8080/status | \
  jq '.[] | {name, key_count, max_keys, cache_duration}'
```

### Test Cache Headers

```bash
# Check cache duration for specific IDP
curl -I http://localhost:8080/jwks/auth0 | grep Cache-Control

# Check merged endpoint cache
curl -I http://localhost:8080/.well-known/jwks.json | grep Cache-Control
```

### Verify Key Limiting

```bash
# Check if keys were truncated
kubectl logs deployment/idp-caller | grep "Truncating keys"

# View original vs actual count
curl http://localhost:8080/status | \
  jq '.[] | select(.key_count == .max_keys) | {name, key_count, max_keys}'
```

## Migration from Previous Version

### No Breaking Changes

If you don't specify `max_keys` or `cache_duration`, defaults are applied:

**Old config:**
```yaml
idps:
  - name: "auth0"
    url: "https://..."
    refresh_interval: 3600
```

**Still works! Equivalent to:**
```yaml
idps:
  - name: "auth0"
    url: "https://..."
    refresh_interval: 3600
    max_keys: 10        # Auto-applied default
    cache_duration: 900  # Auto-applied default
```

### Recommended: Update Config

Add explicit values for clarity:

```yaml
idps:
  - name: "auth0"
    url: "https://..."
    refresh_interval: 3600
    max_keys: 10
    cache_duration: 900
```

## Monitoring

### Key Metrics to Track

1. **Key count vs max_keys** - Are any IDPs hitting limits?
2. **Cache hit rates** - Are cache durations appropriate?
3. **Truncation warnings** - Are IDPs returning too many keys?
4. **Update frequency** - Is refresh_interval optimal?

### Alerting Recommendations

```yaml
# Alert if keys are consistently truncated
- alert: IDPKeysTruncated
  expr: idp_keys_truncated_total > 0
  annotations:
    summary: "IDP {{ $labels.idp }} is returning more keys than max_keys limit"
    
# Alert if cache duration is too short
- alert: IDPCacheTooShort
  expr: idp_cache_duration_seconds < 300
  annotations:
    summary: "IDP {{ $labels.idp }} has very short cache duration"
```

## FAQ

### Q: What happens if an IDP returns more than max_keys?

**A:** The service keeps only the first `max_keys` and logs a warning. This protects against memory issues and misconfigured IDPs.

### Q: Why does the merged endpoint use minimum cache duration?

**A:** To ensure the response is never stale for any IDP. If one IDP requires fresh keys every 5 minutes, the merged response respects that.

### Q: Can I disable key limiting?

**A:** Set `max_keys: 0` or omit it for unlimited keys (not recommended for production).

### Q: How do I know if an IDP is returning too many keys?

**A:** Check logs for "Truncating keys to max limit" warnings, or monitor the `/status` endpoint for `key_count == max_keys`.

### Q: What if I need more than 10 keys from an IDP?

**A:** Increase `max_keys` for that specific IDP. Most IDPs rarely need more than 10 keys.

### Q: Does cache_duration affect when JWKS are fetched?

**A:** No. `refresh_interval` controls fetching. `cache_duration` only affects the `Cache-Control` header sent to clients.

## Summary

✅ **Per-IDP Protection**: Each IDP has independent limits  
✅ **Standard Default**: 10 keys per IDP (industry standard)  
✅ **Flexible Caching**: Configure cache per IDP needs  
✅ **Backward Compatible**: Existing configs work with defaults  
✅ **Better Observability**: Track key counts and cache status  
✅ **Memory Safe**: Prevents abuse from excessive keys  

This makes the service more robust and production-ready!

