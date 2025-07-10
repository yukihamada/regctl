#!/bin/bash
# batch_update_dns.sh - 複数ドメインのDNSレコードを一括更新

set -e

# 設定
OLD_IP="192.0.2.1"
NEW_IP="192.0.2.100"
LOG_FILE="dns_update_$(date +%Y%m%d_%H%M%S).log"

echo "DNS Batch Update Script"
echo "======================"
echo "Updating IP from $OLD_IP to $NEW_IP"
echo "Log file: $LOG_FILE"
echo ""

# ログ関数
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# メイン処理
log "Starting DNS batch update"

# すべてのドメインを取得
domains=$(regctl domains list --output json | jq -r '.domains[].name')

for domain in $domains; do
    log "Processing domain: $domain"
    
    # 対象のAレコードを検索
    records=$(regctl dns list $domain --type A --output json | \
        jq -r ".records[] | select(.content == \"$OLD_IP\") | .id")
    
    if [ -z "$records" ]; then
        log "  No matching records found for $domain"
        continue
    fi
    
    # レコードを更新
    for record_id in $records; do
        log "  Updating record $record_id"
        
        if regctl dns update $domain $record_id --content "$NEW_IP"; then
            log "  ✓ Successfully updated record $record_id"
        else
            log "  ✗ Failed to update record $record_id"
        fi
        
        # レート制限対策
        sleep 1
    done
done

log "DNS batch update completed"

# サマリー表示
echo ""
echo "Update Summary"
echo "=============="
grep -c "Successfully updated" "$LOG_FILE" | xargs echo "Success:"
grep -c "Failed to update" "$LOG_FILE" | xargs echo "Failed:"