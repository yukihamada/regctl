# 🚀 regctl Cloud ドメイン取得ガイド

## 📋 ドメイン取得に必要なもの

### 🔧 **技術的要件**
1. **regctl CLI** - インストール済み
2. **VALUE-DOMAINアカウント** - API Key取得用
3. **regctl Cloudアカウント** - SaaS利用登録
4. **クレジットカード** - 支払い用（VALUE-DOMAIN経由）

### 📝 **個人情報（WHOIS情報）**
ドメイン登録時に以下の情報が必要です：

#### 👤 **個人の場合**
- 氏名（英語表記）
- 住所（英語表記）
- 電話番号
- メールアドレス
- 国・都道府県・郵便番号

#### 🏢 **法人の場合**
- 会社名（英語表記）
- 代表者名
- 会社住所（英語表記）
- 電話番号・FAX番号
- 担当者メールアドレス

#### 🇯🇵 **.co.jp ドメインの場合（追加要件）**
- 日本国内の法人登記が必要
- 登記簿謄本（原本）
- 印鑑登録証明書
- 代表者の身分証明書

## 🛠️ セットアップ手順

### ステップ1: regctl CLIインストール

```bash
# macOS (Homebrew)
brew install regctl

# Linux/Windows (直接ダウンロード)
curl -L https://github.com/yukihamada/regctl/releases/latest/download/regctl-linux-amd64 -o regctl
chmod +x regctl
sudo mv regctl /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/yukihamada/regctl/releases/latest/download/regctl-windows-amd64.exe" -OutFile "regctl.exe"
```

### ステップ2: VALUE-DOMAINアカウント作成

1. **VALUE-DOMAIN公式サイト**にアクセス
   - URL: https://www.value-domain.com/
2. **新規会員登録**をクリック
3. **必要情報入力**:
   - メールアドレス
   - パスワード
   - 個人情報（WHOIS用）
4. **メール認証**を完了
5. **API Key取得**:
   - 管理画面 → API設定
   - API Keyを生成・記録

### ステップ3: regctl Cloud登録

```bash
# regctl Cloudアカウント登録
regctl login --device

# ブラウザで認証完了後、以下のコマンドで確認
regctl config show
```

### ステップ4: VALUE-DOMAIN API Key設定

```bash
# 環境変数で設定（推奨）
export VALUE_DOMAIN_API_KEY="your_api_key_here"

# または設定ファイルで永続化
regctl config set value_domain_api_key "your_api_key_here"
```

## 🌐 ドメイン取得実行手順

### 1. ドメイン可用性チェック

```bash
# 単一ドメインチェック
regctl domains check example.com

# 複数TLD一括チェック
regctl domains check example.{com,net,org,jp,dev,io}

# 価格情報付きチェック
regctl domains check example.com --show-price
```

**出力例**:
```
✅ example.com - Available (¥790/year)
❌ google.com - Unavailable
✅ example.dev - Available (¥2,663/year)
💰 premium.com - Available (¥50,000/year) [Premium Domain]
```

### 2. WHOIS情報設定

```bash
# 個人情報設定テンプレート作成
regctl config whois create-template

# テンプレート編集
regctl config whois edit

# 設定確認
regctl config whois show
```

**設定例** (`~/.regctl/whois.yaml`):
```yaml
registrant:
  name: "Taro Yamada"
  organization: "Example Corp"
  address: "1-1-1 Shibuya, Shibuya-ku"
  city: "Tokyo"
  state: "Tokyo"
  postal_code: "150-0002"
  country: "JP"
  phone: "+81.3.1234.5678"
  email: "admin@example.com"
admin:
  # 管理者連絡先（通常は登録者と同じ）
  name: "Taro Yamada"
  email: "admin@example.com"
tech:
  # 技術担当者連絡先
  name: "Tech Support"
  email: "tech@example.com"
```

### 3. ドメイン登録実行

```bash
# 基本的な登録
regctl domains register example.com \
  --registrar value-domain \
  --years 1 \
  --auto-renew true \
  --privacy true

# 詳細オプション付き登録
regctl domains register example.com \
  --registrar value-domain \
  --years 2 \
  --auto-renew true \
  --privacy true \
  --nameservers ns1.example.com,ns2.example.com \
  --confirm
```

**パラメータ説明**:
- `--registrar`: プロバイダー選択（value-domain/route53/porkbun）
- `--years`: 登録年数（1-10年）
- `--auto-renew`: 自動更新有効化
- `--privacy`: WHOIS情報保護サービス
- `--nameservers`: カスタムネームサーバー
- `--confirm`: 確認プロンプトをスキップ

### 4. 登録確認

```bash
# 登録済みドメイン一覧
regctl domains list

# 特定ドメイン詳細
regctl domains info example.com

# DNS設定確認
regctl dns list example.com
```

## 💳 支払い・請求について

### 💰 **支払いフロー**
1. **regctl**でドメイン登録を実行
2. **VALUE-DOMAIN**に登録リクエスト送信
3. **VALUE-DOMAIN**から請求メール受信
4. **クレジットカード**または**銀行振込**で支払い
5. **支払い確認後**にドメイン登録完了

### 📧 **支払い通知**
```bash
# 未払い請求確認
regctl billing list

# 請求詳細表示
regctl billing show invoice-12345

# 支払い状況確認
regctl domains list --filter status=pending-payment
```

### 💡 **支払い方法**

#### 1. **クレジットカード（推奨）**
- VISA、MasterCard、JCB対応
- 即座決済・自動更新対応
- VALUE-DOMAIN管理画面で設定

#### 2. **銀行振込**
- 3営業日以内の振込が必要
- 振込手数料は顧客負担
- 海外送金の場合は追加手数料

#### 3. **プリペイド残高**
- VALUE-DOMAINポイント使用
- 事前チャージ方式

## ⚠️ 重要な注意事項

### 🔒 **セキュリティ**
1. **API Key保護**: 環境変数で管理、GitHubにコミット禁止
2. **2FA設定**: VALUE-DOMAINアカウントで2段階認証有効化
3. **ドメインロック**: 移管防止のため自動ロック有効化

### 📅 **期限管理**
1. **自動更新設定**: 期限切れ防止のため必須
2. **期限監視**: `regctl domains list --expiring 30` で30日以内期限を確認
3. **更新通知**: メール・Slack通知設定

### 🌐 **DNS設定**
```bash
# 基本的なDNS設定例
regctl dns create example.com \
  --type A \
  --name @ \
  --content 192.0.2.1 \
  --ttl 300

regctl dns create example.com \
  --type CNAME \
  --name www \
  --content example.com \
  --ttl 300
```

## 🆘 トラブルシューティング

### ❌ **よくあるエラー**

#### 1. **API認証エラー**
```bash
Error: API authentication failed
```
**解決方法**:
```bash
# API Key再設定
regctl config set value_domain_api_key "new_api_key"

# 認証テスト
regctl auth test
```

#### 2. **ドメイン登録失敗**
```bash
Error: Domain registration failed - insufficient funds
```
**解決方法**:
- VALUE-DOMAINアカウントにクレジットカード登録
- プリペイド残高のチャージ確認

#### 3. **WHOIS情報エラー**
```bash
Error: Invalid registrant information
```
**解決方法**:
```bash
# WHOIS情報再設定
regctl config whois edit

# 英語表記確認（特に住所・氏名）
regctl config whois validate
```

### 📞 **サポート連絡先**

#### regctl Cloud サポート
- **メール**: support@regctl.cloud
- **GitHub Issues**: https://github.com/yukihamada/regctl/issues
- **Discord**: https://discord.gg/regctl

#### VALUE-DOMAIN サポート
- **サポートページ**: https://www.value-domain.com/support/
- **メール**: support@value-domain.com
- **電話**: 050-3684-3371（平日 10:00-17:00）

## 🎯 成功のコツ

### 💡 **ベストプラクティス**
1. **事前準備**: WHOIS情報を英語で正確に準備
2. **テスト実行**: 高額ドメインの前に安価ドメインでテスト
3. **バックアップ**: 複数TLDで同じドメイン名を確保
4. **自動化**: GitHub Actionsで定期的な期限チェック

### 🚀 **効率的な運用**
```bash
# スクリプト例: 複数ドメイン一括管理
#!/bin/bash

DOMAINS=("example.com" "example.jp" "example.dev")

for domain in "${DOMAINS[@]}"; do
  echo "Checking $domain..."
  regctl domains info $domain
  regctl dns list $domain
done
```

---

**📚 関連ドキュメント**:
- [ドメイン価格一覧](./DOMAIN_PRICING.md)
- [DNS管理ガイド](./DNS_MANAGEMENT.md)
- [API リファレンス](./API.md)

**🎉 これで完了！** あなたも regctl Cloud でドメイン管理を始めましょう！