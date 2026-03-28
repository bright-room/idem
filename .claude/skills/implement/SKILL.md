---
name: implement
description: Issue 上の実装プラン、PR のレビュー指摘、またはローカル Markdown を元にコードを実装する。Issue 番号、PR 番号、またはファイルパスを指定する。
argument-hint: "<issue-number> | <pr-number> --pr | <markdown-file-path> [--branch <branch-name>]"
---

# Implement Skill

以下の3つの入力ソースに基づいてコードを実装する:
- **Issue 番号**: Issue コメント上の実装プラン（`<!-- claude:plan -->` マーカー）を読み取って実装する
- **PR 番号 + `--pr`**: PR のレビュー指摘（インラインコメント）に対して修正を行う
- **Markdown ファイルパス**: ローカルの Markdown ファイルを読み込んで実装する

## 前提条件

- `gh` CLI が認証済みであること
- Docker が起動していること（ビルド・テストは Docker 内で実行される）
- `make build` で Docker イメージがビルド済みであること

## 引数

```
$ARGUMENTS = <issue-number> | <pr-number> --pr | <markdown-file-path> [--branch <branch-name>]
```

### 入力モードの判定

| 引数パターン | モード | 動作 |
|-------------|--------|------|
| 数値のみ（例: `42`） | **Issue プラン実装** | Issue コメントからプランを読み取り実装 |
| 数値 + `--pr`（例: `15 --pr`） | **PR レビュー指摘対応** | PR のレビュー指摘を読み取り修正 |
| ファイルパス [+ `--branch`]（例: `plan.md`） | **Markdown 実装** | ローカル Markdown を読み込んで実装 |

引数なしの場合はエラーとする。

## 手順

### 1. 入力ソースの取得

#### モード A: Issue プラン実装

Issue コメントから `<!-- claude:plan -->` マーカー付きのプランを抽出する。

```bash
# Issue のコメント一覧からプランコメントを取得
gh api repos/{owner}/{repo}/issues/<issue-number>/comments \
  --jq '.[] | select(.body | contains("<!-- claude:plan -->"))'
```

- プランコメントが見つからない場合はエラーメッセージを出力して終了する
- 複数のプランコメントが見つかった場合は、最新（最後に投稿された）コメントを採用する
- プランの Phase / Step の構成、対象ファイル、変更内容を把握する

#### モード B: PR レビュー指摘対応

PR のレビュー指摘（未解決のもの）を取得する。

##### B-1. 未解決のレビュースレッドを取得

```bash
gh api graphql -f query='
{
  repository(owner: "{owner}", name: "{repo}") {
    pullRequest(number: <pr-number>) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 10) {
            nodes {
              body
              path
              line
              diffHunk
              author { login }
            }
          }
        }
      }
    }
  }
}'
```

- `isResolved: false` のスレッドのみを対象とする
- bot コメント（CI 等）は除外する
- 各指摘の `path`, `line`, `diffHunk` から修正箇所を特定する
- 解釈不能な指摘はスキップし、対応できなかった旨をユーザーに報告する

##### B-2. PR のブランチ情報を取得

```bash
gh pr view <pr-number> --json headRefName,baseRefName
```

#### モード C: Markdown 実装

指定された Markdown ファイルを読み込み、内容を深く理解する。

- **実装プランの場合**: Phase / Step の構成、対象ファイル、変更内容を把握する
- **レビュー指摘の場合**: 指摘事項、修正案、対象ファイル・行番号を把握する
- **その他の Markdown**: 記述された要件・仕様を把握する

### 2. ブランチの準備

#### モード A（Issue プラン実装）: 新規ブランチを作成

1. `main` ブランチの最新を取得し、そこから新規ブランチを作成する
2. ブランチ名: `feat/<issue-number>-<概要のケバブケース>`
   - 例: `feat/42-add-middleware-support`

#### モード B（PR レビュー指摘対応）: PR のブランチにチェックアウト

PR の `headRefName` にチェックアウトし、最新を pull する。

#### モード C（Markdown 実装）: 引数に依存

| 引数 | 動作 |
|------|------|
| `--branch` なし | Markdown の内容からブランチ名を自動生成し、`main` から新規ブランチを作成 |
| `--branch <existing-branch>` | 指定されたブランチにチェックアウトし、最新を pull する |

ブランチ名の自動生成ルール:
- 実装プランの場合: `feat/<issue-number>-<概要のケバブケース>`
- レビュー指摘修正の場合: `fix/<issue-number>-<概要のケバブケース>`
- その他: `feat/<概要のケバブケース>`

### 3. コードの実装

入力ソースの内容に基づいてコードを実装する。

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

### 4. ビルドとフォーマットの確認

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

### 5. コミット

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

### 6. Push と PR 作成

#### モード A（Issue プラン実装）: Push して PR 作成

1. リモートに Push する

```bash
git push -u origin <branch-name>
```

2. PR を作成する。Issue を紐づけるため **PR 本文** に `Closes #<issue-number>` を記載する。

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

3. Issue に `Kind: *` ラベルが付与されている場合、同じラベルを PR にも付与する

```bash
LABELS=$(gh issue view <issue-number> --json labels --jq '[.labels[].name | select(startswith("Kind: "))] | join(",")')
if [ -n "$LABELS" ]; then
  gh pr edit --add-label "$LABELS"
fi
```

- PR の URL をユーザーに返すこと

#### モード B（PR レビュー指摘対応）: Push のみ

1. リモートに Push する
2. 対応した指摘と対応できなかった指摘をユーザーに報告すること

#### モード C（Markdown 実装）: 引数に依存

**`--branch` なしの場合（新規実装）:**
- Push して PR を作成する（モード A と同様のフロー）

**`--branch` ありの場合（既存ブランチでの修正）:**
- Push のみ行い、完了をユーザーに報告する

## 注意事項

- 入力ソースの内容を正確に理解し、過不足のない実装を行うこと
- 推測ではなく、実際のコードを読んで確認した事実に基づいて実装すること
- 実装中に不明点や判断が必要な事項があればユーザーに確認すること
- ビルドが通らない状態でコミット・Push しないこと
- `make fmt` を忘れずに実行すること（`go fmt` ではなく `make fmt`）
