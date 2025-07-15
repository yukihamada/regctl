# 🚀 regctl.com 完全サービス展開ガイド

## 📋 現在の状況

✅ **完了済み**:
- regctl.com ドメイン登録完了 (VALUE-DOMAIN、¥790)
- Worker API デプロイ完了 (api.regctl.com対応)
- サブドメイン構成設計完了
- Cloudflare DNS設定手順準備完了

## 🎯 次の実行ステップ

### ステップ1: Cloudflareでドメイン管理設定

```bash
# 1. Cloudflare ダッシュボードアクセス
open https://dash.cloudflare.com/

# 2. "Add a Site" で regctl.com を追加

# 3. ネームサーバーをメモ（例: ns1.cloudflare.com, ns2.cloudflare.com）
```

### ステップ2: VALUE-DOMAINでネームサーバー変更

```bash
# VALUE-DOMAIN コントロールパネルアクセス
open https://www.value-domain.com/

# ログイン後:
# ドメイン管理 > regctl.com > ネームサーバー設定
# Cloudflareから提供されたネームサーバーに変更
```

### ステップ3: Cloudflare Pages プロジェクト作成

```bash
# スクリプト実行で一括作成
cd /Users/yuki/RegiOps
./deploy/pages-setup.sh

# または手動で実行:
cd site
wrangler pages project create regctl-site
wrangler pages deploy . --project-name=regctl-site

cd app  
npm run build
wrangler pages project create regctl-app
wrangler pages deploy dist --project-name=regctl-app

cd ../docs
wrangler pages project create regctl-docs  
wrangler pages deploy . --project-name=regctl-docs
```

### ステップ4: DNS レコード設定

CloudflareのAPI設定:
```bash
# 環境変数設定
export CLOUDFLARE_API_TOKEN="あなたのトークン"
export ZONE_ID="regctl.comのゾーンID"

# Root domain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"@","content":"regctl-site.pages.dev","proxied":true}'

# API subdomain (Worker用)
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"api","content":"regctl-api.yukihamada.workers.dev","proxied":true}'

# App subdomain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"app","content":"regctl-app.pages.dev","proxied":true}'

# Docs subdomain
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"docs","content":"regctl-docs.pages.dev","proxied":true}'

# WWW redirect
curl -X POST "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
  -H "Content-Type: application/json" \
  --data '{"type":"CNAME","name":"www","content":"regctl-site.pages.dev","proxied":true}'
```

### ステップ5: カスタムドメイン設定

各Pagesプロジェクトにカスタムドメインを追加:

```bash
# Cloudflare ダッシュボードで実行:
# Pages > regctl-site > Custom domains > "regctl.com" 追加
# Pages > regctl-site > Custom domains > "www.regctl.com" 追加
# Pages > regctl-app > Custom domains > "app.regctl.com" 追加  
# Pages > regctl-docs > Custom domains > "docs.regctl.com" 追加
```

### ステップ6: Worker ルート確認

現在のwrangler.tomlには既にルートが設定済み:
```toml
routes = [
  "api.regctl.com/*",
  "regctl.com/api/*", 
  "api.regctl.cloud/*",
  "regctl.cloud/api/*"
]
```

## 🔍 動作確認コマンド

### DNS伝播確認
```bash
# 各サブドメインの確認
dig regctl.com
dig api.regctl.com
dig app.regctl.com
dig docs.regctl.com
dig www.regctl.com

# または
nslookup regctl.com
nslookup api.regctl.com
```

### API動作確認
```bash
# API エンドポイント確認
curl https://api.regctl.com/health
curl https://api.regctl.com/api/v1/health
curl https://api.regctl.com/api/v1/subdomains

# ドメイン価格確認
curl https://api.regctl.com/api/v1/test/value-domain/pricing/regctl.com

# VALUE-DOMAIN連携確認
curl https://api.regctl.com/api/v1/test/value-domain/connectivity
```

### サイト動作確認
```bash
# 各サイトアクセス確認
curl -I https://regctl.com
curl -I https://www.regctl.com
curl -I https://app.regctl.com
curl -I https://docs.regctl.com
```

## 📊 期待される結果

完了後のサービス構成:

```
regctl.com ecosystem:
├── https://regctl.com          → マーケティングサイト
├── https://www.regctl.com      → regctl.com へリダイレクト  
├── https://api.regctl.com      → regctl API (Worker)
├── https://app.regctl.com      → ダッシュボード (React)
└── https://docs.regctl.com     → ドキュメント
```

### SSL証明書
- Cloudflareが自動発行
- Let's Encrypt証明書
- A+ SSL Labs評価

### パフォーマンス
- Cloudflare CDN経由で高速配信
- 世界中のエッジロケーション活用
- 99.9%+ 稼働率

## ⏰ 予想時間

| タスク | 時間 |
|-------|------|
| Cloudflareドメイン追加 | 5分 |
| ネームサーバー変更 | 5分 |
| DNS伝播待機 | 15-60分 |
| Pages プロジェクト作成 | 15分 |
| カスタムドメイン設定 | 10分 |
| SSL証明書発行 | 15分 |
| **合計** | **65-130分** |

## 🎉 完了後の次ステップ

1. **CLI設定更新**: `regctl login`でapi.regctl.comを使用
2. **ドキュメント公開**: docs.regctl.comでAPI仕様公開
3. **ユーザー登録**: app.regctl.comでユーザー管理開始
4. **プロダクション運用**: 監視・ログ・バックアップ設定

---

**🚀 regctl Cloud SaaS 本格稼働開始！**