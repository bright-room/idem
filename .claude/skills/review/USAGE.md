# /review スキル 使い方ガイド

## 概要

コードレビューを実施するスキル。ブランチ差分または main 全体のレビューを行い、ローカルファイルとして出力する。

## 使い方

```
/review                          # feature ブランチ → main との差分レビュー（カテゴリ自動選択）
                                 # main ブランチ → コードベース全体レビュー（全カテゴリ）
/review develop                  # develop ブランチとの差分レビュー（カテゴリ自動選択）
/review --full                   # 差分レビュー（全カテゴリ）
/review --category security,code # セキュリティとコード品質のみ
/review --full-codebase          # コードベース全体レビュー（全カテゴリ）
```

## レビューモード

| 条件 | モード | レビュー対象 |
|------|--------|-------------|
| `--full-codebase` 指定 | コードベース全体レビュー | main ブランチの全コード |
| feature ブランチ + 引数なし | main との差分レビュー | `main` との差分 |
| feature ブランチ + ブランチ指定 | 指定ブランチとの差分レビュー | 指定ブランチとの差分 |
| main ブランチ + 引数なし | コードベース全体レビュー | main ブランチの全コード |

## カテゴリ選択

| 条件 | カテゴリ |
|------|---------|
| `--full` 指定 | 全6カテゴリ |
| `--category` 指定 | 指定されたカテゴリのみ |
| コードベース全体レビュー | 全6カテゴリ |
| 差分レビュー（指定なし） | 変更ファイルから自動選択 |

### カテゴリ一覧

| カテゴリ名 | 説明 |
|-----------|------|
| `architecture` | 設計パターン・パッケージ分割・依存関係 |
| `code` | ロジック・命名・Go の慣習への準拠 |
| `test` | カバレッジ・網羅性・品質 |
| `security` | 認証・入力バリデーション・並行処理の安全性 |
| `docs` | GoDoc・README・コード内コメント |
| `build` | go.mod・.golangci.yml・Makefile・Dockerfile |

### 自動選択の例

| 変更ファイル | 選択されるカテゴリ |
|-------------|-------------------|
| `middleware.go` | architecture, code |
| `redis/locker.go` | code, security |
| `internal/recorder.go` | code |
| `.golangci.yml` | build |
| `idem_test.go` | test |

## 出力先

| モード | 出力先 |
|--------|--------|
| 差分レビュー | `.claude/outputs/reviews/REVIEW-<branch-name>.md` |
| 全体レビュー | `.claude/outputs/reviews/REVIEW-main-YYYY-MM-DD.md` |

## 指摘の優先度

| 優先度 | プレフィックス | 項番例 |
|--------|--------------|--------|
| 🔴 Critical | C | C-1, C-1-1 |
| 🟠 High | H | H-1, H-1-1 |
| 🟡 Medium | M | M-1, M-1-1 |
| 🟢 Low | L | L-1, L-1-1 |

## 定義ファイル

[SKILL.md](SKILL.md)