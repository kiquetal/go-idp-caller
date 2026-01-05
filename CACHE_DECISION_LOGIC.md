# Cache Duration Decision Logic

## Understanding the Problem

When fetching JWKS from IDPs, there's a potential conflict between two cache durations:

1. **Your Config** (`cache_duration: 900`) - What YOU want clients to cache
2. **IDP's Suggestion** (`Cache-Control: max-age=86400`) - What the IDP recommends

### The Problem Scenarios

#### Scenario 1: IDP Cache is LOWER than Config
```yaml
# Your config
idps:
  - name: "auth0"
    cache_duration: 900      # You want 15 min cache

# IDP returns
Cache-Control: max-age=300   # IDP suggests 5 min cache
```

**What this means:** IDP rotates keys more frequently (every 5 minutes). They know their rotation schedule better than you do.

**Solution:** Use IDP's shorter duration (300 seconds)

**Why:** If you cache for 15 minutes but IDP rotates every 5 minutes, you'll have stale keys and token validation will fail!

#### Scenario 2: IDP Cache is HIGHER than Config
```yaml
# Your config
idps:
  - name: "auth0"
    cache_duration: 900      # You want 15 min cache

# IDP returns
Cache-Control: max-age=86400 # IDP suggests 24 hours
```

**What this means:** IDP has stable keys that don't rotate frequently.

**Solution:** Use YOUR config duration (900 seconds)

**Why:** You may want fresher data for other reasons (security policy, compliance, monitoring). Your config is the minimum freshness requirement.

---

## Current Implementation Logic

```go
func determineCacheDuration(idpMaxAge int, configDuration int) int {
    // No cache header from IDP? Use config
    if idpMaxAge <= 0 {
        return configDuration
    }
    
    // IDP suggests SHORTER cache?
    if idpMaxAge < configDuration {
        // IDP knows best - they rotate keys more often
        return idpMaxAge  // Use IDP's shorter duration
    }
    
    // IDP suggests LONGER cache?
    // You want fresher data than IDP suggests
    return configDuration  // Use your config
}
```

### Decision Table

| IDP max-age | Config cache_duration | Final Duration Used | Reason |
|-------------|----------------------|---------------------|---------|
| 0 (none) | 900 | **900** | IDP didn't specify, use config |
| 300 | 900 | **300** | IDP rotates fast, respect that |
| 900 | 900 | **900** | Same value, use either |
| 86400 | 900 | **900** | Your config is more conservative |
| 3600 | 300 | **300** | Your config wants even fresher data |

**Rule:** Always use the **MINIMUM** to ensure maximum freshness!

---

## Real-World Examples

### Example 1: Auth0 with Stable Keys

```bash
# What Auth0 returns
$ curl -I https://tenant.auth0.com/.well-known/jwks.json
Cache-Control: max-age=86400  # 24 hours
```

```yaml
# Your config
idps:
  - name: "auth0"
    refresh_interval: 3600  # Fetch hourly
    cache_duration: 900     # Clients cache 15 min
```

**Result:**
- IDP suggests: 86400 seconds (24 hours)
- Config says: 900 seconds (15 min)
- **Used: 900 seconds** ✅
- Clients get fresh data every 15 minutes
- Service fetches every hour

**Log Output:**
```json
{
  "level": "INFO",
  "msg": "Using config cache duration (more conservative than IDP)",
  "idp": "auth0",
  "idp_max_age": 86400,
  "config_duration": 900,
  "using": 900,
  "reason": "config requires fresher data than IDP suggests"
}
```

### Example 2: Custom IDP with Fast Rotation

```bash
# What custom IDP returns
$ curl -I https://custom-idp.com/.well-known/jwks.json
Cache-Control: max-age=300  # 5 minutes!
```

```yaml
# Your config
idps:
  - name: "custom"
    refresh_interval: 3600
    cache_duration: 900
```

**Result:**
- IDP suggests: 300 seconds (5 min)
- Config says: 900 seconds (15 min)
- **Used: 300 seconds** ✅
- Clients get fresh data every 5 minutes (IDP knows they rotate fast!)
- Service fetches every hour

**Log Output:**
```json
{
  "level": "INFO",
  "msg": "Using IDP's shorter cache duration (IDP rotates keys faster)",
  "idp": "custom",
  "idp_max_age": 300,
  "config_duration": 900,
  "using": 300,
  "reason": "IDP rotates keys more frequently"
}
```

### Example 3: Okta with No Cache Header

```bash
# What Okta returns
$ curl -I https://domain.okta.com/oauth2/default/v1/keys
# (no Cache-Control header)
```

```yaml
# Your config
idps:
  - name: "okta"
    refresh_interval: 3600
    cache_duration: 900
```

**Result:**
- IDP suggests: 0 (nothing)
- Config says: 900 seconds
- **Used: 900 seconds** ✅

**Log Output:**
```json
{
  "level": "DEBUG",
  "msg": "No cache control from IDP, using config",
  "idp": "okta",
  "cache_duration": 900
}
```

---

## How to Configure Properly

### Strategy 1: Let IDPs Guide You (Recommended)

```yaml
idps:
  - name: "auth0"
    refresh_interval: 3600
    cache_duration: 3600   # Set high, let IDP override if needed
```

**Benefit:** If IDP rotates keys fast, they'll tell you with a lower max-age.

### Strategy 2: Enforce Strict Freshness

```yaml
idps:
  - name: "auth0"
    refresh_interval: 1800   # 30 min
    cache_duration: 600      # 10 min - very fresh
```

**Benefit:** You always get fresh data, regardless of IDP suggestions.

### Strategy 3: Balanced Approach (Best for Most Cases)

```yaml
idps:
  - name: "auth0"
    refresh_interval: 3600   # 1 hour
    cache_duration: 900      # 15 min
```

**Benefit:** 
- If IDP rotates fast (< 15 min), you respect that
- If IDP is stable (> 15 min), you still get fresh data every 15 min

---

## Monitoring Cache Decisions

### Check Status Endpoint

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  cache_duration,
  idp_suggested_cache,
  refresh_interval
}'
```

**Output:**
```json
{
  "name": "auth0",
  "cache_duration": 900,
  "idp_suggested_cache": 86400,
  "refresh_interval": 3600
}
```

**What this tells you:**
- `cache_duration: 900` - What we're actually using (15 min)
- `idp_suggested_cache: 86400` - What IDP suggested (24 hours)
- `refresh_interval: 3600` - How often we fetch from IDP (1 hour)

### Check Logs for Decisions

```bash
kubectl logs deployment/idp-caller | grep "cache duration"
```

You'll see log entries explaining WHY each decision was made.

---

## Common Questions

### Q: Why not always use IDP's suggestion?

**A:** Because:
1. Some IDPs don't provide cache headers
2. Your security policy may require fresher data
3. You might be testing and want rapid updates

### Q: Why not always use my config?

**A:** Because:
1. IDP knows their rotation schedule better
2. If IDP rotates every 5 minutes but you cache for 15, tokens will fail
3. Respecting IDP's cache prevents unnecessary load

### Q: What if IDP cache is much lower?

**Example:**
```yaml
cache_duration: 3600  # You want 1 hour
# IDP says: max-age=60  # 1 minute!
```

**Current behavior:** Uses 60 seconds (IDP's value)

**Impact:**
- Clients fetch every 1 minute (high load)
- But ensures valid tokens (IDP rotates fast!)

**Solution if you want to override:**
```yaml
cache_duration: 300  # Compromise at 5 min
# Even if IDP says 60, we use 300 as our floor
```

Then modify the code to use `max(idpMaxAge, minFloor)`:
```go
// Set minimum cache floor
minFloor := 300  // Never cache less than 5 min

if idpMaxAge > 0 && idpMaxAge < minFloor {
    return minFloor  // Use our minimum
}
```

### Q: Should I set cache_duration high or low?

**Recommended:** Set it to your **minimum acceptable freshness**

```yaml
# For high-security, fast-changing environments
cache_duration: 300   # 5 minutes

# For normal production
cache_duration: 900   # 15 minutes (standard)

# For stable, low-traffic
cache_duration: 1800  # 30 minutes
```

The system will use IDP's value if it's lower (IDP knows best about rotation).

---

## Configuration Examples

### High Security Environment
```yaml
idps:
  - name: "auth0"
    refresh_interval: 1800   # Fetch every 30 min
    cache_duration: 600      # Cache max 10 min
```

**Behavior:** Even if IDP says 24 hours, clients only cache 10 min max.

### Standard Production
```yaml
idps:
  - name: "auth0"
    refresh_interval: 3600   # Fetch every hour
    cache_duration: 900      # Cache max 15 min
```

**Behavior:** Balanced between performance and freshness.

### Low-Traffic / Stable Keys
```yaml
idps:
  - name: "auth0"
    refresh_interval: 7200   # Fetch every 2 hours
    cache_duration: 1800     # Cache max 30 min
```

**Behavior:** Less frequent updates, good for stable environments.

---

## Summary

✅ **The Logic:**
```
if idpMaxAge < configDuration:
    use idpMaxAge    # IDP rotates fast, respect it
else:
    use configDuration  # Your config is more conservative
```

✅ **The Goal:**
- Always ensure clients have fresh enough keys
- Respect IDP's rotation schedule
- Allow you to enforce minimum freshness

✅ **The Result:**
- No stale keys causing auth failures
- Optimal cache based on actual IDP behavior
- Full visibility via logs and status endpoint

🎯 **Best Practice:**
Set `cache_duration` to your **minimum acceptable freshness**, and let the system use IDP's value if it's lower!

