# 🔑 Cloudflare API認証情報取得ガイド

## 手順1: APIトークン取得

1. https://dash.cloudflare.com/profile/api-tokens にアクセス
2. "Create Token" をクリック
3. "Edit zone DNS" テンプレートを選択
4. Zone Resources: Include - Specific zone - regctl.com を選択
5. "Continue to summary" → "Create Token"
6. 生成されたトークンをコピー

## 手順2: ゾーンID取得

1. https://dash.cloudflare.com/ でregctl.comを選択
2. 右サイドバーの"Zone ID"をコピー

## 手順3: 環境変数設定とコマンド実行

```bash
# 認証情報設定
export CLOUDFLARE_API_TOKEN="your_api_token_here"
export ZONE_ID="your_zone_id_here"

# DNS設定実行
./deploy/dns-setup-commands.sh
```

## または、手動でcurlコマンド実行も可能です。