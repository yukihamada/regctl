#!/bin/bash

# regctl.com Cloudflare Pages セットアップスクリプト
# 使用方法: ./deploy/pages-setup.sh

set -e

echo "🚀 regctl.com Cloudflare Pages セットアップ開始"

# 色付きログ関数
log_info() {
    echo -e "\033[36m[INFO]\033[0m $1"
}

log_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

log_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

# 1. メインサイト (regctl.com) の作成
log_info "1. メインサイト用 Pages プロジェクト作成"

if ! command -v wrangler &> /dev/null; then
    log_error "wrangler CLI が見つかりません。インストールしてください: npm install -g wrangler"
    exit 1
fi

# site ディレクトリへ移動
cd site

# pages プロジェクト作成
log_info "regctl-site プロジェクト作成中..."
wrangler pages project create regctl-site --compatibility-date=2024-01-01 || log_error "プロジェクト作成失敗"

# 初期デプロイ
log_info "初期デプロイ実行中..."
wrangler pages deploy . --project-name=regctl-site --compatibility-date=2024-01-01

log_success "regctl-site デプロイ完了"

# 2. アプリ (app.regctl.com) の作成
log_info "2. アプリ用 Pages プロジェクト作成"

cd app

# React アプリをビルド
log_info "React アプリビルド中..."
npm install
npm run build

# pages プロジェクト作成
log_info "regctl-app プロジェクト作成中..."
wrangler pages project create regctl-app --compatibility-date=2024-01-01 || log_error "プロジェクト作成失敗"

# デプロイ
log_info "regctl-app デプロイ実行中..."
wrangler pages deploy dist --project-name=regctl-app --compatibility-date=2024-01-01

log_success "regctl-app デプロイ完了"

cd ..

# 3. ドキュメント (docs.regctl.com) の作成
log_info "3. ドキュメント用 Pages プロジェクト作成"

# docs プロジェクト作成
log_info "regctl-docs プロジェクト作成中..."
wrangler pages project create regctl-docs --compatibility-date=2024-01-01 || log_error "プロジェクト作成失敗"

# docsディレクトリデプロイ
log_info "regctl-docs デプロイ実行中..."
wrangler pages deploy docs --project-name=regctl-docs --compatibility-date=2024-01-01

log_success "regctl-docs デプロイ完了"

# 4. カスタムドメイン設定
log_info "4. カスタムドメイン設定"

echo "以下のカスタムドメインを手動で設定してください："
echo ""
echo "📋 Cloudflare ダッシュボード > Pages > 各プロジェクト > Custom domains"
echo ""
echo "regctl-site:"
echo "  - regctl.com"
echo "  - www.regctl.com"
echo ""
echo "regctl-app:"
echo "  - app.regctl.com"
echo ""
echo "regctl-docs:"
echo "  - docs.regctl.com"
echo ""

# 5. DNS設定コマンド表示
log_info "5. DNS設定コマンド"

echo "以下のDNS設定を実行してください："
echo ""
echo "# 環境変数設定（実際の値に置き換えてください）"
echo "export CLOUDFLARE_API_TOKEN=\"your_token\""
echo "export ZONE_ID=\"your_zone_id\""
echo ""
echo "# Root domain"
echo "curl -X POST \"https://api.cloudflare.com/client/v4/zones/\${ZONE_ID}/dns_records\" \\"
echo "  -H \"Authorization: Bearer \${CLOUDFLARE_API_TOKEN}\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  --data '{\"type\":\"CNAME\",\"name\":\"@\",\"content\":\"regctl-site.pages.dev\",\"proxied\":true}'"
echo ""
echo "# API subdomain"
echo "curl -X POST \"https://api.cloudflare.com/client/v4/zones/\${ZONE_ID}/dns_records\" \\"
echo "  -H \"Authorization: Bearer \${CLOUDFLARE_API_TOKEN}\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  --data '{\"type\":\"CNAME\",\"name\":\"api\",\"content\":\"regctl-api.yukihamada.workers.dev\",\"proxied\":true}'"
echo ""
echo "# App subdomain"
echo "curl -X POST \"https://api.cloudflare.com/client/v4/zones/\${ZONE_ID}/dns_records\" \\"
echo "  -H \"Authorization: Bearer \${CLOUDFLARE_API_TOKEN}\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  --data '{\"type\":\"CNAME\",\"name\":\"app\",\"content\":\"regctl-app.pages.dev\",\"proxied\":true}'"
echo ""
echo "# Docs subdomain"
echo "curl -X POST \"https://api.cloudflare.com/client/v4/zones/\${ZONE_ID}/dns_records\" \\"
echo "  -H \"Authorization: Bearer \${CLOUDFLARE_API_TOKEN}\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  --data '{\"type\":\"CNAME\",\"name\":\"docs\",\"content\":\"regctl-docs.pages.dev\",\"proxied\":true}'"

log_success "🎉 Cloudflare Pages セットアップ完了！"
echo ""
echo "次のステップ："
echo "1. Cloudflareダッシュボードでカスタムドメインを設定"
echo "2. 上記のDNS設定コマンドを実行"
echo "3. 動作確認とテスト"