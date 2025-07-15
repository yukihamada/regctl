#!/bin/bash

# Cloudflare for SaaS Setup Script
# regctl.com ドメインでのカスタムホスト名設定

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="regctl.com"
ZONE_NAME="regctl.com"
API_SUBDOMAIN="api.regctl.com"
APP_SUBDOMAIN="app.regctl.com"

echo -e "${BLUE}🌐 Cloudflare for SaaS Setup for regctl.com${NC}"
echo "============================================="

# Check if required environment variables are set
if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    echo -e "${RED}❌ CLOUDFLARE_API_TOKEN is not set${NC}"
    echo "Please set your Cloudflare API token:"
    echo "export CLOUDFLARE_API_TOKEN=your_token_here"
    exit 1
fi

if [ -z "$CLOUDFLARE_ACCOUNT_ID" ]; then
    echo -e "${RED}❌ CLOUDFLARE_ACCOUNT_ID is not set${NC}"
    echo "Please set your Cloudflare Account ID:"
    echo "export CLOUDFLARE_ACCOUNT_ID=your_account_id_here"
    exit 1
fi

# Function to make Cloudflare API calls
cf_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -n "$data" ]; then
        curl -s -X "$method" \
            -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
            -H "Content-Type: application/json" \
            --data "$data" \
            "https://api.cloudflare.com/client/v4/$endpoint"
    else
        curl -s -X "$method" \
            -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
            -H "Content-Type: application/json" \
            "https://api.cloudflare.com/client/v4/$endpoint"
    fi
}

echo -e "${YELLOW}📋 Step 1: Getting Zone ID for $ZONE_NAME${NC}"
ZONE_RESPONSE=$(cf_api "GET" "zones?name=$ZONE_NAME")
ZONE_ID=$(echo "$ZONE_RESPONSE" | jq -r '.result[0].id // empty')

if [ -z "$ZONE_ID" ] || [ "$ZONE_ID" = "null" ]; then
    echo -e "${RED}❌ Zone $ZONE_NAME not found. Please ensure the domain is added to Cloudflare.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Zone ID found: $ZONE_ID${NC}"

echo -e "${YELLOW}📋 Step 2: Setting up Custom Hostnames for SaaS${NC}"

# Setup custom hostname for main domain
echo -e "${BLUE}🔧 Setting up custom hostname for $DOMAIN${NC}"
CUSTOM_HOSTNAME_DATA='{
  "hostname": "'$DOMAIN'",
  "ssl": {
    "method": "http",
    "type": "dv",
    "settings": {
      "http2": "on",
      "min_tls_version": "1.2",
      "tls_1_3": "on"
    }
  }
}'

HOSTNAME_RESPONSE=$(cf_api "POST" "zones/$ZONE_ID/custom_hostnames" "$CUSTOM_HOSTNAME_DATA")
HOSTNAME_ID=$(echo "$HOSTNAME_RESPONSE" | jq -r '.result.id // empty')

if [ -n "$HOSTNAME_ID" ] && [ "$HOSTNAME_ID" != "null" ]; then
    echo -e "${GREEN}✅ Custom hostname created for $DOMAIN: $HOSTNAME_ID${NC}"
else
    echo -e "${YELLOW}⚠️ Custom hostname may already exist or error occurred${NC}"
    echo "Response: $HOSTNAME_RESPONSE"
fi

# Setup custom hostname for API subdomain
echo -e "${BLUE}🔧 Setting up custom hostname for $API_SUBDOMAIN${NC}"
API_HOSTNAME_DATA='{
  "hostname": "'$API_SUBDOMAIN'",
  "ssl": {
    "method": "http",
    "type": "dv",
    "settings": {
      "http2": "on",
      "min_tls_version": "1.2",
      "tls_1_3": "on"
    }
  }
}'

API_HOSTNAME_RESPONSE=$(cf_api "POST" "zones/$ZONE_ID/custom_hostnames" "$API_HOSTNAME_DATA")
API_HOSTNAME_ID=$(echo "$API_HOSTNAME_RESPONSE" | jq -r '.result.id // empty')

if [ -n "$API_HOSTNAME_ID" ] && [ "$API_HOSTNAME_ID" != "null" ]; then
    echo -e "${GREEN}✅ Custom hostname created for $API_SUBDOMAIN: $API_HOSTNAME_ID${NC}"
else
    echo -e "${YELLOW}⚠️ API custom hostname may already exist or error occurred${NC}"
    echo "Response: $API_HOSTNAME_RESPONSE"
fi

# Setup custom hostname for app subdomain
echo -e "${BLUE}🔧 Setting up custom hostname for $APP_SUBDOMAIN${NC}"
APP_HOSTNAME_DATA='{
  "hostname": "'$APP_SUBDOMAIN'",
  "ssl": {
    "method": "http",
    "type": "dv",
    "settings": {
      "http2": "on",
      "min_tls_version": "1.2",
      "tls_1_3": "on"
    }
  }
}'

APP_HOSTNAME_RESPONSE=$(cf_api "POST" "zones/$ZONE_ID/custom_hostnames" "$APP_HOSTNAME_DATA")
APP_HOSTNAME_ID=$(echo "$APP_HOSTNAME_RESPONSE" | jq -r '.result.id // empty')

if [ -n "$APP_HOSTNAME_ID" ] && [ "$APP_HOSTNAME_ID" != "null" ]; then
    echo -e "${GREEN}✅ Custom hostname created for $APP_SUBDOMAIN: $APP_HOSTNAME_ID${NC}"
else
    echo -e "${YELLOW}⚠️ App custom hostname may already exist or error occurred${NC}"
    echo "Response: $APP_HOSTNAME_RESPONSE"
fi

echo -e "${YELLOW}📋 Step 3: Setting up DNS Records${NC}"

# Create A record for main domain pointing to Cloudflare Pages
echo -e "${BLUE}🔧 Creating DNS A record for $DOMAIN${NC}"
DNS_RECORD_DATA='{
  "type": "CNAME",
  "name": "'$DOMAIN'",
  "content": "regctl-site.pages.dev",
  "ttl": 1,
  "proxied": true
}'

DNS_RESPONSE=$(cf_api "POST" "zones/$ZONE_ID/dns_records" "$DNS_RECORD_DATA")
DNS_RECORD_ID=$(echo "$DNS_RESPONSE" | jq -r '.result.id // empty')

if [ -n "$DNS_RECORD_ID" ] && [ "$DNS_RECORD_ID" != "null" ]; then
    echo -e "${GREEN}✅ DNS record created for $DOMAIN${NC}"
else
    echo -e "${YELLOW}⚠️ DNS record may already exist or error occurred${NC}"
    echo "Response: $DNS_RESPONSE"
fi

# Create CNAME record for API subdomain
echo -e "${BLUE}🔧 Creating DNS CNAME record for $API_SUBDOMAIN${NC}"
API_DNS_DATA='{
  "type": "CNAME",
  "name": "api",
  "content": "regctl-api.yukihamada.workers.dev",
  "ttl": 1,
  "proxied": true
}'

API_DNS_RESPONSE=$(cf_api "POST" "zones/$ZONE_ID/dns_records" "$API_DNS_DATA")
API_DNS_ID=$(echo "$API_DNS_RESPONSE" | jq -r '.result.id // empty')

if [ -n "$API_DNS_ID" ] && [ "$API_DNS_ID" != "null" ]; then
    echo -e "${GREEN}✅ DNS record created for $API_SUBDOMAIN${NC}"
else
    echo -e "${YELLOW}⚠️ API DNS record may already exist or error occurred${NC}"
    echo "Response: $API_DNS_RESPONSE"
fi

# Create CNAME record for app subdomain
echo -e "${BLUE}🔧 Creating DNS CNAME record for $APP_SUBDOMAIN${NC}"
APP_DNS_DATA='{
  "type": "CNAME",
  "name": "app",
  "content": "regctl-app.pages.dev",
  "ttl": 1,
  "proxied": true
}'

APP_DNS_RESPONSE=$(cf_api "POST" "zones/$ZONE_ID/dns_records" "$APP_DNS_DATA")
APP_DNS_ID=$(echo "$APP_DNS_RESPONSE" | jq -r '.result.id // empty')

if [ -n "$APP_DNS_ID" ] && [ "$APP_DNS_ID" != "null" ]; then
    echo -e "${GREEN}✅ DNS record created for $APP_SUBDOMAIN${NC}"
else
    echo -e "${YELLOW}⚠️ App DNS record may already exist or error occurred${NC}"
    echo "Response: $APP_DNS_RESPONSE"
fi

echo -e "${YELLOW}📋 Step 4: Checking SSL Certificate Status${NC}"

# Check SSL status for custom hostnames
echo -e "${BLUE}🔧 Checking SSL status...${NC}"
sleep 5

HOSTNAMES_RESPONSE=$(cf_api "GET" "zones/$ZONE_ID/custom_hostnames")
echo "$HOSTNAMES_RESPONSE" | jq -r '.result[] | "Hostname: \(.hostname) | SSL Status: \(.ssl.status) | SSL Method: \(.ssl.method)"'

echo ""
echo -e "${GREEN}🎉 Cloudflare for SaaS setup completed!${NC}"
echo ""
echo -e "${BLUE}📝 Next Steps:${NC}"
echo "1. Wait for SSL certificates to be issued (usually 1-15 minutes)"
echo "2. Update your applications to use the new domains:"
echo "   - Main site: https://$DOMAIN"
echo "   - API: https://$API_SUBDOMAIN"
echo "   - App: https://$APP_SUBDOMAIN"
echo "3. Test the domains to ensure they're working correctly"
echo ""
echo -e "${YELLOW}💡 To check SSL status:${NC}"
echo "   curl -H 'Authorization: Bearer \$CLOUDFLARE_API_TOKEN' \\"
echo "        'https://api.cloudflare.com/client/v4/zones/$ZONE_ID/custom_hostnames' | jq"
echo ""
echo -e "${GREEN}✨ Setup complete! Your regctl.com domain is now configured with Cloudflare for SaaS.${NC}"