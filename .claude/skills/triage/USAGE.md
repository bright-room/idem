# /triage スキル 使い方ガイド

## 概要

GitHub Issue の棚卸を一括実行するスキル。マイルストーンクローズ後のリリースタイミングでの実行を想定。5つのステップを順番に実行し、結果を GitHub Discussions に投稿する。

## 使い方

```
/triage              # 最新のクローズ済みマイルストーンを対象に棚卸
/triage v0.5.0       # 指定マイルストーンを対象に棚卸
```

## 処理の流れ

1. **対応済み Issue のクローズ** — コードベースと PR を確認し、対応済みの Issue をクローズ
2. **既存 Issue のラベル付与** — ラベル不足の Issue に Kind・Priority・補助ラベルを付与
3. **実装プランからの Issue 作成** — 対象マイルストーンに紐づく Issue のプランコメント（`<!-- claude:plan -->`）から「今後の展望」を読み取り、新規 Issue を作成
4. **マイルストーンの平準化** — 全 Issue（既存 + 新規）を対象に Semver ルールと Priority に基づいてマイルストーンを割り当て・平準化
5. **棚卸レポートの出力** — GitHub Discussions（Triage Reports カテゴリ）に投稿

## Step 3 の対象範囲

| 条件 | 対象 |
|------|------|
| 引数あり | 指定マイルストーンに紐づくクローズ済み Issue |
| 引数なし | 最新のクローズ済みマイルストーンに紐づくクローズ済み Issue |

- 正常にクローズされた Issue（`COMPLETED`）のみ対象
- `Close as not planned` や `Close as duplicate` は除外

## ラベル体系

### Kind ラベル（種別）

| ラベル | 内容 |
|--------|------|
| `Kind: Bug Fix` | バグ報告・不具合修正 |
| `Kind: Feature` | 新しい機能の追加 |
| `Kind: Enhancement` | 既存機能の改善・拡張 |
| `Kind: Refactoring` | コードの整理・改善（API 破壊なし） |
| `Kind: Tests` | テストの追加・改善 |
| `Kind: Documentation` | ドキュメントの追加・改善 |

### Priority ラベル（優先度）

| ラベル | 基準 |
|--------|------|
| `Priority: Critical` | サービス停止や重大なセキュリティ問題 |
| `Priority: High` | ユーザーに直接影響する |
| `Priority: Medium` | 品質向上に寄与するが、緊急性は低い |
| `Priority: Low` | あると良いが、なくても問題ない |

### 補助ラベル

| ラベル | 条件 |
|--------|------|
| `Impact: Breaking` | 後方互換性が失われる変更 |
| `Need: Discussion` | 設計上の議論が必要 |
| `Need: Help Wanted` | 他者の協力が必要 |

## Semver ルール

Kind と Impact の組み合わせでバージョン区分を決定:

| 区分 | 対象ラベル |
|------|----------|
| **Patch** | `Kind: Bug Fix`, `Kind: Tests`, `Kind: Documentation` |
| **Minor** | `Kind: Feature`, `Kind: Enhancement`, `Kind: Refactoring` |
| **Major** | `Impact: Breaking` + Feature/Enhancement/Refactoring |

## マイルストーン配置の優先度

| Priority | 配置位置 |
|----------|---------|
| Critical / High | 直近のマイルストーン |
| Medium | 中間のマイルストーン |
| Low | 後方のマイルストーン |

## 平準化の目安

- 1 マイルストーンあたり 5〜8 Issue
- 8 Issue 以上の場合は分散を検討

## 前提条件

- `gh` CLI が認証済みであること
- GitHub Discussions が有効で「Triage Reports」カテゴリが存在すること（なければローカルファイルにフォールバック）

## 定義ファイル

[SKILL.md](SKILL.md)