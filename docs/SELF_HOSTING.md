# セルフホスティングガイド

regctl Cloudを自分のインフラストラクチャで運用するための完全ガイドです。

## 前提条件

- Cloudflareアカウント（Workers, D1, KV, R2が利用可能）
- 各レジストラのAPIキー
  - VALUE-DOMAIN API Key
  - AWS Access Key (Route 53用)
  - Porkbun API Key & Secret
- Node.js 18以上
- Go 1.23以上
- Git

## セットアップ手順

### 1. リポジトリのクローン

```bash
git clone https://github.com/yukihamada/regctl.git
cd regctl
```

### 2. Cloudflareの準備

#### 2.1 Cloudflare CLIのインストール

```bash
npm install -g wrangler
wrangler login
```

#### 2.2 D1データベースの作成

```bash
cd edge/workers

# データベースを作成
wrangler d1 create regctl-db

# 出力されたdatabase_idを wrangler.toml に記載
```

#### 2.3 KV Namespaceの作成

```bash
# セッション用KV
wrangler kv:namespace create "SESSIONS"

# キャッシュ用KV  
wrangler kv:namespace create "CACHE"

# 出力されたIDを wrangler.toml に記載
```

### 3. 環境変数の設定

```bash
# プロジェクトルートで
cp .env.example .env
```

`.env`ファイルを編集：

```env
# Cloudflare
CLOUDFLARE_ACCOUNT_ID=your-account-id
CLOUDFLARE_API_TOKEN=your-api-token

# JWT Secret (必ず変更してください)
JWT_SECRET=$(openssl rand -base64 32)

# レジストラAPI
VALUE_DOMAIN_API_KEY=your-value-domain-key
AWS_ACCESS_KEY_ID=your-aws-key
AWS_SECRET_ACCESS_KEY=your-aws-secret
PORKBUN_API_KEY=your-porkbun-key
PORKBUN_API_SECRET=your-porkbun-secret

# Stripe (オプション)
STRIPE_SECRET_KEY=sk_test_...
```

### 4. データベースの初期化

```bash
cd edge/workers

# ローカルでスキーマを適用
wrangler d1 execute regctl-db --local --file=./schema.sql

# 本番環境に適用
wrangler d1 execute regctl-db --file=./schema.sql
```

### 5. Workersのデプロイ

```bash
# 開発環境で確認
wrangler dev

# 本番環境にデプロイ
wrangler deploy
```

### 6. カスタムドメインの設定（オプション）

Cloudflareダッシュボードで：

1. Workers & Pages → 作成したWorker選択
2. Settings → Triggers
3. Custom Domains → Add Custom Domain
4. `api.yourdomain.com` を追加

### 7. CLIツールのビルド

```bash
cd cmd/regctl
go build -o regctl
```

## 設定のカスタマイズ

### APIエンドポイントの変更

CLIで独自のAPIエンドポイントを使用：

```bash
regctl config set api_url https://api.yourdomain.com
```

### レート制限の調整

`edge/workers/src/middleware/rate-limit.ts`で調整：

```typescript
const response = await stub.fetch(
    `https://rate-limiter.internal/?key=${key}&limit=1000&window=60`
)
```

### 新しいレジストラの追加

1. `edge/workers/src/services/providers/`に新しいプロバイダーを実装
2. `base.ts`のインターフェースを実装
3. `domains.ts`の`getProvider`関数に追加

## 監視とロギング

### Cloudflare Analytics

Workersダッシュボードで以下を確認：
- リクエスト数
- エラー率
- レスポンス時間
- 地域別アクセス

### カスタムメトリクス

```typescript
// カスタムメトリクスの送信例
c.executionCtx.waitUntil(
  c.env.ANALYTICS.writeDataPoint({
    blobs: ['domain_registered', domain.registrar],
    doubles: [1],
    indexes: [Date.now()],
  })
)
```

## バックアップとリストア

### D1データベースのバックアップ

```bash
# エクスポート
wrangler d1 export regctl-db > backup.sql

# インポート
wrangler d1 execute regctl-db --file=backup.sql
```

### KVデータのバックアップ

```bash
# 一覧取得
wrangler kv:key list --namespace-id=<namespace-id>

# 個別エクスポート
wrangler kv:key get <key> --namespace-id=<namespace-id>
```

## トラブルシューティング

### よくある問題

1. **D1データベースエラー**
   - `wrangler.toml`のdatabase_idを確認
   - マイグレーションが適用されているか確認

2. **認証エラー**
   - JWT_SECRETが設定されているか確認
   - 環境変数が正しくWorkersに渡されているか確認

3. **レジストラAPI接続エラー**
   - APIキーの有効性を確認
   - IPホワイトリストの設定を確認

### デバッグモード

```bash
# Workersのログを確認
wrangler tail

# CLIのデバッグモード
regctl --debug domains list
```

## セキュリティのベストプラクティス

1. **シークレット管理**
   - 本番環境では環境変数を使用
   - `wrangler secret put`でセキュアに保存

2. **アクセス制御**
   - Cloudflare Access でWorkers APIを保護
   - API Key認証の実装

3. **監査ログ**
   - すべてのAPIアクセスをログ記録
   - 定期的な監査の実施

## アップデート手順

```bash
# 最新版を取得
git pull origin main

# 依存関係を更新
npm install
go mod download

# テストを実行
make test

# デプロイ
wrangler deploy
```

## サポート

- GitHub Issues: https://github.com/yukihamada/regctl/issues
- Discussions: https://github.com/yukihamada/regctl/discussions
- Security: security@regctl.cloud