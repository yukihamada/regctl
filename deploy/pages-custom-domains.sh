#!/bin/bash

# Pages カスタムドメイン設定ガイド

echo "🔗 Cloudflare Pages カスタムドメイン設定"
echo ""
echo "ブラウザでCloudflare Pagesを開いています..."

# Pagesダッシュボードを開く
open "https://dash.cloudflare.com/pages"

echo ""
echo "📋 以下の設定を行ってください:"
echo ""
echo "1. regctl-site プロジェクト:"
echo "   - Custom domains → 'regctl.com' を追加"
echo "   - Custom domains → 'www.regctl.com' を追加"
echo ""
echo "2. regctl-dashboard プロジェクト:"
echo "   - Custom domains → 'app.regctl.com' を追加"
echo ""
echo "3. regctl-documentation プロジェクト:"
echo "   - Custom domains → 'docs.regctl.com' を追加"
echo ""
echo "⏱️ 設定時間: 約3分"
echo ""
echo "完了後、確認コマンドを実行:"
echo "./deploy/check-cloudflare-setup.sh"