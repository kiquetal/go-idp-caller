# Understanding cache_duration vs refresh_interval

## The Two Parameters Explained

### `refresh_interval` - Server-Side Fetching
**Who uses it:** Your service (goroutines)  
**What it controls:** How often your service fetches JWKS from the IDP  
**Where it happens:** Background goroutine for each IDP

```yaml
refresh_interval: 3600  # Service fetches from IDP every 3600 seconds (1 hour)
```

### `cache_duration` - Client-Side Caching
**Who uses it:** Your clients (KrakenD, apps, browsers)  
**What it controls:** How long clients should cache the JWKS they get from your service  
**Where it happens:** HTTP `Cache-Control: max-age=` header in responses

```yaml
cache_duration: 900  # Clients cache for 900 seconds (15 minutes)
```

---

## They Are Completely Independent!

```
┌─────────────────────────────────────────────────────────┐
│                        IDP (Auth0)                      │
│                  Has the actual keys                    │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ refresh_interval: 3600s (1 hour)
                     │ ← Service fetches every hour
                     ↓
┌─────────────────────────────────────────────────────────┐
│                  Your IDP Caller Service                │
│              Stores keys in memory cache                │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ cache_duration: 900s (15 min)
                     │ ← Clients cache for 15 minutes
                     ↓
┌─────────────────────────────────────────────────────────┐
│                    Clients (KrakenD)                    │
│              Cache JWKS they receive                    │
└─────────────────────────────────────────────────────────┘
```

---

## Timeline Example: Understanding the Relationship

### Configuration
```yaml
idps:
  - name: "auth0"
    refresh_interval: 3600    # Fetch every 1 hour
    cache_duration: 900       # Clients cache 15 min
```

### What Happens Over Time

```
Time    Service Action              Client Action              Data Age
─────────────────────────────────────────────────────────────────────────
0:00    [Fetch from Auth0] ●        -                         Fresh (0 min)
        Stores in memory
        
0:01    -                           [Requests JWKS] ●         Fresh (1 min)
                                    Gets data + Cache-Control: 900
                                    Caches for 15 min
                                    
0:05    -                           Uses cached data          5 min old
                                    (no network call)
                                    
0:10    -                           Uses cached data          10 min old
                                    (no network call)
                                    
0:15    -                           Uses cached data          15 min old
                                    (no network call)
                                    
0:16    -                           [Cache expired!]          16 min old
                                    [Requests JWKS] ●
                                    Gets from service memory
                                    Caches again for 15 min
                                    
0:31    -                           [Cache expired!]          31 min old
                                    [Requests JWKS] ●
                                    Gets from service memory
                                    Caches again for 15 min
                                    
0:46    -                           [Cache expired!]          46 min old
                                    [Requests JWKS] ●
                                    Gets from service memory
                                    Caches again for 15 min
                                    
1:00    [Fetch from Auth0] ●        -                         Fresh (0 min)
        Updates memory cache
        
1:01    -                           [Cache expired!]          1 min old
                                    [Requests JWKS] ●
                                    Gets UPDATED data!
                                    Caches for 15 min
```

### Key Observations

**Service fetches:** 1 time (at T=0:00 and T=1:00)  
**Client fetches:** 4 times (at T=0:01, T=0:16, T=0:31, T=0:46, T=1:01)

**Client gets updates:**
- Every 15 minutes from the service
- But service only hits Auth0 every hour
- Service responds instantly from memory

---

## Different Relationships

### Relationship 1: cache_duration < refresh_interval (Recommended)

```yaml
refresh_interval: 3600  # 1 hour
cache_duration: 900     # 15 min
```

**Result:** Clients get "fresh" data every 15 min, but it might be up to 15 min stale from IDP's perspective.

**Timeline:**
```
0:00  Service fetches from IDP → Memory has fresh keys
0:15  Client cache expires → Gets keys from service memory (15 min old)
0:30  Client cache expires → Gets keys from service memory (30 min old)
0:45  Client cache expires → Gets keys from service memory (45 min old)
1:00  Service fetches from IDP → Memory updated with fresh keys
1:00  Client cache expires → Gets fresh keys
```

**Maximum staleness:** 1 hour (refresh_interval) + 15 min (cache_duration) = 1 hour 15 min

**Pros:**
- Clients get relatively fresh data
- Service only hits IDP once per hour (low load on IDP)
- Balanced approach

**Good for:** Most production scenarios

### Relationship 2: cache_duration = refresh_interval

```yaml
refresh_interval: 3600  # 1 hour
cache_duration: 3600    # 1 hour
```

**Result:** Clients cache as long as service caches

**Timeline:**
```
0:00  Service fetches from IDP → Memory fresh
      Client gets data → Caches for 1 hour
1:00  Service fetches from IDP → Memory updated
      Client cache expires → Gets fresh data
```

**Maximum staleness:** 2 hours (service's data is 1h old + client cached for 1h)

**Pros:**
- Minimal requests to service
- Simple to understand

**Cons:**
- Clients might have very stale data
- If IDP rotates keys, clients won't know until 2 hours later

**Good for:** Very stable IDPs with infrequent rotation

### Relationship 3: cache_duration > refresh_interval (Anti-pattern)

```yaml
refresh_interval: 1800  # 30 min
cache_duration: 3600    # 1 hour
```

**Result:** Service fetches more often than clients request! Wasteful!

**Timeline:**
```
0:00  Service fetches → Client caches for 1 hour
0:30  Service fetches again (but client still has cached data!)
1:00  Service fetches again → Client cache expires, gets data
```

**Maximum staleness:** 1 hour 30 min

**Cons:**
- Service does unnecessary fetches
- Wastes resources
- No benefit to clients

**Avoid this!**

### Relationship 4: cache_duration = 0 (No Client Caching)

```yaml
refresh_interval: 3600  # 1 hour
cache_duration: 0       # No caching!
```

**Result:** Clients fetch every single time they need to validate a token

**Pros:**
- Clients always get latest data service has
- Good for debugging

**Cons:**
- Very high load on your service
- High latency for token validation
- Not practical for production

**Good for:** Testing only

---

## Common Configuration Patterns

### Pattern 1: Standard Production (Recommended)
```yaml
refresh_interval: 3600  # 1 hour
cache_duration: 900     # 15 minutes
```

**Flow:**
- Service: 24 fetches/day from IDP
- Clients: Up to 96 fetches/day per client from service
- Max staleness: ~1h 15min
- Good balance of freshness and performance

### Pattern 2: High Security / Fast Rotation
```yaml
refresh_interval: 1800  # 30 minutes
cache_duration: 300     # 5 minutes
```

**Flow:**
- Service: 48 fetches/day from IDP
- Clients: Up to 288 fetches/day per client from service
- Max staleness: ~35 min
- More load, but fresher data

### Pattern 3: Stable / Low Traffic
```yaml
refresh_interval: 7200  # 2 hours
cache_duration: 1800    # 30 minutes
```

**Flow:**
- Service: 12 fetches/day from IDP
- Clients: Up to 48 fetches/day per client from service
- Max staleness: ~2h 30min
- Reduced load, acceptable for stable keys

### Pattern 4: High Traffic / Stable Keys
```yaml
refresh_interval: 7200  # 2 hours
cache_duration: 3600    # 1 hour
```

**Flow:**
- Service: 12 fetches/day from IDP
- Clients: Up to 24 fetches/day per client from service
- Max staleness: ~3 hours
- Minimum load on both IDP and service

---

## Adding IDP's Cache-Control to the Mix

### How It Changes Things

When IDP returns `Cache-Control: max-age=X`, the system uses:
```
actual_cache_duration = min(config.cache_duration, idp.max_age)
```

### Example: IDP Suggests Shorter Cache

```yaml
# Your config
refresh_interval: 3600  # 1 hour
cache_duration: 900     # 15 min
```

```bash
# IDP returns
Cache-Control: max-age=300  # 5 min!
```

**Result:**
```
actual_cache_duration = min(900, 300) = 300 seconds
```

**New Timeline:**
```
0:00  Service fetches → Clients cache for 5 min
0:05  Client cache expires → Fetch from service
0:10  Client cache expires → Fetch from service
0:15  Client cache expires → Fetch from service
...
1:00  Service fetches fresh keys from IDP
```

**Now:**
- Service: Still 24 fetches/day from IDP
- Clients: Up to 288 fetches/day from service (every 5 min!)
- Max staleness: ~1h 5min
- **More accurate to IDP's rotation schedule**

---

## The Math: Maximum Staleness

```
Maximum Staleness = refresh_interval + actual_cache_duration
```

### Examples

**Config 1:**
```yaml
refresh_interval: 3600
cache_duration: 900
IDP suggests: 86400 (24h)
Actual cache: min(900, 86400) = 900
```
**Max staleness:** 3600 + 900 = 4500 seconds (75 minutes)

**Config 2:**
```yaml
refresh_interval: 3600
cache_duration: 900
IDP suggests: 300 (5 min)
Actual cache: min(900, 300) = 300
```
**Max staleness:** 3600 + 300 = 3900 seconds (65 minutes)

**Config 3:**
```yaml
refresh_interval: 600  # 10 min
cache_duration: 300    # 5 min
IDP suggests: 300
Actual cache: min(300, 300) = 300
```
**Max staleness:** 600 + 300 = 900 seconds (15 minutes) ✅ Very fresh!

---

## Best Practices

### 1. Keep cache_duration ≤ refresh_interval

```yaml
# ✅ Good
refresh_interval: 3600
cache_duration: 900

# ✅ Also good
refresh_interval: 3600
cache_duration: 3600

# ❌ Bad (wasteful)
refresh_interval: 1800
cache_duration: 3600  # Service fetches more than clients need
```

### 2. For Critical Systems: Use Smaller Values

```yaml
# Critical payment system
refresh_interval: 600   # 10 min
cache_duration: 300     # 5 min
```

### 3. For Stable Systems: Use Larger Values

```yaml
# Internal tool with stable keys
refresh_interval: 7200  # 2 hours
cache_duration: 1800    # 30 min
```

### 4. Let IDPs Guide You

```yaml
# Set reasonable defaults
refresh_interval: 3600
cache_duration: 900

# System automatically uses IDP's max-age if shorter
# So if IDP rotates every 5 min, clients will cache only 5 min
```

---

## Monitoring the Relationship

### Check Current Settings

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
  refresh_interval,
  cache_duration,
  idp_suggested_cache,
  actual_used: (
    if .idp_suggested_cache > 0 and .idp_suggested_cache < .cache_duration
    then .idp_suggested_cache
    else .cache_duration
    end
  )
}'
```

### Calculate Maximum Staleness

```bash
curl http://localhost:8080/status | jq '.[] | {
  name,
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

## Quick Reference Table

| refresh_interval | cache_duration | IDP max-age | Actual Cache | Max Staleness | Use Case |
|-----------------|----------------|-------------|--------------|---------------|----------|
| 3600 (1h) | 900 (15m) | - | 900 | 4500 (75m) | Standard production |
| 3600 (1h) | 900 (15m) | 300 (5m) | 300 | 3900 (65m) | IDP rotates fast |
| 1800 (30m) | 300 (5m) | - | 300 | 2100 (35m) | High security |
| 600 (10m) | 300 (5m) | - | 300 | 900 (15m) | Critical systems |
| 7200 (2h) | 1800 (30m) | - | 1800 | 9000 (150m) | Stable keys |
| 3600 (1h) | 3600 (1h) | - | 3600 | 7200 (120m) | Simple setup |

---

## Summary

### ✅ Key Takeaways

1. **refresh_interval** = How often YOUR SERVICE fetches from IDP
2. **cache_duration** = How long YOUR CLIENTS cache from your service
3. **They are independent** but work together
4. **Best practice:** cache_duration ≤ refresh_interval
5. **IDP's max-age** can override cache_duration if shorter
6. **Max staleness** = refresh_interval + actual_cache_duration

### ✅ Recommended Starting Point

```yaml
idps:
  - name: "your-idp"
    refresh_interval: 3600  # 1 hour
    cache_duration: 900     # 15 minutes
    max_keys: 10
```

This gives you:
- Reasonable load on IDP (24 fetches/day)
- Reasonable freshness (clients update every 15 min)
- Automatic adjustment if IDP rotates faster

### ✅ Rule of Thumb

```
For every 1 hour refresh_interval:
  Set cache_duration to 15-30 minutes

For every 30 min refresh_interval:
  Set cache_duration to 5-15 minutes

For every 10 min refresh_interval:
  Set cache_duration to 3-5 minutes
```

Your service now intelligently manages both server-side and client-side caching to balance freshness, performance, and IDP rotation schedules! 🚀

