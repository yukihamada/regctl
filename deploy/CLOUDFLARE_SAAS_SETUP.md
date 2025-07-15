# Cloudflare for SaaS Setup Guide

regctl.com ドメインでのCloudflare for SaaS設定手順です。

## 前提条件

1. regctl.com ドメインがCloudflareに追加済み
2. Cloudflare API Token が設定済み
3. Account ID が取得済み

## 環境変数設定

```bash
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export CLOUDFLARE_ACCOUNT_ID="your_account_id_here"
```

## 自動セットアップ（推奨）

### 1. Cloudflare for SaaS設定

```bash
cd /Users/yuki/RegiOps/deploy
./cloudflare-saas-setup.sh
```

このスクリプトで以下が自動設定されます：
- Custom Hostname APIの設定
- SSL証明書の自動発行
- DNS レコードの作成
- 各サブドメインの設定

### 2. Pages設定の更新

```bash
./pages-saas-config.sh
```

このスクリプトで以下が設定されます：
- regctl.com をregctl-siteプロジェクトに追加
- www.regctl.com の設定
- app.regctl.com の設定

## 手動設定（参考）

### 1. Custom Hostnames API設定

```bash
# Zone ID取得
ZONE_ID=$(curl -s -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  "https://api.cloudflare.com/client/v4/zones?name=regctl.com" | \
  jq -r '.result[0].id')

# Custom Hostname作成
curl -X POST \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "hostname": "regctl.com",
    "ssl": {
      "method": "http",
      "type": "dv"
    }
  }' \
  "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/custom_hostnames"
```

### 2. DNS設定

```bash
# メインドメイン（CNAME to Pages）
curl -X POST \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME",
    "name": "regctl.com",
    "content": "regctl-site.pages.dev",
    "proxied": true
  }' \
  "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records"

# API サブドメイン（CNAME to Workers）
curl -X POST \
  -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{
    "type": "CNAME", 
    "name": "api",
    "content": "regctl-api.yukihamada.workers.dev",
    "proxied": true
  }' \
  "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/dns_records"
```

## 設定確認

### 1. SSL証明書ステータス確認

```bash
curl -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/custom_hostnames" | \
  jq -r '.result[] | "Hostname: \(.hostname) | SSL Status: \(.ssl.status)"'
```

### 2. ドメインアクセステスト

```bash
# メインサイト
curl -I https://regctl.com

# API エンドポイント
curl -I https://api.regctl.com/health

# アプリ
curl -I https://app.regctl.com
```

## トラブルシューティング

### SSL証明書が発行されない場合

1. DNS設定が正しいか確認
2. 10-15分待ってから再確認
3. HTTP認証が完了するまで待機

### ドメイン検証が失敗する場合

1. DNS伝播を確認: `dig regctl.com`
2. Cloudflare Dashboard でステータス確認
3. 必要に応じてDNS設定を再作成

### Pages カスタムドメインエラー

1. プロジェクト名が正しいか確認
2. ドメインが他のプロジェクトで使用されていないか確認
3. DNS レコードがConflictしていないか確認

## 設定後の状態

設定完了後、以下のドメインが利用可能になります：

- **https://regctl.com** - メインウェブサイト（Pages）
- **https://www.regctl.com** - WWWリダイレクト
- **https://api.regctl.com** - API エンドポイント（Workers）
- **https://app.regctl.com** - アプリケーション（Pages）

## 注意事項

1. **SSL証明書**: 自動更新されますが、期限切れ前の通知設定を推奨
2. **DNS設定**: Cloudflare Proxyを有効にすることでパフォーマンス向上
3. **モニタリング**: UptimeRobotやPingdomでの監視設定を推奨
4. **バックアップ**: 設定のエクスポートを定期的に実行

## 関連リンク

- [Cloudflare for SaaS Documentation](https://developers.cloudflare.com/cloudflare-for-platforms/)
- [Custom Hostnames API](https://developers.cloudflare.com/ssl/ssl-for-saas/)
- [Pages Custom Domains](https://developers.cloudflare.com/pages/platform/custom-domains/)