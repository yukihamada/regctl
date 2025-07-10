# regctl Examples

実際の使用例とユースケースを紹介します。

## 基本的な使用例

### 初回セットアップ

```bash
# インストール
curl -fsSL https://regctl.cloud/install.sh | sh

# ログイン（デバイスフロー推奨）
regctl login --device

# 設定確認
regctl config show
```

### ドメイン管理の基本

```bash
# ドメイン一覧
regctl domains list

# 特定のレジストラのドメインのみ表示
regctl domains list --registrar value-domain

# ドメイン詳細情報
regctl domains info example.com

# 新規ドメイン登録
regctl domains register mynewsite.com --registrar porkbun --years 2
```

## 実践的なユースケース

### 1. 複数ドメインの一括管理

```bash
#!/bin/bash
# domains_report.sh - すべてのドメインの状態をレポート

echo "Domain Status Report - $(date)"
echo "================================"

# JSONフォーマットで取得して処理
regctl domains list --output json | jq -r '.domains[] | 
  "\(.name) | \(.registrar) | \(.expires_at) | \(.auto_renew)"' | 
  column -t -s '|'

# 期限が近いドメインを警告
echo -e "\nDomains expiring soon:"
regctl domains list --output json | jq -r '.domains[] | 
  select(.expires_at < (now + 30*24*60*60 | strftime("%Y-%m-%d"))) | 
  .name + " expires on " + .expires_at'
```

### 2. DNS設定の自動化

```bash
#!/bin/bash
# setup_dns.sh - 新しいドメインのDNS設定を自動化

DOMAIN=$1
IP_ADDRESS=$2

if [ -z "$DOMAIN" ] || [ -z "$IP_ADDRESS" ]; then
  echo "Usage: $0 <domain> <ip_address>"
  exit 1
fi

echo "Setting up DNS for $DOMAIN..."

# Aレコード設定
regctl dns add $DOMAIN --type A --name @ --content $IP_ADDRESS
regctl dns add $DOMAIN --type A --name www --content $IP_ADDRESS

# MXレコード設定（Google Workspace）
regctl dns add $DOMAIN --type MX --name @ --content aspmx.l.google.com --priority 1
regctl dns add $DOMAIN --type MX --name @ --content alt1.aspmx.l.google.com --priority 5
regctl dns add $DOMAIN --type MX --name @ --content alt2.aspmx.l.google.com --priority 5

# SPFレコード
regctl dns add $DOMAIN --type TXT --name @ --content "v=spf1 include:_spf.google.com ~all"

# 確認
echo -e "\nDNS records for $DOMAIN:"
regctl dns list $DOMAIN
```

### 3. ドメイン移管の自動化

```bash
#!/bin/bash
# migrate_domains.sh - 複数ドメインを一括移管

FROM_REGISTRAR="value-domain"
TO_REGISTRAR="route53"
DOMAINS=("example1.com" "example2.com" "example3.com")

for domain in "${DOMAINS[@]}"; do
  echo "Processing $domain..."
  
  # 現在のステータス確認
  status=$(regctl domains info $domain --output json | jq -r '.status')
  
  if [ "$status" != "active" ]; then
    echo "Skipping $domain - status is $status"
    continue
  fi
  
  # 移管実行
  echo "Initiating transfer for $domain..."
  regctl domains transfer $domain --from $FROM_REGISTRAR --to $TO_REGISTRAR
  
  # 結果を記録
  echo "$domain transfer initiated at $(date)" >> transfer_log.txt
  
  # API制限回避のため少し待機
  sleep 5
done
```

### 4. CI/CDパイプラインでの使用

```yaml
# .github/workflows/dns_deploy.yml
name: Deploy DNS Changes

on:
  push:
    paths:
      - 'dns-config/*.yaml'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install regctl
        run: |
          curl -fsSL https://regctl.cloud/install.sh | sh
          
      - name: Login to regctl
        env:
          REGCTL_TOKEN: ${{ secrets.REGCTL_TOKEN }}
        run: |
          echo "$REGCTL_TOKEN" | regctl login --token-stdin
          
      - name: Apply DNS configurations
        run: |
          for config in dns-config/*.yaml; do
            domain=$(basename $config .yaml)
            echo "Applying DNS config for $domain"
            regctl dns import $domain --file $config
          done
```

### 5. 監視とアラート

```bash
#!/bin/bash
# monitor_domains.sh - ドメインの状態を監視してSlackに通知

SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

check_domains() {
  # 期限切れ間近のドメインをチェック
  expiring=$(regctl domains list --output json | jq -r '
    .domains[] | 
    select(.expires_at < (now + 30*24*60*60 | strftime("%Y-%m-%d"))) | 
    .name + " expires on " + .expires_at
  ')
  
  if [ -n "$expiring" ]; then
    message="⚠️ Domains expiring soon:\n$expiring"
    curl -X POST -H 'Content-type: application/json' \
      --data "{\"text\":\"$message\"}" \
      $SLACK_WEBHOOK_URL
  fi
  
  # 移管中のドメインをチェック
  transferring=$(regctl domains list --status transferring --output json | jq -r '
    .domains[] | .name
  ')
  
  if [ -n "$transferring" ]; then
    message="🔄 Domains in transfer:\n$transferring"
    curl -X POST -H 'Content-type: application/json' \
      --data "{\"text\":\"$message\"}" \
      $SLACK_WEBHOOK_URL
  fi
}

# 毎日実行（cronで設定）
check_domains
```

### 6. バックアップとリストア

```bash
#!/bin/bash
# backup_dns.sh - すべてのDNSレコードをバックアップ

BACKUP_DIR="dns_backup_$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

# すべてのドメインのDNSレコードをエクスポート
regctl domains list --output json | jq -r '.domains[].name' | while read domain; do
  echo "Backing up DNS for $domain..."
  regctl dns list $domain --output json > "$BACKUP_DIR/${domain}_dns.json"
  
  # ゾーンファイル形式でもエクスポート
  regctl dns export $domain > "$BACKUP_DIR/${domain}.zone"
done

# アーカイブ作成
tar -czf "${BACKUP_DIR}.tar.gz" $BACKUP_DIR
rm -rf $BACKUP_DIR

echo "Backup completed: ${BACKUP_DIR}.tar.gz"
```

### 7. ドメインコスト分析

```bash
#!/bin/bash
# cost_analysis.sh - ドメインのコストを分析

echo "Domain Cost Analysis Report"
echo "=========================="

total_cost=0

regctl domains list --output json | jq -r '.domains[]' | while read -r domain_json; do
  name=$(echo "$domain_json" | jq -r '.name')
  registrar=$(echo "$domain_json" | jq -r '.registrar')
  
  # レジストラごとの年間費用（例）
  case $registrar in
    "value-domain")
      annual_cost=1500
      ;;
    "route53")
      annual_cost=1200
      ;;
    "porkbun")
      annual_cost=900
      ;;
    *)
      annual_cost=1000
      ;;
  esac
  
  echo "$name: ¥$annual_cost/year ($registrar)"
  total_cost=$((total_cost + annual_cost))
done

echo "=========================="
echo "Total Annual Cost: ¥$total_cost"
```

## 高度な使用例

### プログラマティックな使用（Go）

```go
// main.go - regctl APIをGoから直接使用

package main

import (
    "fmt"
    "log"
    "github.com/yukihamada/regctl/pkg/client"
)

func main() {
    // クライアント作成
    c, err := client.NewClient(client.Config{
        APIEndpoint: "https://api.regctl.cloud",
        Token:       os.Getenv("REGCTL_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // ドメイン一覧取得
    domains, err := c.Domains.List(client.ListOptions{
        Registrar: "porkbun",
    })
    if err != nil {
        log.Fatal(err)
    }

    // DNSレコード追加
    for _, domain := range domains {
        _, err := c.DNS.CreateRecord(domain.Name, client.DNSRecord{
            Type:    "TXT",
            Name:    "_verification",
            Content: "site-verification=xyz123",
            TTL:     300,
        })
        if err != nil {
            log.Printf("Failed to add record to %s: %v", domain.Name, err)
        }
    }
}
```

### Terraformプロバイダー（将来実装予定）

```hcl
# main.tf - Terraformでドメインを管理

terraform {
  required_providers {
    regctl = {
      source  = "yukihamada/regctl"
      version = "~> 1.0"
    }
  }
}

provider "regctl" {
  api_endpoint = "https://api.regctl.cloud"
}

resource "regctl_domain" "example" {
  name           = "example.com"
  registrar      = "porkbun"
  auto_renew     = true
  privacy_enabled = true
}

resource "regctl_dns_record" "www" {
  domain  = regctl_domain.example.name
  type    = "CNAME"
  name    = "www"
  content = "example.com"
  ttl     = 3600
  proxied = true
}
```

## ベストプラクティス

1. **APIキーの管理**
   - 環境変数や安全なシークレット管理ツールを使用
   - コードにハードコードしない

2. **エラーハンドリング**
   - 常に戻り値をチェック
   - リトライロジックの実装

3. **レート制限対策**
   - バッチ処理時は適切な待機時間を設定
   - 並行処理の制限

4. **監査ログ**
   - 重要な操作はログに記録
   - 変更履歴の保持

詳細なAPIドキュメントは[API Reference](../docs/API.md)を参照してください。