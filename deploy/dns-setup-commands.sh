#!/bin/bash

# regctl.com DNS設定コマンド
# 使用前に環境変数を設定してください:
# export CLOUDFLARE_API_TOKEN="your_token"
# export ZONE_ID="your_zone_id"

echo "🌐 regctl.com DNS設定コマンド"
echo ""
echo "実際のデプロイされたPages URLs:"
echo "- Main site: https://regctl-site.pages.dev"
echo "- Dashboard: https://regctl-dashboard.pages.dev"  
echo "- Documentation: https://regctl-documentation.pages.dev"
echo "- API Worker: regctl-api.yukihamada.workers.dev"
echo ""

# 環境変数チェック
if [ -z "$CLOUDFLARE_API_TOKEN" ] || [ -z "$ZONE_ID" ]; then
    echo "❌ 環境変数を設定してください:"
    echo "export CLOUDFLARE_API_TOKEN=\"your_token\""
    echo "export ZONE_ID=\"your_zone_id\""
    echo ""
    echo "トークンは以下で取得: https://dash.cloudflare.com/profile/api-tokens"
    echo "ゾーンIDは regctl.com のダッシュボード右サイドバーで確認"
    echo ""
    echo "設定後、このスクリプトを再実行してください。"
    exit 1
fi

echo "🚀 DNS レコードを作成中..."

# Root domain
echo "📍 Root domain (regctl.com) → regctl-site.pages.dev"
curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"@","content":"regctl-site.pages.dev","proxied":true}' | jq '.success'

# WWW subdomain
echo "📍 WWW subdomain (www.regctl.com) → regctl-site.pages.dev"
curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"www","content":"regctl-site.pages.dev","proxied":true}' | jq '.success'

# API subdomain (Worker)
echo "📍 API subdomain (api.regctl.com) → regctl-api.yukihamada.workers.dev"
curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"api","content":"regctl-api.yukihamada.workers.dev","proxied":true}' | jq '.success'

# App subdomain
echo "📍 App subdomain (app.regctl.com) → regctl-dashboard.pages.dev"
curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"app","content":"regctl-dashboard.pages.dev","proxied":true}' | jq '.success'

# Docs subdomain
echo "📍 Docs subdomain (docs.regctl.com) → regctl-documentation.pages.dev"
curl -s -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"docs","content":"regctl-documentation.pages.dev","proxied":true}' | jq '.success'

echo ""
echo "✅ DNS レコード作成完了！"
echo ""
echo "次のステップ:"
echo "1. CloudflareダッシュボードでSSL設定確認"
echo "2. 各Pagesプロジェクトにカスタムドメイン追加"
echo "3. DNS伝播待機 (15-60分)"
echo "4. 動作確認"
echo ""
echo "カスタムドメイン設定:"
echo "- regctl-site: regctl.com, www.regctl.com"
echo "- regctl-dashboard: app.regctl.com"
echo "- regctl-documentation: docs.regctl.com"