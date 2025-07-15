# Changelog

All notable changes to regctl Cloud will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-07-14 🚀

### 🚀 MVP Launch - AI開発者向けドメイン管理SaaS

#### Added
- **新しいランディングページ**: AI開発者向けに完全リニューアル
  - Claude Code統合を前面に打ち出したデザイン
  - インタラクティブなデモタブ (CLI/JSON API/Claude Code)
  - ポイント制料金体系の詳細説明
  - ワンライン インストール&登録機能

- **A/Bテストシステム**: Cloudflare Workers によるエッジレベル実装
  - ヒーロCTA文言のテスト (4バリアント)
  - 料金プランの表示順序テスト (3バリアント)  
  - デモタブのデフォルト選択テスト (3バリアント)
  - Google Analytics 4 連携

- **SEO最適化**: 
  - JSON-LD構造化データ実装
  - サイトマップ&robots.txt
  - Core Web Vitals最適化
  - ターゲットキーワード対応

- **マーケティング基盤**:
  - コンテンツマーケティング戦略策定
  - 技術ブログプラットフォーム準備
  - Developer-Firstリードジェネレーション設計

#### Core Features
- **Cloudflare Workers API**: 本番環境でのAPI稼働
  - VALUE-DOMAIN, Route53, Porkbun 統合
  - JSON出力対応
  - バッチ処理機能
  - 認証・認可システム

- **regctl CLI**: AI開発最適化
  - ワンライン インストール
  - 確認なし実行 (`-y` フラグ)
  - 環境変数による自動化
  - ポイント制課金システム

- **ダッシュボード**: React/TypeScript実装
  - ドメイン一覧・管理
  - DNS設定インターフェース
  - 使用量・請求管理
  - API キー管理

#### Infrastructure
- **Cloudflare Pages**: 静的サイトホスティング
  - regctl.com (メインサイト)
  - docs.regctl.com (ドキュメント)
  - app.regctl.com (ダッシュボード)

- **監視・分析**:
  - Google Analytics 4
  - Cloudflare Analytics
  - A/Bテスト結果追跡

#### Security
- JWT-based authentication
- PBKDF2 password hashing
- Role-based access control
- API rate limiting

#### Target Metrics
- **Month 1**: DAU 100+, 新規登録率 15%
- **Month 3**: MRR ¥150,000+, 課金ユーザー 500+
- **Month 6**: Break-even ¥500,000, Enterprise 10社

## [0.9.0] - 2025-07-13 (Pre-launch)

### Added
- Initial project structure
- Cloudflare Workers-based API Gateway
- CLI tool (regctl) with device flow authentication
- Multi-registrar support (VALUE-DOMAIN, Route 53, Porkbun)
- Domain management (list, register, transfer, update)
- DNS record management (CRUD operations)
- D1 database schema for data persistence
- KV-based session caching
- Durable Objects for rate limiting
- Comprehensive API documentation
- Development setup scripts

## [0.1.0] - 2024-12-01 (Development Start)

### Added
- プロジェクト開始
- アーキテクチャ設計
- 技術スタック選定

---

## Release Types

- **Major** (X.0.0) - Breaking API changes
- **Minor** (0.X.0) - New features, backwards compatible
- **Patch** (0.0.X) - Bug fixes, backwards compatible