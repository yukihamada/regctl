# regctl.sh - ホスティング統合ガイド

## クイックスタート

```bash
# 1. ドメイン検索・登録・ホスティングを一括実行
regctl deploy example.com --template static

# 2. 既存ドメインにホスティング追加
regctl host add example.com --type fly

# 3. DNS設定を自動更新
regctl dns sync example.com
```

## アーキテクチャ

```
regctl.sh
    ├── register    ドメイン登録（既存機能）
    ├── deploy      登録→ホスティング→DNS自動化（NEW）
    │   ├── fly     Fly.io アプリ作成
    │   ├── cloudflare  Cloudflare Pages（静的サイト）
    │   └── vercel  Vercel デプロイ
    └── dns         DNS管理（NEW）
        ├── sync    自動DNS設定
        └── check   設定確認
```

## コスト比較

| プラン | ドメイン | ホスティング | 合計/月 | IP割り当て速度 |
|--------|---------|-------------|---------|---------------|
| **無料枠** | $7.95（.com） | $0（Fly.io無料枠） | $0.66/月 | 30秒 |
| **Pro** | $7.95 | $1.94（Fly.io Dedicated） | $2.60/月 | 10秒 |
| **Static** | $7.95 | $0（Cloudflare Pages） | $0.66/月 | 5秒 |

## 対応ホスティングプロバイダー

### 1. Fly.io（推奨 - 動的サイト）
- **無料枠**: 3 shared-cpu VMs, 160GB転送/月
- **IP割り当て**: 即座（アプリ作成時に自動）
- **グローバル展開**: 34リージョン
- **API**: flyctl CLI経由

```bash
# 自動デプロイ例
regctl deploy myapp.com --template nextjs --region nrt
```

### 2. Cloudflare Pages（推奨 - 静的サイト）
- **無料枠**: 無制限リクエスト、500ビルド/月
- **IP割り当て**: 即座（Anycast IP）
- **CDN**: グローバル自動
- **API**: Cloudflare API v4

```bash
# 静的サイト自動デプロイ
regctl deploy myblog.com --template static --github yukihamada/myblog
```

### 3. Vercel（オプション - Next.js特化）
- **無料枠**: 100GB転送/月
- **IP割り当て**: 即座（Edge Network）
- **最適化**: Next.js自動最適化

## 実装詳細

### Phase 1: Fly.io統合（最優先）

**ファイル構成**:
```
regctl.sh/
├── internal/
│   ├── hosting/
│   │   ├── fly.go          # Fly.io API クライアント
│   │   ├── provider.go     # ホスティングプロバイダー抽象化
│   │   └── templates.go    # デプロイテンプレート
│   └── dns/
│       ├── cloudflare.go   # Cloudflare DNS API
│       └── manager.go      # DNS管理ロジック
├── cmd/
│   └── deploy.go           # deploy コマンド実装
└── templates/              # アプリテンプレート
    ├── static/
    ├── nextjs/
    └── go/
```

**ワークフロー**:
```
1. ドメイン登録（既存API）
   ↓
2. Fly.ioアプリ作成
   $ flyctl apps create myapp --org personal
   ↓ レスポンス: IPv4 66.241.124.100, IPv6 2a09:8280:1::
   ↓
3. DNS自動設定（Cloudflare API）
   A    example.com → 66.241.124.100
   AAAA example.com → 2a09:8280:1::
   ↓
4. アプリデプロイ
   $ flyctl deploy --app myapp
   ↓
5. 完了（合計30-60秒）
```

### Phase 2: DNS自動化

**Cloudflare API統合**:
```go
type CloudflareDNS struct {
    apiToken string
    client   *http.Client
}

func (c *CloudflareDNS) SetRecords(domain string, records []DNSRecord) error {
    // 1. ゾーンID取得
    zoneID, err := c.getZoneID(domain)

    // 2. 既存レコード削除
    c.deleteExistingRecords(zoneID, domain)

    // 3. 新レコード作成
    for _, record := range records {
        c.createRecord(zoneID, record)
    }

    return nil
}
```

### Phase 3: テンプレートシステム

**デプロイテンプレート**:
```yaml
# templates/static/fly.toml
app = "{{.AppName}}"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true

[[statics]]
  guest_path = "/app/public"
  url_prefix = "/"
```

```dockerfile
# templates/static/Dockerfile
FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 8080
```

## セキュリティ

- **API Key管理**: `~/.regctl/config.yaml` に暗号化保存
- **DNS検証**: DNSSEC対応確認
- **HTTPS**: Let's Encrypt自動証明書（Fly.io/Cloudflare両対応）

## 使用例

### ケース1: 個人ブログをゼロから立ち上げ
```bash
# 1. ドメイン検索・登録・ホスティング一括
regctl deploy myblog.dev --template static --source ./my-blog

# 内部処理:
# ✓ myblog.dev を最安値レジストラで登録（Porkbun $4.18）
# ✓ Cloudflare Pages にデプロイ
# ✓ DNS自動設定（A/AAAA/CNAME）
# ✓ SSL証明書自動取得
# → https://myblog.dev が30秒で稼働

# 月額コスト: $0.35（ドメイン代のみ）
```

### ケース2: Next.jsアプリを高速展開
```bash
# GitHub リポジトリから自動デプロイ
regctl deploy myapp.com \
  --template nextjs \
  --github yukihamada/myapp \
  --region nrt,sin,lax

# 内部処理:
# ✓ myapp.com を登録
# ✓ Fly.io に3リージョン展開（Tokyo, Singapore, LA）
# ✓ GitHub Actions で自動ビルド設定
# ✓ DNS設定完了
# → https://myapp.com が60秒で稼働

# 月額コスト: $0.66（無料枠内）
```

### ケース3: マイクロサービスを量産
```bash
# 複数ドメインを一括セットアップ
cat domains.txt | xargs -I {} regctl deploy {} --template go

# domains.txt:
# api.myapp.com
# admin.myapp.com
# webhook.myapp.com

# 各ドメインが独立したFly.ioアプリとして起動
# 合計90秒で3サービス稼働
```

## ロードマップ

- [x] **Phase 0**: ドメイン登録機能（完了）
- [ ] **Phase 1**: Fly.io統合（今週実装）
  - [ ] `flyctl` API ラッパー
  - [ ] テンプレート3種（static, nextjs, go）
  - [ ] `regctl deploy` コマンド
- [ ] **Phase 2**: DNS自動化（来週）
  - [ ] Cloudflare API統合
  - [ ] 他レジストラDNS API対応
  - [ ] `regctl dns` コマンド
- [ ] **Phase 3**: 高度な機能（2週間後）
  - [ ] Vercel統合
  - [ ] カスタムテンプレート
  - [ ] マルチリージョン展開UI
  - [ ] コスト最適化アドバイザー

## API設定

### 必要なAPI Key

1. **Fly.io**: `fly auth token`で取得
2. **Cloudflare**: ダッシュボード → API Tokens → Create Token
3. **GitHub**（オプション）: Personal Access Token

### 設定方法

```bash
# 初回セットアップ
regctl config set fly.token <FLY_TOKEN>
regctl config set cloudflare.token <CF_TOKEN>
regctl config set github.token <GH_TOKEN>

# 設定確認
regctl config list
```

## パフォーマンス

| 操作 | 所要時間 | 備考 |
|-----|---------|------|
| ドメイン検索 | 2-5秒 | 4レジストラ並行検索 |
| ドメイン登録 | 10-30秒 | レジストラAPIレスポンス待ち |
| Fly.ioアプリ作成 | 5-10秒 | IP即座割り当て |
| DNS伝播 | 1-5分 | Cloudflare高速（通常1分） |
| **合計（初回）** | **30-60秒** | DNS伝播除く |

## トラブルシューティング

### DNS伝播が遅い
```bash
# 伝播状況確認
regctl dns check example.com

# 出力例:
# ✓ Cloudflare NS: 1.1.1.1 → 66.241.124.100 (OK)
# ✓ Google NS: 8.8.8.8 → 66.241.124.100 (OK)
# ⚠ ISP NS: xxx.xxx.xxx.xxx → NXDOMAIN (伝播中)
```

### Fly.io無料枠を使い切った
```bash
# 使用状況確認
regctl hosting status

# 出力例:
# Fly.io: 3/3 VMs使用中
# 推奨: 不要なアプリを削除、または有料プランへ
```

## まとめ

**regctl.sh + Fly.io + Cloudflare** で、ドメイン登録からホスティング起動まで**30秒、月額$0.66**で実現できます。

次のステップ: Phase 1実装に着手しますか？
