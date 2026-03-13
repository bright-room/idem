---
name: triage
description: Issue の棚卸を行う。対応済み Issue のクローズ、ラベル未付与 Issue へのラベル付与、マイルストーン未割り当て Issue へのマイルストーン紐づけと平準化、実装プランの今後の展望から新規 Issue を作成する。
---

# Issue Triage Skill

GitHub Issue の棚卸を行い、プロジェクトの Issue 管理を整理する。

## 前提条件

- `gh` CLI が認証済みであること

## 手順

### 1. 対応済み Issue のクローズ

Open な Issue を一覧取得し、それぞれの Issue が既に対応済みかを判定する。

```bash
gh issue list --state open --json number,title,labels,milestone,body --limit 100
```

#### 判定方法

各 Issue について以下を確認する:

1. **コードベースの確認**: Issue の要件に対応するコードが既に実装されているかを調査する
   - 関連するファイル、関数、テストの存在を確認
   - `git log --oneline --all --grep="#<issue-number>"` で関連コミットを検索
2. **PR の確認**: Issue に紐づく PR がマージ済みかを確認する
   ```bash
   gh pr list --state merged --search "close #<issue-number> OR closes #<issue-number> OR fix #<issue-number> OR fixes #<issue-number>" --json number,title
   ```
3. **Issue 内容との照合**: コードベースの現状が Issue の要件を満たしているかを総合的に判断する

#### クローズ処理

対応済みと判定した Issue は、理由を添えてクローズする。

```bash
gh issue close <issue-number> --comment "$(cat <<'EOF'
棚卸によりクローズします。

**対応済みの根拠:**
- <対応済みと判断した具体的な根拠を記載>

🤖 *Triaged by Claude Code*
EOF
)"
```

**注意事項:**
- Renovate の Dependency Dashboard（#23 など）はクローズ対象外とする
- 判定に迷う場合はクローズせず、後述のレポートで報告するに留める
- `Close: Duplicate` や `Close: WontFix` ラベルが付いた Issue もクローズ対象とする

### 2. 既存 Issue のラベル付与

ラベルが付与されていない既存の Open Issue を特定し、適切なラベルを付与する。

#### 2.1 ラベル未付与 Issue の特定

```bash
gh issue list --state open --json number,title,labels,body --limit 100 --jq '.[] | select(.labels | length == 0)'
```

Dependency Dashboard（Renovate）など自動管理されている Issue は対象外とする。

#### 2.2 ラベルの判定と付与

Issue のタイトルと本文を読み、以下のルールに基づいて適切なラベルを判定する。

**種別ラベル（必須・1つ選択）:**

| 内容の種別 | ラベル |
|-----------|--------|
| バグ報告・不具合修正 | `Type: Bug` |
| 新しい機能の追加 | `Type: Feature` |
| 既存機能の改善・拡張 | `Type: Enhancement` |
| コードの整理・改善 | `Type: Refactoring` |
| テストの追加・改善 | `Type: Test` |
| ドキュメントの追加・改善 | `Type: Document` |
| リリース作業 | `Type: Publishing` |

**優先度ラベル（必須・1つ選択）:**

| 優先度 | ラベル | 基準 |
|--------|--------|------|
| 高 | `Priority: High` | ユーザーに直接影響する、またはバグの温床となりうる |
| 中 | `Priority: Medium` | 品質向上に寄与するが、緊急性は低い |
| 低 | `Priority: Low` | あると良いが、なくても問題ない |

**その他のラベル（任意）:**

Issue の内容に応じて、`Close: Duplicate` や `Close: WontFix` などの既存ラベルが適切な場合は付与する。

#### 2.3 ラベルの付与

```bash
gh issue edit <issue-number> --add-label "<ラベル1>,<ラベル2>"
```

**注意事項:**
- 既にラベルが付いている Issue のラベルは変更しない（この手順はラベル未付与の Issue のみが対象）
- 判断に迷う場合は `Type: Enhancement` + `Priority: Medium` をデフォルトとする
- ラベルがリポジトリに存在しない場合は、`gh label create` で作成してから付与する

### 3. マイルストーンの平準化

#### 3.1 現状の把握

マイルストーンの一覧と各マイルストーンの Issue 数を確認する。

```bash
gh api repos/{owner}/{repo}/milestones --jq '.[] | select(.state=="open") | {title, open_issues, closed_issues, due_on}'
```

#### 3.2 マイルストーン未割り当て Issue の処理

マイルストーンが未割り当ての Issue（Dependency Dashboard を除く）を特定する。

#### 3.3 ラベルに基づくマイルストーン割り当てルール

Issue のラベルに基づいて、適切なマイルストーンを判断する:

| ラベル | マイルストーンの目安 |
|--------|---------------------|
| `Type: Bug` | 直近のマイルストーン（バグは早期修正） |
| `Type: Feature` | 機能の規模に応じて適切なマイルストーンに配置 |
| `Type: Enhancement` | 機能の規模に応じて適切なマイルストーンに配置 |
| `Type: Refactoring` | 大きなリリースの前（v1.0.0 など）に配置 |
| `Type: Document` | 関連する機能と同じマイルストーン、または大きなリリースの前 |
| `Type: Test` | 関連する機能と同じマイルストーン |
| `Type: Publishing` | リリース作業用のマイルストーン |

#### 3.4 マイルストーンの作成

必要に応じて新しいマイルストーンを作成する。

- マイルストーン名はセマンティクスバージョニング形式: `vX.Y.Z`
- 既存のマイルストーン一覧を確認し、次のバージョン番号を決定する
- 1つのマイルストーンに Issue が偏りすぎないようにする（目安: 1マイルストーンあたり 5〜8 Issue）
- マイルストーンの作成:
  ```bash
  gh api repos/{owner}/{repo}/milestones --method POST --field title="vX.Y.Z"
  ```

#### 3.5 Issue のマイルストーン割り当て

```bash
gh issue edit <issue-number> --milestone "vX.Y.Z"
```

#### 3.6 平準化の基準

- 1つのマイルストーンに 8 Issue 以上ある場合は、一部を別のマイルストーンに移動する
- ラベルの種類と依存関係を考慮して、論理的にまとまりのあるグループにする
- `Type: Publishing`（リリース作業）は、そのバージョンのマイルストーンに残す

### 4. 実装プランの今後の展望からの Issue 作成

#### 4.1 実装プランの読み込み

`.claude/outputs/plans/` ディレクトリが存在する場合、`PLAN-*.md` ファイルを読み込み、「今後の展望」セクションを抽出する。

ディレクトリやファイルが存在しない場合は、この手順（手順 3）全体をスキップする。

#### 4.2 既存 Issue との重複チェック

抽出した各展望項目について、既存の Open/Closed Issue と重複していないかを確認する。

```bash
gh issue list --state all --json number,title,body,state --limit 200
```

重複判定の基準:
- Issue のタイトルが同じ、またはほぼ同一の内容を指している
- Issue の本文に同じ機能・改善が記載されている
- 完全一致でなくても、実質的に同じ作業を指している場合は重複とみなす

#### 4.3 新規 Issue の作成

重複していない展望項目について、Issue を作成する。

プランのファイル名は `PLAN-<Issue番号>-<タイトル>.md` の形式であるため、ファイル名から元の Issue 番号を抽出し、Issue 本文で紐づける。

```bash
gh issue create --title "<タイトル>" --label "<ラベル>" --body "$(cat <<'EOF'
## 概要

<展望項目の内容を具体的に記述>

## 背景

#<元Issue番号> の実装プランにおける今後の展望から抽出。

---
🤖 *Created by Claude Code (Issue Triage)*
EOF
)"
```

**ラベルの割り当てルール（必須）:**

展望項目の内容に応じて、適切なラベルを**必ず**付与する:

| 内容の種別 | ラベル |
|-----------|--------|
| 新しい機能の追加 | `Type: Feature` |
| 既存機能の改善・拡張 | `Type: Enhancement` |
| コードの整理・改善 | `Type: Refactoring` |
| テストの追加・改善 | `Type: Test` |
| ドキュメントの追加・改善 | `Type: Document` |

さらに、優先度ラベルも付与する:

| 優先度 | ラベル | 基準 |
|--------|--------|------|
| 高 | `Priority: High` | ユーザーに直接影響する、またはバグの温床となりうる |
| 中 | `Priority: Medium` | 品質向上に寄与するが、緊急性は低い |
| 低 | `Priority: Low` | あると良いが、なくても問題ない |

#### 4.4 作成した Issue のマイルストーン割り当て

作成した Issue にも、ラベルに基づいて適切なマイルストーンを割り当てる（手順 3 のルールに従う）。

### 5. 棚卸レポートの出力

棚卸の結果を `.claude/outputs/triage/` ディレクトリにファイルとして出力する。

- ディレクトリが存在しない場合は作成すること
- ファイル名: `TRIAGE-YYYY-MM-DD-HHmmss.md`（実行日時のタイムスタンプ）
  - 例: `TRIAGE-2026-03-13-143025.md`

以下のフォーマットで出力する:

```markdown
## 📋 Issue 棚卸レポート

> 📅 実施日時: YYYY-MM-DD HH:mm:ss

### クローズした Issue

| # | タイトル | 理由 |
|---|---------|------|
| #XX | <タイトル> | <クローズ理由の要約> |

（対象がない場合は「対象なし」と記載）

### ラベルを付与した Issue

| # | タイトル | 付与したラベル |
|---|---------|--------------|
| #XX | <タイトル> | <付与したラベル> |

（対象がない場合は「対象なし」と記載）

### マイルストーンの変更

| # | タイトル | 変更前 | 変更後 |
|---|---------|--------|--------|
| #XX | <タイトル> | <旧マイルストーン or なし> | <新マイルストーン> |

新規作成したマイルストーン: <あれば記載>

### 新規作成した Issue

| # | タイトル | ラベル | マイルストーン | 元プラン |
|---|---------|--------|---------------|---------|
| #XX | <タイトル> | <ラベル> | <マイルストーン> | PLAN-XX |

### 棚卸後のマイルストーン状況

| マイルストーン | Open | Closed | 合計 |
|--------------|------|--------|------|
| vX.Y.Z | N | N | N |
```

## 注意事項

- Issue のクローズは慎重に行うこと。判断に迷う場合はクローズしない
- Renovate が管理する Issue（Dependency Dashboard）は操作しない
- マイルストーンの割り当ては、Issue の内容とラベルを総合的に判断して行う
- 新規 Issue には**必ず**ラベルを付けること（ラベルなしの Issue は作成しない）
- 既存 Issue と重複する展望項目は Issue 化しない
- すべての操作は `gh` CLI を通じて行う
