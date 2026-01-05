# IDP JWKS Caller Service

A Go service that fetches and maintains JSON Web Key Sets (JWKS) from multiple Identity Providers (IDPs). Features independent goroutines per IDP, intelligent caching, and a merged JWKS endpoint for seamless multi-IDP JWT validation.

## 📚 Documentation

- **[MERGED_JWKS_GUIDE.md](MERGED_JWKS_GUIDE.md)** - Using the merged JWKS endpoint (recommended for multi-IDP)
- **[CONFIGURATION.md](CONFIGURATION.md)** - Complete configuration reference and caching strategy
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System design and how it works
- **[KRAKEND_INTEGRATION.md](KRAKEND_INTEGRATION.md)** - Integrating with KrakenD API Gateway

## 🚀 Quick Start

```bash
# 1. Configure your IDPs
cp config.example.yaml config.yaml
# Edit config.yaml with your IDP URLs

# 2. Run locally
go run main.go

# 3. Test the merged endpoint
curl http://localhost:8080/.well-known/jwks.json
```

## Features

- ✅ **Merged JWKS endpoint** - Single `/.well-known/jwks.json` for all IDPs
- ✅ **Independent goroutines** - Each IDP fetches on its own schedule
- ✅ **Intelligent caching** - Respects IDP's Cache-Control headers
- ✅ **Per-IDP configuration** - Different refresh intervals and cache settings
- ✅ **Memory protection** - Configurable key limits per IDP
- ✅ **Kubernetes ready** - Deployment manifests included
- ✅ **Production logging** - Structured JSON logs with full metadata
- ✅ **Health & status endpoints** - Monitor each IDP independently

See [ARCHITECTURE.md](ARCHITECTURE.md) for complete system design.

## API Endpoints

### Health Check
```bash
GET /health
```
Returns service health status.

### Get Merged JWKS (All IDPs Combined) - **JOSE JWT Compatible**
```bash
GET /.well-known/jwks.json  # Standard OIDC endpoint (recommended)
```
**Returns all keys from all IDPs merged into a single array.** This is the standard OIDC Discovery endpoint format expected by JOSE JWT libraries, KrakenD, and most JWT validators.

**Response format:**
```json
{
  "keys": [
    { "kid": "key1", "kty": "RSA", "alg": "RS256", "use": "sig", "n": "...", "e": "AQAB" },
    { "kid": "key2", "kty": "RSA", "alg": "RS256", "use": "sig", "n": "...", "e": "AQAB", "x5c": ["..."] },
    { "kid": "key3", "kty": "RSA", "alg": "RSA-OAEP", "use": "enc", "n": "...", "e": "AQAB" }
  ]
}
```

**Response Headers:**
- `Cache-Control: public, max-age=900` (uses minimum cache duration from all IDPs)
- `X-Total-Keys: 9` (total number of keys across all IDPs)
- `X-IDP-Count: 3` (number of configured IDPs)

### Get All JWKS (Separated by IDP)
```bash
GET /jwks
```
Returns JWKS from all configured IDPs as a map with IDP names as keys.

**Response format:**
```json
{
  "auth0": { "keys": [...] },
  "okta": { "keys": [...] }
}
```

### Get IDP-Specific JWKS
```bash
GET /jwks/{idp-name}
```
Returns JWKS for a specific IDP (e.g., `/jwks/auth0`).

**Response Headers:**
- `Cache-Control: public, max-age=900` (IDP-specific cache duration)
- `X-Key-Count: 3` (number of keys for this IDP)
- `X-Max-Keys: 10` (configured maximum)
- `X-Last-Updated: 2026-01-05T10:30:00Z` (last successful fetch)

### Get All IDP Status
```bash
GET /status
```
Returns detailed status for all IDPs including:
- Last update timestamp
- Update count
- Last error (if any)
- JWKS data

### Get IDP-Specific Status
```bash
GET /status/{idp-name}
```
Returns detailed status for a specific IDP.

## Configuration

Edit `config.yaml` to configure your IDPs:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

idps:
  - name: "auth0"
    url: "https://YOUR_AUTH0_DOMAIN/.well-known/jwks.json"
    refresh_interval: 3600  # seconds - how often to fetch from IDP
    max_keys: 10            # maximum keys to maintain per IDP (standard: 10)
    cache_duration: 900     # cache time in seconds (default: 900 = 15 min)
  - name: "okta"
    url: "https://YOUR_OKTA_DOMAIN/oauth2/default/v1/keys"
    refresh_interval: 3600
    max_keys: 10
    cache_duration: 900
  - name: "keycloak"
    url: "https://YOUR_KEYCLOAK_DOMAIN/auth/realms/YOUR_REALM/protocol/openid-connect/certs"
    refresh_interval: 3600
    max_keys: 10
    cache_duration: 900

logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json or text
```

### Configuration Parameters

| Parameter | Description | Default | Recommended |
|-----------|-------------|---------|-------------|
| `name` | Unique IDP identifier | - | Short, descriptive |
| `url` | JWKS endpoint URL | - | HTTPS only |
| `refresh_interval` | Fetch interval (seconds) | - | 3600 (1 hour) |
| `max_keys` | Maximum keys per IDP | 10 | 10 (standard) |
| `cache_duration` | Cache time (seconds) | 900 | 900 (15 min) |

**Why 10 keys?** Most IDPs maintain 2-5 active keys. 10 keys provides buffer for rotation while preventing memory abuse.

**Understanding refresh_interval vs cache_duration:**
- `refresh_interval`: How often YOUR SERVICE fetches from the IDP
- `cache_duration`: How long YOUR CLIENTS cache data from your service
- They are **independent** but work together
- 📘 See [CACHE_VS_REFRESH_INTERVAL.md](CACHE_VS_REFRESH_INTERVAL.md) for complete explanation with timelines and examples

## Local Development

### Prerequisites
- Go 1.24 or later

### Run Locally
```bash
# Install dependencies
go mod download

# Run the service
go run main.go
```

### Test the API
```bash
# Health check
curl http://localhost:8080/health

# Get all JWKS
curl http://localhost:8080/jwks

# Get specific IDP JWKS
curl http://localhost:8080/jwks/auth0

# Get status
curl http://localhost:8080/status
curl http://localhost:8080/status/auth0
```

## Docker Build

```bash
docker build -t idp-caller:latest .
docker run -p 8080:8080 idp-caller:latest
```

## Kubernetes Deployment

### 1. Update the ConfigMap
Edit `k8s/configmap.yaml` with your IDP URLs.

### 2. Deploy to Kubernetes
```bash
# Apply ConfigMap
kubectl apply -f k8s/configmap.yaml

# Deploy the service
kubectl apply -f k8s/deployment.yaml
```

### 3. Verify Deployment
```bash
# Check pods
kubectl get pods -l app=idp-caller

# Check service
kubectl get svc idp-caller

# View logs
kubectl logs -f deployment/idp-caller

# Port forward for testing
kubectl port-forward svc/idp-caller 8080:80
```

## KrakenD Integration

The service is designed to work seamlessly with KrakenD for JWT validation. Configure KrakenD to use the standard OIDC endpoint:

**Option 1: Single IDP via standard endpoint (Recommended)**
```json
{
  "endpoint": "/api/protected",
  "extra_config": {
    "auth/validator": {
      "alg": "RS256",
      "jwk_url": "http://idp-caller.default.svc.cluster.local/jwks/auth0",
      "cache": true,
      "cache_duration": 900
    }
  }
}
```

**Option 2: All IDPs merged (Multi-IDP support)**
```json
{
  "endpoint": "/api/protected",
  "extra_config": {
    "auth/validator": {
      "alg": "RS256",
      "jwk_url": "http://idp-caller.default.svc.cluster.local/.well-known/jwks.json",
      "cache": true,
      "cache_duration": 900
    }
  }
}
```

This allows JWTs from any configured IDP (Auth0, Okta, Keycloak, etc.) to be validated.

See `k8s/krakend-integration.yaml` for more examples.

## Logging

The service logs all important events in JSON format (configurable):

- Service startup and shutdown
- JWKS update attempts with goroutine per IDP
- Update successes with key count and metadata
- Key truncation warnings when exceeding max_keys
- Update failures with error details
- HTTP requests with timing and status codes
- Last update timestamp for each IDP
- Cache metadata (cache_until, cache_duration)

Example log output:
```json
{
  "time": "2026-01-05T10:30:00Z",
  "level": "INFO",
  "msg": "Successfully updated JWKS",
  "idp": "auth0",
  "key_count": 3,
  "max_keys": 10,
  "cache_duration": 900,
  "cache_until": "2026-01-05T10:45:00Z",
  "last_updated": "2026-01-05T10:30:00Z",
  "update_count": 15
}
```

**Warning for key limits:**
```json
{
  "time": "2026-01-05T10:30:00Z",
  "level": "WARN",
  "msg": "Truncating keys to max limit",
  "idp": "auth0",
  "original_count": 25,
  "max_keys": 10
}
```

## Architecture

- **Manager**: Thread-safe storage for JWKS data with RWMutex
- **Updater**: Goroutine-based periodic fetcher for each IDP
- **Server**: HTTP REST API with middleware
- **Config**: YAML-based configuration management

Each IDP has its own goroutine that:
1. Fetches JWKS immediately on startup
2. Updates periodically based on `refresh_interval`
3. Logs all operations with timestamps
4. Tracks update count and errors
5. **NEW:** Reads IDP's `Cache-Control` headers to optimize caching dynamically

**📘 Caching Details:** See [CACHING_STRATEGY.md](CACHING_STRATEGY.md) for complete information on how the two-level caching system works.

**🔄 Goroutines & Cache Optimization:** See [INDEPENDENT_GOROUTINES_CACHE.md](INDEPENDENT_GOROUTINES_CACHE.md) to understand how each IDP's independent goroutine works with different intervals and how the system uses IDP's `max-age` headers to optimize caching.

## Monitoring

Monitor the service using:
- Health endpoint: `/health`
- Status endpoint: `/status` - shows last update times, error states
- Kubernetes liveness/readiness probes
- Structured logs in JSON format

## License

MIT

