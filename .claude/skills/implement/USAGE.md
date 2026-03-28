# /implement スキル 使い方ガイド

## 概要

Issue 上の実装プラン、PR のレビュー指摘、またはローカル Markdown を元にコードを実装するスキル。

## 使い方

```
/implement 42                    # Issue #42 のプランコメントから実装
/implement 15 --pr               # PR #15 の未解決レビュー指摘に対応
/implement plan.md               # ローカル Markdown ファイルから実装
/implement plan.md --branch feat/42-xxx  # 既存ブランチ上で実装
```

## 入力モード

| 引数パターン | モード | 動作 |
|-------------|--------|------|
| 数値のみ（例: `42`） | Issue プラン実装 | Issue コメントから `<!-- claude:plan -->` マーカー付きプランを読み取り実装 |
| 数値 + `--pr`（例: `15 --pr`） | PR レビュー指摘対応 | 未解決のレビュー指摘を読み取り修正 |
| ファイルパス（例: `plan.md`） | Markdown 実装 | ローカル Markdown を読み込んで実装 |

## ブランチの動作

| モード | ブランチ |
|--------|---------|
| Issue プラン実装 | `main` から `feat/<issue-number>-<概要>` を自動作成 → PR 作成 |
| PR レビュー指摘対応 | PR のブランチにチェックアウト → Push のみ |
| Markdown（`--branch` なし） | `main` から自動生成ブランチを作成 → PR 作成 |
| Markdown（`--branch` あり） | 指定ブランチにチェックアウト → Push のみ |

## 処理の流れ

1. **入力ソースの取得** — Issue コメント / PR レビュースレッド / Markdown ファイル
2. **ブランチの準備** — モードに応じて新規作成 or チェックアウト
3. **コードの実装** — 入力ソースに基づいて段階的に実装
4. **ビルド確認** — `make fmt` + `make lint` + `make test-unit`
5. **コミット** — 変更内容を適切なメッセージでコミット
6. **Push / PR 作成** — モードに応じて Push のみ or PR 作成

## ユースケース

| ユースケース | コマンド例 |
|-------------|-----------|
| Issue のプランから新規実装 | `/implement 42` |
| PR のレビュー指摘を修正 | `/implement 15 --pr` |
| ローカルプランから新規実装 | `/implement .claude/outputs/plans/PLAN-42-xxx.md` |
| 既存ブランチで修正 | `/implement plan.md --branch feat/42-xxx` |

## 定義ファイル

[SKILL.md](SKILL.md)
