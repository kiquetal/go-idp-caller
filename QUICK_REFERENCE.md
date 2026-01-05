# 🚀 Quick Reference: Merged JWKS Endpoint

## What Changed?

✅ **NEW: Merged JWKS Endpoint** - All IDP keys combined into one response

## Endpoints Available

### For JOSE JWT Libraries (Multi-IDP Support)
```
GET /.well-known/jwks.json   ⭐ RECOMMENDED
GET /jwks.json
GET /jwks/all
```

**Returns:** Single array with ALL keys from ALL IDPs
```json
{
  "keys": [
    { "kid": "auth0-key-1", ... },
    { "kid": "okta-key-1", ... },
    { "kid": "keycloak-key-1", ... }
  ]
}
```

### For Single IDP
```
GET /jwks/auth0
GET /jwks/okta
GET /jwks/keycloak
```

**Returns:** Keys from specific IDP only

### For Debugging
```
GET /jwks           # All IDPs as map: {"auth0": {...}, "okta": {...}}
GET /status         # Status of all IDPs
GET /status/auth0   # Status of specific IDP
GET /health         # Health check
```

## KrakenD Configuration

### Option 1: Accept tokens from ALL configured IDPs
```json
{
  "auth/validator": {
    "alg": "RS256",
    "jwk_url": "http://idp-caller/.well-known/jwks.json",
    "cache": true,
    "cache_duration": 900
  }
}
```

### Option 2: Accept tokens from specific IDP only
```json
{
  "auth/validator": {
    "alg": "RS256",
    "jwk_url": "http://idp-caller/jwks/auth0",
    "cache": true,
    "cache_duration": 900,
    "issuer": "https://your-tenant.auth0.com/"
  }
}
```

## Testing

```bash
# Get merged keys from all IDPs
curl http://localhost:8080/.well-known/jwks.json | jq '.keys | length'

# Get keys from specific IDP
curl http://localhost:8080/jwks/auth0 | jq '.keys | length'

# Check which IDPs are configured
curl http://localhost:8080/status | jq 'keys'

# Find a specific key by kid
curl http://localhost:8080/.well-known/jwks.json | \
  jq '.keys[] | select(.kid=="YOUR_KID")'
```

## How It Works

1. Service fetches JWKS from each configured IDP
2. Keys are stored separately per IDP
3. Merged endpoint combines all keys into single array
4. JWT library uses `kid` from token to find correct key
5. No need to know which IDP issued the token!

## Benefits

✅ Single endpoint for all IDPs  
✅ Standard OIDC `.well-known` path  
✅ JOSE JWT library compatible  
✅ Automatic key selection via `kid`  
✅ Cache-Control headers included  
✅ Zero config changes when adding/removing IDPs  

## Configuration

No changes needed! Just configure your IDPs in `config.yaml`:

```yaml
idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600
    
  - name: "okta"
    url: "https://domain.okta.com/oauth2/default/v1/keys"
    refresh_interval: 3600
```

All keys from both IDPs will be available at `/.well-known/jwks.json`

## Kubernetes Service URL

```
http://idp-caller.default.svc.cluster.local/.well-known/jwks.json
```

Replace `default` with your namespace if different.

## More Info

- **CACHING_AND_LIMITS.md** - Complete guide on per-IDP caching and key limits
- **MERGED_JWKS_GUIDE.md** - Complete guide with examples in multiple languages
- **KRAKEND_INTEGRATION.md** - Detailed KrakenD setup
- **README.md** - Full API documentation

