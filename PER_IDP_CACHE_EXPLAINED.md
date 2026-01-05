# How Cache Duration Works: Per-IDP vs Merged Endpoint

## Your Question

**"When you say for all IDP uses the lower, is it still correct? I was understanding that this will be different for IDP"**

## ✅ You Are CORRECT!

Each IDP has **its own cache_duration** that can be **different** from other IDPs!

---

## How It Actually Works

### Step 1: Each IDP Determines Its Own cache_duration

```
┌─────────────────────────────────────────────────────┐
│ Auth0                                               │
│ Config: cache_duration = 900 (15 min)              │
│ IDP says: max-age=86400 (24 hours)                 │
│ Uses: min(900, 86400) = 900 ✅                     │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Okta                                                │
│ Config: cache_duration = 1800 (30 min)             │
│ IDP says: max-age=3600 (1 hour)                    │
│ Uses: min(1800, 3600) = 1800 ✅                    │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Keycloak                                            │
│ Config: cache_duration = 900 (15 min)              │
│ IDP says: max-age=300 (5 minutes!)                 │
│ Uses: min(900, 300) = 300 ✅                       │
└─────────────────────────────────────────────────────┘
```

**Result:** Each IDP has **different** cache_duration!
- Auth0: 900 seconds
- Okta: 1800 seconds
- Keycloak: 300 seconds

---

### Step 2: Individual IDP Endpoints Use Their Own Duration

```
GET /jwks/auth0
Response: Cache-Control: max-age=900

GET /jwks/okta  
Response: Cache-Control: max-age=1800

GET /jwks/keycloak
Response: Cache-Control: max-age=300
```

**Each endpoint is different!** ✅

---

### Step 3: Merged Endpoint Uses MINIMUM of All

```
GET /.well-known/jwks.json

Calculation:
  min(auth0: 900, okta: 1800, keycloak: 300) = 300

Response: Cache-Control: max-age=300
```

**Why the minimum?**
- Merged endpoint contains keys from **all** IDPs
- Keycloak rotates every 5 minutes (300s)
- If clients cache for 15+ minutes, they'll have stale Keycloak keys
- Must use shortest duration to keep **all** keys fresh

---

## Complete Flow Diagram

```
Configuration:
┌──────────────────────────────────────────────────────┐
│ config.yaml:                                         │
│   auth0:     cache_duration: 900                     │
│   okta:      cache_duration: 1800                    │
│   keycloak:  cache_duration: 900                     │
└──────────────────────────────────────────────────────┘
                        ↓
IDP Responses:
┌──────────────────────────────────────────────────────┐
│ Auth0:     Cache-Control: max-age=86400              │
│ Okta:      Cache-Control: max-age=3600               │
│ Keycloak:  Cache-Control: max-age=300                │
└──────────────────────────────────────────────────────┘
                        ↓
System Calculates (per IDP):
┌──────────────────────────────────────────────────────┐
│ Auth0:     min(900, 86400) = 900                     │
│ Okta:      min(1800, 3600) = 1800                    │
│ Keycloak:  min(900, 300) = 300                       │
└──────────────────────────────────────────────────────┘
                        ↓
API Responses:
┌──────────────────────────────────────────────────────┐
│ /jwks/auth0    → max-age=900                         │
│ /jwks/okta     → max-age=1800                        │
│ /jwks/keycloak → max-age=300                         │
│                                                       │
│ /.well-known/jwks.json → max-age=300 (minimum!)     │
└──────────────────────────────────────────────────────┘
```

---

## Monitoring: See Different Durations Per IDP

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  cache_duration,
  idp_suggested_cache
}'
```

**Output:**
```json
{
  "name": "auth0",
  "cache_duration": 900,
  "idp_suggested_cache": 86400
}
{
  "name": "okta",
  "cache_duration": 1800,
  "idp_suggested_cache": 3600
}
{
  "name": "keycloak",
  "cache_duration": 300,
  "idp_suggested_cache": 300
}
```

**Notice:** Each IDP has **different** `cache_duration`! ✅

---

## Why This Matters

### Example: Client Using Merged Endpoint

**Scenario:**
```
Client fetches: /.well-known/jwks.json
Response: Cache-Control: max-age=300
Client caches for 5 minutes
```

**Timeline:**
```
0:00  Client fetches merged JWKS
      Gets keys from: Auth0, Okta, Keycloak
      Caches for 5 minutes

0:05  Keycloak rotates keys (new key!)
      Client cache expires
      Client fetches again → gets new Keycloak key ✅

0:10  Keycloak rotates again
      Client fetches again → gets new key ✅
```

**Result:** Client always has fresh keys from **all** IDPs

### What Would Happen Without Minimum?

**Bad scenario:**
```
If merged used max(900, 1800, 300) = 1800:

0:00  Client caches for 30 minutes
0:05  Keycloak rotates
0:10  Keycloak rotates
0:15  Keycloak rotates
0:20  Keycloak rotates
...
0:30  Client cache expires → gets keys
      But Keycloak rotated 6 times! ❌
      Client had stale Keycloak keys!
```

---

## Summary

✅ **Each IDP has its OWN cache_duration**
- Determined by: `min(config.cache_duration, idp.max_age)`
- Can be **different** for each IDP
- Based on that IDP's rotation schedule

✅ **Individual endpoints use their own duration**
- `/jwks/auth0` → uses Auth0's cache_duration
- `/jwks/okta` → uses Okta's cache_duration
- `/jwks/keycloak` → uses Keycloak's cache_duration

✅ **Merged endpoint uses MINIMUM**
- `/.well-known/jwks.json` → uses shortest duration
- Ensures clients get fresh keys from **all** IDPs
- Protects against stale keys from fast-rotating IDPs

---

## Configuration Impact

```yaml
idps:
  - name: "auth0"
    cache_duration: 900     # Individual endpoint: 900 (or less if IDP says so)
    
  - name: "keycloak"
    cache_duration: 900     # Individual endpoint: might be 300 if IDP rotates fast
```

**Result:**
- Individual endpoints: Each uses its own calculated duration
- Merged endpoint: Uses minimum of all calculated durations

**You were right to question this!** Each IDP **does** have different cache durations based on its behavior. The merged endpoint then uses the minimum to be safe for all IDPs. 🎯

