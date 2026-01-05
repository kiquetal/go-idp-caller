#!/bin/bash

# Test script for IDP caller service
# Usage: ./test.sh [localhost|k8s]

set -e

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
echo "======================================"
echo "  IDP Caller Service - Test Suite"
echo "======================================"

echo ""
echo "=== Health Check ==="
curl -s "$BASE_URL/health" | jq .

echo ""
echo "=== Get Merged JWKS (Standard OIDC Endpoint) ==="
echo "Testing: /.well-known/jwks.json"
echo ""
echo "Response Headers:"
curl -s -D - "$BASE_URL/.well-known/jwks.json" | grep -E "(Cache-Control|X-Total-Keys|X-IDP-Count)"
echo ""
echo "Response Body (first 2 keys):"
curl -s "$BASE_URL/.well-known/jwks.json" | jq '{keys: .keys[:2]}'
echo ""
echo "Total key count:"
curl -s "$BASE_URL/.well-known/jwks.json" | jq '.keys | length'

echo ""
echo "=== Get All Status ==="
curl -s "$BASE_URL/status" | jq 'to_entries | map({idp: .key, key_count: .value.key_count, max_keys: .value.max_keys, last_updated: .value.last_updated, update_count: .value.update_count})'

echo ""
echo "=== Get All JWKS (Separated by IDP) ==="
curl -s "$BASE_URL/jwks" | jq 'to_entries | map({idp: .key, key_count: (.value.keys | length)})'

echo ""
echo "=== Get Individual IDP - Auth0 ==="
echo "Response Headers:"
curl -s -D - "$BASE_URL/jwks/auth0" 2>&1 | grep -E "(Cache-Control|X-Key-Count|X-Max-Keys|X-Last-Updated)" || echo "Headers not available (might be 404)"
echo ""
echo "Response Body:"
curl -s "$BASE_URL/jwks/auth0" | jq . 2>/dev/null || echo "Auth0 not configured or not available"

echo ""
echo "=== Get Status for Auth0 ==="
curl -s "$BASE_URL/status/auth0" | jq . 2>/dev/null || echo "Auth0 not configured or not available"

echo ""
echo "=== Test 404 - Non-existent IDP ==="
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/jwks/nonexistent")
echo "HTTP Status: $HTTP_CODE"
if [ "$HTTP_CODE" = "404" ]; then
    echo "✅ Correctly returns 404 for non-existent IDP"
else
    echo "❌ Expected 404, got $HTTP_CODE"
fi

echo ""
echo "=== Verify JOSE JWT Compatibility ==="
echo "Checking merged JWKS structure..."
JWKS=$(curl -s "$BASE_URL/.well-known/jwks.json")
HAS_KEYS=$(echo "$JWKS" | jq 'has("keys")')
if [ "$HAS_KEYS" = "true" ]; then
    echo "✅ Has 'keys' array (JOSE compatible)"

    # Check first key has required fields
    FIRST_KEY=$(echo "$JWKS" | jq '.keys[0]')
    HAS_KTY=$(echo "$FIRST_KEY" | jq 'has("kty")')
    HAS_KID=$(echo "$FIRST_KEY" | jq 'has("kid")')

    if [ "$HAS_KTY" = "true" ] && [ "$HAS_KID" = "true" ]; then
        echo "✅ Keys have required fields (kty, kid)"
    else
        echo "❌ Keys missing required fields"
    fi
else
    echo "❌ Missing 'keys' array"
fi

echo ""
echo "=== Test Complete ==="

# Cleanup
if [ "$TARGET" = "k8s" ] && [ -n "$PID" ]; then
    kill $PID 2>/dev/null || true
fi

echo ""
echo "Tests completed!"

if [ ! -z "$PID" ]; then
    kill $PID 2>/dev/null
fi

