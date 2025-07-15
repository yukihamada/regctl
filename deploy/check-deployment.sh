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
