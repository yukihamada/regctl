#!/bin/bash

# Cloudflare Pages Configuration for SaaS
# Updates Pages projects to use custom domains

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}📄 Configuring Cloudflare Pages for SaaS${NC}"
echo "=========================================="

# Check if required environment variables are set
if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    echo -e "${RED}❌ CLOUDFLARE_API_TOKEN is not set${NC}"
    exit 1
fi

if [ -z "$CLOUDFLARE_ACCOUNT_ID" ]; then
    echo -e "${RED}❌ CLOUDFLARE_ACCOUNT_ID is not set${NC}"
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

echo -e "${YELLOW}📋 Step 1: Adding custom domain to Pages project${NC}"

# Add custom domain to regctl-site Pages project
echo -e "${BLUE}🔧 Adding regctl.com to regctl-site project${NC}"
DOMAIN_DATA='{
  "name": "regctl.com"
}'

DOMAIN_RESPONSE=$(cf_api "POST" "accounts/$CLOUDFLARE_ACCOUNT_ID/pages/projects/regctl-site/domains" "$DOMAIN_DATA")
echo "Response: $DOMAIN_RESPONSE"

# Add www subdomain
echo -e "${BLUE}🔧 Adding www.regctl.com to regctl-site project${NC}"
WWW_DOMAIN_DATA='{
  "name": "www.regctl.com"
}'

WWW_DOMAIN_RESPONSE=$(cf_api "POST" "accounts/$CLOUDFLARE_ACCOUNT_ID/pages/projects/regctl-site/domains" "$WWW_DOMAIN_DATA")
echo "Response: $WWW_DOMAIN_RESPONSE"

echo -e "${YELLOW}📋 Step 2: Configuring app subdomain${NC}"

# Check if regctl-app project exists, if not create it
echo -e "${BLUE}🔧 Checking regctl-app project${NC}"
APP_PROJECT_RESPONSE=$(cf_api "GET" "accounts/$CLOUDFLARE_ACCOUNT_ID/pages/projects/regctl-app")
APP_PROJECT_ID=$(echo "$APP_PROJECT_RESPONSE" | jq -r '.result.id // empty')

if [ -z "$APP_PROJECT_ID" ] || [ "$APP_PROJECT_ID" = "null" ]; then
    echo -e "${YELLOW}⚠️ regctl-app project not found, you may need to create it manually${NC}"
else
    echo -e "${GREEN}✅ regctl-app project found${NC}"
    
    # Add app.regctl.com domain to app project
    echo -e "${BLUE}🔧 Adding app.regctl.com to regctl-app project${NC}"
    APP_DOMAIN_DATA='{
      "name": "app.regctl.com"
    }'
    
    APP_DOMAIN_RESPONSE=$(cf_api "POST" "accounts/$CLOUDFLARE_ACCOUNT_ID/pages/projects/regctl-app/domains" "$APP_DOMAIN_DATA")
    echo "Response: $APP_DOMAIN_RESPONSE"
fi

echo -e "${YELLOW}📋 Step 3: Listing all configured domains${NC}"

# List all domains for regctl-site
echo -e "${BLUE}📄 Domains for regctl-site:${NC}"
SITE_DOMAINS=$(cf_api "GET" "accounts/$CLOUDFLARE_ACCOUNT_ID/pages/projects/regctl-site/domains")
echo "$SITE_DOMAINS" | jq -r '.result[] | "  - \(.name) (Status: \(.status))"'

echo ""
echo -e "${GREEN}🎉 Pages SaaS configuration completed!${NC}"
echo ""
echo -e "${BLUE}📝 Next Steps:${NC}"
echo "1. Wait for domain verification (usually 1-5 minutes)"
echo "2. Ensure DNS records are properly configured"
echo "3. Check SSL certificate status"
echo "4. Test all domains:"
echo "   - https://regctl.com"
echo "   - https://www.regctl.com" 
echo "   - https://app.regctl.com"
echo ""
echo -e "${YELLOW}💡 To check domain status:${NC}"
echo "   curl -H 'Authorization: Bearer \$CLOUDFLARE_API_TOKEN' \\"
echo "        'https://api.cloudflare.com/client/v4/accounts/$CLOUDFLARE_ACCOUNT_ID/pages/projects/regctl-site/domains' | jq"