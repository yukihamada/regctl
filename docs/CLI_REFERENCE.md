# regctl CLI Reference

完全なCLIコマンドリファレンス

## Global Options

すべてのコマンドで使用可能なオプション：

```
--config string     設定ファイルのパス (default: $HOME/.regctl/config.yaml)
--debug            デバッグモードを有効化
--api-url string   APIエンドポイントURL (default: https://api.regctl.cloud)
```

## Commands

### regctl login

regctl cloudに認証します。

```bash
regctl login [flags]
```

**Flags:**
- `-e, --email string` - メールアドレス
- `-p, --password string` - パスワード
- `--device` - デバイスフロー認証を使用

**Examples:**
```bash
# インタラクティブログイン
regctl login

# メール/パスワードでログイン
regctl login -e user@example.com -p password

# デバイスフローでログイン（推奨）
regctl login --device
```

### regctl domains

ドメインを管理します。

#### regctl domains list

所有するドメインの一覧を表示します。

```bash
regctl domains list [flags]
```

**Flags:**
- `-r, --registrar string` - レジストラでフィルタ
- `-s, --status string` - ステータスでフィルタ
- `-p, --page int` - ページ番号 (default: 1)
- `-l, --limit int` - 1ページあたりの件数 (default: 20)

**Examples:**
```bash
# すべてのドメインを表示
regctl domains list

# VALUE-DOMAINのドメインのみ表示
regctl domains list --registrar value-domain

# アクティブなドメインのみ表示
regctl domains list --status active
```

#### regctl domains info

ドメインの詳細情報を表示します。

```bash
regctl domains info <domain> [flags]
```

**Examples:**
```bash
regctl domains info example.com
```

#### regctl domains register

新しいドメインを登録します。

```bash
regctl domains register <domain> [flags]
```

**Flags:**
- `-r, --registrar string` - 使用するレジストラ (default: "value-domain")
- `-y, --years int` - 登録年数 (default: 1)
- `--auto-renew` - 自動更新を有効化 (default: true)
- `--privacy` - WHOISプライバシーを有効化 (default: true)

**Examples:**
```bash
# デフォルト設定で登録
regctl domains register mynewdomain.com

# Porkbunで3年間登録
regctl domains register mynewdomain.com --registrar porkbun --years 3
```

#### regctl domains transfer

ドメインをレジストラ間で移管します。

```bash
regctl domains transfer <domain> [flags]
```

**Flags:**
- `-t, --to string` - 移管先レジストラ (required)
- `--auth-code string` - 認証コード（省略時は自動取得）

**Examples:**
```bash
# VALUE-DOMAINからRoute53へ移管
regctl domains transfer example.com --to route53

# 認証コード指定で移管
regctl domains transfer example.com --to porkbun --auth-code ABC123
```

#### regctl domains update

ドメイン設定を更新します。

```bash
regctl domains update <domain> [flags]
```

**Flags:**
- `--auto-renew` - 自動更新の有効/無効
- `--privacy` - WHOISプライバシーの有効/無効
- `--nameservers strings` - ネームサーバーを更新

**Examples:**
```bash
# 自動更新を無効化
regctl domains update example.com --auto-renew=false

# ネームサーバーを変更
regctl domains update example.com --nameservers ns1.custom.com,ns2.custom.com
```

### regctl dns

DNSレコードを管理します。

#### regctl dns list

DNSレコードの一覧を表示します。

```bash
regctl dns list <domain> [flags]
```

**Flags:**
- `-t, --type string` - レコードタイプでフィルタ

**Examples:**
```bash
# すべてのDNSレコードを表示
regctl dns list example.com

# Aレコードのみ表示
regctl dns list example.com --type A
```

#### regctl dns add

DNSレコードを追加します。

```bash
regctl dns add <domain> [flags]
```

**Flags:**
- `-t, --type string` - レコードタイプ (required)
- `-n, --name string` - レコード名 (required)
- `-c, --content string` - レコード値 (required)
- `--ttl int` - TTL秒数 (default: 3600)
- `--priority int` - 優先度（MX/SRVレコード用）
- `--proxied` - Cloudflareプロキシを有効化

**Examples:**
```bash
# Aレコードを追加
regctl dns add example.com --type A --name @ --content 192.0.2.1

# MXレコードを追加
regctl dns add example.com --type MX --name @ --content mail.example.com --priority 10

# CNAMEレコードを追加（プロキシ有効）
regctl dns add example.com --type CNAME --name www --content example.com --proxied
```

#### regctl dns update

DNSレコードを更新します。

```bash
regctl dns update <domain> <record-id> [flags]
```

**Flags:**
- `-c, --content string` - 新しい値
- `--ttl int` - 新しいTTL
- `--priority int` - 新しい優先度
- `--proxied` - プロキシの有効/無効

**Examples:**
```bash
# IPアドレスを変更
regctl dns update example.com dns_123 --content 192.0.2.2

# TTLを変更
regctl dns update example.com dns_123 --ttl 300
```

#### regctl dns delete

DNSレコードを削除します。

```bash
regctl dns delete <domain> <record-id> [flags]
```

**Flags:**
- `-f, --force` - 確認をスキップ

**Examples:**
```bash
# 確認付きで削除
regctl dns delete example.com dns_123

# 確認なしで削除
regctl dns delete example.com dns_123 --force
```

#### regctl dns import

ゾーンファイルからDNSレコードをインポートします。

```bash
regctl dns import <domain> [flags]
```

**Flags:**
- `-f, --file string` - ゾーンファイルのパス (required)

**Examples:**
```bash
regctl dns import example.com --file zone.txt
```

### regctl config

設定を管理します。

#### regctl config show

現在の設定を表示します。

```bash
regctl config show
```

#### regctl config set

設定値を変更します。

```bash
regctl config set <key> <value>
```

**Available Keys:**
- `api_url` - APIエンドポイントURL
- `debug` - デバッグモード (true/false)
- `default_registrar` - デフォルトレジストラ
- `output_format` - 出力形式 (table/json/yaml)

**Examples:**
```bash
# APIエンドポイントを変更
regctl config set api_url https://api.myregctl.com

# デバッグモードを有効化
regctl config set debug true
```

#### regctl config get

設定値を取得します。

```bash
regctl config get <key>
```

**Examples:**
```bash
regctl config get api_url
```

#### regctl config reset

設定をリセットします。

```bash
regctl config reset [flags]
```

**Flags:**
- `--all` - すべての設定をリセット（認証情報も含む）

**Examples:**
```bash
# 認証情報のみクリア
regctl config reset

# すべての設定をリセット
regctl config reset --all
```

### regctl version

バージョン情報を表示します。

```bash
regctl version
```

### regctl completion

シェル補完スクリプトを生成します。

```bash
regctl completion <shell>
```

**Supported Shells:**
- bash
- zsh
- fish
- powershell

**Examples:**
```bash
# Bashの補完を有効化
source <(regctl completion bash)

# Zshの補完を有効化
source <(regctl completion zsh)
```

## Environment Variables

環境変数で設定を上書きできます：

- `REGCTL_API_URL` - APIエンドポイントURL
- `REGCTL_DEBUG` - デバッグモード
- `REGCTL_CONFIG` - 設定ファイルパス

## Exit Codes

- `0` - 成功
- `1` - 一般的なエラー
- `2` - 誤った使用方法
- `3` - 認証エラー
- `4` - APIエラー
- `5` - ネットワークエラー