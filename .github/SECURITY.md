# Security Policy

## Supported Versions

以下のバージョンについてセキュリティアップデートを提供しています：

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

セキュリティ脆弱性を発見した場合は、責任ある開示にご協力ください。

### 報告方法

1. **公開しないでください** - GitHubのIssueやDiscussionsには投稿しないでください
2. **メールで報告** - security@regctl.cloud にメールを送信してください
3. **暗号化** - 可能であれば、GPGで暗号化してください（公開鍵は[こちら](https://keybase.io/yukihamada)）

### 報告に含めてほしい情報

- 脆弱性の種類（XSS、SQLインジェクション、認証バイパスなど）
- 影響を受けるコンポーネント（CLI、API、特定のエンドポイントなど）
- 再現手順
- 影響の範囲（情報漏洩、権限昇格、サービス拒否など）
- 可能であれば、修正案やパッチ

### 対応プロセス

1. **受領確認** - 48時間以内に受領確認のメールを送信します
2. **初期評価** - 7日以内に初期評価を行い、深刻度を判定します
3. **修正開発** - 深刻度に応じて修正を開発します
   - Critical: 24時間以内
   - High: 7日以内
   - Medium: 30日以内
   - Low: 90日以内
4. **通知** - 修正がリリースされたら報告者に通知します
5. **公開** - 修正リリース後、脆弱性情報を公開します

### 報奨金プログラム

現在、報奨金プログラムは実施していませんが、貢献者としてクレジットさせていただきます。

### セキュリティのベストプラクティス

regctlを使用する際は、以下のセキュリティプラクティスに従ってください：

1. **最新版を使用** - 常に最新のバージョンを使用してください
2. **APIキーの保護** - APIキーを安全に管理し、コミットしないでください
3. **HTTPS使用** - API通信は常にHTTPSを使用してください
4. **最小権限の原則** - 必要最小限の権限のみを付与してください

## セキュリティ関連の更新

セキュリティ関連の更新情報は以下で確認できます：

- [GitHub Security Advisories](https://github.com/yukihamada/regctl/security/advisories)
- [Release Notes](https://github.com/yukihamada/regctl/releases)

## 連絡先

セキュリティチーム: security@regctl.cloud

GPG Fingerprint: `[Your GPG Fingerprint]`