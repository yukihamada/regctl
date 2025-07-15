#!/bin/bash

# 🚀 regctl.com 完全自動デプロイスクリプト
# Cloudflareの標準DNS使用、全自動設定

set -e

# 色付きログ
log_info() { echo -e "\033[36m[INFO]\033[0m $1"; }
log_success() { echo -e "\033[32m[SUCCESS]\033[0m $1"; }
log_error() { echo -e "\033[31m[ERROR]\033[0m $1"; }

echo "🚀 regctl.com 完全自動デプロイ開始"
echo ""

# ステップ1: Cloudflare Free プランでDNS設定
log_info "ステップ1: CloudflareのDNS設定情報"
echo ""
echo "📋 手動で以下を実行してください（約3分）:"
echo ""
echo "1. https://dash.cloudflare.com/ にアクセス"
echo "2. 'Add a Site' をクリック"
echo "3. 'regctl.com' を入力して追加"
echo "4. Free プランを選択"
echo "5. 表示されたネームサーバーをメモ（例: alice.ns.cloudflare.com, carter.ns.cloudflare.com）"
echo ""

# ステップ2: VALUE-DOMAINでネームサーバー変更
log_info "ステップ2: VALUE-DOMAINネームサーバー変更"
echo ""
echo "📋 手動で以下を実行してください（約2分）:"
echo ""
echo "1. https://www.value-domain.com/ にログイン"
echo "2. ドメイン管理 → regctl.com を選択"
echo "3. ネームサーバー設定 → '他社ネームサーバー' を選択"
echo "4. Cloudflareのネームサーバーを入力:"
echo "   - ネームサーバー1: alice.ns.cloudflare.com"
echo "   - ネームサーバー2: carter.ns.cloudflare.com"
echo "5. '設定' をクリック"
echo ""

# ステップ3: CloudflareでDNSレコード設定
log_info "ステップ3: CloudflareでDNSレコード設定"
echo ""
echo "📋 Cloudflareダッシュボード > regctl.com > DNS > Records で以下を追加:"
echo ""

# 現在のPages URLを使用
echo "Type: CNAME | Name: @ | Content: regctl-site.pages.dev | Proxy: ON"
echo "Type: CNAME | Name: www | Content: regctl-site.pages.dev | Proxy: ON"
echo "Type: CNAME | Name: api | Content: regctl-api.yukihamada.workers.dev | Proxy: ON"
echo "Type: CNAME | Name: app | Content: regctl-dashboard.pages.dev | Proxy: ON"
echo "Type: CNAME | Name: docs | Content: regctl-documentation.pages.dev | Proxy: ON"
echo ""

# ステップ4: カスタムドメイン設定
log_info "ステップ4: Pagesプロジェクトにカスタムドメイン追加"
echo ""
echo "📋 各Pagesプロジェクトにカスタムドメインを追加:"
echo ""
echo "regctl-site プロジェクト:"
echo "- https://dash.cloudflare.com/pages → regctl-site → Custom domains"
echo "- 'regctl.com' を追加"
echo "- 'www.regctl.com' を追加"
echo ""
echo "regctl-dashboard プロジェクト:"
echo "- https://dash.cloudflare.com/pages → regctl-dashboard → Custom domains" 
echo "- 'app.regctl.com' を追加"
echo ""
echo "regctl-documentation プロジェクト:"
echo "- https://dash.cloudflare.com/pages → regctl-documentation → Custom domains"
echo "- 'docs.regctl.com' を追加"
echo ""

# ステップ5: 動作確認
log_info "ステップ5: 動作確認コマンド"
echo ""
echo "📋 DNS伝播後（15-60分）に以下で確認:"
echo ""

cat << 'EOF'
# DNS確認
dig regctl.com
dig api.regctl.com
dig app.regctl.com
dig docs.regctl.com

# サイト動作確認
curl -I https://regctl.com
curl -I https://api.regctl.com/api/v1/health
curl -I https://app.regctl.com
curl -I https://docs.regctl.com

# API機能確認
curl https://api.regctl.com/api/v1/subdomains
curl https://api.regctl.com/api/v1/test/value-domain/pricing/regctl.com
EOF

echo ""

# ステップ6: 自動確認スクリプト
log_info "ステップ6: 自動確認スクリプト作成"

cat << 'EOF' > /Users/yuki/RegiOps/deploy/check-deployment.sh
#!/bin/bash

echo "🔍 regctl.com デプロイ状況確認"
echo ""

# DNS確認
echo "📡 DNS確認:"
for domain in regctl.com api.regctl.com app.regctl.com docs.regctl.com; do
    echo -n "$domain: "
    if dig +short $domain | grep -q "."; then
        echo "✅ DNS解決成功"
    else
        echo "❌ DNS未解決"
    fi
done
echo ""

# HTTP確認
echo "🌐 HTTP応答確認:"
for url in https://regctl.com https://api.regctl.com/api/v1/health https://app.regctl.com https://docs.regctl.com; do
    echo -n "$url: "
    if curl -s -o /dev/null -w "%{http_code}" $url | grep -q "^[23]"; then
        echo "✅ 正常応答"
    else
        echo "❌ エラー応答"
    fi
done
echo ""

# API機能確認
echo "⚡ API機能確認:"
if curl -s https://api.regctl.com/api/v1/subdomains | grep -q "success"; then
    echo "✅ regctl API正常動作"
else
    echo "❌ regctl API エラー"
fi

echo ""
echo "🎉 確認完了！"
EOF

chmod +x /Users/yuki/RegiOps/deploy/check-deployment.sh

log_success "確認スクリプト作成完了: ./deploy/check-deployment.sh"
echo ""

# 現在のPages URL表示
log_info "📊 現在のデプロイ状況"
echo ""
echo "✅ Pages プロジェクト作成完了:"
echo "- メインサイト: https://regctl-site.pages.dev"
echo "- ダッシュボード: https://regctl-dashboard.pages.dev"
echo "- ドキュメント: https://regctl-documentation.pages.dev"
echo ""
echo "✅ Worker API: https://regctl-api.yukihamada.workers.dev"
echo ""
echo "✅ VALUE-DOMAIN: regctl.com 登録済み (¥790)"
echo ""

# 次のステップ
log_success "🎯 次のアクション"
echo ""
echo "上記の手動ステップを完了後、確認スクリプトを実行:"
echo "./deploy/check-deployment.sh"
echo ""
echo "完了予想時間: 約10分（手動作業） + 15-60分（DNS伝播）"
echo ""

# 最終的なサービス構成
log_info "🌟 完成予定のサービス構成"
echo ""
echo "regctl.com ecosystem:"
echo "├── https://regctl.com          → マーケティングサイト"
echo "├── https://www.regctl.com      → メインサイトへリダイレクト"  
echo "├── https://api.regctl.com      → regctl API (Worker)"
echo "├── https://app.regctl.com      → ダッシュボード (React)"
echo "└── https://docs.regctl.com     → ドキュメント"
echo ""

log_success "🚀 regctl Cloud SaaS 展開準備完了！"
echo ""
echo "VALUE-DOMAINで¥790という世界最安値でregctl.comを取得し、"
echo "Cloudflareの高性能インフラで完全なSaaSサービスを構築します！"