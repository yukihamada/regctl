# Purpose
- Build **regctl Cloud** — a “CLI‑First ✕ Multi‑Registrar API Proxy SaaS” that lets DevOps teams manage domains/DNS/Edge CDN entirely from the command line.
- Phase 1 (MVP): wrap VALUE‑DOMAIN / Route 53 / Porkbun with a unified REST+gRPC API and OSS CLI.
- Goal: break‑even MRR ￥6 M by 2026‑06, then gradually in‑house registrar functions.

# Tech Stack (locked versions)
- Go 1.23           # CLI, API Gateway, workers
- TypeScript 5.4    # Cloudflare Workers (edge proxy)
- Supabase 1.49     # Postgres + Auth + Storage
- NATS JetStream    # Event bus (audit logs, webhooks)
- Rust 1.79         # Edge DNS PoP microservice
- Terraform 1.9 + Terragrunt   # IaC
- Fly.io Anycast PoP / Cloudflare R2 (object store)

# Directory Anatomy
- `/cmd/regctl`           : OSS CLI (Go + cobra)
- `/cmd/api-gw`           : Go API Gateway (REST/gRPC)
- `/edge/workers`         : CF Workers Typescript source
- `/internal/providers`   : VALUE‑DOMAIN, AWS, Porkbun wrappers
- `/deploy/terraform`     : IaC modules (supabase, edge, monitoring)
- `/docs`                 : design docs, draw.io diagrams

# Canonical Commands
- `make dev`              # hot‑reload API + CLI
- `make lint`             # go vet, golangci‑lint, eslint
- `make test`             # unit + integration (Docker compose)
- `make e2e`              # spin up local PoP + run CLI smoke tests
- `make release v=x.y.z`  # cross‑compile binaries & publish GitHub release

# Code Style
- Go: `go fmt`, `revive`; TS: `eslint --fix`
- API structs use **snake_case** JSON; internal var names **camelCase**.
- Return errors, never `panic()` outside `main`.

# AI Collaboration Workflow
1. **/plan → /code**   : Every task starts with a plan checklist.
2. Claude writes failing tests first (TDD) in `/test/…`.
3. After each Edit: `make lint && make test`. Tests must pass.
4. PRs >200 LOC require Claude‑generated review using `/diff summary`.
5. When green: `claude /deploy staging` triggers GitHub Action.

# Permissions & Safety
- Allowed tools: `Edit`, `Write`, `Bash(make* git* go* npm*)`.
- Deny: `Bash(curl https://*prod*)` unless `--dry‑run`.
- Never edit `/internal/providers/aws/generated/`.
- SQL migrations must append file & changelog in `/deploy/migrations`.

# Secrets & Env
- Secrets live in 1Password + `.envrc` (direnv). Claude must `source` them; **never** paste into chat.
- Local API keys stored via OS keychain (Keychain/secret‑tool/DPAPI).

# Extended Thinking Budget
- Use `think hard` for multi‑provider abstractions.
- Use `ultrathink` when designing auth/crypto.

# Glossary
- **Provider** = Upstream registrar API (VALUE‑DOMAIN etc.)
- **Edge PoP** = Anycast DNS+CDN micro POP on Fly.io
- **RBAC** = Role‑based access control (JSON policy)

# Claude Code 7つのルール
1. まず、問題を徹底的に考え、コードベースで関連ファイルを読み、tasks/todo.mdに計画を書き出します。
2. 計画には、完了したらチェックマークを付けられるToDo項目のリストを含めます。
3. 作業を始める前に、私に連絡を取り、計画を確認します。
4. 次に、ToDo項目に取り組み始め、完了したら完了マークを付けます。
5. 各ステップごとに、どのような変更を加えたのか、概要を説明してください。
6. タスクやコード変更はできる限りシンプルにしてください。大規模で複雑な変更は避けたいと考えています。変更はコードへの影響を最小限に抑えるべきです。シンプルさが何よりも重要です。
7. 最後に、todo.mdファイルにレビューセクションを追加し、変更内容の概要とその他の関連情報を記載してください。

# TODO seed
- [ ] MVP: VALUE‑DOMAIN domain list → unified `/v1/domains` endpoint
- [ ] CLI: `regctl login` (Supabase OAuth device flow)
- [ ] Billing: Stripe usage metering PoC
- [ ] Edge DNS: deploy Rust DoH microservice to `nrt` PoP
- [ ] GitHub Action: `regctl-action@v0` initial release
