#!/bin/bash

# 🚀 regctl 簡単テスト

echo "🔧 regctl 動作確認"
echo ""

# 1. CLI テスト
echo "1️⃣ CLI確認"
if ./regctl --help >/dev/null 2>&1; then
    echo "✅ CLI動作OK"
else
    echo "❌ CLI未ビルド → go build -o regctl"
fi

# 2. API テスト
echo ""
echo "2️⃣ API確認"
if curl -s https://api.regctl.com/api/v1/health >/dev/null 2>&1; then
    echo "✅ API稼働中"
else
    echo "❌ API接続エラー"
fi

# 3. サイト確認
echo ""
echo "3️⃣ サイト確認"
if curl -s https://regctl.com >/dev/null 2>&1; then
    echo "✅ メインサイト応答OK"
else
    echo "❌ サイト接続エラー"
fi

echo ""
echo "🎯 次のステップ:"
echo "   ./regctl login              # 初回認証"
echo "   ./regctl domains list       # ドメイン確認"
echo "   ./regctl dns list regctl.com # DNS確認"
echo ""
echo "📋 サイト: https://regctl.com"