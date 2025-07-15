# 🌐 Cloudflare DNS設定ガイド - regctl.com

## 📋 概要

regctl.comをVALUE-DOMAINで登録完了後、CloudflareでDNS管理とサブドメイン設定を行います。

## ⚡ ステップ1: Cloudflareにドメインを追加

### 1.1 Cloudflareダッシュボードでドメイン追加

```bash
# Cloudflare ダッシュボードにアクセス
https://dash.cloudflare.com/

# "Add a Site" をクリック
# "regctl.com" を入力
# "Add Site" をクリック
```

### 1.2 DNS レコードスキャン

Cloudflareが自動的にDNSレコードをスキャンします。

## ⚡ ステップ2: Cloudflare ネームサーバーの設定

### 2.1 Cloudflareのネームサーバーを確認

Cloudflareから提供されるネームサーバー（例）：
```
ns1.cloudflare.com
ns2.cloudflare.com
```

### 2.2 VALUE-DOMAINでネームサーバー変更

```bash
# VALUE-DOMAIN API経由でネームサーバー変更
curl -X PUT "https://api.value-domain.com/v1/domains/regctl.com" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "nameservers": [
      "ns1.cloudflare.com",
      "ns2.cloudflare.com"
    ]
  }'
```

または、VALUE-DOMAINコントロールパネルから手動で変更：
1. VALUE-DOMAINにログイン
2. ドメイン管理 → regctl.com
3. ネームサーバー設定 → Cloudflareのネームサーバーを入力

## ⚡ ステップ3: regctl.com サブドメイン設定

### 3.1 必要なDNSレコード

```bash
# Root domain
Type: CNAME
Name: regctl.com
Content: regctl-site.pages.dev
Proxied: Yes
TTL: Auto

# API subdomain  
Type: CNAME
Name: api
Content: regctl-api.yukihamada.workers.dev
Proxied: Yes
TTL: Auto

# App subdomain
Type: CNAME
Name: app  
Content: regctl-app.pages.dev
Proxied: Yes
TTL: Auto

# Docs subdomain
Type: CNAME
Name: docs
Content: regctl-docs.pages.dev
Proxied: Yes
TTL: Auto

# WWW redirect
Type: CNAME
Name: www
Content: regctl-site.pages.dev
Proxied: Yes
TTL: Auto

# CDN subdomain
Type: CNAME
Name: cdn
Content: regctl-cdn.r2.dev
Proxied: Yes
TTL: Auto

# Status subdomain
Type: CNAME
Name: status
Content: regctl-status.pages.dev
Proxied: Yes
TTL: Auto
```

### 3.2 Cloudflare APIでDNSレコード作成

```bash
# 環境変数設定
export CLOUDFLARE_API_TOKEN="your_cloudflare_api_token"
export ZONE_ID="your_zone_id"

# Root domain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME",
    "name": "regctl.com",
    "content": "regctl-site.pages.dev",
    "proxied": true
  }'

# API subdomain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME",
    "name": "api",
    "content": "regctl-api.yukihamada.workers.dev",
    "proxied": true
  }'

# App subdomain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME",
    "name": "app",
    "content": "regctl-app.pages.dev",
    "proxied": true
  }'

# Docs subdomain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME",
    "name": "docs",
    "content": "regctl-docs.pages.dev",
    "proxied": true
  }'

# WWW subdomain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME",
    "name": "www",
    "content": "regctl-site.pages.dev",
    "proxied": true
  }'
```

## ⚡ ステップ4: Workers ルート設定

### 4.1 現在の設定確認

```bash
# 現在のWorkerルート確認
wrangler route list
```

### 4.2 新しいルート追加

```bash
# api.regctl.com ルート追加
wrangler route add api.regctl.com/* regctl-api

# regctl.com/api ルート追加  
wrangler route add regctl.com/api/* regctl-api
```

または、wrangler.tomlで設定済み：
```toml
routes = [
  "api.regctl.com/*",
  "regctl.com/api/*",
  "api.regctl.cloud/*",
  "regctl.cloud/api/*"
]
```

## ⚡ ステップ5: SSL/TLS設定

### 5.1 SSL設定

Cloudflareダッシュボードで：
1. SSL/TLS タブ
2. 暗号化モード: **Full (strict)**
3. Always Use HTTPS: **有効**
4. HTTP Strict Transport Security (HSTS): **有効**

### 5.2 カスタムドメイン証明書

Cloudflareが自動的にSSL証明書を発行します。

## ⚡ ステップ6: 確認とテスト

### 6.1 DNS伝播確認

```bash
# DNS伝播確認
dig regctl.com
dig api.regctl.com
dig app.regctl.com
dig docs.regctl.com

# または
nslookup regctl.com
nslookup api.regctl.com
```

### 6.2 API動作確認

```bash
# API endpoint test
curl https://api.regctl.com/health
curl https://api.regctl.com/api/v1/health

# Domain check
curl https://api.regctl.com/api/v1/subdomains
```

## 📊 予想される処理時間

| ステップ | 時間 |
|---------|------|
| Cloudflareドメイン追加 | 5分 |
| ネームサーバー変更 | 10分 |
| DNS伝播 | 15-60分 |
| SSL証明書発行 | 15分 |
| 総時間 | **45分-90分** |

## 🔧 トラブルシューティング

### DNS伝播が遅い場合
```bash
# 伝播状況確認
https://www.whatsmydns.net/#CNAME/api.regctl.com
```

### SSL証明書エラー
1. SSL/TLS設定を確認
2. Full (strict) モードに設定
3. Origin証明書の設定確認

## 📝 次のステップ

DNS設定完了後：
1. Cloudflare Pages プロジェクト作成
2. カスタムドメイン設定
3. 各サービスのデプロイ
4. 動作確認とテスト

---

🎯 **目標**: regctl.com での完全なサービス稼働