---
name: implement
description: 指定された Markdown（実装プラン等）を読み込み、内容に基づいてコードを実装する。ファイルパスで実装ソースを指定する。
argument-hint: "[markdown-file-path] [--branch <branch-name>]"
---

# Implement Skill

指定された Markdown ファイルを読み込み、その内容に基づいてコードを実装する。

## 前提条件

- `gh` CLI が認証済みであること
- Docker が起動していること（ビルド・テストは Docker 内で実行される）
- `make build` で Docker イメージがビルド済みであること

## 引数

```
$ARGUMENTS = <markdown-file-path> [--branch <branch-name>]
```

- `<markdown-file-path>`: 実装の元となる Markdown ファイルのパス（必須）
- `--branch <branch-name>`: 作業ブランチの指定（任意）

引数なしの場合はエラーとする。

### ブランチの動作

| 引数 | 動作 |
|------|------|
| `--branch` なし | Markdown の内容からブランチ名を自動生成し、`main` から新規ブランチを作成。実装完了後に PR を作成する |
| `--branch <existing-branch>` | 指定されたブランチにチェックアウトし、そのブランチ上で修正を実施。実装完了後に Push する（PR は作成しない） |

## 手順

### 1. 引数の解析

- `--branch` オプションがあればブランチ名を取得する
- 残りの引数から Markdown ファイルパスを取得する
- ファイルが存在しない場合はエラーメッセージを出力して終了する

### 2. 実装ソースの理解

取得したソースを深く理解する。

- **実装プランの場合**: Phase / Step の構成、対象ファイル、変更内容を把握する
- **レビュー指摘の場合**: 指摘事項、修正案、対象ファイル・行番号を把握する
- **その他の Markdown**: 記述された要件・仕様を把握する

### 3. ブランチの準備

#### `--branch` なしの場合（新規実装）

1. `main` ブランチの最新を取得し、そこから新規ブランチを作成する
2. ブランチ名はソースの内容から自動生成する:
   - 実装プランの場合: `feat/<issue-number>-<概要のケバブケース>`（例: `feat/42-add-middleware-support`）
   - レビュー指摘修正の場合: `fix/<issue-number>-<概要のケバブケース>`
   - その他: `feat/<概要のケバブケース>`

#### `--branch` ありの場合（既存ブランチでの修正）

指定されたブランチにチェックアウトし、最新を pull する。

### 4. コードの実装

ソースの内容に基づいてコードを実装する。

#### 実装時の注意事項

- 既存のコードベースのパターン・命名規則に従うこと
- `CLAUDE.md` に記載されたアーキテクチャとパッケージ構成を遵守すること
- 実装プランがある場合は Phase / Step の順序に従って段階的に実装すること
- 各ステップの実装後、コンパイルエラーがないことを確認すること
- **GoDoc コメントは必ず英語で記述すること**（実装プランに日本語で記載されていても英語に翻訳する）
- **パッケージコメント（`// Package xxx ...`）は `doc.go` に記述すること**（実装ファイルには書かない）
- **`internal/cmd/genrecorder/` を変更した場合は `go generate ./...` を実行し、`recorder_gen.go` と `recorder_gen_test.go` を再生成してコミットに含めること**

#### 実装の進め方

1. **プロダクトコードの実装**: 新規ファイルの作成、既存ファイルの修正
2. **テストコードの実装**: ユニットテスト、統合テストの作成
3. **ドキュメントの更新**: GoDoc コメント、README、CLAUDE.md など（ソースに記載がある場合）

### 5. ビルドとフォーマットの確認

このプロジェクトは Docker 内でビルド・テストを実行する。すべてのコマンドは `make` 経由で実行すること（内部で `docker compose run --rm dev` を使用）。

```bash
# コード生成（internal/cmd/genrecorder/ を変更した場合のみ）
docker compose run --rm dev go generate ./...

# フォーマット（gofumpt による厳密なフォーマット。go fmt とは結果が異なる）
make fmt

# 静的解析（golangci-lint: errcheck, govet, staticcheck, gocritic, revive 等 11 チェッカー）
make lint

# ユニットテスト（gotestsum + race detector + coverage）
make test-unit

# ビルド確認（lint/test で暗黙的にビルドされるが、明示的に確認したい場合）
docker compose run --rm dev go build ./...
```

- **`go fmt` ではなく `make fmt` を使うこと**。プロジェクトは gofumpt を使用しており、`go fmt` ではフォーマットが不十分で CI が失敗する
- **`go vet` ではなく `make lint` を使うこと**。golangci-lint は `go vet` の上位互換で、追加の 10 チェッカーを含む
- ビルドやテストが失敗した場合は原因を特定し修正すること。修正後に再度実行し、成功するまで繰り返す
- 統合テストは Redis 環境が必要なため、CI に委ねてよい。ローカルで実行する場合は `make test-integration` を使用する

### 6. コミット

変更内容をコミットする。

- コミットメッセージは変更内容を適切に要約すること
- 実装プランの場合は Issue 番号をコミットメッセージに含めること
  - 例: `feat: add middleware support (#42)`
- レビュー指摘修正の場合は修正内容を簡潔に記載すること
  - 例: `fix: improve error handling and add missing tests`
- 複数の論理的なまとまりがある場合は、適切にコミットを分割すること
- Co-Authored-By には実行時のモデル情報を使用すること

```bash
git add <files>
git commit -m "$(cat <<'EOF'
<commit message>

Co-Authored-By: <実行中のモデル名> <noreply@anthropic.com>
EOF
)"
```

### 7. Push と PR 作成

#### `--branch` なしの場合（新規実装）

1. リモートに Push する

```bash
git push -u origin <branch-name>
```

2. PR を作成する。Issue を紐づける場合は **PR 本文** に `Closes #<issue-number>` を記載する。

```bash
gh pr create --title "<PR title>" --body "$(cat <<'EOF'
## Summary
<変更内容の箇条書き>

Closes #<issue-number>

## Test plan
<テスト方針のチェックリスト>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- PR タイトルは 70 文字以内に収めること
- PR の URL をユーザーに返すこと

#### `--branch` ありの場合（既存ブランチでの修正）

1. リモートに Push する
2. Push が完了した旨をユーザーに報告すること

## 注意事項

- ソースの内容を正確に理解し、過不足のない実装を行うこと
- 推測ではなく、実際のコードを読んで確認した事実に基づいて実装すること
- 実装中に不明点や判断が必要な事項があればユーザーに確認すること
- ビルドが通らない状態でコミット・Push しないこと
- `make fmt` を忘れずに実行すること（`go fmt` ではなく `make fmt`）
