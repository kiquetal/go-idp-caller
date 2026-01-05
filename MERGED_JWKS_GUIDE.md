# JWKS Merged Endpoint - Usage Guide

## Overview

The IDP Caller service provides a **merged JWKS endpoint** that combines keys from all configured Identity Providers into a single JSON array. This format is compatible with JOSE JWT libraries and standard JWT validators.

## Why Use Merged JWKS?

### Problem
When you have multiple Identity Providers (Auth0, Okta, Keycloak, etc.), each provides their own JWKS endpoint. JWT libraries need to know which IDP to query based on the token's issuer.

### Solution
The merged JWKS endpoint combines all keys from all IDPs into a single endpoint. The JWT library uses the `kid` (Key ID) claim in the JWT header to automatically find the correct public key, regardless of which IDP issued the token.

## Merged Endpoint URLs

All three URLs return the same merged response:

- `GET /.well-known/jwks.json` (Standard OIDC path)
- `GET /jwks.json` (Simple path)
- `GET /jwks/all` (Explicit path)

## Response Format

### Example Merged Response

```json
{
  "keys": [
    {
      "kid": "ZWliPr4t9ciW0FS",
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "e": "AQAB",
      "n": "x5kvoAVGraJQ0xDOihwrSkcKaK1Aw8WlfYNJwE_99VjDbJlkuovDFHvCiE9USiIvFlvRrNozUKKnsepOTCnD5Q0DUMQz6Yox-Z2E2xVyEJSWWEssWwXeDkqAYbmKZ7z1YDEN1Z5G3ug3FqpfpswtJsi9N1iekKl1cThmReJfDodD_7Q6vw72AIZXnGa3SrFpmPyOBELJNrUW2kfFwWHl-IdeTXAKmG0GgAkEGsXgW7GxYtdajeOzuoF-yzAA-4VQCQ4lDFfmBshax8DY6T9QAdHyFBM9pMlwSz6x1Jh8s25iPDkNeeRghK4-A1mP0NtxeSZhnl3RTovwo_yGOa1qWQ"
    },
    {
      "kty": "RSA",
      "use": "sig",
      "n": "yooxUH7Ky4X3QopBi7oX9HyAJSU-y_2dfyjV9Bv6odTKSpXRFZGyFgThnFS6vQBqwTOWLNW2G2d7duhpV211P71GebOTjLX3ndtLBPiqs7KT8DlMWNfYAs_Tx2ncqaI0Gxr85gOqVbKMMrnzDH6e0FgSMQXUXoP4iXC9snLUldpjVvRBaqB0tdUdKJChUbR0JyLB0i0e_ohnNgYWc7FkR2FHoW-H_DwJWhoifDOS5sj-AH7brO5bpBHHh8dQ02HSrEI0U8DnHYjy_C6b_AcPWoU7ZLfIct9UQdKYUzHuvk8Cq12C4JRLFTU_Mf7NH7SgaBuhu6gyNzZPZdMlLo1zkw",
      "e": "AQAB",
      "kid": "qjPcUaB_zp5mw_GBBR2Cy",
      "x5t": "lxzRNmKF8gNtzdWu15Ysb3EnpLo",
      "x5c": [
        "MIIDETCCAfmgAwIBAgIJIxctA6pQC7bdMA0GCSqGSIb3DQEBCwUAMCYxJDAiBgNVBAMTG3RpZ29pZC1wcm9kLWhuLnVzLmF1dGgwLmNvbTAeFw0yMjA4MTEyMzI2MTlaFw0zNjA0MTkyMzI2MTlaMCYxJDAiBgNVBAMTG3RpZ29pZC1wcm9kLWhuLnVzLmF1dGgwLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMqKMVB+ysuF90KKQYu6F/R8gCUlPsv9nX8o1fQb+qHUykqV0RWRshYE4ZxUur0AasEzlizVthtne3boaVdtdT+9Rnmzk4y1953bSwT4qrOyk/A5TFjX2ALP08dp3KmiNBsa/OYDqlWyjDK58wx+ntBYEjEF1F6D+IlwvbJy1JXaY1b0QWqgdLXVHSiQoVG0dCciwdItHv6IZzYGFnOxZEdhR6Fvh/w8CVoaInwzkubI/gB+26zuW6QRx4fHUNNh0qxCNFPA5x2I8vwum/wHD1qFO2S3yHLfVEHSmFMx7r5PAqtdguCUSxU1PzH+zR+0oGgbobuoMjc2T2XTJS6Nc5MCAwEAAaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUjEuWROWNeKK+HFTflLhMoHex5O0wDgYDVR0PAQH/BAQDAgKEMA0GCSqGSIb3DQEBCwUAA4IBAQCfjMHKFH+LtFo0OtRAJGpvWbD0p0r30aImsCQJninvrUhLUSGi39mAceoPZpywgxOqpnkI9n4LP35+6WQMaumBYoQAH7ywdSsZ9ymZFgpYhoycqKH2vxoiPj+rPrLFuWhiLBX9FZtOt4Qp5IoXkuD/H+B6wHr8vmSKvKm0rfdGv9nM+PEDBGSSOgzFqCsVGj6eh6uWPLJxdMKHchhiSERsLVRzEHdUrvue6ce1WTBAjcMRHk40ytO/uHdpH0br13qHnCnVREEwU2WgDA6FJq0Hr+Av9z+qw/Plr8uOMCRdMO0nBJc8dMmPNeGgFbejjTzYn4Vj5bD5PbPHK4PNAF1n"
      ],
      "alg": "RS256"
    },
    {
      "kty": "RSA",
      "use": "sig",
      "n": "zbakHv_5KK_Kyx69mbze33UMtWMf8blEAl_tfhnr40DdZwcAxQFUiWn8n3QhUbaq63Cpn7dqM4naMRcxhDGx5zXKVGmsdoD3zdMlUDinNByPuvuT15Z6EwCj_OHr7CeHGhT6nQwanJKB_1Qy8sb7BtuIZdjGiyfbn-bzBoHndAnj8IXmADdHmegWZGJcy3leodkUYu3p_1UqwMv7EXMfzmX8nB2mZzSg5aYoaJQ_p1F2Ww8IyD4SGqHL07obUcOvR3AZ0tl07dcC6431D6_gmEhNKzUox5cp9KkBg2MtzH_0nER96z90sIfuEWBSiQsyNGBDZuYczoTnvQs3NQUNGw",
      "e": "AQAB",
      "kid": "IREH-BMO8CJ_iWhSpO-Uv",
      "x5t": "vJsqFqQty0zz8-O0vJW3VhTxOzs",
      "x5c": [
        "MIIDETCCAfmgAwIBAgIJNnUIOvh+hytTMA0GCSqGSIb3DQEBCwUAMCYxJDAiBgNVBAMTG3RpZ29pZC1wcm9kLWhuLnVzLmF1dGgwLmNvbTAeFw0yMjA4MTEyMzI2MTlaFw0zNjA0MTkyMzI2MTlaMCYxJDAiBgNVBAMTG3RpZ29pZC1wcm9kLWhuLnVzLmF1dGgwLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM22pB7/+SivyssevZm83t91DLVjH/G5RAJf7X4Z6+NA3WcHAMUBVIlp/J90IVG2qutwqZ+3ajOJ2jEXMYQxsec1ylRprHaA983TJVA4pzQcj7r7k9eWehMAo/zh6+wnhxoU+p0MGpySgf9UMvLG+wbbiGXYxosn25/m8waB53QJ4/CF5gA3R5noFmRiXMt5XqHZFGLt6f9VKsDL+xFzH85l/Jwdpmc0oOWmKGiUP6dRdlsPCMg+Ehqhy9O6G1HDr0dwGdLZdO3XAuuN9Q+v4JhITSs1KMeXKfSpAYNjLcx/9JxEfes/dLCH7hFgUokLMjRgQ2bmHM6E570LNzUFDRsCAwEAAaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUyZrVvy+zzjILkfMnUfR1IXqiU6cwDgYDVR0PAQH/BAQDAgKEMA0GCSqGSIb3DQEBCwUAA4IBAQBM1avOrp030E8lBBSsIl73Vz/C41FbDetQpdcPG1vVWC1iKEztp7SVbdvEQGhX5MomOH27+uYvGaSTAqPviygfciz1L5m8L+sAKY9wgktDw+Q/ZZE//svZYQx165E5oc52K2rbraebXHVvG33+Zc3xX6z543F4J9Lef2jk5Jcl7eLUYOQazhvoe45B+ElKrgo/8jSDXbCVyMkUQC6V7KKD1pc0rlGw91+ARmU+Q0tuXnSrYRAuHH8ug9kRWlf5Qv9187PSKCvQ8zVbD15xSnlRPbeo/LSvXKvRBxlFWPqDA+ZFxA49NIp4AVy/D9nTIQ7DCES72PDnCQCSQZauHNP6"
      ],
      "alg": "RS256"
    },
    {
      "kid": "HU6QyrltTBhzTJBo57zmqWRztm5a6J-7OBBNbWivZAM",
      "kty": "RSA",
      "alg": "RSA-OAEP",
      "use": "enc",
      "x5c": [
        "MIICnTCCAYUCBgGWaAY+tDANBgkqhkiG9w0BAQsFADASMRAwDgYDVQQDDAdwaG9lbml4MB4XDTI1MDQyNDEzMzc0MVoXDTM1MDQyNDEzMzkyMVowEjEQMA4GA1UEAwwHcGhvZW5peDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAItZccCvXBbBiI+0jWvfVNbfIRzkrcl7I3/w+misJmEKH6a43bpR9TypRGlEfhYpgUpKBL/uyDhsJT/KIQ/4OjXQzhgvOsWrnPx7AMC3gO9w/usqZLeLD8K00znr5LlJVvgpr62Qz82jiSKMB7MmU2STmQigBB4eqI3oQcYh/tb3YED+sNtIs6VKZQXlcxmtE8rA6Dc5er1EdDZrdYvuvXhf0o+CeJSv2eZQLuLunA+ZMpmR31k9gDWQ2GOKZ3l+jJ1AyRNEm1JMWb8MKUJShGCADKh+vLj8d6Gpj7yZEvY6MavIs5nD4zL7lFQW9FGAOhEu2dKMtb9ALUk380Z9gpUCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEAbD/HLY6s/ueCp7nalV64xnYrnGXwLV5AxtHEZ1r4Q0LSU9LAFRSCJI2evWank8hrjwewswS4+U7u2UA7d6ycTOKISDY1zqjONa/dET8FlZ7Vw+LHkT8rXjXc35jMr26+Y3qhPXLpQnUDu1D5nMUXluQKFMzVs9NSrULLg9XZrKI8G6CUB5qH9zpdBmv9+JVrf9cznpOYMQ37mrQ9314FnVlLJ4zTdz/wzPDJNb1DOzwLvoUbggvdxE/BEH8Ck252tsWs83uyOoPpXHWMgKlK3rcvdtlU2ypkFPk5m1OWjhfuYwoiuyPluPgGNnm+eoOYbrbLBgE2/3ZivSS/LRPzaA=="
      ],
      "x5t": "T11HNbfGX92zZ_RmvaUJ2hiTFY0",
      "x5t#S256": "uwEvp-AbY4R5nYxZWCsVoqCe7SxYQ9odVknT9BKKJEQ",
      "n": "i1lxwK9cFsGIj7SNa99U1t8hHOStyXsjf_D6aKwmYQofprjdulH1PKlEaUR-FimBSkoEv-7IOGwlP8ohD_g6NdDOGC86xauc_HsAwLeA73D-6ypkt4sPwrTTOevkuUlW-CmvrZDPzaOJIowHsyZTZJOZCKAEHh6ojehBxiH-1vdgQP6w20izpUplBeVzGa0TysDoNzl6vUR0Nmt1i-69eF_Sj4J4lK_Z5lAu4u6cD5kymZHfWT2ANZDYY4pneX6MnUDJE0SbUkxZvwwpQlKEYIAMqH68uPx3oamPvJkS9joxq8izmcPjMvuUVBb0UYA6ES7Z0oy1v0AtSTfzRn2ClQ",
      "e": "AQAB"
    },
    {
      "kid": "ToYW17Cf6E7EcbFl4-8RXBifujtWqQuW3DDG9wPFjCo",
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "x5c": [
        "MIICnTCCAYUCBgGWaAZB0zANBgkqhkiG9w0BAQsFADASMRAwDgYDVQQDDAdwaG9lbml4MB4XDTI1MDQyNDEzMzc0MloXDTM1MDQyNDEzMzkyMlowEjEQMA4GA1UEAwwHcGhvZW5peDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANIxmoSefbQLeI4oduHdont1uw9YAPneYJU9pmQOSj0NQLSTGPpdNLZfh/oRwqxgNEdF1SWxeungo+/uXqZm7b7xIG3YDWEmVVfwRBoP3X8rHhMj8G6qBoYSQFl5lUVUKqmrh9DHu0G8EHFz9DiFpzliQLZ16qTn1t761noFps5wXn4EBRCuQNtUVfubRiaxNEplyzqUCkoPFsIcmZ0lf1debCfKPVoUcfxtTtjsM2kSasQ3qd4vrNndhMBVVhlCXVLjsw8zjPDe6OUeXJLL6c4+emM593TKDJFYIzXLE5zf1alyK/vZnczoU9QtER4EWI7WArBPbaL56E+DaqQMyRUCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEAfPtBou4igMh1SWmZfou6U8g3Y2TizuZi1o15fRcT90Ok+tfem4nCHXAlVGDpCtvVu4CbqpjyojfeaCuLEOHigKN6v/iuhZEEQiz/6WumKWJT8BsQ2xdEgllBdkNtGyv6xt4PFqrrdHxdKet0EUX0XyDaHbe0l5r+0RA4HuOHMhtmo7vVg+IZWZ4xWJtPG9jh+CJb0C2xdubxtUvcTgXr9+X/4yf5f/jT3G1GR+t1vTKrc/zTCEg7iUmIu/c1xPOSRY0f8dm2cYGZg8SXXLt/RYB2ZNOerQrVj/OSnVnB6YJKWESAOtPR06q39dJm+r0acONF6KIBfsCn+gXnzx1hkg=="
      ],
      "x5t": "ui9QWdikW76oeW5MbltZG60HbWU",
      "x5t#S256": "_qed8A4cRm7uuAJBwBBxQwfUYiVi-SAPz9moaZEQPbo",
      "n": "0jGahJ59tAt4jih24d2ie3W7D1gA-d5glT2mZA5KPQ1AtJMY-l00tl-H-hHCrGA0R0XVJbF66eCj7-5epmbtvvEgbdgNYSZVV_BEGg_dfyseEyPwbqoGhhJAWXmVRVQqqauH0Me7QbwQcXP0OIWnOWJAtnXqpOfW3vrWegWmznBefgQFEK5A21RV-5tGJrE0SmXLOpQKSg8WwhyZnSV_V15sJ8o9WhRx_G1O2OwzaRJqxDep3i-s2d2EwFVWGUJdUuOzDzOM8N7o5R5cksvpzj56Yzn3dMoMkVgjNcsTnN_VqXIr-9mdzOhT1C0RHgRYjtYCsE9tovnoT4NqpAzJFQ",
      "e": "AQAB"
    }
  ]
}
```

**Key Features:**
- All keys from all configured IDPs merged into a single `keys` array
- Each key includes all fields from the original IDP (kid, kty, alg, use, n, e, x5c, x5t, x5t#S256, etc.)
- Note: Some keys may have `kid` at the end of the object (field order varies by IDP)
- Full X.509 certificate chains (`x5c`) included when provided by IDP
- Supports both signing keys (`use: "sig"`) and encryption keys (`use: "enc"`)
- Compatible with JOSE JWT libraries that expect RFC 7517 JWKS format

## Usage Examples

### 1. KrakenD Configuration

```json
{
  "endpoints": [
    {
      "endpoint": "/api/protected",
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller//.well-known/jwks.json",
          "cache": true,
          "cache_duration": 900
        }
      }
    }
  ]
}
```

### 2. Node.js with jose Library

```javascript
const { createRemoteJWKSet, jwtVerify } = require('jose');

const JWKS = createRemoteJWKSet(
  new URL('http://idp-caller/.well-known/jwks.json')
);

async function verifyToken(token) {
  try {
    const { payload } = await jwtVerify(token, JWKS);
    console.log('Token verified:', payload);
    return payload;
  } catch (error) {
    console.error('Token verification failed:', error);
    throw error;
  }
}
```

### 3. Python with python-jose

```python
from jose import jwt
import requests

# Fetch JWKS
jwks_url = 'http://idp-caller/.well-known/jwks.json'
jwks = requests.get(jwks_url).json()

# Verify token
try:
    payload = jwt.decode(
        token,
        jwks,
        algorithms=['RS256'],
        options={'verify_aud': False}
    )
    print('Token verified:', payload)
except jwt.JWTError as e:
    print('Token verification failed:', e)
```

### 4. Go with golang-jwt

```go
package main

import (
    "github.com/golang-jwt/jwt/v5"
    "github.com/lestrrat-go/jwx/v2/jwk"
)

func main() {
    // Fetch JWKS
    set, err := jwk.Fetch(context.Background(), 
        "http://idp-caller/.well-known/jwks.json")
    if err != nil {
        log.Fatal(err)
    }

    // Parse and verify token
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        kid := token.Header["kid"].(string)
        key, found := set.LookupKeyID(kid)
        if !found {
            return nil, fmt.Errorf("key not found")
        }
        
        var pubKey interface{}
        if err := key.Raw(&pubKey); err != nil {
            return nil, err
        }
        return pubKey, nil
    })
}
```

### 5. Java with nimbus-jose-jwt

```java
import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jwt.SignedJWT;
import com.nimbusds.jose.JWSVerifier;
import com.nimbusds.jose.crypto.RSASSAVerifier;
import java.net.URL;

public class JWTVerifier {
    public static void main(String[] args) throws Exception {
        // Load JWKS
        JWKSet jwkSet = JWKSet.load(
            new URL("http://idp-caller/.well-known/jwks.json")
        );
        
        // Parse JWT
        SignedJWT signedJWT = SignedJWT.parse(tokenString);
        
        // Get key by kid
        String kid = signedJWT.getHeader().getKeyID();
        RSAKey rsaKey = (RSAKey) jwkSet.getKeyByKeyId(kid);
        
        // Verify signature
        JWSVerifier verifier = new RSASSAVerifier(rsaKey);
        if (signedJWT.verify(verifier)) {
            System.out.println("Token verified!");
        }
    }
}
```

## How It Works

1. **Token arrives** with a JWT containing:
   ```
   Header: { "kid": "qjPcUaB_zp5mw_GBBR2Cy", "alg": "RS256" }
   ```

2. **JWT library fetches JWKS** from merged endpoint

3. **Library finds matching key** by comparing `kid` from token header with keys in JWKS

4. **Verification happens** using the matched public key

5. **No need to know** which IDP issued the token - the `kid` is unique across all IDPs

## Benefits

### ✅ Simplified Configuration
- Single endpoint for all IDPs
- No need to configure multiple JWKS URLs
- Automatic IDP detection via `kid`

### ✅ Zero Downtime IDP Changes
- Add/remove IDPs without changing application config
- Keys automatically merged in real-time
- No application restart needed

### ✅ Standard Compliance
- Follows OIDC/OAuth 2.0 standards
- Compatible with all JOSE JWT libraries
- Standard `.well-known/jwks.json` path available

### ✅ Performance
- Single HTTP request for all keys
- Client-side caching (Cache-Control header: 15 minutes)
- Reduced network latency

## Comparison: Merged vs Separate Endpoints

### Merged Endpoint (Recommended)
```
GET /.well-known/jwks.json
→ Returns ALL keys from ALL IDPs
→ Works with tokens from ANY configured IDP
→ Automatic key selection via kid
```

**Use when:** You accept tokens from multiple IDPs

### Separate Endpoints
```
GET /jwks/auth0 → Only Auth0 keys
GET /jwks/okta  → Only Okta keys
```

**Use when:** You want to restrict to a specific IDP

## Configuration Example

### config.yaml
```yaml
idps:
  - name: "auth0"
    url: "https://tenant.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600
    
  - name: "keycloak"
    url: "https://keycloak.example.com/realms/master/protocol/openid-connect/certs"
    refresh_interval: 3600
    
  - name: "okta"
    url: "https://domain.okta.com/oauth2/default/v1/keys"
    refresh_interval: 3600
```

All keys from these three IDPs will be merged and available at:
- `/.well-known/jwks.json`
- `/jwks.json`
- `/jwks/all`

## Caching

The merged endpoint includes cache headers:
```
Cache-Control: public, max-age=900
```

This allows clients to cache the response for 15 minutes, reducing load on the service.

## Monitoring

Check which keys are currently in the merged set:

```bash
# View merged keys
curl http://idp-caller/.well-known/jwks.json | jq '.keys[] | {kid, alg, use}'

# Count total keys
curl http://idp-caller/.well-known/jwks.json | jq '.keys | length'

# Check status of each IDP
curl http://idp-caller/status | jq
```

## Troubleshooting

### Issue: Token verification fails

**Check:**
1. Is the key present in merged JWKS?
   ```bash
   curl http://idp-caller/.well-known/jwks.json | jq '.keys[] | select(.kid=="YOUR_KID")'
   ```

2. Has the IDP been updated recently?
   ```bash
   curl http://idp-caller/status/your-idp | jq '.last_updated'
   ```

3. Check for errors:
   ```bash
   curl http://idp-caller/status | jq '.[] | select(.last_error != "")'
   ```

### Issue: Old keys still present

The service caches keys until the next refresh. Check `refresh_interval` in config.yaml.

### Issue: Missing keys from an IDP

Check if the IDP is reachable:
```bash
kubectl logs deployment/idp-caller | grep "your-idp"
```

## Best Practices

1. **Use caching** in your JWT library (15-30 minutes)
2. **Set appropriate refresh intervals** (30-60 minutes) in config
3. **Monitor the `/status` endpoint** for update failures
4. **Test with tokens from each IDP** after deployment
5. **Use `.well-known/jwks.json`** for standard compliance

## Summary

The merged JWKS endpoint is the **recommended approach** when working with multiple IDPs. It provides:
- ✅ Standard JOSE JWT compatibility
- ✅ Automatic IDP detection
- ✅ Simplified configuration
- ✅ Better performance
- ✅ Easier maintenance

Use it as your primary JWKS endpoint for JWT validation across all your services!

