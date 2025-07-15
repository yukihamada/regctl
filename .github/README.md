# GitHub Actions CI/CD Configuration

このディレクトリには、regctl CloudのCI/CDパイプラインの設定が含まれています。

## ワークフロー

### deploy.yml
メインのデプロイワークフロー。以下を実行します：
1. サイトのテスト（HTML/CSS/JavaScript検証）
2. Cloudflare Workersへのデプロイ
3. Cloudflare Pagesへのサイトデプロイ

### deploy-with-tests.yml
拡張版のデプロイワークフロー。以下の追加テストを含みます：
- Lighthouse CI（パフォーマンステスト）
- リンクチェック
- PRプレビューデプロイ

## 必要なGitHub Secrets

以下のシークレットをリポジトリに設定してください：

1. **CLOUDFLARE_API_TOKEN**
   - Cloudflareの[APIトークン](https://dash.cloudflare.com/profile/api-tokens)を作成
   - 必要な権限：
     - Account:Cloudflare Pages:Edit
     - Zone:Page Rules:Edit
     - Zone:DNS:Edit

2. **CLOUDFLARE_ACCOUNT_ID**
   - CloudflareダッシュボードのアカウントIDをコピー

3. **その他のシークレット**（Workersデプロイ用）
   - JWT_SECRET
   - VALUE_DOMAIN_API_KEY
   - AWS_ACCESS_KEY_ID
   - AWS_SECRET_ACCESS_KEY
   - PORKBUN_API_KEY
   - PORKBUN_API_SECRET
   - STRIPE_SECRET_KEY

## セットアップ手順

1. GitHub リポジトリの Settings > Secrets and variables > Actions に移動
2. 上記のシークレットを追加
3. mainブランチにプッシュすると自動的にデプロイが開始されます

## テストの実行

ローカルでテストを実行する場合：

```bash
cd site
npm install
npm test
```

## トラブルシューティング

- **デプロイが失敗する場合**
  - GitHub Actionsのログを確認
  - Cloudflare APIトークンの権限を確認
  - シークレットが正しく設定されているか確認

- **テストが失敗する場合**
  - HTMLバリデーションエラー: `.htmlvalidate.json`でルールを調整
  - CSSエラー: `.stylelintrc.json`でルールを調整
  - JSエラー: `.eslintrc.json`でルールを調整