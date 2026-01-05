# SOLUTION: How the System Handles IDP Key Rotation

## Your Question

**"How is it possible to obtain the new key from IDP if it rotates every 10 minutes, but I cache 15 minutes?"**

## The Answer: Automatic Cache Adjustment

The system **automatically uses the shorter duration** to prevent this exact problem!

---

## Step-by-Step: What Actually Happens

### Your Configuration
```yaml
idps:
  - name: "keycloak"
    refresh_interval: 3600  # Fetch every 1 hour
    cache_duration: 900     # You think 15 min is OK
```

### What IDP Says
```bash
$ curl -I https://keycloak.example.com/.well-known/jwks.json
HTTP/1.1 200 OK
Cache-Control: max-age=600  # IDP rotates every 10 minutes!
```

### System Logic (Automatic)

```
Step 1: Service fetches from IDP
  ↓
Step 2: Service sees Cache-Control: max-age=600
  ↓
Step 3: Service compares:
        config.cache_duration = 900
        idp.max_age = 600
  ↓
Step 4: System uses MINIMUM:
        actual_cache = min(900, 600) = 600 ✅
  ↓
Step 5: Service returns to clients:
        Cache-Control: max-age=600
```

**Result:** Clients cache for **10 minutes**, not 15!

---

## Complete Timeline: IDP Rotates Every 10 Minutes

```yaml
# Configuration
refresh_interval: 3600  # Service fetches hourly
cache_duration: 900     # You configured 15 min
# But IDP says: max-age=600 (10 min)
# System uses: 600 (10 min) ✅
```

### Timeline

```
Time    IDP Keys        Service             Client              
─────────────────────────────────────────────────────────────────
0:00    [Key A, B]      FETCH ●             -
                        Gets: max-age=600
                        Uses: 600 (not 900!)
                        
0:01    [Key A, B]      -                   REQUEST ●
                                            Gets: Cache-Control: 600
                                            Caches for 10 min (not 15!)
                                            
0:10    [Key B, C] ←    -                   (cache still valid)
        NEW Key C!                          Using old keys A, B
        
0:11    [Key B, C]      -                   CACHE EXPIRED ✅
                                            REQUEST ●
                                            Gets NEW keys B, C!
                                            Caches for 10 min
                                            
0:20    [Key C, D] ←    -                   (cache still valid)
        NEW Key D!                          Using keys B, C
        
0:21    [Key C, D]      -                   CACHE EXPIRED ✅
                                            REQUEST ●
                                            Gets NEW keys C, D!
                                            Caches for 10 min
                                            
0:30    [Key D, E] ←    -                   (cache still valid)
        
0:31    [Key D, E]      -                   CACHE EXPIRED ✅
                                            REQUEST ●
                                            Gets keys D, E!
```

### Key Points

✅ **Client cache expires every 10 minutes** (not 15!)
✅ **Client gets new keys before IDP rotates**
✅ **No stale key problem**

But there's still a potential issue...

---

## The Remaining Problem: Service refresh_interval

### Notice the Issue

```
Time    IDP Keys        Service             Client
────────────────────────────────────────────────────
0:00    [Key A, B]      FETCH ● (fresh!)    Gets A, B
0:10    [Key B, C] ←    (memory has A, B)   Gets A, B (stale!)
0:20    [Key C, D] ←    (memory has A, B)   Gets A, B (VERY stale!)
0:30    [Key D, E] ←    (memory has A, B)   Gets A, B (VERY VERY stale!)
1:00    [Key E, F]      FETCH ● (updates!)  Gets E, F (fresh again)
```

**Problem:** Service only fetches every hour, but IDP rotates every 10 minutes!

**Result:** Between T=0:10 and T=1:00, clients get **stale keys** even though they fetch every 10 minutes.

---

## The REAL Solution: Match refresh_interval to IDP Rotation

### Current Configuration (WRONG)
```yaml
refresh_interval: 3600  # Service fetches hourly
cache_duration: 900     # Clients cache 15 min
# IDP rotates: 600 (10 min)
```

**Problem:** Service memory gets stale after 10 minutes.

### Correct Configuration
```yaml
refresh_interval: 600   # Service fetches every 10 min ✅
cache_duration: 900     # Clients cache up to 15 min
# IDP rotates: 600 (10 min)
# System uses: 600 for client cache ✅
```

**Now:**
- Service fetches every 10 minutes (matches IDP rotation)
- Clients cache for 10 minutes (system uses min(900, 600))
- **Service always has fresh keys**
- **Clients always get fresh keys**

### New Timeline (CORRECT)

```
Time    IDP Keys        Service             Client
─────────────────────────────────────────────────────
0:00    [Key A, B]      FETCH ● (A, B)      Gets A, B ✅
0:10    [Key B, C]      FETCH ● (B, C) ✅   Gets B, C ✅
0:20    [Key C, D]      FETCH ● (C, D) ✅   Gets C, D ✅
0:30    [Key D, E]      FETCH ● (D, E) ✅   Gets D, E ✅
```

**Perfect!** Service and clients always have fresh keys.

---

## Configuration Rules

### Rule 1: Match or Beat IDP Rotation

```yaml
# If IDP rotates every X seconds
refresh_interval: X  # Or less!
```

### Rule 2: Let System Handle Client Cache

```yaml
# Set cache_duration as your maximum staleness
cache_duration: 900

# System automatically uses IDP's max-age if shorter
# You don't need to manually match it!
```

### Example Configurations

#### IDP Rotates Every 10 Minutes
```yaml
idps:
  - name: "fast-keycloak"
    refresh_interval: 600   # Match IDP rotation ✅
    cache_duration: 900     # System will use 600 from IDP
```

#### IDP Rotates Every Hour
```yaml
idps:
  - name: "stable-auth0"
    refresh_interval: 3600  # Match IDP rotation ✅
    cache_duration: 900     # System keeps using 900
```

#### IDP Rotates Every 5 Minutes (Very Fast!)
```yaml
idps:
  - name: "dev-idp"
    refresh_interval: 300   # Match IDP rotation ✅
    cache_duration: 900     # System will use 300 from IDP
```

---

## How to Find IDP's Rotation Schedule

### Method 1: Check Cache-Control Header

```bash
curl -I https://your-idp.com/.well-known/jwks.json | grep -i cache-control
```

**Output:**
```
Cache-Control: max-age=600
```

**IDP rotates every:** 600 seconds (10 minutes)

### Method 2: Monitor Your Service Logs

After deployment, check logs:

```bash
kubectl logs deployment/idp-caller | grep "idp_suggested_cache"
```

**Output:**
```json
{
  "idp": "keycloak",
  "idp_max_age": 600,
  "config_duration": 900,
  "using": 600
}
```

**IDP suggests:** 600 seconds

### Method 3: Check Status Endpoint

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  refresh_interval,
  cache_duration,
  idp_suggested_cache
}'
```

**Output:**
```json
{
  "name": "keycloak",
  "refresh_interval": 3600,
  "cache_duration": 900,
  "idp_suggested_cache": 600  ← IDP rotates every 10 min!
}
```

**Action needed:** Change `refresh_interval` to 600 or less!

---

## Recommended Configuration Strategy

### Step 1: Deploy with Conservative Defaults

```yaml
idps:
  - name: "new-idp"
    refresh_interval: 3600  # Start conservative
    cache_duration: 900
```

### Step 2: Monitor IDP's Suggestion

```bash
# After deployment, check what IDP suggests
curl http://localhost:8080/status/new-idp | jq '.idp_suggested_cache'
```

### Step 3: Adjust refresh_interval

```yaml
# If IDP suggests 600 (10 min)
idps:
  - name: "new-idp"
    refresh_interval: 600   # ✅ Match IDP rotation
    cache_duration: 900     # Keep as max staleness
```

### Step 4: Verify

```bash
# Check max staleness
curl http://localhost:8080/status/new-idp | jq '{
  refresh_interval,
  actual_cache: (
    if .idp_suggested_cache > 0 and .idp_suggested_cache < .cache_duration
    then .idp_suggested_cache
    else .cache_duration
    end
  ),
  max_staleness_seconds: (
    .refresh_interval + (
      if .idp_suggested_cache > 0 and .idp_suggested_cache < .cache_duration
      then .idp_suggested_cache
      else .cache_duration
      end
    )
  )
}'
```

---

## Quick Reference: Matching Configuration

| IDP Rotation | Your refresh_interval | Your cache_duration | Actual Client Cache | Max Staleness |
|--------------|----------------------|---------------------|---------------------|---------------|
| 600 (10m) | 600 ✅ | 900 | 600 | 1200s (20m) |
| 600 (10m) | 3600 ❌ | 900 | 600 | 4200s (70m) |
| 3600 (1h) | 3600 ✅ | 900 | 900 | 4500s (75m) |
| 300 (5m) | 300 ✅ | 900 | 300 | 600s (10m) |
| 300 (5m) | 3600 ❌ | 900 | 300 | 3900s (65m) |

**✅ = Optimal**  
**❌ = Service memory gets stale**

---

## Code That Handles This

### In internal/jwks/updater.go

```go
// Fetches from IDP
func (u *Updater) fetch() (*JWKS, int, error) {
    resp, err := u.client.Do(req)
    
    // Parse Cache-Control header
    cacheControl := resp.Header.Get("Cache-Control")
    idpMaxAge := parseCacheControl(cacheControl)  // Extract max-age
    
    return &jwks, idpMaxAge, nil
}

// Determines cache duration
func (u *Updater) determineCacheDuration(idpMaxAge int) int {
    configDuration := u.config.GetCacheDuration()
    
    if idpMaxAge < configDuration {
        // IDP rotates faster - use IDP's value
        return idpMaxAge  ✅
    }
    
    // Use config (more conservative)
    return configDuration
}
```

---

## Summary

### ✅ What the System Does Automatically

1. **Reads IDP's `Cache-Control: max-age=X`**
2. **Uses minimum of (config, IDP's max-age)** for client caching
3. **Logs the decision** so you can see what's happening

### ❌ What You Must Do Manually

1. **Set `refresh_interval` to match IDP's rotation**
2. **Monitor `idp_suggested_cache` in status endpoint**
3. **Adjust configuration if needed**

### 🎯 The Answer to Your Question

**"How is it possible to obtain new keys if IDP rotates every 10 min but I cache 15 min?"**

**Answer:** 
1. System **automatically** uses 10 min for client cache (not 15)
2. But you **must** set `refresh_interval: 600` so SERVICE fetches every 10 min
3. Otherwise service memory gets stale even though clients fetch frequently

### 📋 Action Items

```bash
# 1. Check what IDP suggests
curl http://localhost:8080/status | jq '.[] | {
  name,
  idp_suggested_cache
}'

# 2. Update config.yaml
# Set refresh_interval = idp_suggested_cache (or less)

# 3. Redeploy and verify
kubectl apply -f k8s/
kubectl logs -f deployment/idp-caller | grep "Successfully updated"
```

**Your system will then always have fresh keys!** ✅

