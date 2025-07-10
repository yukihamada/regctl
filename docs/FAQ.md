# よくある質問 (FAQ)

## 一般的な質問

### Q: regctlとは何ですか？

A: regctlは、複数のドメインレジストラ（VALUE-DOMAIN、Route 53、Porkbunなど）を統一されたCLIインターフェースで管理できるツールです。ドメインの登録、移管、DNS管理などをコマンドラインから簡単に行えます。

### Q: なぜregctlを使うべきですか？

A: 
- **統一インターフェース**: 複数のレジストラを一つのツールで管理
- **自動化**: スクリプトやCI/CDパイプラインに組み込み可能
- **高速**: CLIベースで軽量・高速な操作
- **セキュア**: 認証情報はOSのキーチェーンに安全に保管

### Q: どのレジストラに対応していますか？

A: 現在対応しているレジストラ：
- VALUE-DOMAIN
- AWS Route 53
- Porkbun

今後、Cloudflare Registrar、Google Domains、Namecheapなどの対応も予定しています。

### Q: 料金はかかりますか？

A: regctl自体はオープンソースで無料です。ただし：
- 各レジストラのドメイン登録料金は別途必要です
- セルフホスティングする場合はインフラ費用がかかります
- 将来的にマネージドサービス版を提供する可能性があります

## インストールと設定

### Q: 対応しているOSは？

A: 以下のOSに対応しています：
- macOS (Intel/Apple Silicon)
- Linux (x86_64/ARM64)
- Windows (x86_64)

### Q: インストールできません

A: 以下を確認してください：

1. **権限エラーの場合**:
   ```bash
   sudo curl -fsSL https://regctl.cloud/install.sh | sh
   ```

2. **curlがない場合**:
   ```bash
   wget -qO- https://regctl.cloud/install.sh | sh
   ```

3. **PATHが通っていない場合**:
   ```bash
   echo 'export PATH="$PATH:/usr/local/bin"' >> ~/.bashrc
   source ~/.bashrc
   ```

### Q: 設定ファイルはどこに保存されますか？

A: 
- 設定ファイル: `~/.regctl/config.yaml`
- 認証トークン: OSのキーチェーン（Keychain/Secret Service/Credential Manager）

## 認証

### Q: ログインできません

A: 以下を試してください：

1. **デバイスフロー認証を使用**:
   ```bash
   regctl login --device
   ```

2. **プロキシ環境の場合**:
   ```bash
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   regctl login
   ```

3. **認証情報をリセット**:
   ```bash
   regctl config reset
   regctl login
   ```

### Q: 認証トークンの有効期限は？

A: 
- 通常のログイン: 7日間
- CLIデバイスフロー: 30日間
- 期限が切れたら再度ログインが必要です

## ドメイン管理

### Q: ドメインの移管にかかる時間は？

A: レジストラによって異なりますが、通常5-7日かかります：
1. 移管申請
2. 現在のレジストラでの承認
3. 移管先レジストラでの処理
4. レジストリでの更新

### Q: 移管時の注意点は？

A: 
- ドメインがロックされていないことを確認
- 登録/更新から60日以上経過していること
- WHOISの管理者メールアドレスが有効であること
- 認証コード（EPPコード）を取得していること

### Q: 自動更新の設定は？

A: 
```bash
# 自動更新を有効化
regctl domains update example.com --auto-renew=true

# 自動更新を無効化
regctl domains update example.com --auto-renew=false
```

## DNS管理

### Q: DNSの変更が反映されません

A: DNSの変更には時間がかかります：
1. TTLの確認（デフォルト3600秒 = 1時間）
2. DNSキャッシュのクリア:
   ```bash
   # macOS
   sudo dscacheutil -flushcache
   
   # Linux
   sudo systemctl restart systemd-resolved
   
   # Windows
   ipconfig /flushdns
   ```

3. 伝播の確認:
   ```bash
   dig example.com
   nslookup example.com
   ```

### Q: メールが届かなくなりました

A: MXレコードを確認してください：
```bash
# MXレコードを表示
regctl dns list example.com --type MX

# MXレコードが削除された場合は追加
regctl dns add example.com --type MX --name @ --content mail.example.com --priority 10
```

## トラブルシューティング

### Q: "API error: 429 Too Many Requests"

A: レート制限に達しています：
- 認証済みユーザー: 100リクエスト/分
- 未認証ユーザー: 10リクエスト/分

少し待ってから再試行してください。

### Q: "Error: not authenticated"

A: 認証が必要です：
```bash
regctl login --device
```

### Q: デバッグ情報を見たい

A: デバッグモードを有効化：
```bash
# 一時的に有効化
regctl --debug domains list

# 永続的に有効化
regctl config set debug true
```

## セキュリティ

### Q: APIキーはどこに保存されますか？

A: 
- macOS: Keychain Access
- Linux: Secret Service (gnome-keyring)
- Windows: Credential Manager

設定ファイルには保存されません。

### Q: 2要素認証は対応していますか？

A: 現在は未対応ですが、今後の実装を予定しています。

## セルフホスティング

### Q: 自分のサーバーで動かせますか？

A: はい、完全にオープンソースなのでセルフホスティング可能です。
詳細は[セルフホスティングガイド](SELF_HOSTING.md)を参照してください。

### Q: 必要なCloudflareのサービスは？

A: 
- Workers（必須）
- D1 Database（必須）
- KV Storage（必須）
- Durable Objects（必須）
- R2 Storage（オプション）

## その他

### Q: バグを見つけました

A: [GitHub Issues](https://github.com/yukihamada/regctl/issues)で報告してください。

### Q: 機能リクエストがあります

A: [GitHub Discussions](https://github.com/yukihamada/regctl/discussions)で提案してください。

### Q: コントリビュートしたい

A: 大歓迎です！[コントリビューションガイド](../CONTRIBUTING.md)をご覧ください。

### Q: 商用利用は可能ですか？

A: はい、MITライセンスなので商用利用も可能です。