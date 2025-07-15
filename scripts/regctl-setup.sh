#!/bin/bash

# 🚀 regctl セットアップ & 動作確認ツール

echo "🔧 regctl Cloud セットアップ確認"
echo ""

# 1. 基本確認
echo "📋 1. 基本設定確認"
echo "   ドメイン: regctl.com"
echo "   API: api.regctl.com"
echo "   ダッシュボード: app.regctl.com"
echo "   ドキュメント: docs.regctl.com"
echo ""

# 2. DNS確認
echo "📡 2. DNS解決確認"
for domain in regctl.com api.regctl.com app.regctl.com docs.regctl.com; do
    if nslookup $domain >/dev/null 2>&1; then
        echo "   ✅ $domain"
    else
        echo "   ❌ $domain"
    fi
done
echo ""

# 3. HTTP確認
echo "🌐 3. サービス応答確認"
for url in "https://regctl.com" "https://api.regctl.com/api/v1/health"; do
    status=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
    if [[ $status == "200" ]]; then
        echo "   ✅ $url"
    else
        echo "   ⏳ $url (HTTP $status)"
    fi
done
echo ""

# 4. CLI確認
echo "🛠️ 4. CLI動作確認"
if command -v regctl >/dev/null 2>&1; then
    echo "   ✅ regctl CLI インストール済み"
    echo "   バージョン: $(regctl --version 2>/dev/null || echo "不明")"
else
    echo "   ❌ regctl CLI が見つかりません"
    echo "   インストール: make build"
fi
echo ""

# 5. 設定ファイル確認
echo "⚙️ 5. 設定ファイル確認"
config_file="$HOME/.config/regctl/config.yaml"
if [[ -f "$config_file" ]]; then
    echo "   ✅ 設定ファイル: $config_file"
else
    echo "   ❌ 設定ファイルなし"
    echo "   作成: regctl login"
fi
echo ""

echo "🎯 次のステップ:"
echo "   1. regctl login    # 初回認証"
echo "   2. regctl domains list    # ドメイン一覧"
echo "   3. regctl dns list regctl.com    # DNS確認"
echo ""