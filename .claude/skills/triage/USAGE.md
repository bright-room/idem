# /triage スキル 使い方ガイド

## 概要

GitHub Issue の棚卸を一括実行するスキル。4つのサブスキルを順番に実行し、統合レポートを出力する。

## 使い方

```
/triage
```

引数は不要。実行すると自動的にすべての棚卸作業を行う。

## サブスキル

棚卸の各工程は独立したスキルとしても利用できる。自然言語で依頼すると自動的に呼び出される。

| スキル | 単独での使い方 | 説明 |
|--------|--------------|------|
| `close-resolved-issues` | 「対応済み Issue を閉じて」 | コードベースと PR を確認し、対応済みの Issue をクローズ |
| `label-issues` | 「ラベルついてない Issue にラベルつけて」 | ラベル未付与の Issue に種別・優先度ラベルを付与 |
| `balance-milestones` | 「マイルストーン整理して」 | マイルストーン未割り当て Issue の紐づけと平準化 |
| `create-issues-from-plans` | 「プランから Issue 作って」 | 実装プランの「今後の展望」から新規 Issue を作成 |

## 処理の流れ

1. **対応済み Issue のクローズ** (`close-resolved-issues`)
2. **既存 Issue のラベル付与** (`label-issues`)
3. **マイルストーンの平準化** (`balance-milestones`)
4. **実装プランからの Issue 作成** (`create-issues-from-plans`)
5. **棚卸レポートの出力** — `.claude/outputs/triage/TRIAGE-YYYY-MM-DD-HHmmss.md` に結果を出力

## マイルストーンの割り当て基準

ラベルに基づいて自動判断:

| ラベル | 配置の目安 |
|--------|-----------|
| `Type: Bug` | 直近のマイルストーン |
| `Type: Feature` / `Type: Enhancement` | 機能規模に応じて配置 |
| `Type: Refactoring` | 大きなリリースの前 |
| `Type: Document` | 関連機能と同じ、またはリリース前 |
| `Type: Publishing` | リリース作業用マイルストーン |

## 平準化の目安

- 1 マイルストーンあたり 5〜8 Issue
- 8 Issue 以上の場合は分散を検討

## 定義ファイル

- [triage SKILL.md](SKILL.md) — オーケストレーター
- [close-resolved-issues](../close-resolved-issues/SKILL.md)
- [label-issues](../label-issues/SKILL.md)
- [balance-milestones](../balance-milestones/SKILL.md)
- [create-issues-from-plans](../create-issues-from-plans/SKILL.md)
