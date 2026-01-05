#!/bin/bash

# Test script for IDP caller service
# Usage: ./test.sh [localhost|k8s]

TARGET="${1:-localhost}"
BASE_URL=""

if [ "$TARGET" = "k8s" ]; then
    echo "Testing Kubernetes service..."
    kubectl port-forward svc/idp-caller 8080:80 &
    PID=$!
    sleep 2
    BASE_URL="http://localhost:8080"
elif [ "$TARGET" = "localhost" ]; then
    echo "Testing local service..."
    BASE_URL="http://localhost:8080"
else
    echo "Unknown target: $TARGET"
    echo "Usage: $0 [localhost|k8s]"
    exit 1
fi

echo ""
echo "=== Health Check ==="
curl -s "$BASE_URL/health" | jq .

echo ""
echo "=== Get All Status ==="
curl -s "$BASE_URL/status" | jq .

echo ""
echo "=== Get All JWKS ==="
curl -s "$BASE_URL/jwks" | jq .

echo ""
echo "=== Get Auth0 Status ==="
curl -s "$BASE_URL/status/auth0" | jq .

echo ""
echo "=== Get Auth0 JWKS ==="
curl -s "$BASE_URL/jwks/auth0" | jq .

echo ""
echo "=== Test 404 - Non-existent IDP ==="
curl -s -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/jwks/nonexistent"

echo ""
echo "Tests completed!"

if [ ! -z "$PID" ]; then
    kill $PID 2>/dev/null
fi

