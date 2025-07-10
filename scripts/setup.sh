#!/bin/bash

set -e

echo "🚀 Setting up regctl Cloud development environment..."

# Check dependencies
echo "📋 Checking dependencies..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go 1.23 or later."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✅ Go $GO_VERSION found"

# Check Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js not found. Please install Node.js 18 or later."
    exit 1
fi

NODE_VERSION=$(node --version)
echo "✅ Node.js $NODE_VERSION found"

# Check Wrangler
if ! command -v wrangler &> /dev/null; then
    echo "📦 Installing Wrangler..."
    npm install -g wrangler
fi

WRANGLER_VERSION=$(wrangler --version | awk '{print $2}')
echo "✅ Wrangler $WRANGLER_VERSION found"

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod download

# Install Node dependencies
echo "📦 Installing Node dependencies..."
cd edge/workers
npm install
cd ../..

# Setup Cloudflare D1 database
echo "🗄️ Setting up Cloudflare D1 database..."
cd edge/workers

# Create D1 database (if not exists)
if ! wrangler d1 list | grep -q "regctl-db"; then
    echo "Creating D1 database..."
    wrangler d1 create regctl-db
fi

# Run migrations
echo "Running database migrations..."
wrangler d1 execute regctl-db --local --file=./schema.sql

cd ../..

# Build CLI
echo "🔨 Building regctl CLI..."
cd cmd/regctl
go build -o ../../regctl
cd ../..

# Setup config directory
echo "📁 Setting up configuration..."
mkdir -p ~/.regctl

# Create example .env file
if [ ! -f .env ]; then
    cat > .env.example << EOF
# Cloudflare Workers
CLOUDFLARE_ACCOUNT_ID=your-account-id
CLOUDFLARE_API_TOKEN=your-api-token

# Registrar API Keys
VALUE_DOMAIN_API_KEY=your-value-domain-key
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
PORKBUN_API_KEY=your-porkbun-key
PORKBUN_API_SECRET=your-porkbun-secret

# Other services
STRIPE_SECRET_KEY=your-stripe-key
JWT_SECRET=your-jwt-secret
EOF
    echo "📝 Created .env.example - Please copy to .env and fill in your credentials"
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Copy .env.example to .env and fill in your API credentials"
echo "2. Run 'make dev' to start the development server"
echo "3. Run './regctl login' to authenticate with the API"
echo ""
echo "Happy coding! 🎉"