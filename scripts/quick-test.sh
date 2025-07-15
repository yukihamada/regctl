#!/bin/bash

# 🚀 regctl クイックテスト

echo "⚡ regctl 動作テスト"

# API テスト
echo "📡 API接続テスト..."
if curl -s https://api.regctl.com/api/v1/health >/dev/null 2>&1; then
    echo "✅ API稼働中"
else
    echo "❌ API接続エラー"
fi

# CLI テスト  
echo "🛠️ CLI テスト..."
if ./cmd/regctl/regctl --help >/dev/null 2>&1; then
    echo "✅ CLI動作確認"
    echo "💡 使用例:"
    echo "   ./cmd/regctl/regctl login"
    echo "   ./cmd/regctl/regctl domains list"
else
    echo "❌ CLI未ビルド"
    echo "💡 ビルド: make build"
fi

echo ""
echo "🌐 サービス確認: https://regctl.com"