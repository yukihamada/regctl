#!/bin/bash

# 🚀 regctl風 ドメイン登録コマンド

DOMAIN="$1"
REGISTRAR="value-domain"
YEARS="1"

if [ -z "$DOMAIN" ]; then
    echo "使用法: $0 <domain>"
    echo "例: $0 mobility360.jp"
    exit 1
fi

echo "🔧 regctl domains register $DOMAIN --registrar=$REGISTRAR --years=$YEARS"
echo ""

# VALUE-DOMAIN API Key
API_KEY="0LuCVBY2sy0LvKbTsDNB0NBa6zTfap0gSmHrANxT07WaCamHGZ9pydQwSVzn3ruWGFqRas99VBBfbsBz1FsDxgFFmA6f2Y4UbMU4"

# ドメインをSLDとTLDに分解
if [[ "$DOMAIN" == *.jp ]]; then
    SLD="${DOMAIN%.jp}"
    TLD="jp"
    REGISTRAR_API="JPRS"
elif [[ "$DOMAIN" == *.com ]]; then
    SLD="${DOMAIN%.com}"
    TLD="com"
    REGISTRAR_API="GMO"
else
    echo "❌ サポートされていないTLD: $DOMAIN"
    exit 1
fi

echo "📋 登録情報:"
echo "   ドメイン: $DOMAIN"
echo "   SLD: $SLD"
echo "   TLD: $TLD"
echo "   レジストラー: $REGISTRAR_API"
echo "   年数: $YEARS"
echo ""

# 利用可能確認
echo "🔍 利用可能確認中..."
AVAILABILITY=$(curl -s "https://api.value-domain.com/v1/domainsearch?domainnames=$DOMAIN" \
  -H "Authorization: Bearer $API_KEY" \
  | jq -r '.results[0].status')

if [[ "$AVAILABILITY" == "210 Domain name available" ]]; then
    echo "✅ $DOMAIN 取得可能"
else
    echo "❌ $DOMAIN 取得不可: $AVAILABILITY"
    exit 1
fi

echo ""
read -p "登録を実行しますか？ (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🔄 登録中..."
    
    # ドメイン登録
    RESPONSE=$(curl -s -X POST "https://api.value-domain.com/v1/domains" \
      -H "Authorization: Bearer $API_KEY" \
      -H "Content-Type: application/json" \
      -d "{
        \"registrar\": \"$REGISTRAR_API\",
        \"sld\": \"$SLD\",
        \"tld\": \"$TLD\",
        \"years\": $YEARS,
        \"whois_proxy\": 0,
        \"ns\": [\"lou.ns.cloudflare.com\", \"violet.ns.cloudflare.com\"],
        \"contact\": {
          \"registrant\": {
            \"firstname\": \"Yuki\",
            \"lastname\": \"Hamada\",
            \"organization\": \"RegiOps\",
            \"country\": \"JP\",
            \"postalcode\": \"100-0001\",
            \"state\": \"Tokyo\",
            \"city\": \"Tokyo\",
            \"address1\": \"1-1-1 Chiyoda\",
            \"address2\": \"\",
            \"email\": \"yuki@hamada.dev\",
            \"phone\": \"+81.9012345678\",
            \"fax\": \"+81.9012345678\"
          },
          \"admin\": {
            \"firstname\": \"Yuki\",
            \"lastname\": \"Hamada\",
            \"organization\": \"RegiOps\",
            \"country\": \"JP\",
            \"postalcode\": \"100-0001\",
            \"state\": \"Tokyo\",
            \"city\": \"Tokyo\",
            \"address1\": \"1-1-1 Chiyoda\",
            \"address2\": \"\",
            \"email\": \"yuki@hamada.dev\",
            \"phone\": \"+81.9012345678\",
            \"fax\": \"+81.9012345678\"
          }
        }
      }")
    
    echo ""
    echo "📋 登録結果:"
    echo "$RESPONSE" | jq .
    
    # 成功確認
    if echo "$RESPONSE" | jq -e '.domainid' >/dev/null 2>&1; then
        echo ""
        echo "🎉 $DOMAIN 登録完了！"
        echo "   ドメインID: $(echo "$RESPONSE" | jq -r '.domainid')"
        echo "   有効期限: $(echo "$RESPONSE" | jq -r '.expirationdate')"
        echo ""
        echo "🌐 次のステップ:"
        echo "   1. Cloudflareにゾーン追加"
        echo "   2. DNS設定"
        echo "   3. サービスデプロイ"
    else
        echo ""
        echo "❌ 登録に失敗しました"
    fi
else
    echo "登録をキャンセルしました"
fi