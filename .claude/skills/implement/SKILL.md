---
name: implement
description: 指定された Markdown（実装プラン等）を読み込み、内容に基づいてコードを実装する。新規実装時は main からブランチを切り PR を作成し、既存ブランチ指定時はそのブランチ上で修正・Push を行う。
argument-hint: "<markdown-file-path> [--branch <branch-name>]"
---

# Implement Skill

指定された Markdown ファイル（実装プラン、レビュー指摘など）を読み込み、その内容に基づいてコードを実装する。

## 前提条件

- 引数 `$ARGUMENTS` に Markdown ファイルのパスが指定されていること（必須）
- `gh` CLI が認証済みであること（PR 作成時）

## 引数

```
$ARGUMENTS = <markdown-file-path> [--branch <branch-name>]
```

- `<markdown-file-path>`: 実装の元となる Markdown ファイルのパス（必須）
- `--branch <branch-name>`: 作業ブランチの指定（任意）

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

- Markdown ファイルパスが指定されていない場合はエラーメッセージを出力して終了すること
- `--branch` オプションがあればブランチ名を取得する

### 2. Markdown ファイルの読み込みと理解

指定された Markdown ファイルを読み込み、内容を深く理解する。

- **実装プランの場合**: Phase / Step の構成、対象ファイル、変更内容を把握する
- **レビュー指摘の場合**: 指摘事項、修正案、対象ファイル・行番号を把握する
- **その他の Markdown**: 記述された要件・仕様を把握する

### 3. ブランチの準備

#### `--branch` なしの場合（新規実装）

1. `main` ブランチの最新を取得する

```bash
git fetch origin main
git checkout main
git pull origin main
```

2. Markdown の内容からブランチ名を自動生成する
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

### 4. コードの実装

Markdown の内容に基づいてコードを実装する。

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
3. **ドキュメントの更新**: GoDoc コメント、README、CLAUDE.md など（Markdown に記載がある場合）

### 5. ビルドとフォーマットの確認

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

### 6. コミット

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

### 7. Push と PR 作成

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

- Markdown ファイルの内容を正確に理解し、過不足のない実装を行うこと
- 推測ではなく、実際のコードを読んで確認した事実に基づいて実装すること
- 実装中に不明点や判断が必要な事項があればユーザーに確認すること
- ビルドが通らない状態でコミット・Push しないこと
- `golangci-lint fmt ./...` を忘れずに実行すること
