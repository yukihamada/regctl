#!/bin/bash

# 🔧 デモユーザー作成スクリプト

echo "👤 デモユーザー作成"
echo ""

# Cloudflare Workers wrangler でデモユーザーを直接DBに挿入
echo "🔑 デモユーザー情報:"
echo "   メール: demo@regctl.com"
echo "   パスワード: demo12345"
echo "   名前: Demo User"
echo ""

# Workers環境でSQL実行
echo "📊 DB接続中..."

# パスワードハッシュ生成 (bcryptの代わりにプレーンテキストでテスト)
PASSWORD_HASH='$2b$10$K2KvLxtbXKmBgvJkYhkOCeCX5JLmUJOJN5.4rJOlBVGlQvLWQvhJa'

# SQL実行
SQL="INSERT INTO users (id, email, password_hash, name, role, plan, api_keys_limit, created_at, updated_at) VALUES (
  'demo-user-001',
  'demo@regctl.com',
  '$PASSWORD_HASH',
  'Demo User',
  'user',
  'free',
  5,
  datetime('now'),
  datetime('now')
) ON CONFLICT(email) DO UPDATE SET
  password_hash = '$PASSWORD_HASH',
  name = 'Demo User',
  updated_at = datetime('now');"

echo "📝 実行SQL:"
echo "$SQL"
echo ""

echo "💡 手動実行手順:"
echo "1. Cloudflare Dashboard → Workers & Pages → regctl-api"
echo "2. Settings → Variables → Database binding"
echo "3. D1 Console でSQL実行"
echo "4. または wrangler d1 execute で実行"
echo ""

echo "🔄 wrangler コマンド例:"
echo "cd edge/workers"
echo "wrangler d1 execute regctl-db --command=\"$SQL\""
echo ""

echo "✅ 完了後、以下でテスト:"
echo "./regctl login -e demo@regctl.com -p demo12345"