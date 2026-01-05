# Quick Start Guide

Get up and running with the IDP JWKS Caller in 5 minutes!

## 1. Prerequisites

- Go 1.24+ installed
- Docker (optional, for containerized deployment)
- Kubernetes cluster (optional, for k8s deployment)

## 2. Local Development Setup

### Step 1: Clone and Configure

```bash
# Navigate to the project
cd go-idp-caller

# Copy example config
cp config.example.yaml config.yaml

# Edit config.yaml with your IDP URLs
nano config.yaml
```

### Step 2: Update Configuration

Edit `config.yaml` with your actual IDP endpoints:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

idps:
  - name: "auth0"
    url: "https://YOUR-TENANT.auth0.com/.well-known/jwks.json"
    refresh_interval: 3600
    max_keys: 10
    cache_duration: 900
```

### Step 3: Run Locally

```bash
# Install dependencies
go mod download

# Run the service
go run main.go
```

You should see:
```
INFO Starting IDP JWS caller service
INFO Starting updater for IDP name=auth0 url=https://...
INFO Starting HTTP server addr=0.0.0.0:8080
```

### Step 4: Test the Endpoints

Open another terminal:

```bash
# Health check
curl http://localhost:8080/health

# Get merged JWKS (all IDPs)
curl http://localhost:8080/.well-known/jwks.json

# Get status
curl http://localhost:8080/status

# Run full test suite
chmod +x test.sh
./test.sh localhost
```

## 3. Docker Deployment

```bash
# Build image
docker build -t idp-caller:latest .

# Run container
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/root/config.yaml \
  idp-caller:latest

# Test
curl http://localhost:8080/.well-known/jwks.json
```

## 4. Kubernetes Deployment

### Step 1: Update ConfigMap

Edit `k8s/configmap.yaml` with your IDP URLs:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: idp-caller-config
data:
  config.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"
    idps:
      - name: "auth0"
        url: "https://YOUR-TENANT.auth0.com/.well-known/jwks.json"
        refresh_interval: 3600
        max_keys: 10
        cache_duration: 900
```

### Step 2: Deploy

```bash
# Apply ConfigMap
kubectl apply -f k8s/configmap.yaml

# Deploy application
kubectl apply -f k8s/deployment.yaml

# Verify
kubectl get pods -l app=idp-caller
kubectl get svc idp-caller
```

### Step 3: Test in Kubernetes

```bash
# Port forward
kubectl port-forward svc/idp-caller 8080:80

# Test (in another terminal)
curl http://localhost:8080/.well-known/jwks.json

# Or run test suite
./test.sh k8s
```

## 5. Integrate with KrakenD

Add to your KrakenD configuration:

```json
{
  "version": 3,
  "endpoints": [
    {
      "endpoint": "/api/protected",
      "method": "GET",
      "backend": [
        {
          "url_pattern": "/protected-resource",
          "host": ["http://backend:8080"]
        }
      ],
      "extra_config": {
        "auth/validator": {
          "alg": "RS256",
          "jwk_url": "http://idp-caller.default.svc.cluster.local/.well-known/jwks.json",
          "cache": true,
          "cache_duration": 900,
          "disable_jwk_security": false
        }
      }
    }
  ]
}
```

## 6. Verify It's Working

### Check Health
```bash
curl http://localhost:8080/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "time": "2026-01-05T10:30:00Z"
}
```

### Check Merged JWKS
```bash
curl http://localhost:8080/.well-known/jwks.json | jq '.keys | length'
```

**Expected:** Number of keys from all IDPs combined

### Check Status
```bash
curl http://localhost:8080/status | jq '.[] | {name, key_count, last_updated}'
```

**Expected Response:**
```json
{
  "name": "auth0",
  "key_count": 3,
  "last_updated": "2026-01-05T10:30:00Z"
}
```

### Verify Headers
```bash
curl -I http://localhost:8080/.well-known/jwks.json
```

**Expected Headers:**
```
Cache-Control: public, max-age=900
X-Total-Keys: 9
X-IDP-Count: 3
```

## 7. Common Issues

### Issue: "Failed to load configuration"
**Solution:** Make sure `config.yaml` exists and is valid YAML

```bash
# Validate YAML syntax
cat config.yaml | python3 -c "import yaml, sys; yaml.safe_load(sys.stdin)"
```

### Issue: "Failed to update JWKS" in logs
**Solution:** Check IDP URL is accessible

```bash
# Test IDP URL directly
curl https://YOUR-TENANT.auth0.com/.well-known/jwks.json
```

### Issue: Service returns 404 for specific IDP
**Solution:** Verify IDP name matches config.yaml

```bash
# List configured IDPs
curl http://localhost:8080/status | jq 'keys'
```

### Issue: No keys in merged endpoint
**Solution:** Wait for initial fetch (up to refresh_interval) or check logs

```bash
# Check logs for errors
kubectl logs -f deployment/idp-caller

# Or in Docker
docker logs <container-id>
```

## 8. Next Steps

- ✅ Read [MERGED_JWKS_GUIDE.md](MERGED_JWKS_GUIDE.md) for detailed usage
- ✅ See [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- ✅ Check [KRAKEND_INTEGRATION.md](KRAKEND_INTEGRATION.md) for API gateway setup
- ✅ Review [README.md](README.md) for full documentation

## Production Checklist

Before deploying to production:

- [ ] Use HTTPS URLs for all IDP endpoints
- [ ] Set appropriate `refresh_interval` (recommended: 3600)
- [ ] Configure `max_keys: 10` per IDP
- [ ] Set `cache_duration: 900` (15 minutes)
- [ ] Use JSON logging format (`format: "json"`)
- [ ] Set up monitoring on `/health` endpoint
- [ ] Configure Kubernetes liveness/readiness probes
- [ ] Set resource limits (CPU: 100m, Memory: 64Mi)
- [ ] Test with actual JWT tokens from each IDP
- [ ] Monitor logs for "Truncating keys" warnings
- [ ] Set up alerts for update failures

## Quick Commands Reference

```bash
# Local Development
go run main.go                           # Run locally
./test.sh localhost                      # Test local service

# Docker
docker build -t idp-caller:latest .      # Build image
docker run -p 8080:8080 idp-caller       # Run container

# Kubernetes
kubectl apply -f k8s/                    # Deploy all
kubectl logs -f deployment/idp-caller    # View logs
kubectl port-forward svc/idp-caller 8080:80  # Port forward
./test.sh k8s                           # Test k8s service

# Testing
curl http://localhost:8080/health        # Health check
curl http://localhost:8080/.well-known/jwks.json  # Merged JWKS
curl http://localhost:8080/status        # Status of all IDPs
```

---

**Need Help?** Check the logs first:
```bash
# Kubernetes
kubectl logs -f deployment/idp-caller

# Docker
docker logs -f <container-id>

# Local
# Logs are output to stdout
```

