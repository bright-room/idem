---
name: implement
description: 指定された Markdown（実装プラン等）を読み込み、内容に基づいてコードを実装する。ローカルではファイルパス指定、CI 環境では Issue コメントの内容を読み取って実装する。
argument-hint: "[markdown-file-path] [--branch <branch-name>]"
---

# Implement Skill

指定されたソース（Markdown ファイルまたは GitHub Issue のコメント）を読み込み、その内容に基づいてコードを実装する。

## 前提条件

- `gh` CLI が認証済みであること

## 引数

```
$ARGUMENTS = [markdown-file-path] [--branch <branch-name>]
```

- `<markdown-file-path>`: 実装の元となる Markdown ファイルのパス（任意）
- `--branch <branch-name>`: 作業ブランチの指定（任意）

### ソースの解決ルール

引数と実行環境に応じて、実装の入力ソースを以下の優先順で決定する:

| 条件 | ソース | 例 |
|------|--------|-----|
| 引数がファイルパス（`/` を含む or `.md` で終わる） | 指定された Markdown ファイルを読み込む | `/implement .claude/outputs/plans/PLAN-42-xxx.md` |
| 引数なし + `CI=true` | 現在の Issue のコメント内容を読み取る | `/implement` |
| 引数なし + ローカル | エラー（ソースの指定が必要） | — |

### ブランチの動作

| 引数 | 動作 |
|------|------|
| `--branch` なし | Markdown の内容からブランチ名を自動生成し、`main` から新規ブランチを作成。実装完了後に PR を作成する |
| `--branch <existing-branch>` | 指定されたブランチにチェックアウトし、そのブランチ上で修正を実施。実装完了後に Push する（PR は作成しない） |

## 手順

### 1. 引数の解析

```
ARGUMENTS = "$ARGUMENTS"
```

- `--branch` オプションがあればブランチ名を取得する
- 残りの引数からソース種別を判定する（ファイルパス / Issue 番号 / なし）

### 2. 実装ソースの取得

ソースの解決ルールに従い、実装内容を取得する。

#### パターン A: ファイルパスが指定された場合

指定された Markdown ファイルを読み込む。ファイルが存在しない場合はエラーメッセージを出力して終了する。

#### パターン B: 引数なし + CI 環境

現在の Issue のコンテキストからソースを取得する。

```bash
# 現在の Issue 番号を取得（GitHub Actions のコンテキストから）
ISSUE_NUMBER=${GITHUB_ISSUE_NUMBER:-}

# Issue 番号が取得できない場合
if [ -z "$ISSUE_NUMBER" ]; then
  echo "Error: Issue number not found in CI context"
  exit 1
fi

# Issue 本文とコメントの取得
gh issue view ${ISSUE_NUMBER} --json body,title,comments
```

以下の優先順でソースを特定する:

1. Issue コメントの中から「実装プラン」セクション（`## 実装プラン` で始まるコメント）を検索する
2. 実装プランコメントが見つからない場合は、Issue 本文を実装ソースとして使用する
3. トリガーとなったコメント（`@claude /implement` を含むコメント）自体に実装指示が含まれている場合は、そのコメント内容も実装ソースに加味する
4. いずれも実装可能な内容を含まない場合はエラーメッセージを出力して終了する

### 3. 実装ソースの理解

取得したソースを深く理解する。

- **実装プランの場合**: Phase / Step の構成、対象ファイル、変更内容を把握する
- **レビュー指摘の場合**: 指摘事項、修正案、対象ファイル・行番号を把握する
- **Issue 本文の場合**: 要件・仕様を把握し、コードベースを調査した上で実装方針を決定する
- **その他の Markdown**: 記述された要件・仕様を把握する

### 4. ブランチの準備

#### `--branch` なしの場合（新規実装）

1. `main` ブランチの最新を取得する

```bash
git fetch origin main
git checkout main
git pull origin main
```

2. ソースの内容からブランチ名を自動生成する
   - 実装プランの場合: `feat/<issue-number>-<概要のケバブケース>` の形式
     - 例: Issue #42 "Add middleware support" → `feat/42-add-middleware-support`
   - レビュー指摘修正の場合: `fix/<issue-number>-<概要のケバブケース>` の形式
   - その他: `feat/<概要のケバブケース>` の形式

3. ブランチを作成する

```bash
git checkout -b <branch-name>
```

#### `--branch` ありの場合（既存ブランチでの修正）

1. 指定されたブランチにチェックアウトする

```bash
git checkout <branch-name>
git pull origin <branch-name>
```

### 5. コードの実装

ソースの内容に基づいてコードを実装する。

#### 実装時の注意事項

- 既存のコードベースのパターン・命名規則に従うこと
- `CLAUDE.md` に記載されたアーキテクチャとパッケージ構成を遵守すること
- 実装プランがある場合は Phase / Step の順序に従って段階的に実装すること
- 各ステップの実装後、コンパイルエラーがないことを確認すること
- **GoDoc コメントは必ず英語で記述すること**（実装プランに日本語で記載されていても英語に翻訳する）
- **パッケージコメント（`// Package xxx ...`）は `doc.go` に記述すること**（実装ファイルには書かない）

#### 実装の進め方

1. **プロダクトコードの実装**: 新規ファイルの作成、既存ファイルの修正
2. **テストコードの実装**: ユニットテスト、統合テストの作成
3. **ドキュメントの更新**: GoDoc コメント、README、CLAUDE.md など（ソースに記載がある場合）

### 6. ビルドとフォーマットの確認

実装完了後、以下を実行する:

```bash
# フォーマットの確認と適用
go fmt ./...

# 静的解析
go vet ./...

# ビルドの確認
go build ./...

# テストの実行
go test ./...
```

- `go fmt` は必ずコミット前に実行すること
- ビルドやテストが失敗した場合は原因を特定し修正すること。修正後に再度実行し、成功するまで繰り返す

### 7. コミット

変更内容をコミットする。

- コミットメッセージは変更内容を適切に要約すること
- 実装プランの場合は Issue 番号をコミットメッセージに含めること
  - 例: `Close #42: Add middleware support`
- レビュー指摘修正の場合は修正内容を簡潔に記載すること
  - 例: `Fix review comments: improve error handling and add missing tests`
- 複数の論理的なまとまりがある場合は、適切にコミットを分割すること

```bash
git add <files>
git commit -m "$(cat <<'EOF'
<commit message>

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

### 8. Push と PR 作成

#### `--branch` なしの場合（新規実装）

1. リモートに Push する

```bash
git push -u origin <branch-name>
```

2. PR を作成する

```bash
gh pr create --title "<PR title>" --body "$(cat <<'EOF'
## Summary
<変更内容の箇条書き>

## Test plan
<テスト方針のチェックリスト>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- PR タイトルは Issue を閉じる場合 `Close #<issue-number>: <概要>` の形式にする
- PR タイトルは 70 文字以内に収めること

3. PR の URL をユーザーに返すこと

#### `--branch` ありの場合（既存ブランチでの修正）

1. リモートに Push する

```bash
git push origin <branch-name>
```

2. Push が完了した旨をユーザーに報告すること

## 注意事項

- ソースの内容を正確に理解し、過不足のない実装を行うこと
- 推測ではなく、実際のコードを読んで確認した事実に基づいて実装すること
- 実装中に不明点や判断が必要な事項があればユーザーに確認すること
- ビルドが通らない状態でコミット・Push しないこと
- `golangci-lint fmt ./...` を忘れずに実行すること
- CI 環境で Issue 本文から実装する場合は、要件が曖昧な場合に Issue にコメントで確認してから実装を進めること
