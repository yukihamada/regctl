#!/bin/bash

# 🚀 regctl クイック インストール & ドメイン登録
# curl -fsSL https://regctl.com/quick.sh | bash -s mobility360.jp

set -e

DOMAIN="$1"

# 色付きメッセージ
red() { printf "\033[31m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }
blue() { printf "\033[34m%s\033[0m\n" "$*"; }
bold() { printf "\033[1m%s\033[0m\n" "$*"; }

# バナー
echo ""
bold "🚀 regctl クイック セットアップ"
echo ""

if [ -z "$DOMAIN" ]; then
    blue "使用法:"
    echo "   curl -fsSL https://regctl.com/quick.sh | bash -s <domain>"
    echo ""
    blue "例:"
    echo "   curl -fsSL https://regctl.com/quick.sh | bash -s mobility360.jp"
    echo "   curl -fsSL https://regctl.com/quick.sh | bash -s example.com"
    echo ""
    blue "サポート対象:"
    echo "   ✅ .jp ドメイン (¥2,035/年)"
    echo "   ✅ .com ドメイン (¥790/年)"
    echo ""
    exit 1
fi

# 1. regctl インストール
echo "📦 Step 1: regctl CLI インストール"
if ! command -v regctl >/dev/null 2>&1; then
    curl -fsSL https://regctl.com/install.sh | bash
else
    green "✅ regctl 既にインストール済み"
fi

echo ""

# 2. ドメイン登録
echo "🏁 Step 2: $DOMAIN ドメイン登録"

# VALUE-DOMAIN API設定
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
    red "❌ サポートされていないTLD: $DOMAIN"
    red "   サポート対象: .jp, .com"
    exit 1
fi

# 利用可能確認
blue "🔍 $DOMAIN の利用可能性を確認中..."
AVAILABILITY=$(curl -s "https://api.value-domain.com/v1/domainsearch?domainnames=$DOMAIN" \
  -H "Authorization: Bearer $API_KEY" \
  | jq -r '.results[0].status' 2>/dev/null || echo "error")

if [[ "$AVAILABILITY" == "210 Domain name available" ]]; then
    green "✅ $DOMAIN 取得可能 ($PRICE)"
    echo ""
    
    yellow "💰 登録費用: $PRICE"
    yellow "🏢 レジストラー: $REGISTRAR"
    yellow "🌐 ネームサーバー: Cloudflare"
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
            echo ""
            bold "📋 登録情報:"
            echo "   ドメインID: $(echo "$RESPONSE" | jq -r '.domainid')"
            echo "   有効期限: $(echo "$RESPONSE" | jq -r '.expirationdate')"
            echo "   費用: $PRICE"
            echo ""
            bold "🌐 次のステップ:"
            echo "   1. DNS確認: regctl dns list $DOMAIN"
            echo "   2. レコード追加: regctl dns add $DOMAIN A www 192.168.1.1"
            echo "   3. SSL証明書: 自動取得済み"
            echo ""
            blue "📖 詳細: https://docs.regctl.com"
        else
            echo ""
            red "❌ 登録に失敗しました"
            echo "$RESPONSE" | jq .
        fi
    else
        echo ""
        yellow "登録をキャンセルしました"
        echo ""
        blue "💡 他のドメインを確認:"
        echo "   curl -fsSL https://regctl.com/quick.sh | bash -s example.com"
    fi
else
    echo ""
    red "❌ $DOMAIN 取得不可: $AVAILABILITY"
    echo ""
    blue "💡 他のドメインを試してください:"
    echo "   curl -fsSL https://regctl.com/quick.sh | bash -s ${DOMAIN}2"
fi