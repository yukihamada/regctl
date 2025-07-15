#!/bin/bash

# 🚀 Mobility360.jp ドメイン登録スクリプト

echo "🏁 Mobility360.jp ドメイン登録"
echo ""

# 1. 利用可能確認
echo "1️⃣ 利用可能確認"
AVAILABILITY=$(curl -s "https://api.value-domain.com/v1/domainsearch?domainnames=mobility360.jp" \
  -H "Authorization: Bearer 0LuCVBY2sy0LvKbTsDNB0NBa6zTfap0gSmHrANxT07WaCamHGZ9pydQwSVzn3ruWGFqRas99VBBfbsBz1FsDxgFFmA6f2Y4UbMU4" \
  | jq -r '.results[0].status')

if [[ "$AVAILABILITY" == "210 Domain name available" ]]; then
    echo "✅ Mobility360.jp 取得可能"
else
    echo "❌ Mobility360.jp 取得不可: $AVAILABILITY"
    exit 1
fi

echo ""
echo "2️⃣ 登録情報"
echo "   ドメイン: mobility360.jp"
echo "   価格: ¥2,035/年"
echo "   登録者: Yuki Hamada (RegiOps)"
echo "   住所: 1-1-1 Chiyoda, Tokyo 100-0001"
echo ""

# 3. 登録実行
echo "3️⃣ 登録実行"
read -p "登録を実行しますか？ (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🔄 登録中..."
    
    # VALUE-DOMAIN API でドメイン登録
    RESPONSE=$(curl -s -X POST "https://api.value-domain.com/v1/domains" \
      -H "Authorization: Bearer 0LuCVBY2sy0LvKbTsDNB0NBa6zTfap0gSmHrANxT07WaCamHGZ9pydQwSVzn3ruWGFqRas99VBBfbsBz1FsDxgFFmA6f2Y4UbMU4" \
      -H "Content-Type: application/json" \
      -d '{
        "registrar": "JPRS",
        "sld": "mobility360",
        "tld": "jp",
        "years": 1,
        "whois_proxy": 0,
        "ns": ["lou.ns.cloudflare.com", "violet.ns.cloudflare.com"],
        "contact": {
          "registrant": {
            "firstname": "Yuki",
            "lastname": "Hamada",
            "organization": "RegiOps",
            "country": "JP",
            "postalcode": "100-0001",
            "state": "Tokyo",
            "city": "Tokyo",
            "address1": "1-1-1 Chiyoda",
            "address2": "",
            "email": "yuki@hamada.dev",
            "phone": "+81.9012345678",
            "fax": "+81.9012345678"
          },
          "admin": {
            "firstname": "Yuki",
            "lastname": "Hamada",
            "organization": "RegiOps",
            "country": "JP",
            "postalcode": "100-0001",
            "state": "Tokyo",
            "city": "Tokyo",
            "address1": "1-1-1 Chiyoda",
            "address2": "",
            "email": "yuki@hamada.dev",
            "phone": "+81.9012345678",
            "fax": "+81.9012345678"
          }
        }
      }')
    
    echo "📋 登録結果:"
    echo "$RESPONSE" | jq .
    
    # 成功確認
    if echo "$RESPONSE" | jq -e '.domainid' >/dev/null 2>&1; then
        echo ""
        echo "🎉 Mobility360.jp 登録完了！"
        echo "   ドメインID: $(echo "$RESPONSE" | jq -r '.domainid')"
        echo "   有効期限: $(echo "$RESPONSE" | jq -r '.expirationdate')"
        echo "   ネームサーバー: Cloudflare"
        echo ""
        echo "🌐 次のステップ:"
        echo "   1. Cloudflare にゾーン追加"
        echo "   2. DNS設定"
        echo "   3. サービスデプロイ"
    else
        echo ""
        echo "❌ 登録に失敗しました"
        echo "エラー詳細を確認してください"
    fi
else
    echo "登録をキャンセルしました"
fi