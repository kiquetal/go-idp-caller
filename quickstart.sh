#!/bin/bash

# Quick Start Script for IDP Caller Service
# This script helps you get started quickly

set -e

echo "🚀 IDP Caller Service - Quick Start"
echo "===================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.24 or later."
    exit 1
fi

GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
echo "✅ Go $GO_VERSION detected"
echo ""

# Step 1: Configure IDPs
echo "📝 Step 1: Configuration"
echo "-----------------------"
if [ ! -f "config.yaml" ]; then
    echo "⚠️  config.yaml not found. Please create it from config.example.yaml"
    echo ""
    echo "Example:"
    echo "  cp config.example.yaml config.yaml"
    echo "  # Then edit config.yaml with your IDP URLs"
    exit 1
fi
echo "✅ config.yaml found"
echo ""

# Step 2: Install dependencies
echo "📦 Step 2: Installing dependencies"
echo "-----------------------------------"
go mod download
echo "✅ Dependencies installed"
echo ""

# Step 3: Build
echo "🔨 Step 3: Building application"
echo "--------------------------------"
go build -o idp-caller .
echo "✅ Build successful"
echo ""

# Step 4: Instructions
echo "🎯 Next Steps:"
echo "-------------"
echo ""
echo "1️⃣  Run the service locally:"
echo "   ./idp-caller"
echo ""
echo "2️⃣  Test the endpoints:"
echo "   curl http://localhost:8080/health"
echo "   curl http://localhost:8080/.well-known/jwks.json  # Merged JWKS (JOSE compatible)"
echo "   curl http://localhost:8080/status"
echo "   curl http://localhost:8080/jwks/auth0  # Single IDP"
echo ""
echo "3️⃣  Deploy to Kubernetes:"
echo "   kubectl apply -f k8s/configmap.yaml"
echo "   kubectl apply -f k8s/deployment.yaml"
echo ""
echo "4️⃣  Use with KrakenD:"
echo "   See KRAKEND_INTEGRATION.md for detailed instructions"
echo ""
echo "📚 Documentation:"
echo "   - QUICKSTART.md - 5-minute quick start guide"
echo "   - README.md - Complete usage guide"
echo "   - MERGED_JWKS_GUIDE.md - JOSE JWT integration"
echo "   - KRAKEND_INTEGRATION.md - KrakenD integration"
echo "   - ARCHITECTURE.md - System architecture"
echo ""
echo "🧪 Run Tests:"
echo "   ./test.sh localhost"
echo ""
echo "🎉 Setup complete! Ready to start the service."

