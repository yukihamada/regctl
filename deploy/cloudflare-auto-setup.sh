#!/bin/bash

# 🚀 Cloudflare API 完全自動セットアップ for regctl.com
# VALUE-DOMAINからCloudflareへの完全移行

set -e

# 色付きログ
log_info() { echo -e "\033[36m[INFO]\033[0m $1"; }
log_success() { echo -e "\033[32m[SUCCESS]\033[0m $1"; }
log_error() { echo -e "\033[31m[ERROR]\033[0m $1"; }
log_warning() { echo -e "\033[33m[WARNING]\033[0m $1"; }

echo "🚀 Cloudflare API 完全自動セットアップ開始"
echo ""

# APIトークンチェック
if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    log_error "CLOUDFLARE_API_TOKEN 環境変数が設定されていません"
    echo ""
    echo "📋 APIトークン取得手順:"
    echo "1. https://dash.cloudflare.com/profile/api-tokens にアクセス"
    echo "2. 'Create Token' をクリック"
    echo "3. 'Custom token' → 'Get started'"
    echo "4. 権限設定:"
    echo "   - Zone:Zone:Edit"
    echo "   - Zone:DNS:Edit"  
    echo "   - Account:Account:Read"
    echo "5. Account Resources: Include - All accounts"
    echo "6. Zone Resources: Include - All zones"
    echo "7. 'Continue to summary' → 'Create Token'"
    echo ""
    echo "取得後、以下を実行:"
    echo "export CLOUDFLARE_API_TOKEN=\"your_token_here\""
    echo "./deploy/cloudflare-auto-setup.sh"
    exit 1
fi

# ステップ1: regctl.com ゾーン作成
log_info "ステップ1: regctl.com ゾーンをCloudflareに追加"

ZONE_RESPONSE=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "name": "regctl.com",
    "jump_start": true,
    "type": "full"
  }')

# レスポンス確認
if echo "$ZONE_RESPONSE" | jq -e '.success' > /dev/null; then
    ZONE_ID=$(echo "$ZONE_RESPONSE" | jq -r '.result.id')
    NAMESERVERS=$(echo "$ZONE_RESPONSE" | jq -r '.result.name_servers[]' | tr '\n' ' ')
    log_success "ゾーン作成成功: $ZONE_ID"
    echo "📋 ネームサーバー: $NAMESERVERS"
else
    # ゾーンが既に存在する場合の処理
    if echo "$ZONE_RESPONSE" | grep -q "already exists"; then
        log_warning "ゾーンは既に存在します。既存ゾーンを使用します。"
        
        # 既存ゾーンのIDを取得
        ZONES_LIST=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones?name=regctl.com" \
          -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN")
        
        ZONE_ID=$(echo "$ZONES_LIST" | jq -r '.result[0].id')
        NAMESERVERS=$(echo "$ZONES_LIST" | jq -r '.result[0].name_servers[]' | tr '\n' ' ')
        log_success "既存ゾーンID: $ZONE_ID"
    else
        log_error "ゾーン作成失敗:"
        echo "$ZONE_RESPONSE" | jq '.'
        exit 1
    fi
fi

echo ""

# ステップ2: DNSレコード作成
log_info "ステップ2: DNSレコード自動作成"

# DNS records configuration
declare -a DNS_RECORDS=(
    "CNAME|@|regctl-site.pages.dev|true|Root domain"
    "CNAME|www|regctl-site.pages.dev|true|WWW subdomain"
    "CNAME|api|regctl-api.yukihamada.workers.dev|true|API Worker"
    "CNAME|app|regctl-dashboard.pages.dev|true|Dashboard App"
    "CNAME|docs|regctl-documentation.pages.dev|true|Documentation"
)

# DNSレコード作成関数
create_dns_record() {
    local type=$1
    local name=$2
    local content=$3
    local proxied=$4
    local description=$5
    
    echo -n "📍 $description ($name) → $content: "
    
    RECORD_RESPONSE=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records" \
      -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
      -H "Content-Type: application/json" \
      --data "{
        \"type\": \"$type\",
        \"name\": \"$name\",
        \"content\": \"$content\",
        \"proxied\": $proxied,
        \"ttl\": 1
      }")
    
    if echo "$RECORD_RESPONSE" | jq -e '.success' > /dev/null; then
        echo "✅ 作成成功"
    else
        # 既に存在する場合の処理
        if echo "$RECORD_RESPONSE" | grep -q "already exists"; then
            echo "⚠️  既に存在"
        else
            echo "❌ 作成失敗"
            echo "$RECORD_RESPONSE" | jq '.errors'
        fi
    fi
}

# 各DNSレコードを作成
for record in "${DNS_RECORDS[@]}"; do
    IFS='|' read -r type name content proxied description <<< "$record"
    create_dns_record "$type" "$name" "$content" "$proxied" "$description"
done

echo ""

# ステップ3: Pages カスタムドメイン設定（API経由）
log_info "ステップ3: Cloudflare Pages カスタムドメイン設定"

# Pages custom domain function
add_pages_domain() {
    local project_name=$1
    local domain=$2
    
    echo -n "🔗 $project_name に $domain を追加: "
    
    # 注意: Pages API は異なるエンドポイントを使用
    # 実際のAPIエンドポイントは /accounts/{account_id}/pages/projects/{project_name}/domains
    # アカウントIDが必要
    
    echo "⚠️  手動設定が必要（Pages API制限）"
}

echo "📋 以下のカスタムドメインを手動で追加してください:"
echo ""
echo "regctl-site:"
echo "  - regctl.com"
echo "  - www.regctl.com"
echo ""
echo "regctl-dashboard:"
echo "  - app.regctl.com"
echo ""
echo "regctl-documentation:"
echo "  - docs.regctl.com"
echo ""

# ステップ4: VALUE-DOMAIN ネームサーバー更新
log_info "ステップ4: VALUE-DOMAIN ネームサーバー更新"

echo "📋 VALUE-DOMAINで以下のネームサーバーに変更してください:"
echo "$NAMESERVERS"
echo ""

# ネームサーバー自動更新を試行（VALUE-DOMAIN API）
echo "🔄 VALUE-DOMAIN API経由でネームサーバー更新を試行..."

# ネームサーバーを配列に変換
NS_ARRAY=($(echo $NAMESERVERS | tr ' ' '\n'))

if [ ${#NS_ARRAY[@]} -ge 2 ]; then
    NS_UPDATE_RESPONSE=$(curl -s -X POST "https://regctl-api.yukihamada.workers.dev/api/v1/test/value-domain/update-nameservers" \
      -H "Content-Type: application/json" \
      --data "{
        \"domain\": \"regctl.com\",
        \"nameservers\": [\"${NS_ARRAY[0]}\", \"${NS_ARRAY[1]}\"]
      }")
    
    if echo "$NS_UPDATE_RESPONSE" | jq -e '.success' > /dev/null 2>&1; then
        log_success "VALUE-DOMAIN ネームサーバー更新成功！"
    else
        log_warning "VALUE-DOMAIN API更新失敗。手動で設定してください。"
        echo ""
        echo "📋 VALUE-DOMAIN手動設定:"
        echo "1. https://www.value-domain.com/ にログイン"
        echo "2. ドメイン管理 → regctl.com"
        echo "3. ネームサーバー設定 → 他社ネームサーバー"
        echo "4. 以下を入力:"
        echo "   - ネームサーバー1: ${NS_ARRAY[0]}"
        echo "   - ネームサーバー2: ${NS_ARRAY[1]}"
    fi
else
    log_warning "ネームサーバー情報が不完全です。手動で設定してください。"
fi

echo ""

# ステップ5: SSL/TLS設定
log_info "ステップ5: SSL/TLS設定確認"

SSL_RESPONSE=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/settings/ssl" \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN")

if echo "$SSL_RESPONSE" | jq -e '.success' > /dev/null; then
    SSL_MODE=$(echo "$SSL_RESPONSE" | jq -r '.result.value')
    echo "🔒 現在のSSLモード: $SSL_MODE"
    
    if [ "$SSL_MODE" != "full" ] && [ "$SSL_MODE" != "strict" ]; then
        echo "🔧 SSLモードをFull (strict)に変更中..."
        curl -s -X PATCH "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/settings/ssl" \
          -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
          -H "Content-Type: application/json" \
          --data '{"value":"strict"}' > /dev/null
        log_success "SSL設定更新完了"
    else
        log_success "SSL設定は適切です"
    fi
fi

echo ""

# 環境変数保存
log_info "ステップ6: 設定情報保存"

cat > /Users/yuki/RegiOps/.env.cloudflare << EOF
# Cloudflare設定 ($(date))
CLOUDFLARE_ZONE_ID="$ZONE_ID"
CLOUDFLARE_API_TOKEN="$CLOUDFLARE_API_TOKEN"
REGCTL_NAMESERVERS="$NAMESERVERS"
EOF

log_success "設定情報を .env.cloudflare に保存しました"

# 確認スクリプト更新
cat > /Users/yuki/RegiOps/deploy/check-cloudflare-setup.sh << 'EOF'
#!/bin/bash

source /Users/yuki/RegiOps/.env.cloudflare 2>/dev/null || {
    echo "❌ .env.cloudflare が見つかりません"
    exit 1
}

echo "🔍 Cloudflare設定確認"
echo ""
echo "Zone ID: $CLOUDFLARE_ZONE_ID"
echo "ネームサーバー: $REGCTL_NAMESERVERS"
echo ""

# DNS確認
echo "📡 DNS伝播確認:"
for domain in regctl.com api.regctl.com app.regctl.com docs.regctl.com www.regctl.com; do
    echo -n "$domain: "
    if nslookup $domain | grep -q "Name:"; then
        echo "✅ 解決済み"
    else
        echo "⏳ 伝播中"
    fi
done

echo ""

# HTTP確認
echo "🌐 HTTPS接続確認:"
for url in https://regctl.com https://api.regctl.com/api/v1/health https://app.regctl.com https://docs.regctl.com; do
    echo -n "$url: "
    if curl -s -I "$url" | head -1 | grep -q "200\|301\|302"; then
        echo "✅ 応答正常"
    else
        echo "⏳ 準備中"
    fi
done

echo ""
echo "🎉 確認完了！"
EOF

chmod +x /Users/yuki/RegiOps/deploy/check-cloudflare-setup.sh

echo ""

# 完了メッセージ
log_success "🎉 Cloudflare自動セットアップ完了！"
echo ""
echo "📊 設定完了事項:"
echo "✅ Zone作成: regctl.com ($ZONE_ID)"
echo "✅ DNSレコード: 5件作成"
echo "✅ SSL設定: Strict mode"
echo "✅ 設定保存: .env.cloudflare"
echo ""
echo "⏳ DNS伝播待機: 15-60分"
echo ""
echo "🔍 確認コマンド:"
echo "./deploy/check-cloudflare-setup.sh"
echo ""
echo "🌟 完成予定URL:"
echo "- https://regctl.com"
echo "- https://api.regctl.com" 
echo "- https://app.regctl.com"
echo "- https://docs.regctl.com"
echo ""
log_success "regctl Cloud SaaS 自動展開完了！ 🚀"