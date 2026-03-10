# /implement スキル 使い方ガイド

## 概要

指定された Markdown ファイルを読み込み、内容に基づいてコードを実装するスキル。

## 使い方

```
/implement .claude/outputs/plans/PLAN-42-add-webflux-support.md
/implement .claude/outputs/plans/PLAN-42-add-webflux-support.md --branch feat/42-add-webflux-support
/implement .claude/outputs/reviews/REVIEW-feat-42-add-webflux-support.md --branch feat/42-add-webflux-support
```

### 新規実装（main からブランチを切って PR を作成）

```
/implement <markdown-file-path>
```

- `main` から新規ブランチを自動作成
- 実装完了後に PR を作成し、URL を返す

### 既存ブランチでの修正（レビュー指摘対応など）

```
/implement <markdown-file-path> --branch <branch-name>
```

- 指定ブランチにチェックアウトして修正を実施
- 実装完了後に Push する（PR は作成しない）

## 処理の流れ

1. **引数の解析** — Markdown ファイルパスとブランチオプションを取得
2. **Markdown の読み込み** — 実装プラン・レビュー指摘等の内容を理解
3. **ガイドラインの読み込み** — `.claude/guidelines/coding.md` を把握
4. **ブランチの準備** — 新規作成 or 既存ブランチにチェックアウト
5. **コードの実装** — Markdown の内容に基づいて段階的に実装
6. **ビルド確認** — `spotlessApply` + `build` の実行
7. **コミット** — 変更内容を適切なメッセージでコミット
8. **Push / PR 作成** — ブランチモードに応じて Push のみ or PR 作成

## ユースケース

| ユースケース | コマンド例 |
|-------------|-----------|
| 実装プランに基づく新規実装 | `/implement .claude/outputs/plans/PLAN-42-xxx.md` |
| レビュー指摘の修正 | `/implement .claude/outputs/reviews/REVIEW-feat-42-xxx.md --branch feat/42-xxx` |
| 任意の仕様書に基づく実装 | `/implement docs/spec.md` |
| 既存ブランチへの追加実装 | `/implement .claude/outputs/plans/PLAN-42-xxx.md --branch feat/42-xxx` |

## 定義ファイル

[SKILL.md](SKILL.md)
