---
name: triage
description: Issue の棚卸を一括実行する。対応済み Issue のクローズ、ラベル付与、実装プランからの Issue 作成、マイルストーン平準化をすべて行い、棚卸レポートを出力する。
---

# Issue Triage Skill

GitHub Issue の棚卸を一括で行い、プロジェクトの Issue 管理を整理する。

5つのステップを順番に実行し、最後に統合レポートを出力する。途中のステップが失敗した場合でも、エラーを記録して次のステップに進むこと。

## 前提条件

- `gh` CLI が認証済みであること

## 共通の除外ルール

以下は全ステップで操作対象外とする:
- Renovate が管理する Issue（Dependency Dashboard）
- bot が自動生成した Issue

## 手順

### 1. 対応済み Issue のクローズ

Open な Issue を一覧取得し、それぞれの Issue が既に対応済みかを判定してクローズする。

#### 1-1. Open Issue の取得

```bash
gh issue list --state open --json number,title,labels,milestone,body --limit 100
```

#### 1-2. 対応済み判定

各 Issue について以下を確認する:

1. **コードベースの確認**: Issue の要件に対応するコードが既に実装されているかを調査する
   - 関連するファイル、関数、テストの存在を確認
   - `git log --oneline --all --grep="#<issue-number>"` で関連コミットを検索
2. **PR の確認**: Issue に紐づく PR がマージ済みかを確認する
   ```bash
   gh pr list --state merged --search "close #<N> OR closes #<N> OR fix #<N> OR fixes #<N> OR resolve #<N> OR resolves #<N>" --json number,title
   ```
3. **Issue 内容との照合**: コードベースの現状が Issue の要件を満たしているかを総合的に判断する

#### 1-3. クローズ処理

対応済みと判定した Issue は、理由を添えてクローズする。**判定に迷う場合はクローズしない。**

```bash
gh issue close <issue-number> --comment "$(cat <<'EOF'
棚卸によりクローズします。

**対応済みの根拠:**
- <対応済みと判断した具体的な根拠を記載>

🤖 *Triaged by Claude Code*
EOF
)"
```

`Close: Duplicate` や `Close: WontFix` ラベルが付いた Issue もクローズ対象とする。

#### 1-4. 結果の記録

クローズした Issue の一覧を記録する。対象がない場合は「対応済みの Issue はありませんでした」と記録する。

---

### 2. 既存 Issue のラベル付与

ラベルが不足している既存の Open Issue を特定し、適切なラベルを付与する。

#### 2-1. ラベル不足 Issue の特定

```bash
gh issue list --state open --json number,title,labels,body --limit 100
```

結果を確認し、Kind ラベルまたは Priority ラベルが付いていない Issue を対象とする。

#### 2-2. ラベルの判定と付与

Issue のタイトルと本文を読み、以下のルールに基づいてラベルを判定する。

**Kind ラベル（必須・1つ選択）:**

| 内容の種別 | ラベル |
|-----------|--------|
| バグ報告・不具合修正 | `Kind: Bug Fix` |
| 新しい機能の追加 | `Kind: Feature` |
| 既存機能の改善・拡張 | `Kind: Enhancement` |
| コードの整理・改善（API 破壊なし） | `Kind: Refactoring` |
| テストの追加・改善 | `Kind: Tests` |
| ドキュメントの追加・改善 | `Kind: Documentation` |

**Priority ラベル（必須・1つ選択）:**

| 優先度 | ラベル | 基準 |
|--------|--------|------|
| 緊急 | `Priority: Critical` | サービス停止や重大なセキュリティ問題 |
| 高 | `Priority: High` | ユーザーに直接影響する、またはバグの温床となりうる |
| 中 | `Priority: Medium` | 品質向上に寄与するが、緊急性は低い |
| 低 | `Priority: Low` | あると良いが、なくても問題ない |

**補助ラベル（任意・複数選択可）:**

| 条件 | ラベル |
|------|--------|
| 後方互換性が失われる変更 | `Impact: Breaking` |
| 設計上の議論が必要 | `Need: Discussion` |
| 他者の協力が必要 | `Need: Help Wanted` |

**判定ルール:**

- Kind の判断に迷う場合は `Kind: Enhancement` をデフォルトとする
- Priority の判断に迷う場合は `Priority: Medium` をデフォルトとする
- Issue 本文に「破壊的変更」「後方互換性」「メジャーバージョン」等の記述がある場合は `Impact: Breaking` を付与する
- Issue 本文に「検討」「議論」「要設計」等の記述があり、実装方針が未確定の場合は `Need: Discussion` を付与する
- `Need: Discussion` のみで Kind が確定しない場合は Kind ラベルを付与しない
- ラベルがリポジトリに存在しない場合は付与せず、その旨を記録する
- 既に付いているラベルは変更しない

```bash
gh issue edit <issue-number> --add-label "<ラベル1>,<ラベル2>"
```

#### 2-3. 結果の記録

ラベルを付与した Issue の一覧を記録する。

---

### 3. 実装プランの今後の展望からの Issue 作成

特定マイルストーンに紐づく Issue の実装プラン（「今後の展望」セクション）を読み取り、既存 Issue と重複しないものを新規 Issue として作成する。

#### 3-0. 対象マイルストーンの決定

triage スキルの引数でマイルストーンタイトル（バージョン）が指定されている場合はそのマイルストーンを対象にする。引数がない場合は、最もバージョンが新しいクローズ済みマイルストーンを自動的に対象とする。

```bash
# クローズ済みマイルストーン一覧を取得
gh api repos/{owner}/{repo}/milestones?state=closed --jq '.[].title'
```

取得したマイルストーン一覧から Semver でソートし、最新バージョンを選択する。

#### 3-1. 対象 Issue の取得

対象マイルストーンに紐づく Issue のうち、**正常にクローズされたもの（Done / Closed / Fixed / Resolved）のみ**を取得する。`Close as not planned`（Won't fix, Can't repro, Stale）や `Close as duplicate` でクローズされた Issue は対象外とする。

```bash
# マイルストーンに紐づくクローズ済み Issue を取得
gh issue list --state closed --milestone "<マイルストーンタイトル>" --json number,title,stateReason --limit 100
```

- `stateReason` が `COMPLETED` の Issue のみを対象とする（`NOT_PLANNED` は除外）

#### 3-2. 実装プランの読み込み

対象 Issue のコメントから `<!-- claude:plan -->` マーカー付きコメントを検索し、「今後の展望」セクションを抽出する。

```bash
gh api repos/{owner}/{repo}/issues/<issue-number>/comments \
  --jq '.[] | select(.body | contains("<!-- claude:plan -->"))'
```

- 複数のプランコメントが存在する場合は、最新（最後に投稿された）コメントを採用する
- 「今後の展望」セクションが含まれないプランはスキップする
- プランコメントを含む Issue が見つからない場合はスキップする
- 対象マイルストーンにプランコメントを含む Issue が1つもない場合は「実装プランが見つかりませんでした」と記録してスキップする

#### 3-2. 既存 Issue との重複チェック

```bash
gh issue list --state all --search "<keyword>" --json number,title,state --limit 50
```

重複判定: タイトルが同じ、またはほぼ同一の内容を指している場合は重複とみなす。

#### 3-3. 新規 Issue の作成

プランコメントが投稿されている Issue の番号を元 Issue として紐づける。

```bash
gh issue create --title "<タイトル>" --label "<Kind>,<Priority>" --body "$(cat <<'EOF'
## 概要

<展望項目の内容を具体的に記述>

## 背景

#<元Issue番号> の実装プランにおける今後の展望から抽出。

---
🤖 *Created by Claude Code (Issue Triage)*
EOF
)"
```

- 新規 Issue には**必ず** Kind ラベルと Priority ラベルを付けること
- 補助ラベル（`Impact: Breaking`, `Need: Discussion`）も該当する場合は付与すること
- **マイルストーンはこの時点ではアタッチしない**（Step 4 で一括処理するため）

#### 3-4. 結果の記録

作成した Issue の一覧を記録する。

---

### 4. マイルストーンの平準化

全 Open Issue（既存 + Step 3 で新規作成したもの）を対象に、マイルストーンの割り当てと平準化を行う。

#### 4-1. 現状の把握

```bash
gh api repos/:owner/:repo/milestones?state=all --jq '.[] | {title, state, open_issues, closed_issues}'
```

#### 4-2. Semver ルールに基づく分類

Kind と Impact ラベルの組み合わせでバージョン区分を決定する。

**Patch (x.y.Z):**

| ラベル | ケース |
|--------|-------|
| `Kind: Bug Fix` | 常にパッチ対象 |
| `Kind: Tests` | テスト追加・改善 |
| `Kind: Documentation` | ドキュメント追加・改善 |

**Minor (x.Y.0):**

| ラベル | ケース |
|--------|-------|
| `Kind: Feature` | 新機能の追加 |
| `Kind: Enhancement` | 既存機能の改善・拡張 |
| `Kind: Refactoring` | 内部的なリファクタリング |
| `Impact: Breaking` + `Kind: Bug Fix` | 構造上の問題に起因するバグ修正（低頻度） |

**Major (X.0.0):**

| ラベル | ケース |
|--------|-------|
| `Impact: Breaking` + `Kind: Feature` | 破壊的変更を伴う新機能 |
| `Impact: Breaking` + `Kind: Enhancement` | 破壊的変更を伴う機能改善 |
| `Impact: Breaking` + `Kind: Refactoring` | 破壊的変更を伴うリファクタリング |

**マイルストーン割り当て対象外:**

- Kind ラベルが付いていない Issue（`Need: Discussion` のみ等）
- Dependency Dashboard

#### 4-3. Priority に基づくマイルストーン配置

同一バージョン区分（Patch/Minor/Major）内で、Priority によって配置するマイルストーンの位置を決定する。

| Priority | 配置位置 |
|----------|---------|
| `Priority: Critical` | 最も直近のマイルストーン |
| `Priority: High` | 直近のマイルストーン |
| `Priority: Medium` | 中間のマイルストーン |
| `Priority: Low` | 後方のマイルストーン |

#### 4-4. マイルストーンの作成（必要な場合）

- 既存のマイルストーン一覧（**クローズ済みを含む**）を確認し、次のバージョン番号を決定する
- バージョン区分に対応するマイルストーンが不足している場合は新設する
- 1マイルストーンに Issue が偏りすぎないようにする（目安: 5〜8 Issue）

```bash
gh api repos/:owner/:repo/milestones --method POST --field title="vX.Y.Z"
```

#### 4-5. Issue のマイルストーン割り当て

```bash
gh issue edit <issue-number> --milestone "vX.Y.Z"
```

- マイルストーン未割り当ての Issue を対象に割り当てる
- 既にマイルストーンが割り当てられている Issue は、Semver ルールと Priority に照らして不整合がない限り変更しない
- 不整合がある場合（例: Patch 区分の Issue が Minor マイルストーンに入っている）は適切なマイルストーンに移動する

#### 4-6. 結果の記録

マイルストーンの変更内容を記録する。

---

### 5. 棚卸レポートの出力

棚卸の結果を `.claude/outputs/triage/` ディレクトリにファイルとして出力する。

- ディレクトリが存在しない場合は作成すること
- ファイル名: `TRIAGE-YYYY-MM-DD-HHmmss.md`（実行日時のタイムスタンプ）

レポートフォーマット:

```markdown
## Issue 棚卸レポート

> 実施日時: YYYY-MM-DD HH:mm:ss

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

### 新規作成した Issue

| # | タイトル | ラベル | 元 Issue |
|---|---------|--------|---------|
| #XX | <タイトル> | <ラベル> | #YY |

（対象がない場合は「対象なし」と記載）

### マイルストーンの変更

| # | タイトル | 変更前 | 変更後 |
|---|---------|--------|--------|
| #XX | <タイトル> | <旧マイルストーン or なし> | <新マイルストーン> |

新規作成したマイルストーン: <あれば記載>

### 棚卸後のマイルストーン状況

| マイルストーン | Open | Closed | 合計 |
|--------------|------|--------|------|
| vX.Y.Z | N | N | N |
```

## 注意事項

- Issue のクローズは慎重に行うこと。判断に迷う場合はクローズしない
- すべての操作は `gh` CLI を通じて行う
- 途中のステップでエラーが発生した場合は、エラー内容をレポートに記録し、次のステップに進む
