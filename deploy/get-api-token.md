# 🔑 Cloudflare APIトークン取得手順 (30秒)

現在ブラウザでCloudflare APIトークンページが開いています。

## 手順:

1. **"Create Token"** をクリック
2. **"Custom token"** → **"Get started"** をクリック  
3. **権限設定**:
   - Token name: `regctl-api`
   - Permissions:
     - `Zone` : `Zone` : `Edit`
     - `Zone` : `DNS` : `Edit`
     - `Account` : `Account` : `Read`
4. **Account Resources**: `Include` - `All accounts`
5. **Zone Resources**: `Include` - `All zones`
6. **"Continue to summary"** → **"Create Token"**
7. **生成されたトークンをコピー**

## 取得後、即座に実行:

```bash
export CLOUDFLARE_API_TOKEN="your_token_here"
./deploy/cloudflare-auto-setup.sh
```

これで完全自動化が開始されます！ 🚀