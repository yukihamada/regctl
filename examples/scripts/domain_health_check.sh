#!/bin/bash
# domain_health_check.sh - ドメインの健全性をチェック

set -e

# カラー定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# チェック関数
check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# ドメインチェック
check_domain() {
    local domain=$1
    echo ""
    echo "Checking domain: $domain"
    echo "================================"
    
    # ドメイン情報取得
    domain_info=$(regctl domains info $domain --output json)
    
    # ステータスチェック
    status=$(echo "$domain_info" | jq -r '.status')
    if [ "$status" = "active" ]; then
        check_pass "Domain status: $status"
    else
        check_fail "Domain status: $status"
    fi
    
    # 有効期限チェック
    expires_at=$(echo "$domain_info" | jq -r '.expires_at')
    expires_timestamp=$(date -j -f "%Y-%m-%d" "$expires_at" +%s 2>/dev/null || date -d "$expires_at" +%s)
    current_timestamp=$(date +%s)
    days_until_expiry=$(( (expires_timestamp - current_timestamp) / 86400 ))
    
    if [ $days_until_expiry -gt 60 ]; then
        check_pass "Expiry: $expires_at ($days_until_expiry days)"
    elif [ $days_until_expiry -gt 30 ]; then
        check_warn "Expiry: $expires_at ($days_until_expiry days)"
    else
        check_fail "Expiry: $expires_at ($days_until_expiry days) - URGENT!"
    fi
    
    # 自動更新チェック
    auto_renew=$(echo "$domain_info" | jq -r '.auto_renew')
    if [ "$auto_renew" = "true" ]; then
        check_pass "Auto-renewal: Enabled"
    else
        check_warn "Auto-renewal: Disabled"
    fi
    
    # DNSチェック
    echo ""
    echo "DNS Configuration:"
    
    # Aレコード存在チェック
    a_records=$(regctl dns list $domain --type A --output json | jq -r '.records | length')
    if [ $a_records -gt 0 ]; then
        check_pass "A records: $a_records found"
    else
        check_fail "A records: None found"
    fi
    
    # MXレコード存在チェック
    mx_records=$(regctl dns list $domain --type MX --output json | jq -r '.records | length')
    if [ $mx_records -gt 0 ]; then
        check_pass "MX records: $mx_records found"
    else
        check_warn "MX records: None found (mail may not work)"
    fi
    
    # ネームサーバー応答チェック
    echo ""
    echo "Nameserver Check:"
    nameservers=$(echo "$domain_info" | jq -r '.nameservers[]')
    for ns in $nameservers; do
        if dig @$ns $domain A +short &>/dev/null; then
            check_pass "Nameserver $ns: Responding"
        else
            check_fail "Nameserver $ns: Not responding"
        fi
    done
    
    # DNSSEC チェック（オプション）
    echo ""
    echo "Security:"
    if dig +dnssec $domain | grep -q "ad;" 2>/dev/null; then
        check_pass "DNSSEC: Enabled"
    else
        check_warn "DNSSEC: Not enabled"
    fi
}

# メイン処理
echo "Domain Health Check Report"
echo "========================="
echo "Date: $(date)"

if [ $# -eq 0 ]; then
    # 引数なしの場合、すべてのドメインをチェック
    domains=$(regctl domains list --output json | jq -r '.domains[].name')
    for domain in $domains; do
        check_domain "$domain"
    done
else
    # 特定のドメインをチェック
    check_domain "$1"
fi

echo ""
echo "Check completed."