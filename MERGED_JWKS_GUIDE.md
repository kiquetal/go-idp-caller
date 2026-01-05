# Merged JWKS Endpoint Guide

## 🎯 Primary Endpoint for Multi-IDP JWT Validation

The merged JWKS endpoint is the **recommended** way to use this service. It combines keys from all configured Identity Providers into a single standard OIDC-compliant endpoint.

---

## Why Use Merged JWKS?

### The Problem: Multiple IDPs

**Traditional approach:**
```
JWT from Auth0     → Validate against Auth0's JWKS
JWT from Okta      → Validate against Okta's JWKS  
JWT from Keycloak  → Validate against Keycloak's JWKS
```

❌ Need to know which IDP issued each token  
❌ Configure multiple JWKS URLs  
❌ Complex routing logic

### The Solution: One Endpoint

**Merged approach:**
```
ANY JWT → Validate against merged JWKS at /.well-known/jwks.json
System automatically finds correct key by kid
```

✅ Single endpoint for all IDPs  
✅ Automatic key selection by `kid`  
✅ Zero per-IDP configuration  
✅ Standard OIDC format

---

## The Endpoint

### URL
```
GET /.well-known/jwks.json
```

Standard OpenID Connect Discovery path.

### Example Request
```bash
curl http://idp-caller/.well-known/jwks.json
```

### Response Format

```json
{
  "keys": [
    {
      "kid": "ZWliPr4t9ciW0FS",
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "e": "AQAB",
      "n": "x5kvoAVGraJQ0xDOihwrSkcKa..."
    },
    {
      "kty": "RSA",
      "use": "sig",
      "n": "yooxUH7Ky4X3QopBi7oX9HyAJSU...",
      "e": "AQAB",
      "kid": "qjPcUaB_zp5mw_GBBR2Cy",
      "x5t": "lxzRNmKF8gNtzdWu15Ysb3EnpLo",
      "x5c": ["MIIDETCCAfmgAwIBAgIJIxctA..."],
      "alg": "RS256"
    },
    {
      "kid": "HU6QyrltTBhzTJBo57zmq...",
      "kty": "RSA",
      "alg": "RSA-OAEP",
      "use": "enc",
      "x5c": ["MIICnTCCAYUCBgGWaAY..."],
      "x5t": "T11HNbfGX92zZ_RmvaUJ2hiTFY0",
      "x5t#S256": "uwEvp-AbY4R5nYxZWCsVoq...",
      "n": "i1lxwK9cFsGIj7SNa99U1t8...",
      "e": "AQAB"
    }
  ]
}
```

**Features:**
- ✅ All keys from all IDPs in single `keys` array
- ✅ Preserves all fields from original IDPs (kid, kty, alg, use, n, e, x5c, x5t, x5t#S256)
- ✅ Field order varies by IDP (kid can be first or last)
- ✅ Supports signing (`use: "sig"`) and encryption (`use: "enc"`) keys
- ✅ Includes X.509 certificate chains when provided
- ✅ RFC 7517 compliant

### Response Headers

```http
HTTP/1.1 200 OK
Content-Type: application/json
Cache-Control: public, max-age=300
X-Total-Keys: 9
X-IDP-Count: 3
```

- `Cache-Control`: Minimum cache duration across all IDPs
- `X-Total-Keys`: Total number of keys
- `X-IDP-Count`: Number of configured IDPs

---

## How It Works

### JWT Validation Flow

```
1. JWT Header contains:
   {
     "kid": "qjPcUaB_zp5mw_GBBR2Cy",
     "alg": "RS256"
   }

2. Client fetches merged JWKS

3. Client finds key where kid matches

4. Client verifies JWT signature using that key

5. ✅ Token validated - no need to know which IDP!
```

The `kid` (Key ID) is unique across all IDPs, enabling automatic key selection.

---

## Integration Examples

### 1. KrakenD API Gateway

**Recommended for:** Multi-IDP environments

```json
{
  "version": 3,
  "endpoints": [
    {
      "endpoint": "/api/protected",
      "method": "GET",
      "backend": [{
        "url_pattern": "/resource",
        "host": ["http://backend:8080"]
      }],
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller/.well-known/jwks.json",
          "cache": true,
          "cache_duration": 900
        }
      }
    }
  ]
}
```

✅ Accepts JWTs from Auth0, Okta, Keycloak automatically!

### 2. Node.js with jose

```javascript
const { createRemoteJWKSet, jwtVerify } = require('jose');

const JWKS = createRemoteJWKSet(
  new URL('http://idp-caller/.well-known/jwks.json')
);

async function verifyToken(token) {
  const { payload } = await jwtVerify(token, JWKS);
  return payload;
}
```

### 3. Python with python-jose

```python
from jose import jwt
import requests

jwks_url = 'http://idp-caller/.well-known/jwks.json'
jwks = requests.get(jwks_url).json()

payload = jwt.decode(token, jwks, algorithms=['RS256'])
```

### 4. Go with lestrrat-go/jwx

```go
set, _ := jwk.Fetch(ctx, "http://idp-caller/.well-known/jwks.json")

token, _ := jwt.Parse([]byte(tokenString), 
    jwt.WithKeySet(set))
```

### 5. Java with nimbus-jose-jwt

```java
JWKSet jwkSet = JWKSet.load(
    new URL("http://idp-caller/.well-known/jwks.json")
);

SignedJWT signedJWT = SignedJWT.parse(tokenString);
String kid = signedJWT.getHeader().getKeyID();
RSAKey rsaKey = (RSAKey) jwkSet.getKeyByKeyId(kid);

JWSVerifier verifier = new RSASSAVerifier(rsaKey);
signedJWT.verify(verifier);
```

---

## Cache Behavior

### Per-IDP Cache Duration

Each IDP calculates its own cache duration:
```
Auth0:     min(config: 900, IDP: 86400) = 900
Keycloak:  min(config: 900, IDP: 300) = 300
Okta:      min(config: 1800, IDP: 3600) = 1800
```

### Merged Endpoint Uses Minimum

```
Merged: min(900, 300, 1800) = 300
Response: Cache-Control: max-age=300
```

**Why minimum?** Ensures ALL IDP keys stay fresh, even if one IDP rotates frequently (like Keycloak every 5 minutes).

### Individual vs Merged

```bash
# Individual endpoints use their own cache
GET /jwks/auth0    → Cache-Control: max-age=900
GET /jwks/keycloak → Cache-Control: max-age=300

# Merged uses minimum across all
GET /.well-known/jwks.json → Cache-Control: max-age=300
```

---

## When to Use Merged vs Individual

### Use Merged (Recommended)

✅ Multi-IDP environment  
✅ Don't know which IDP issued tokens  
✅ Want single endpoint  
✅ Standard OIDC compliance

**Example:** API gateway accepting tokens from any of your IDPs

### Use Individual

✅ Single IDP environment  
✅ Want IDP-specific cache settings  
✅ Need to restrict to specific IDP

**Example:** Microservice that only accepts Auth0 tokens

```bash
GET /jwks/auth0
```

---

## Benefits

### ✅ Simplified Configuration

**Before (3 IDPs):**
```json
{
  "jwks_urls": [
    "https://auth0.com/.well-known/jwks.json",
    "https://okta.com/oauth2/v1/keys",
    "https://keycloak.com/certs"
  ]
}
```

**After:**
```json
{
  "jwk_url": "http://idp-caller/.well-known/jwks.json"
}
```

### ✅ Zero Downtime IDP Changes

Add/remove IDPs without changing application config:

```yaml
# Add new IDP in config.yaml
idps:
  - name: "new-azure"
    url: "https://login.microsoftonline.com/..."
```

Applications automatically get new keys on next fetch!

### ✅ Automatic Key Rotation

- Service fetches every `refresh_interval`
- Clients cache based on `cache_duration`
- No manual intervention needed

### ✅ Standard Compliance

- Standard OIDC Discovery path
- RFC 7517 JWKS format
- Compatible with all major JWT libraries

---

## Monitoring

### Check Merged Keys

```bash
# View all keys
curl http://localhost:8080/.well-known/jwks.json | jq '.keys[] | {kid, alg, use}'

# Count total keys
curl http://localhost:8080/.well-known/jwks.json | jq '.keys | length'
```

### Check Per-IDP Status

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  key_count,
  cache_duration,
  idp_suggested_cache
}'
```

### Verify Cache Headers

```bash
curl -I http://localhost:8080/.well-known/jwks.json | grep -i cache
```

---

## Troubleshooting

### Token Validation Fails

**Check if key exists:**
```bash
KID="your-kid-value"
curl http://localhost:8080/.well-known/jwks.json | jq ".keys[] | select(.kid==\"$KID\")"
```

**Check IDP status:**
```bash
curl http://localhost:8080/status | jq '.[] | select(.last_error != "")'
```

### Missing Keys from IDP

**Check logs:**
```bash
kubectl logs deployment/idp-caller | grep "Successfully updated"
```

**Verify IDP is reachable:**
```bash
curl -I https://your-idp.com/.well-known/jwks.json
```

### Stale Keys

**Check last update:**
```bash
curl http://localhost:8080/status | jq '.[] | {name, last_updated}'
```

**Verify refresh_interval matches IDP rotation:**
```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  refresh_interval,
  idp_suggested_cache
}'
```

If `idp_suggested_cache` < `refresh_interval`, update config!

---

## Best Practices

### 1. Use Standard Path

✅ `/.well-known/jwks.json` (OIDC standard)  
❌ Custom paths

### 2. Enable Client Caching

```json
{
  "cache": true,
  "cache_duration": 900
}
```

Reduces load and improves performance.

### 3. Monitor Cache Effectiveness

```bash
# Check cache headers are being sent
curl -I http://localhost:8080/.well-known/jwks.json

# Verify clients are caching
# (check your application logs)
```

### 4. Test with Tokens from Each IDP

```bash
# Verify tokens from all IDPs work
curl -H "Authorization: Bearer $AUTH0_TOKEN" http://api/protected
curl -H "Authorization: Bearer $OKTA_TOKEN" http://api/protected
curl -H "Authorization: Bearer $KEYCLOAK_TOKEN" http://api/protected
```

### 5. Set Up Monitoring

Alert on:
- Merged endpoint returns 0 keys
- Any IDP has `last_error`
- Cache duration drops unexpectedly

---

## Summary

**Endpoint:**
```
GET /.well-known/jwks.json
```

**Response:**
```json
{
  "keys": [/* all keys from all IDPs */]
}
```

**Benefits:**
- Single endpoint for all IDPs
- Automatic key selection by kid
- Standard OIDC compliance
- Zero-config per IDP

**Use Case:**
Multi-IDP environments where you want to validate JWTs from any configured provider without knowing the issuer upfront.

**See also:**
- [README.md](README.md) - General usage
- [CONFIGURATION.md](CONFIGURATION.md) - Configuration details
- [KRAKEND_INTEGRATION.md](KRAKEND_INTEGRATION.md) - API gateway setup

