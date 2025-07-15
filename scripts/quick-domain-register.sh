#!/bin/bash

# 🚀 regctl クイック ドメイン登録
# curl -fsSL https://regctl.com/install.sh | bash && regctl domains register mobility360.jp

set -e

DOMAIN="$1"
if [ -z "$DOMAIN" ]; then
    echo "🚀 regctl クイック ドメイン登録"
    echo ""
    echo "使用法: $0 <domain>"
    echo "例: $0 mobility360.jp"
    echo ""
    echo "📦 1行インストール + 登録:"
    echo "   curl -fsSL https://regctl.com/quick.sh | bash -s mobility360.jp"
    exit 1
fi

# 色付きメッセージ
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }
blue() { printf "\033[34m%s\033[0m\n" "$*"; }

echo "🏁 $DOMAIN クイック登録"
echo ""

# regctl がインストール済みかチェック
if ! command -v regctl >/dev/null 2>&1; then
    yellow "📦 regctl をインストール中..."
    curl -fsSL https://regctl.com/install.sh | bash
    echo ""
fi

# ドメイン登録
blue "🔍 $DOMAIN の登録可能性を確認中..."

# VALUE-DOMAIN API で直接確認・登録
API_KEY="0LuCVBY2sy0LvKbTsDNB0NBa6zTfap0gSmHrANxT07WaCamHGZ9pydQwSVzn3ruWGFqRas99VBBfbsBz1FsDxgFFmA6f2Y4UbMU4"

# ドメイン分解
if [[ "$DOMAIN" == *.jp ]]; then
    SLD="${DOMAIN%.jp}"
    TLD="jp"
    REGISTRAR="JPRS"
    PRICE="¥2,035"
elif [[ "$DOMAIN" == *.com ]]; then
    SLD="${DOMAIN%.com}"
    TLD="com"
    REGISTRAR="GMO"
    PRICE="¥790"
else
    echo "❌ サポートされていないTLD: $DOMAIN"
    exit 1
fi

# 利用可能確認
AVAILABILITY=$(curl -s "https://api.value-domain.com/v1/domainsearch?domainnames=$DOMAIN" \
  -H "Authorization: Bearer $API_KEY" \
  | jq -r '.results[0].status' 2>/dev/null || echo "error")

if [[ "$AVAILABILITY" == "210 Domain name available" ]]; then
    green "✅ $DOMAIN 取得可能 ($PRICE)"
    echo ""
    
    read -p "登録を実行しますか？ (y/N): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        yellow "🔄 $DOMAIN を登録中..."
        
        # ドメイン登録
        RESPONSE=$(curl -s -X POST "https://api.value-domain.com/v1/domains" \
          -H "Authorization: Bearer $API_KEY" \
          -H "Content-Type: application/json" \
          -d "{
            \"registrar\": \"$REGISTRAR\",
            \"sld\": \"$SLD\",
            \"tld\": \"$TLD\",
            \"years\": 1,
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
        
        # 成功確認
        if echo "$RESPONSE" | jq -e '.domainid' >/dev/null 2>&1; then
            echo ""
            green "🎉 $DOMAIN 登録完了！"
            echo "   ドメインID: $(echo "$RESPONSE" | jq -r '.domainid')"
            echo "   有効期限: $(echo "$RESPONSE" | jq -r '.expirationdate')"
            echo "   費用: $PRICE"
            echo ""
            blue "🌐 次のステップ:"
            echo "   1. DNS設定: regctl dns list $DOMAIN"
            echo "   2. サブドメイン追加: regctl dns add $DOMAIN A www 192.168.1.1"
            echo "   3. SSL証明書: 自動取得済み"
        else
            echo ""
            echo "❌ 登録に失敗しました"
            echo "$RESPONSE" | jq .
        fi
    else
        echo "登録をキャンセルしました"
    fi
else
    echo "❌ $DOMAIN 取得不可: $AVAILABILITY"
    exit 1
fi