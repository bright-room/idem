# Project Context

## コードベース調査ガイド

### モジュール構成の把握方法

- `go.mod` でモジュール名と依存関係を確認する
- ルートパッケージ（`idem`）、サブモジュール（`gin/`、`redis/`）、内部パッケージ（`internal/`）の構成を把握する
- `gin/` は独自の `go.mod` を持つ別モジュール（`github.com/gin-gonic/gin` 依存の分離のため）

### 既存パターンの調査手順

- **拡張ポイント**: `Storage`、`Locker`、`Validator` インターフェースの実装パターンを確認する
- **Functional Options**: `WithXxx()` 関数による設定パターンを確認する
- **ミドルウェアパターン**: `middleware.go` の `http.Handler` ラッピングパターン
- **コード生成**: `internal/cmd/genrecorder/` が `recorder_gen.go` / `recorder_gen_test.go` を生成する。変更が必要かを確認する

### テスト構成の確認方法

- ユニットテスト: `*_test.go`、`-short` フラグで実行（`make test-unit`）
- 統合テスト: `Integration` prefix のテスト名、Redis 環境が必要（`make test-integration`）
- Docker Compose で Redis Standalone / Cluster / Sentinel 環境を提供

## 実装ガイド

### ビルド・フォーマットコマンド

```bash
# コード生成（internal/cmd/genrecorder/ を変更した場合のみ）
docker compose run --rm dev go generate ./...

# フォーマット（gofumpt。go fmt ではなく必ず make fmt を使うこと）
make fmt

# 静的解析（golangci-lint: errcheck, govet, staticcheck, gocritic, revive 等 11 チェッカー）
make lint

# ユニットテスト（gotestsum + race detector + coverage）
make test-unit

# ビルド確認
docker compose run --rm dev go build ./...
```

**注意**: `go fmt` ではなく `make fmt` を使うこと。gofumpt はより厳密なフォーマットを適用し、`go fmt` では CI が失敗する。同様に `go vet` ではなく `make lint` を使うこと。

### 言語固有の実装規約

- **GoDoc コメントは必ず英語で記述すること**（実装プランに日本語で記載されていても英語に翻訳する）
- **パッケージコメント（`// Package xxx ...`）は `doc.go` に記述すること**（実装ファイルには書かない）
- **`internal/cmd/genrecorder/` を変更した場合は `go generate ./...` を実行し、`recorder_gen.go` と `recorder_gen_test.go` を再生成してコミットに含めること**

### テスト配置ルール

- ユニットテスト: 対象ファイルと同じディレクトリに `*_test.go` として配置
- 統合テスト: 同じ `*_test.go` ファイル内に `Integration` prefix のテスト名で配置
- テーブル駆動テスト、`gotestsum` with testdox output、`-race` flag を使用

### CI に委ねてよい項目

- 統合テスト（Redis Standalone / Cluster / Sentinel 環境が必要）。ローカル実行する場合は `make test-integration`

## レビューガイド

### ファイルパス → カテゴリマッピング

| 変更ファイルのパスパターン | 選択されるカテゴリ |
|--------------------------|-------------------|
| ルート `*.go`（`*_test.go`, `doc.go`, `*_gen.go` を除く） | プロダクトコード |
| 上記のうち公開インターフェース定義（`idem.go`, `storage.go`, `locker.go`, `middleware.go`, `option.go`） | + アーキテクチャ |
| `internal/` | プロダクトコード |
| `redis/`, `gin/`（サブモジュール） | プロダクトコード, セキュリティ |
| `*_gen.go`, `*_gen_test.go` | プロダクトコード（生成コード -- 生成元との同期を確認） |
| `*_test.go` | テストコード |
| `go.mod`, `go.sum`, `.golangci.yml`, `Makefile`, `Dockerfile`, `compose.yml` | ビルド・設定 |
| `**/doc.go` | ドキュメント |
| `*.md`（`.claude/` 配下を除く） | ドキュメント |
| `.claude/skills/`, `.claude/rules/` | ドキュメント |

### カテゴリ別レビュー観点

#### architecture
- 設計パターン、パッケージ分割、依存関係の適切さ
- インターフェースの設計と抽象化の適切さ
- ミドルウェアパターンの一貫性

#### code
- ロジックの正確性、エッジケース
- 命名規則、Go の慣習への準拠
- エラーハンドリングの適切さ
- goroutine / channel の安全な使用

#### test
- テストカバレッジ、テストケースの網羅性
- ユニットテスト / 統合テストの品質
- テストの独立性と再現性

#### security
- 冪等キーの検証とキャッシュポイズニングリスク
- 分散ロックの安全性: TTL 競合、ロック値のランダム性、unlock のアトミック性
- 並行処理の安全性: レースコンディション、goroutine リーク、context キャンセル時のクリーンアップ
- Redis 接続のセキュリティ: タイムアウト設定、TLS、認証
- 機密情報管理: ハードコーディング、ログ出力
- 依存ライブラリの既知脆弱性

#### docs
- GoDoc: 公開 API のドキュメント（英語）
- README / CLAUDE.md の更新
- コード内コメントの正確性

#### build
- go.mod / go.sum の管理
- .golangci.yml、Makefile、Dockerfile の正確性

### セキュリティチェックリスト

| チェック項目 | 結果 | 備考 |
|-------------|:----:|------|
| 冪等キーの検証（長さ制限・形式） | ✅ / ❌ / N/A | |
| キャッシュポイズニング（他ユーザーのキャッシュ応答の返却） | ✅ / ❌ / N/A | |
| 分散ロックの安全性（TTL 競合・ロック値のランダム性） | ✅ / ❌ / N/A | |
| 並行処理の安全性（sync.Map、mutex、レースコンディション） | ✅ / ❌ / N/A | |
| goroutine リーク（context キャンセル時のクリーンアップ） | ✅ / ❌ / N/A | |
| Redis 接続のセキュリティ（タイムアウト、TLS、認証） | ✅ / ❌ / N/A | |
| エラー情報の外部露出（HTTP レスポンスへの内部エラー漏洩） | ✅ / ❌ / N/A | |
| 機密情報のハードコーディング・ログ出力 | ✅ / ❌ / N/A | |
| 依存ライブラリの既知脆弱性 | ✅ / ❌ / N/A | |

### テストカバレッジマトリクス

| 対象パッケージ | 関数/メソッド | ユニットテスト | 統合テスト | 備考 |
|--------------|-------------|:-------------:|:---------:|------|

## プランテンプレート補足

### 影響範囲テーブル

| パッケージ | 影響 | 備考 |
|-----------|------|------|
| `idem`（ルート） | 新規 / 変更 / なし | <概要> |
| `redis/` | 新規 / 変更 / なし | <概要> |
| `internal/cmd/genrecorder/` | 新規 / 変更 / なし | <概要（変更時は go generate が必要）> |

### ファイル構成の記述例

```
.
├── new_file.go              (new)    ← ルートパッケージ (idem)
├── existing_file.go         (modify)
├── redis/
│   └── new_feature.go       (new)
└── internal/
    └── cmd/
        └── genrecorder/
            └── main.go      (modify)  ← 変更時は go generate が必要
```

### テスト戦略テーブル

| テスト種別 | 対象 | テスト内容 | 実行方法 |
|-----------|------|-----------|---------|
| ユニットテスト | `package/func` | <テスト内容> | `make test-unit` |
| 統合テスト | `package/func` | <テスト内容> | `make test-integration`（Redis 環境必要） |

### ドキュメント更新対象

| ドキュメント | 更新条件 |
|-------------|---------|
| `README.md` | 新機能・設定変更・公開 API 変更 |
| `CLAUDE.md` | コマンド変更・CI 変更 |
| `.claude/rules/architecture.md` | パッケージ構成変更・インターフェース変更・設計パターン変更 |
| `.claude/rules/coding.md` | コーディング規約変更 |
| `.claude/skills/references/project-context.md` | パッケージ構成変更・ビルドコマンド変更・レビュー観点変更 |

## ラベル・ワークフロー規約

### Issue/PR ラベルの prefix

`Kind:` prefix を使用する（例: `Kind: Bug Fix`, `Kind: Feature`）。

### コード生成

`recorder_gen.go` と `recorder_gen_test.go` は `go generate ./...` で生成される。`internal/cmd/genrecorder/` を変更した場合は再生成してコミットに含めること。
