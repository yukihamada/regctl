# Claude Codeでドメイン管理を完全自動化する方法

## はじめに

AI開発者の皆さん、ドメイン管理に時間を取られていませんか？

プロジェクトごとにドメインを手動登録し、DNS設定を一つずつ行い、環境ごとにサブドメインを設定する...これらの作業は本来、完全に自動化できるはずです。

今回は、**regctl** を使ってClaude Codeでドメイン管理を完全自動化する実践的な方法をご紹介します。

## 🤖 Claude Codeに最適化されたregctl

regctlは、AI開発ワークフローを念頭に置いて設計されたドメイン管理CLIです。

### 主な特徴

- **JSON出力**: 構造化されたデータでAI処理しやすい
- **確認なし実行**: `-y` フラグで完全自動化
- **環境変数認証**: 設定ファイル不要
- **バッチ処理**: 複数ドメインの一括操作

```bash
# Claude Codeでこのように記述するだけ
regctl domains register $PROJECT_DOMAIN -y --json
```

## 🚀 セットアップ: 3秒で始める

### 1. ワンライン インストール

```bash
curl -fsSL https://regctl.com/install.sh | bash
```

### 2. 環境変数設定

```bash
# Claude Codeでの自動化に必要な設定
export REGCTL_AUTO_CONFIRM="true"
export REGCTL_OUTPUT="json"
export REGCTL_API_TOKEN="your_token"
```

### 3. 即座にドメイン登録

```bash
# プロジェクト名からドメインを自動生成・登録
PROJECT_NAME="ai-assistant"
DOMAIN="${PROJECT_NAME}.com"

regctl domains register $DOMAIN -y --json
```

## 💻 実践例: Claude Codeでの完全自動ワークフロー

### シナリオ: AIアシスタントアプリのデプロイ

Claude Codeで以下のようなプロンプトを実行できます：

```markdown
新しいAIアシスタントプロジェクト "smart-helper" をデプロイしたい。
以下を自動実行してください：

1. smart-helper.com ドメインを登録
2. 開発環境用に dev.smart-helper.com サブドメインを設定
3. API用に api.smart-helper.com を設定
4. DNS設定を確認

費用と実行ログをJSON形式で出力してください。
```

### Claude Codeが生成するスクリプト

```bash
#!/bin/bash
set -e

PROJECT="smart-helper"
DOMAIN="${PROJECT}.com"
SERVER_IP="203.0.113.1"

echo "🚀 ${PROJECT} プロジェクトのドメイン設定開始"

# 1. メインドメイン登録
echo "📋 ドメイン登録: ${DOMAIN}"
REGISTER_RESULT=$(regctl domains register $DOMAIN -y --json)
echo "✅ 登録完了: $(echo $REGISTER_RESULT | jq -r '.domain.name')"

# 2. サブドメイン設定
echo "🔧 サブドメイン設定中..."

# 開発環境
regctl dns create $DOMAIN A dev $SERVER_IP -y --json
echo "✅ dev.${DOMAIN} → ${SERVER_IP}"

# API エンドポイント
regctl dns create $DOMAIN A api $SERVER_IP -y --json
echo "✅ api.${DOMAIN} → ${SERVER_IP}"

# 3. 費用確認
BALANCE=$(regctl balance --json)
echo "💰 現在残高: $(echo $BALANCE | jq -r '.balance') ポイント"
echo "📊 月額費用: $(echo $BALANCE | jq -r '.monthly_fee') ポイント"

# 4. 設定確認
echo "🔍 DNS設定確認:"
regctl dns list $DOMAIN --json | jq '.records[] | "\\(.name): \\(.type) → \\(.content)"'

echo "🎉 ${PROJECT} プロジェクトのドメイン設定完了！"
```

### 実行結果の例

```json
{
  "project": "smart-helper",
  "domain": "smart-helper.com",
  "status": "completed",
  "costs": {
    "registration": "¥790",
    "monthly_points": 10
  },
  "dns_records": [
    {
      "name": "dev.smart-helper.com",
      "type": "A",
      "content": "203.0.113.1"
    },
    {
      "name": "api.smart-helper.com", 
      "type": "A",
      "content": "203.0.113.1"
    }
  ],
  "balance": {
    "current": 290,
    "monthly_cost": 10
  }
}
```

## 🔄 GitHub Actions連携

### .github/workflows/domain-setup.yml

```yaml
name: Domain Setup
on:
  workflow_dispatch:
    inputs:
      domain:
        description: 'Domain to register'
        required: true
        type: string

jobs:
  setup-domain:
    runs-on: ubuntu-latest
    steps:
      - name: Setup regctl
        run: |
          curl -fsSL https://regctl.com/install.sh | bash
          echo "$HOME/.regctl/bin" >> $GITHUB_PATH
      
      - name: Register domain
        env:
          REGCTL_API_TOKEN: ${{ secrets.REGCTL_TOKEN }}
          REGCTL_AUTO_CONFIRM: "true"
          REGCTL_OUTPUT: "json"
        run: |
          # ドメイン登録
          regctl domains register ${{ inputs.domain }} -y --json > registration.json
          
          # DNS設定
          regctl dns create ${{ inputs.domain }} A @ ${{ secrets.SERVER_IP }} -y
          regctl dns create ${{ inputs.domain }} A api ${{ secrets.SERVER_IP }} -y
          regctl dns create ${{ inputs.domain }} A www ${{ secrets.SERVER_IP }} -y
          
          # 結果をアーティファクトとして保存
          cat registration.json
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: domain-setup-results
          path: registration.json
```

## 🧠 高度な自動化テクニック

### 1. プロジェクト設定からの自動推論

```bash
# package.jsonからプロジェクト情報を読み取り
PROJECT_NAME=$(jq -r '.name' package.json)
DOMAIN="${PROJECT_NAME//_/-}.com"

# 既存ドメインチェック
if regctl domains list --json | jq -e ".domains[] | select(.name == \"$DOMAIN\")" > /dev/null; then
    echo "⚠️ $DOMAIN は既に管理中です"
else
    echo "🆕 $DOMAIN を新規登録します"
    regctl domains register $DOMAIN -y --json
fi
```

### 2. 環境別サブドメイン管理

```bash
# 環境リスト
ENVIRONMENTS=("dev" "staging" "prod")
BASE_DOMAIN="smart-helper.com"

for ENV in "${ENVIRONMENTS[@]}"; do
    SUBDOMAIN="${ENV}.${BASE_DOMAIN}"
    
    # 環境変数から対応IPを取得
    IP_VAR="IP_${ENV^^}"
    IP=${!IP_VAR}
    
    if [ -n "$IP" ]; then
        regctl dns create $BASE_DOMAIN A $ENV $IP -y --json
        echo "✅ $SUBDOMAIN → $IP"
    fi
done
```

### 3. コスト監視と自動アラート

```bash
# 残高チェック
BALANCE=$(regctl balance --json | jq -r '.balance')
THRESHOLD=100

if [ $BALANCE -lt $THRESHOLD ]; then
    echo "⚠️ ポイント残高が不足しています: $BALANCE ポイント"
    
    # Slack通知 (webhookがある場合)
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"regctl残高不足: ${BALANCE}ポイント\"}" \
            $SLACK_WEBHOOK
    fi
    
    exit 1
fi
```

## 📊 コスト効率の最大化

### ポイントシステムの活用

regctlのポイント制課金により、以下のコスト最適化が可能です：

```bash
# 月次コスト分析
regctl balance --json | jq '{
    current_balance: .balance,
    monthly_cost: .monthly_fee,
    domains_count: .domains_count,
    cost_per_domain: (.monthly_fee / .domains_count),
    estimated_days: (.balance / (.monthly_fee / 30))
}'
```

### 出力例
```json
{
  "current_balance": 290,
  "monthly_cost": 30,
  "domains_count": 3,
  "cost_per_domain": 10,
  "estimated_days": 290
}
```

## 🎯 まとめ

regctlとClaude Codeの組み合わせにより、ドメイン管理を完全に自動化できます：

- **開発効率**: 手動作業をゼロに
- **コスト管理**: 透明性の高い従量課金  
- **スケーラビリティ**: 大量ドメインの一括管理
- **信頼性**: JSON出力によるエラーハンドリング

次回は、「GitHub Actionsでドメイン登録を自動化」について詳しく解説します。

---

## 今すぐ始める

```bash
# 3秒でインストール
curl -fsSL https://regctl.com/install.sh | bash

# 初回ドメイン登録
regctl domains register your-project.com -y
```

**regctl Cloud**: AI開発者のためのドメイン管理SaaS  
https://regctl.com

#AI開発 #Claude Code #ドメイン管理 #自動化 #DevOps