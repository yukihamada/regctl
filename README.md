# regctl Cloud

<p align="center">
  <img src="site/img/favicon.svg" alt="regctl logo" width="120" height="120">
</p>

<p align="center">
  <strong>CLI-First Multi-Registrar Management Tool</strong><br>
  複数のドメインレジストラを統一されたCLIで管理
</p>

<p align="center">
  <a href="https://github.com/yukihamada/regctl/actions/workflows/ci.yml">
    <img src="https://github.com/yukihamada/regctl/actions/workflows/ci.yml/badge.svg" alt="CI Status">
  </a>
  <a href="https://github.com/yukihamada/regctl/releases">
    <img src="https://img.shields.io/github/v/release/yukihamada/regctl" alt="Latest Release">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
  </a>
</p>

## ✨ Features

- 🌐 **マルチレジストラ対応** - VALUE-DOMAIN、Route 53、Porkbunを一つのインターフェースで管理
- 🚀 **高速なCLI** - Go言語で実装された軽量・高速なコマンドラインツール
- 🔄 **簡単なドメイン移管** - レジストラ間の移管をコマンド一つで実行
- 📝 **DNS管理** - すべてのDNSレコードをCLIから操作
- 🔐 **セキュアな認証** - OAuth 2.0デバイスフロー対応、OSキーチェーン統合
- ⚡ **エッジで動作** - Cloudflare Workersによる低レイテンシーAPI

## 🚀 Quick Start

### インストール

**macOS/Linux:**
```bash
curl -fsSL https://regctl.cloud/install.sh | sh
```

**Windows (PowerShell):**
```powershell
iwr -useb https://regctl.cloud/install.ps1 | iex
```

**Homebrew:**
```bash
brew install regctl
```

### 基本的な使い方

```bash
# ログイン
regctl login

# ドメイン一覧を表示
regctl domains list

# 新しいドメインを登録
regctl domains register example.com --registrar porkbun

# DNSレコードを追加
regctl dns add example.com --type A --name @ --content 192.0.2.1

# ドメインを移管
regctl domains transfer example.com --from value-domain --to route53
```

## 📖 Documentation

- [API リファレンス](docs/API.md) - REST APIの詳細仕様
- [アーキテクチャ](docs/ARCHITECTURE.md) - システム設計の詳細
- [セルフホスティング](docs/SELF_HOSTING.md) - 独自環境での運用方法
- [コントリビューション](CONTRIBUTING.md) - 開発への参加方法

## 🏗️ Architecture

```
┌─────────────┐
│   regctl    │  CLI Tool (Go)
└──────┬──────┘
       │
┌──────▼──────┐
│ Cloudflare  │  API Gateway
│   Workers   │  (TypeScript/Hono.js)
└──────┬──────┘
       │
┌──────┴───────────┬──────────────┐
│  Cloudflare D1   │ Cloudflare   │
│   (Database)     │ KV (Cache)   │
└──────────────────┴──────────────┘
```

## 🛠️ Development

```bash
# リポジトリをクローン
git clone https://github.com/yukihamada/regctl
cd regctl

# セットアップ
./scripts/setup.sh

# 開発サーバーを起動
make dev

# テストを実行
make test

# ビルド
make release v=1.0.0
```

### 必要な環境

- Go 1.23+
- Node.js 18+
- Cloudflareアカウント

## 🤝 Contributing

プルリクエストを歓迎します！[コントリビューションガイド](CONTRIBUTING.md)をご確認ください。

1. このリポジトリをフォーク
2. フィーチャーブランチを作成 (`git checkout -b feature/amazing-feature`)
3. 変更をコミット (`git commit -m 'feat: add amazing feature'`)
4. ブランチにプッシュ (`git push origin feature/amazing-feature`)
5. プルリクエストを作成

## 📝 License

このプロジェクトはMITライセンスの下で公開されています。詳細は[LICENSE](LICENSE)ファイルをご覧ください。

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=yukihamada/regctl&type=Date)](https://star-history.com/#yukihamada/regctl&Date)

## 📞 Support

- [GitHub Issues](https://github.com/yukihamada/regctl/issues) - バグ報告や機能リクエスト
- [GitHub Discussions](https://github.com/yukihamada/regctl/discussions) - 質問やディスカッション

---

<p align="center">
  Built with ❤️ using <a href="https://workers.cloudflare.com">Cloudflare Workers</a>
</p>