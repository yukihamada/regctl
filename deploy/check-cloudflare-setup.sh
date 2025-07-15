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
