# GitLab CI/CD Pipeline Guide

This guide explains how to use the GitLab CI/CD pipeline to automatically build and publish Docker images to the GitLab Container Registry.

## Overview

The pipeline automatically builds AMD64 Docker images and pushes them to GitLab Container Registry whenever you push to `main`/`master` branches or create tags.

## Pipeline Stages

### 1. **Build Stage**
- `build`: Builds Docker image for AMD64 architecture
- `build-version`: Builds and tags version-specific images (only for git tags)

## Triggers

The pipeline runs on:
- ✅ Pushes to `main` or `master` branch
- ✅ Git tags (e.g., `v1.0.0`)

## Generated Image Tags

### For every push to main/master:

```
registry.gitlab.com/your-group/go-idp-caller:latest
registry.gitlab.com/your-group/go-idp-caller:abc1234 (commit SHA)
```

### For git tags (e.g., `v1.0.0`):

```
registry.gitlab.com/your-group/go-idp-caller:1.0.0
registry.gitlab.com/your-group/go-idp-caller:latest
```

## Setup Requirements

### 1. GitLab Runner with Docker Support

Your GitLab instance needs a runner with the `docker` tag.

**Check runners:**
- Go to your project → Settings → CI/CD → Runners
- Ensure you have an active runner with the `docker` tag

### 2. Container Registry Enabled

Ensure GitLab Container Registry is enabled:
- Go to Settings → General → Visibility
- Ensure "Container Registry" is enabled

### 3. Permissions

The pipeline uses built-in GitLab CI/CD variables that are automatically available:
- `CI_REGISTRY` - GitLab Container Registry URL
- `CI_REGISTRY_IMAGE` - Your project's registry image path
- `CI_REGISTRY_USER` - CI job user
- `CI_REGISTRY_PASSWORD` - CI job token (automatic authentication)

**No additional secrets needed!** 🎉

## Usage

### Push to main branch:

```bash
git add .
git commit -m "Update feature"
git push origin main
```

**Result:** Images tagged with `latest` and commit SHA

### Create a version release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

**Result:** Images tagged with `1.0.0` and `latest`

## Pulling Images

### From GitLab Container Registry:

```bash
# Login to GitLab Container Registry
docker login registry.gitlab.com

# Pull the latest image
docker pull registry.gitlab.com/your-group/go-idp-caller:latest

# Pull specific version
docker pull registry.gitlab.com/your-group/go-idp-caller:1.0.0

# Pull by commit SHA
docker pull registry.gitlab.com/your-group/go-idp-caller:abc1234
```

### Authentication for pulling:

**Personal Access Token:**
```bash
echo "YOUR_GITLAB_TOKEN" | docker login registry.gitlab.com -u your-username --password-stdin
```

**Deploy Token (recommended for production):**
1. Go to Settings → Repository → Deploy tokens
2. Create a token with `read_registry` scope
3. Use token to login:
```bash
echo "YOUR_DEPLOY_TOKEN" | docker login registry.gitlab.com -u your-deploy-token-name --password-stdin
```

## Pipeline Variables

You can customize the pipeline behavior using CI/CD variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `CI_REGISTRY` | Registry URL | Automatic (GitLab registry) |
| `CI_REGISTRY_IMAGE` | Full image path | Automatic |
| `DOCKER_DRIVER` | Docker storage driver | `overlay2` |

## Monitoring Pipeline

### View pipeline status:

1. Go to your project in GitLab
2. Click "CI/CD" → "Pipelines"
3. Click on a specific pipeline to see job details

### View published images:

1. Go to your project in GitLab
2. Click "Packages & Registries" → "Container Registry"
3. You'll see all published images and tags

## Troubleshooting

### Pipeline fails with "no space left on device"

**Solution:** Clean up Docker cache on runner:
```bash
docker system prune -af
```

### Runner doesn't have docker tag

**Solution:** Add docker tag to your runner:
1. Go to Settings → CI/CD → Runners
2. Edit your runner
3. Add `docker` to tags

### Images not appearing in registry

**Solution:** 
1. Check if Container Registry is enabled
2. Verify runner has permission to push
3. Check job logs for authentication errors

## Advanced Configuration

### Build only on specific branches:

Edit `.gitlab-ci.yml`:
```yaml
only:
  - main
  - production
  - /^release\/.*$/  # All branches starting with release/
```

### Add additional tags:

Edit the build jobs to add more tags:
```yaml
--tag ${IMAGE_NAME}:my-custom-tag \
```

### Change registry:

To use Docker Hub or another registry, set variables:
```yaml
variables:
  REGISTRY: docker.io
  IMAGE_NAME: yourusername/go-idp-caller
```

And add registry credentials in Settings → CI/CD → Variables:
- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`

## Integration with Kubernetes

Update your Kubernetes deployment to use GitLab registry:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: idp-caller
spec:
  template:
    spec:
      containers:
      - name: idp-caller
        image: registry.gitlab.com/your-group/go-idp-caller:latest
        imagePullPolicy: Always
      imagePullSecrets:
      - name: gitlab-registry
```

Create image pull secret:
```bash
kubectl create secret docker-registry gitlab-registry \
  --docker-server=registry.gitlab.com \
  --docker-username=your-deploy-token-name \
  --docker-password=your-deploy-token \
  --namespace=your-namespace
```

## Best Practices

1. ✅ **Use version tags for production**: `v1.0.0` → Creates stable `1.0.0` tag
2. ✅ **Use commit SHA for rollbacks**: Easy to identify and rollback specific commits
3. ✅ **Use latest for development**: Always points to the most recent main build
4. ✅ **Use deploy tokens for production**: More secure than personal tokens
5. ✅ **Monitor pipeline duration**: Optimize slow builds by reviewing job logs
6. ✅ **Enable registry cleanup**: Settings → Packages & Registries → Container Registry → Cleanup policies

## Support

For issues with:
- **GitLab CI/CD**: Check [GitLab CI/CD documentation](https://docs.gitlab.com/ee/ci/)
- **Container Registry**: Check [GitLab Container Registry docs](https://docs.gitlab.com/ee/user/packages/container_registry/)
- **Docker Buildx**: Check [Docker Buildx documentation](https://docs.docker.com/buildx/working-with-buildx/)

