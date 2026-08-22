# go-service-template

Connect (connect-go) ベースの Go マイクロサービス開発用テンプレートリポジトリ

- HTTP サーバ（ヘルスチェック `GET /healthz` + graceful shutdown）
- API Gateway 発行の内部 JWT の検証（JWKS 取得 + ES256、`INTERNAL_JWKS_URL` で JWKS エンドポイントを指定）
- 開発・テスト用の JWT / JWKS 生成 CLI（`cmd/jwtgen`）と mockgen によるモック生成
- OpenTelemetry によるトレーシング（Connect interceptor + OTLP/HTTP exporter。`OTEL_EXPORTER_OTLP_ENDPOINT` 設定時のみ有効）と Jaeger を含む Compose オーバーライド（`compose.o11y.yml`）
- マルチステージ Dockerfile（distroless）とコンテナイメージ公開ワークフロー、Docker Compose（`compose.yml` + `task up:*`）
- `buf` による proto の lint / コード生成（connect-go・connect-es）
- `.proto` を ORAS で OCI アーティファクト化し GitHub Container Registry へ公開するワークフロー
- connect-es クライアントの npm publish ワークフロー
- 生成物の drift-check（connect-go）ワークフロー
- `mise` による開発ツール（buf / go / task）のバージョン固定

---

## テンプレート利用時の初期設定

開発を始める前にプロジェクト向けに変更する必要がある

### 1. Go モジュールパス

テンプレートの `github.com/pj-hoakari/go-service-template` を新しいモジュールパスへ置換

| ファイル | 箇所 | 内容 |
| --- | --- | --- |
| `go.mod` | `module` 行 | モジュールパス |
| `cmd/server/main.go` / `cmd/jwtgen/main.go` | import | `.../internal/*` の参照 |
| `internal/**`（`internal/infra/connect` / `internal/jwks` など） | import | 内部パッケージ間の参照 |
| `buf.gen.go.yaml` | `go_package_prefix` | 生成コードのパッケージ接頭辞 |

生成物（`gen/`）を除いて一括置換

```bash
OLD=github.com/pj-hoakari/go-service-template
NEW=github.com/<owner>/<repo>

# gen/ は再生成で追従
git grep -lz "$OLD" -- ':!gen' | xargs -0 sed -i "s#$OLD#$NEW#g"

go mod tidy
task proto:gen:go
```

### 2. connect-es クライアント（npm パッケージ）

`clients/connect-es` は GitHub Packages に publish される npm クライアント向け実装

| ファイル | 箇所 | 内容 |
| --- | --- | --- |
| `clients/connect-es/package.json` | `name` | `@<scope>/<repo>-client-es` |
| `clients/connect-es/package.json` | `description` | パッケージ説明 |
| `clients/connect-es/package.json` | `repository.url` | 新しいリポジトリ URL |
| `.github/workflows/publish-client-es.yml` | `scope` | `@<owner>` へ（GitHub Packages のスコープと一致） |

`package.json` の `name` を変更し、ロックファイルを同期

```bash
cd clients/connect-es && npm install
```

### 3. Renovate 設定

```bash
mv renovate.example.json renovate.json
```

### 4. その他

- `cmd/server/main.go` / `internal/infra/connect/service.go` のログ文字列 `go-service-template: ...`
- `internal/telemetry/telemetry.go` の `DefaultServiceName` と `compose.o11y.yml` の `OTEL_SERVICE_NAME`（トレースの `service.name` になる）
- `mise.toml` の Go / buf バージョン
    buf の版を変える場合は `.github/workflows/proto-gen-check.yml` の `version:` も揃える
- with-db ブランチと同期用 workflow（`.github/workflows/sync-with-db.yml`）を削除する

---

## 初期設定チェックリスト

- [ ] Go モジュールパスを `github.com/<owner>/<repo>` に置換（`go.mod` / `cmd/**` / `internal/**` / `buf.gen.go.yaml`）
- [ ] `go mod tidy` を実行
- [ ] `task proto:gen:go` で connect-go を再生成し、`gen/` をコミット

- [ ] connect-es の `package.json`（`name` / `description` / `repository.url`）を更新
- [ ] `clients/connect-es` で `npm install` を実行し `package-lock.json` を同期
- [ ] `publish-client-es.yml` の `scope` を `@<owner>` に変更

- [ ] `renovate.json` を `renovate.example.json` の内容で置き換え（example は削除）
- [ ] `sync-with-db.yml` と with-db ブランチを削除（派生リポジトリでは不要）

- [ ] `main.go` / `service.go` のログ文字列
- [ ] `telemetry.DefaultServiceName` と `compose.o11y.yml` の `OTEL_SERVICE_NAME`
- [ ] README のテンプレート説明を書き換え

---

## ディレクトリ構成

レイヤードアーキテクチャを採用している

```
cmd/server/           エントリポイント（依存の組み立て・DI）
internal/
  domain/             ドメインモデル（エンティティ + 検証。他レイヤに依存しない）
  application/        ユースケース（`GreetUseCases` インターフェース + 実装。domain を使う）
  infra/
    connect/          Connect transport（ハンドラ・authz verifier・interceptor。application に依存）
  jwks/               内部 JWT の検証（JWKS 取得 + ES256）
  jwtgen/             開発・テスト用の JWT / JWKS 生成
  telemetry/          OpenTelemetry トレーシングの配線（OTLP/HTTP exporter + W3C propagator）
  tenantctx/          テナント公開 ID の context 注入・検証
gen/                  buf による生成コード（手動編集しない）
```

---

## 開発

### セットアップ
miseを使用して開発環境をセットアップ

```bash
mise trust
mise install
task proto
```

### Docker Compose での起動

Docker Compose で開発サーバーを起動できる

```bash
docker compose up --build
```

もしくは
```bash
task up:build
```

サーバーは `http://localhost:8080` で待ち受ける（停止は `task down`）  
RPC を呼び出すには内部 JWT が必要なので、`cmd/jwtgen` で生成した JWKS を配信する URL を `INTERNAL_JWKS_URL` で `server` に渡す（後述）

### トレースの確認（Jaeger）

監視スタックはオーバーライドファイル `compose.o11y.yml` を重ねたときだけ有効になる  
Jaeger が起動し、`server` に OTLP エクスポート用の環境変数（`OTEL_EXPORTER_OTLP_ENDPOINT` など）がセットされる

```bash
docker compose -f compose.yml -f compose.o11y.yml up --build
```

もしくは
```bash
task up:build:o11y
```

Jaeger UI は `http://localhost:16686`（停止は `task down:o11y`）

トレーシングの配線は `internal/telemetry` にあり、`cmd/server/main.go` の起動時に `telemetry.Setup` で global tracer provider と W3C Trace Context / Baggage propagator を設定する

- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` または `OTEL_EXPORTER_OTLP_ENDPOINT` が設定されているときだけ OTLP/HTTP で span を export する。未設定なら no-op provider で動作し、エラーにはならない（トレーシングはデプロイ環境の opt-in）
- `service.name` は `OTEL_SERVICE_NAME`（デフォルト: `telemetry.DefaultServiceName` = `go-service-template`）
- ヘッダ・TLS・タイムアウトなどその他の `OTEL_EXPORTER_OTLP_*` は exporter がそのまま解釈する
- Connect の RPC は `otelconnect` interceptor（`internal/infra/connect/server.go`）で span になる。API Gateway の背後で動く前提で `WithTrustRemote()` を指定しており、受信した `traceparent` を span link に落とさず親として継続する
- interceptor は authz interceptor の後段に入るため、認証で拒否されたリクエスト（`CodeUnauthenticated` など）は span にならない
- 終了時は `shutdownTimeout` 内でバッファ済み span を flush する

### connect-es の生成
connect-es の生成（`task proto:gen:es`）はリリース時に CI で行う  
ローカルで実行する場合は `clients/connect-es` の依存（`npm i`）を導入する必要がある

### greet service の authz interceptor（内部 JWT 検証）

`Greet` は proto の policy annotation により `AUTH_LEVEL_AUTHENTICATED` と `greeting.read` スコープを要求する  
`internal/infra/connect` では、生成された `NewGreetServiceHandlerWithAuthz` に、API Gateway 発行の内部 JWT（ES256）を JWKS で検証する verifier を渡す（`AUTH_LEVEL_PUBLIC` の RPC は検証をスキップ）

- JWKS の取得先は環境変数 `INTERNAL_JWKS_URL`（デフォルト: `http://gateway:8080/.well-known/jwks.json`）で指定し、取得結果は 5 分間キャッシュされる
- issuer / audience / token_use の期待値は `internal/infra/connect/auth.go` の定数（`api-gateway` / `go-service-template` / `access`）。サービスに合わせて変更する（RPC ごとに token_use を変える場合は procedure 名で分岐する）
- `src_jti`（変換元外部トークンの jti。IdP 監査ログとの相関用）は必須クレームとして非空を検証する
- ハンドラは `NewHandler(greetService)` → `NewHandlerWithJWKSURL(greetService, jwksURL)` → `NewHandlerWithValidator(greetService, validator)` の段階的コンストラクタで構成され（`greetService` は `application.GreetUseCases`。いずれも `(http.Handler, error)` を返し、tracing interceptor の生成に失敗するとエラー）、テストでは validator（mockgen 生成の `MockJWTValidator`）や JWKS URL を差し替えられる

ローカルでの動作確認には `cmd/jwtgen` でトークンと JWKS を生成する

```bash
# ES256 の内部 JWT と対応する JWKS ドキュメントを JSON で出力
go run ./cmd/jwtgen -scope greeting.read -ttl 10m

# フラグ: -issuer / -audience / -token-use / -tenant-public-id / -scope / -kid / -ttl
# -tenant-public-id は任意（空なら tenant_id クレームを省略）
```

出力の `jwks` を任意の HTTP エンドポイント（例: ローカルのファイルサーバ）で配信し、`INTERNAL_JWKS_URL` にその URL を設定すると、`Authorization: Bearer <token>` で呼び出せる

`JWTValidator` インターフェースのモックは `go generate ./...` で再生成する（`go tool mockgen` を使用）

### テナントIDのコンテキスト注入（tenantctx）

内部 JWT の `tenant_id` クレーム（テナントの 16 文字 hex 公開 ID）は、Connect interceptor（`internal/infra/connect/tenant_id_interceptor.go`、`connect.WithInterceptors` で配線）が検証済みクレームから取り出し、`internal/tenantctx` 経由で request context に注入する

- 注入は fail-closed: 全サービスは原則テナント必須のため、`token_use` が `access` かつ `tenant_id` が非空のトークンを持たないリクエストは `CodeUnauthenticated` で拒否する
- テナント非依存の RPC（セルフサインアップ、サービス間呼び出し、PUBLIC エンドポイント等）を持つサービスは、`tenantIDNotRequired` に procedure 名を列挙して除外する
- ハンドラ / ユースケースは `tenantctx.TenantPublicIDFromContext(ctx)` で参照する。テナント対象操作の認可には `tenantctx.Ensure`（fail-closed）、永続化から復元したモデルの防衛的チェックには `tenantctx.VerifyOwnership`（fail-open）を使う
- 参照は `internal/tenantctx` を直接 import する。

### エラー

内部エラーは `internal` と固定メッセージ `internal error` だけを返し、原因はサーバー側のログ（`go-service-template: internal error: ...`）にのみ記録する  
クライアント都合の中断・締め切り超過は `canceled`／`deadline_exceeded` として返してログには記録せず、内部エラーはトレースが有効なときだけログに `trace_id` を添えてトレースと突き合わせられるようにする  
ハンドラの内部エラーは `InternalError(ctx, err)`（`internal/infra/connect/service.go`）で組み立てる（他の transport からも同じ関数を使う）  
エラーメッセージには内部主キー・テナント名・ユーザー ID などの内部識別子を含めない  
`connectrpc.CodeInternal` の直接使用は golangci-lint の `forbidigo` で禁止しており、許可するのは `InternalError` の中だけである

---

## proto アーティファクトの利用

`.proto` は [ORAS](https://oras.land) で OCI アーティファクト化され、GitHub Container Registry に公開される  
アーティファクト名: `ghcr.io/<owner>/<repo>/proto`

### 取得（pull）

[ORAS CLI](https://oras.land/docs/installation) が必要

```bash
# 出力先ディレクトリに proto を展開（ディレクトリ構造が復元される）
oras pull ghcr.io/pj-hoakari/go-service-template-proto:latest -o proto

# 例: proto/greet/v1/greet.proto として展開される
```

取得した `.proto` は `buf` や `protoc` の入力としてそのまま利用できる
