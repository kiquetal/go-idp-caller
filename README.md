# IDP JWS/JWKS Caller Service

A Go service that fetches and maintains JSON Web Key Sets (JWKS) from multiple Identity Providers (IDPs). It uses goroutines to keep the JWKS updated periodically and provides a REST API for consumption by API gateways like KrakenD.

## Features

- ✅ Fetch JWKS from multiple IDP endpoints
- ✅ Periodic background updates using goroutines
- ✅ Detailed logging with timestamps for each update
- ✅ Track last update time and update count per IDP
- ✅ REST API for retrieving JWKS
- ✅ Health check endpoint
- ✅ Status endpoint with metadata
- ✅ Kubernetes ready with deployment manifests
- ✅ KrakenD integration support
- ✅ Graceful shutdown
- ✅ Thread-safe concurrent access

## API Endpoints

### Health Check
```bash
GET /health
```
Returns service health status.

### Get All JWKS
```bash
GET /jwks
```
Returns JWKS from all configured IDPs.

### Get IDP-Specific JWKS
```bash
GET /jwks/{idp-name}
```
Returns JWKS for a specific IDP (e.g., `/jwks/auth0`).

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
    refresh_interval: 3600  # seconds
  - name: "okta"
    url: "https://YOUR_OKTA_DOMAIN/oauth2/default/v1/keys"
    refresh_interval: 3600
  - name: "keycloak"
    url: "https://YOUR_KEYCLOAK_DOMAIN/auth/realms/YOUR_REALM/protocol/openid-connect/certs"
    refresh_interval: 3600

logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json or text
```

## Local Development

### Prerequisites
- Go 1.22 or later

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

The service is designed to work seamlessly with KrakenD for JWT validation. Configure KrakenD to use the service as the JWKS source:

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

See `k8s/krakend-integration.yaml` for more examples.

## Logging

The service logs all important events in JSON format (configurable):

- Service startup and shutdown
- JWKS update attempts
- Update successes with key count
- Update failures with error details
- HTTP requests with timing
- Last update timestamp for each IDP

Example log output:
```json
{
  "time": "2026-01-05T10:30:00Z",
  "level": "INFO",
  "msg": "Successfully updated JWKS",
  "idp": "auth0",
  "keys_count": 2,
  "last_updated": "2026-01-05T10:30:00Z",
  "update_count": 15
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

## Monitoring

Monitor the service using:
- Health endpoint: `/health`
- Status endpoint: `/status` - shows last update times, error states
- Kubernetes liveness/readiness probes
- Structured logs in JSON format

## License

MIT

