#!/bin/bash

# 🔍 API デバッグスクリプト

echo "🔍 API 接続デバッグ"
echo ""

# 1. 基本接続テスト
echo "1️⃣ 基本接続テスト"
echo "   主サイト: https://regctl.com"
curl -s -I https://regctl.com | head -3
echo ""

echo "   API Health: https://api.regctl.com/api/v1/health"
curl -s -I https://api.regctl.com/api/v1/health | head -3
echo ""

# 2. DNS確認
echo "2️⃣ DNS確認"
echo "   regctl.com:"
nslookup regctl.com | grep "Address" | tail -1
echo "   api.regctl.com:"
nslookup api.regctl.com | grep "Address" | tail -1
echo ""

# 3. Workers 直接テスト
echo "3️⃣ Workers 直接確認"
echo "   Workers URL確認..."
curl -s https://regctl.com | jq -r '.name' 2>/dev/null || echo "   メインサイトからWorkers応答"
echo ""

# 4. 解決策
echo "💡 解決策:"
echo "   - Pages カスタムドメイン設定完了まで待機"
echo "   - または Workers 直接エンドポイントを使用"
echo "   - ダッシュボード: https://dash.cloudflare.com/pages"
echo ""