#!/bin/bash

# 🚀 regctl CLI インストーラー
# curl -fsSL https://regctl.com/install.sh | bash

set -e

# 色付きメッセージ
red() { printf "\033[31m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }
blue() { printf "\033[34m%s\033[0m\n" "$*"; }
bold() { printf "\033[1m%s\033[0m\n" "$*"; }

# バナー
echo ""
bold "🚀 regctl - CLI-First マルチレジストラー ドメイン管理"
echo ""
blue "   VALUE-DOMAIN | Route 53 | Porkbun"
echo ""

# システム情報取得
OS=$(uname -s)
ARCH=$(uname -m)

case $OS in
    "Darwin")
        if [ "$ARCH" = "arm64" ]; then
            PLATFORM="darwin-arm64"
        else
            PLATFORM="darwin-amd64"
        fi
        ;;
    "Linux")
        if [ "$ARCH" = "x86_64" ]; then
            PLATFORM="linux-amd64"
        elif [ "$ARCH" = "aarch64" ]; then
            PLATFORM="linux-arm64"
        else
            red "❌ サポートされていないアーキテクチャ: $ARCH"
            exit 1
        fi
        ;;
    *)
        red "❌ サポートされていないOS: $OS"
        exit 1
        ;;
esac

echo "🔍 システム検出: $OS $ARCH ($PLATFORM)"

# 最新バージョン取得
yellow "📦 最新バージョン確認中..."
LATEST_VERSION=$(curl -s https://api.github.com/repos/yukihamada/regctl/releases/latest | grep '"tag_name"' | cut -d'"' -f4 2>/dev/null || echo "")

if [ -z "$LATEST_VERSION" ]; then
    # GitHub Releases がない場合は直接バイナリをダウンロード
    LATEST_VERSION="v0.1.0"
    DOWNLOAD_URL="https://regctl.com/releases/regctl-${PLATFORM}"
else
    DOWNLOAD_URL="https://github.com/yukihamada/regctl/releases/download/${LATEST_VERSION}/regctl-${PLATFORM}"
fi

echo "📥 ダウンロード: regctl $LATEST_VERSION"

# インストールディレクトリ
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# ダウンロード
TEMP_FILE=$(mktemp)
if curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
    chmod +x "$TEMP_FILE"
    
    # インストール
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TEMP_FILE" "$INSTALL_DIR/regctl"
        green "✅ インストール完了: $INSTALL_DIR/regctl"
    else
        yellow "🔐 管理者権限が必要です"
        sudo mv "$TEMP_FILE" "$INSTALL_DIR/regctl"
        green "✅ インストール完了: $INSTALL_DIR/regctl"
    fi
else
    red "❌ ダウンロードに失敗しました"
    red "   手動インストール: https://regctl.com/install"
    exit 1
fi

# パス確認
if ! echo $PATH | grep -q "$INSTALL_DIR"; then
    yellow "⚠️ PATH設定が必要です:"
    echo "   export PATH=\"$INSTALL_DIR:\$PATH\""
    echo "   ~/.bashrc または ~/.zshrc に追加してください"
    echo ""
fi

# 動作確認
echo "🔧 動作確認..."
if command -v regctl >/dev/null 2>&1; then
    green "✅ regctl コマンドが利用可能です"
    echo ""
    regctl --version
    echo ""
    bold "🎯 使用方法:"
    echo "   regctl login                      # 初回認証"
    echo "   regctl domains list               # ドメイン一覧"
    echo "   regctl domains register example.com  # ドメイン登録"
    echo "   regctl dns list example.com       # DNS確認"
    echo ""
    blue "📖 詳細: https://docs.regctl.com"
    blue "💬 サポート: https://github.com/yukihamada/regctl/issues"
else
    yellow "⚠️ regctl コマンドが見つかりません"
    echo "   PATH設定を確認してください"
fi

echo ""
green "🎉 regctl インストール完了！"