# KrakenD Integration Guide

This guide explains how to integrate the IDP JWS Caller service with KrakenD API Gateway for JWT validation.

## Overview

The IDP JWS Caller service acts as a centralized JWKS provider for KrakenD, offering several benefits:

- **Centralized Management**: Configure all your IDPs in one place
- **Automatic Updates**: JWKS are refreshed automatically in the background
- **Monitoring**: Track last update times and errors for each IDP
- **High Availability**: Deploy as a Kubernetes service with multiple replicas
- **Performance**: KrakenD can cache JWKS locally, reducing latency
- **JOSE JWT Compatible**: Provides merged JWKS endpoint for all IDPs

## Architecture

```
[Client] → [KrakenD] → [Your API]
              ↓
      [IDP Caller Service]
              ↓
      [Multiple IDPs: Auth0, Okta, Keycloak, etc.]
```

## Kubernetes Deployment

### 1. Deploy IDP Caller Service

```bash
# Update config with your IDP URLs
kubectl apply -f k8s/configmap.yaml

# Deploy the service
kubectl apply -f k8s/deployment.yaml

# Verify deployment
kubectl get svc idp-caller
```

The service will be available at: `http://idp-caller.default.svc.cluster.local`

### 2. Configure KrakenD

Add JWT validation to your KrakenD configuration (`krakend.json`):

#### Merged JWKS (All IDPs - Recommended for JOSE JWT)

Use this when you want to accept tokens from **multiple IDPs** and let the JWT library automatically find the correct key by `kid`:

```json
{
  "$schema": "https://www.krakend.io/schema/v3.json",
  "version": 3,
  "endpoints": [
    {
      "endpoint": "/api/protected",
      "method": "GET",
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller.default.svc.cluster.local/.well-known/jwks.json",
          "cache": true,
          "cache_duration": 900,
          "disable_jwk_security": false
        }
      },
      "backend": [
        {
          "url_pattern": "/protected",
          "host": ["http://backend-service"]
        }
      ]
    }
  ]
}
```

**This endpoint returns all keys from all configured IDPs in a single array**, making it compatible with JOSE JWT libraries that expect the standard JWKS format.

#### Single IDP Example

```json
{
  "$schema": "https://www.krakend.io/schema/v3.json",
  "version": 3,
  "endpoints": [
    {
      "endpoint": "/api/protected",
      "method": "GET",
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller.default.svc.cluster.local/jwks/auth0",
          "cache": true,
          "cache_duration": 900,
          "disable_jwk_security": false,
          "issuer": "https://your-tenant.auth0.com/",
          "audience": ["https://your-api.example.com"]
        }
      },
      "backend": [
        {
          "url_pattern": "/protected",
          "host": ["http://backend-service"]
        }
      ]
    }
  ]
}
```

#### Multiple IDPs with Different Endpoints

```json
{
  "endpoints": [
    {
      "endpoint": "/api/auth0-protected",
      "method": "GET",
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller.default.svc.cluster.local/jwks/auth0",
          "cache": true,
          "cache_duration": 900,
          "issuer": "https://your-tenant.auth0.com/"
        }
      },
      "backend": [{"url_pattern": "/resource", "host": ["http://backend"]}]
    },
    {
      "endpoint": "/api/okta-protected",
      "method": "GET",
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller.default.svc.cluster.local/jwks/okta",
          "cache": true,
          "cache_duration": 900,
          "issuer": "https://your-domain.okta.com/oauth2/default"
        }
      },
      "backend": [{"url_pattern": "/resource", "host": ["http://backend"]}]
    }
  ]
}
```

#### With Role-Based Access Control

```json
{
  "endpoint": "/api/admin",
  "method": "POST",
  "extra_config": {
    "auth/validator": {
      "alg": "RS256",
      "jwk_url": "http://idp-caller.default.svc.cluster.local/jwks/auth0",
      "cache": true,
      "cache_duration": 900,
      "roles_key": "roles",
      "roles": ["admin"],
      "issuer": "https://your-tenant.auth0.com/",
      "audience": ["https://your-api.example.com"]
    }
  },
  "backend": [
    {
      "url_pattern": "/admin/action",
      "host": ["http://backend-service"]
    }
  ]
}
```

## Configuration Options

### IDP Caller Service Configuration

| Parameter | Description | Example |
|-----------|-------------|---------|
| `name` | Unique identifier for the IDP | `auth0`, `okta`, `keycloak` |
| `url` | JWKS endpoint URL | `https://tenant.auth0.com/.well-known/jwks.json` |
| `refresh_interval` | Update frequency in seconds | `3600` (1 hour) |

### KrakenD JWT Validator Options

| Parameter | Description | Required |
|-----------|-------------|----------|
| `alg` | Signing algorithm | Yes (typically `RS256`) |
| `jwk_url` | JWKS endpoint URL | Yes |
| `cache` | Enable JWKS caching | Recommended: `true` |
| `cache_duration` | Cache duration in seconds | Recommended: `900` (15 min) |
| `issuer` | Expected token issuer | Yes |
| `audience` | Expected audience(s) | Recommended |
| `roles_key` | JWT claim containing roles | Optional |
| `roles` | Required roles | Optional |

## API Endpoints

### Available Endpoints

| Endpoint | Description | Use Case |
|----------|-------------|----------|
| `GET /health` | Health check | Kubernetes probes |
| `GET /.well-known/jwks.json` | **Merged JWKS (all IDPs)** | **JOSE JWT, KrakenD (multi-IDP)** |
| `GET /jwks.json` | **Merged JWKS (all IDPs)** | **Alternative merged endpoint** |
| `GET /jwks/all` | **Merged JWKS (all IDPs)** | **Alternative merged endpoint** |
| `GET /jwks` | All JWKS by IDP name | Debugging, custom use |
| `GET /jwks/{idp}` | Specific IDP JWKS | KrakenD (single IDP) |
| `GET /status` | All IDP status | Monitoring |
| `GET /status/{idp}` | Specific IDP status | Debugging |

### Example Requests

```bash
# Get merged JWKS from all IDPs (JOSE JWT format)
curl http://idp-caller.default.svc.cluster.local/.well-known/jwks.json

# Get JWKS for specific IDP (Auth0)
curl http://idp-caller.default.svc.cluster.local/jwks/auth0

# Check last update time
curl http://idp-caller.default.svc.cluster.local/status/auth0

# Health check
curl http://idp-caller.default.svc.cluster.local/health
```

## Testing the Integration

### 1. Port Forward the Service

```bash
kubectl port-forward svc/idp-caller 8080:80
```

### 2. Test JWKS Retrieval

```bash
# Get JWKS
curl http://localhost:8080/jwks/auth0 | jq .

# Check status
curl http://localhost:8080/status/auth0 | jq .
```

### 3. Test with KrakenD

```bash
# Start KrakenD
krakend run -c krakend.json

# Test protected endpoint with valid JWT
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/api/protected
```

## Monitoring and Observability

### Health Checks

KrakenD and Kubernetes can monitor the service:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Logs

The service logs all important events in JSON format:

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

View logs:

```bash
kubectl logs -f deployment/idp-caller
```

### Metrics Endpoint (Optional Enhancement)

For production deployments, consider adding Prometheus metrics:

- `idp_jwks_update_count_total` - Total updates per IDP
- `idp_jwks_update_errors_total` - Failed updates per IDP
- `idp_jwks_last_update_timestamp` - Last successful update
- `http_request_duration_seconds` - API request latency

## Troubleshooting

### Issue: KrakenD reports "JWKS not found"

**Solution:**
1. Check if IDP Caller is running: `kubectl get pods -l app=idp-caller`
2. Check logs: `kubectl logs deployment/idp-caller`
3. Verify IDP name matches: `curl http://idp-caller/status`

### Issue: JWT validation fails

**Solution:**
1. Verify JWKS are being fetched: `curl http://idp-caller/jwks/your-idp`
2. Check issuer and audience in KrakenD config match your JWT claims
3. Verify token hasn't expired
4. Check algorithm matches (usually `RS256`)

### Issue: Slow JWT validation

**Solution:**
1. Enable caching in KrakenD: `"cache": true`
2. Increase cache duration: `"cache_duration": 900`
3. Ensure IDP Caller has multiple replicas for HA

## Best Practices

1. **Cache Duration**: Set KrakenD cache to 15-30 minutes to balance freshness and performance
2. **Refresh Interval**: Set IDP Caller refresh to 30-60 minutes
3. **High Availability**: Run at least 2 replicas of IDP Caller
4. **Monitoring**: Set up alerts on update failures
5. **Security**: Use NetworkPolicies to restrict access to IDP Caller
6. **Resource Limits**: Configure appropriate CPU/memory limits
7. **Namespace**: Deploy in the same namespace as KrakenD for optimal networking

## Advanced Configuration

### Using Multiple Namespaces

If KrakenD is in a different namespace:

```json
{
  "jwk_url": "http://idp-caller.idp-namespace.svc.cluster.local/jwks/auth0"
}
```

### TLS/HTTPS

For production, configure TLS:

1. Add TLS termination at Ingress level
2. Update KrakenD config to use `https://`
3. Mount TLS certificates in IDP Caller pods

### Custom Headers

To pass custom headers to IDPs, modify `internal/jwks/updater.go`:

```go
req.Header.Set("User-Agent", "IDP-Caller-Service/1.0")
req.Header.Set("X-Custom-Header", "value")
```

## Support

For issues or questions:
1. Check service logs: `kubectl logs deployment/idp-caller`
2. Review status endpoint: `/status`
3. Verify IDP URLs are accessible from cluster
4. Check KrakenD logs for JWT validation errors

