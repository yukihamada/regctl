# regctl セットアップガイド

このガイドでは、regctlプロジェクトをGitHubにプッシュし、Cloudflareで動作させるまでの手順を説明します。

## 1. GitHubリポジトリの作成

### GitHub CLIを使用する場合
```bash
gh repo create yukihamada/regctl --public --description "CLI-First Multi-Registrar Management Tool" --push
```

### Webインターフェースを使用する場合
1. https://github.com/new にアクセス
2. リポジトリ名: `regctl`
3. 説明: `CLI-First Multi-Registrar Management Tool`
4. Public を選択
5. 「Create repository」をクリック

## 2. リモートリポジトリの設定とプッシュ

```bash
# リモートリポジトリを追加（GitHub CLIを使わない場合）
git remote add origin https://github.com/yukihamada/regctl.git

# コードをプッシュ
git push -u origin main
```

## 3. Cloudflareアカウントのセットアップ

### 3.1 Cloudflareアカウントの作成
1. https://dash.cloudflare.com/sign-up でアカウント作成
2. メールアドレスを確認

### 3.2 Wranglerのインストールと認証
```bash
npm install -g wrangler
wrangler login
```

### 3.3 D1データベースの作成
```bash
cd edge/workers

# データベースを作成
wrangler d1 create regctl-db

# 出力されたdatabase_idをメモして、wrangler.tomlに記載
# 例: database_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

### 3.4 KV Namespaceの作成
```bash
# セッション用
wrangler kv:namespace create "SESSIONS"
# 出力: id = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# キャッシュ用
wrangler kv:namespace create "CACHE"
# 出力: id = "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
```

### 3.5 wrangler.tomlの更新
`edge/workers/wrangler.toml`を編集して、作成したリソースのIDを設定：

```toml
[[d1_databases]]
binding = "DB"
database_name = "regctl-db"
database_id = "実際のdatabase_id"

kv_namespaces = [
  { binding = "SESSIONS", id = "実際のSESSIONS_id" },
  { binding = "CACHE", id = "実際のCACHE_id" }
]
```

## 4. 環境変数の設定

### 4.1 ローカル開発用
```bash
cp .env.example .env
# .envファイルを編集して、必要なAPIキーを設定
```

### 4.2 Cloudflare Workers用
```bash
cd edge/workers

# シークレットを設定
wrangler secret put JWT_SECRET
# ランダムな値を入力（例: openssl rand -base64 32 の出力）

wrangler secret put VALUE_DOMAIN_API_KEY
# VALUE-DOMAINのAPIキーを入力

wrangler secret put AWS_ACCESS_KEY_ID
# AWSのアクセスキーIDを入力

wrangler secret put AWS_SECRET_ACCESS_KEY
# AWSのシークレットアクセスキーを入力

wrangler secret put PORKBUN_API_KEY
# PorkbunのAPIキーを入力

wrangler secret put PORKBUN_API_SECRET
# PorkbunのAPIシークレットを入力
```

## 5. GitHub Secretsの設定

GitHubリポジトリの Settings > Secrets and variables > Actions から以下を設定：

- `CLOUDFLARE_API_TOKEN`: Cloudflare APIトークン
- `CLOUDFLARE_ACCOUNT_ID`: CloudflareアカウントID
- `JWT_SECRET`: JWT署名用シークレット
- `VALUE_DOMAIN_API_KEY`: VALUE-DOMAIN APIキー
- `AWS_ACCESS_KEY_ID`: AWS アクセスキーID
- `AWS_SECRET_ACCESS_KEY`: AWS シークレットアクセスキー
- `PORKBUN_API_KEY`: Porkbun APIキー
- `PORKBUN_API_SECRET`: Porkbun APIシークレット

## 6. 初回デプロイ

### 6.1 データベースの初期化
```bash
cd edge/workers

# スキーマを適用
wrangler d1 execute regctl-db --file=./schema.sql
```

### 6.2 Workersのデプロイ
```bash
# 開発環境で確認
wrangler dev

# 本番環境にデプロイ
wrangler deploy
```

### 6.3 カスタムドメインの設定（オプション）
Cloudflareダッシュボードで：
1. Workers & Pages → regctl-api
2. Settings → Triggers
3. Custom Domains → Add Custom Domain
4. `api.regctl.cloud` などを設定

## 7. CLIツールのビルドとテスト

```bash
# プロジェクトルートで
cd cmd/regctl
go build -o ../../regctl

# テスト実行
cd ../..
./regctl version
```

## 8. サイトのデプロイ（Cloudflare Pages）

```bash
# Cloudflare Pagesプロジェクトを作成
wrangler pages project create regctl-site

# サイトをデプロイ
wrangler pages deploy site --project-name=regctl-site
```

## 9. 初回リリース

```bash
# バージョンタグを作成
git tag v0.1.0
git push --tags

# GitHub Actionsが自動的にリリースビルドを作成
```

## 10. 動作確認

```bash
# CLIをインストール
curl -fsSL https://regctl.cloud/install.sh | sh

# ログイン
regctl login --device

# ドメイン一覧を確認
regctl domains list
```

## トラブルシューティング

### D1データベースエラー
```bash
# データベースの再作成
wrangler d1 delete regctl-db
wrangler d1 create regctl-db
# 新しいIDでwrangler.tomlを更新
```

### KVエラー
```bash
# KV Namespaceの確認
wrangler kv:namespace list
```

### デプロイエラー
```bash
# ログを確認
wrangler tail
```

## 次のステップ

1. ドメインレジストラのAPIキーを取得
2. 実際のドメインでテスト
3. ドキュメントサイトの公開
4. コミュニティへの告知

詳細は[セルフホスティングガイド](docs/SELF_HOSTING.md)を参照してください。