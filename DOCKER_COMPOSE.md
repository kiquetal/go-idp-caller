# Docker Compose Quick Start

This guide shows you how to run the IDP Caller service using Docker Compose.

## Prerequisites

- Docker and Docker Compose installed
- A `config.yaml` file in the project root directory

## Usage

### 1. Edit your configuration

Edit the `config.yaml` file in the project root to configure your IDPs:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

idps:
  - name: "auth0"
    url: "https://YOUR_AUTH0_DOMAIN/.well-known/jwks.json"
    refresh_interval: 3600
  - name: "okta"
    url: "https://YOUR_OKTA_DOMAIN/oauth2/default/v1/keys"
    refresh_interval: 3600

logging:
  level: "info"
  format: "json"
```

### 2. Start the service

```bash
docker-compose up -d
```

### 3. View logs

```bash
docker-compose logs -f
```

### 4. Test the service

```bash
# Check health
curl http://localhost:8080/health

# Get merged JWKS
curl http://localhost:8080/jwks.json

# Get specific IDP keys
curl http://localhost:8080/jwks/auth0
```

### 5. Stop the service

```bash
docker-compose down
```

## Configuration

The docker-compose.yml mounts your local `config.yaml` file into the container at `/etc/idp-caller/config.yaml`.

Any changes to the config file require a container restart:

```bash
docker-compose restart
```

## Environment Variables

You can override environment variables in the docker-compose.yml:

- `CONFIG_PATH`: Path to config file (default: `/etc/idp-caller/config.yaml`)
- `LOG_LEVEL`: Logging level (default: `info`)

## Using a Different Config File

If you want to use a different config file location:

```bash
# Edit docker-compose.yml and change the volumes section:
volumes:
  - /path/to/your/config.yaml:/etc/idp-caller/config.yaml:ro
```

Or create a docker-compose.override.yml:

```yaml
version: '3.8'
services:
  idp-caller:
    volumes:
      - /path/to/your/config.yaml:/etc/idp-caller/config.yaml:ro
```

## Building Locally

To build and run a local image instead of using the published one:

```bash
# Build the image
docker-compose build

# Or specify in docker-compose.yml:
# build: .
# image: go-idp-caller:local
```

## Troubleshooting

### Container won't start
Check the logs:
```bash
docker-compose logs
```

### Config file not found
Ensure `config.yaml` exists in the same directory as `docker-compose.yml`:
```bash
ls -la config.yaml
```

### Permission denied
The config file is mounted read-only (`:ro`). If you need write access, remove `:ro` from the volume mount.

