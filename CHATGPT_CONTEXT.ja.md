# ChatGPT 向けプロジェクト要約

この文書は、`showdown-suite` の現状と構想を ChatGPT に説明しやすくするための要約です。  
下の「コピペ用要約」をそのまま貼れば、かなり早く文脈を共有できます。

## 1. 一言でいうと

`showdown-suite` は、ローカル LAN 上で動く Pokémon Showdown 環境と、それを操作・監視・自動化・AI 学習に使うための Go ツール群をまとめた monorepo です。

## 2. 現在の構成

- `pokemon-showdown-local`
  - Pokémon Showdown サーバー本体
  - LAN 起動/停止スクリプトあり
  - ローカル独自フォーマット `[Gen 9] Showdown Suite Studio` を持つ
- `pokemon-showdown-client-local`
  - ブラウザクライアント
  - `python3 -m http.server` で LAN 配信
- `showdown-go-client`
  - Go の CLI / HTTP API / GUI
  - サーバー状態確認、チーム検証、mock battle、ローカルルール編集ができる
  - 新しく `showtrain` という AI 学習クライアントを追加済み

## 3. 現在できること

### サーバー運用

- LAN サーバー起動
- LAN クライアント配信
- 状態確認
- 停止
- 管理者作成

### Go ツール

- `showcli ping`
  - WebSocket 疎通確認
- `showcli status`
  - 接続状態、ルーム一覧、フォーマット一覧、ローカルフォーマット定義取得
- `showcli gui`
  - ローカル GUI を起動
- `showcli serve`
  - GUI 用 HTTP API を起動
- HTTP API
  - `GET /api/healthz`
  - `GET /api/local-format`
  - `POST /api/local-format`
  - `POST /api/ping`
  - `POST /api/status`
  - `POST /api/validate-team`
  - `POST /api/mock-battle`

### ローカル独自フォーマット

- フォーマット名: `[Gen 9] Showdown Suite Studio`
- GUI または API からルール変更可能
- 変更できる主な項目:
  - singles / doubles プリセット
  - level
  - max team size
  - picked team size
  - Open Team Sheets
  - Terastal 可否
  - `targetPokemon`
  - `bannedPokemon`
  - 追加 custom rules
- 種族名は保存時に正規化される
- typo した種族名はエラーになる
- ルール違反チームは `validate-team` で明示メッセージ付きで弾かれる

### AI 学習ベース

- `showtrain probe`
  - 別 PC から対象サーバーの確認
- `showtrain train`
  - 自己対戦による教師なし強化学習
- `showtrain evaluate`
  - ランダム方策または別モデルとの評価
- 学習モデル
  - Go 実装の小さな MLP
  - policy gradient ベース
- カスタムチーム学習
  - `showcli serve` で API を立てれば別 PC から `validate-team` 経由で可能

## 4. 今のプロジェクトの構想

目指しているものは次の 3 層構成です。

### 1. 対戦基盤

- ローカルまたは LAN で使える Pokémon Showdown サーバー
- 独自フォーマットを柔軟に変えられる
- 人間が普通に対戦できる

### 2. 制御・検証基盤

- GUI でルール変更、状態確認、チーム検証、mock battle を行う
- HTTP API で他アプリや AI からも同じ操作を行える
- 情報取得アプリケーションとして使える

### 3. AI 学習基盤

- 別 PC からサーバーへ接続できる
- 自己対戦を回せる
- 戦闘結果から報酬を計算できる
- モデルを保存・再利用・比較できる
- 将来的にはより強いニューラルネットワークや探索手法に差し替えられる

## 5. 現時点での制約

- API は LAN / ローカル前提で、認証なし
- AI 学習はベースライン段階で、強い AI ではない
- 現在の状態特徴量は簡易版
- 報酬設計も単純な勝敗中心
- doubles の target 選択はまだ単純
- 本格強化には以下の拡張が必要
  - 状態表現の改善
  - 行動空間の改善
  - 報酬設計の改善
  - opponent pool
  - checkpoint 管理
  - replay / dataset 保存
  - 学習速度改善

## 6. ChatGPT に依頼するときに伝えるべき前提

最低限、次を伝えると話が早いです。

- monorepo 名は `showdown-suite`
- 主要ディレクトリは `pokemon-showdown-local`, `pokemon-showdown-client-local`, `showdown-go-client`
- 既に LAN 起動スクリプト、GUI、HTTP API、ローカルフォーマット編集、AI 学習ベースがある
- 変更は monorepo 全体を前提にしてよい
- Showdown の upstream そのものを大きく壊すのではなく、ローカル統合層と Go ツール側で拡張したい
- まずは実装よりも「どこに手を入れるべきか」「既存機能と矛盾しない設計か」を見てほしい

## 7. コピペ用要約

以下をそのまま ChatGPT に貼れます。

```text
今進めているプロジェクトは showdown-suite という monorepo です。

構成は次の 3 つです。
- pokemon-showdown-local: ローカル LAN 用の Pokémon Showdown サーバー
- pokemon-showdown-client-local: LAN 配信用のブラウザクライアント
- showdown-go-client: Go 製の CLI / HTTP API / GUI / AI 学習クライアント

現状できること:
- LAN サーバー起動/停止/状態確認
- 管理者作成
- Go GUI からサーバー状態確認、ローカルルール編集、チーム検証、mock battle
- HTTP API から status / ping / validate-team / mock-battle / local-format の取得と更新
- 独自フォーマット [Gen 9] Showdown Suite Studio の運用
- 別 PC から接続できる AI 学習クライアント showtrain

ローカル独自フォーマットでは次を変更できます:
- singles/doubles プリセット
- level
- max team size
- picked team size
- Open Team Sheets
- Terastal 可否
- targetPokemon
- bannedPokemon
- additional custom rules

仕様:
- サーバーはローカルフォーマット定義を明示配信する
- validate-team でルール違反チームを明示的メッセージ付きで拒否する
- targetPokemon / bannedPokemon は保存時に実在種族名へ正規化し、typo はエラーにする

AI 学習ベース:
- showtrain probe: 接続確認
- showtrain train: 自己対戦による教師なし強化学習
- showtrain evaluate: モデル評価
- モデルは Go 実装の小さな MLP
- 学習は policy gradient ベース
- 本格運用前のベースライン段階

今後の構想:
- 人間用の対戦/GUI基盤
- API ベースの制御/情報取得基盤
- 別 PC から自己対戦・評価・学習ができる AI 基盤

制約:
- API はローカル/LAN 前提で認証なし
- AI はまだ簡易版で、状態表現や報酬設計の改善余地が大きい
- upstream Showdown を大きく壊すより、ローカル統合層と Go ツール側で拡張したい

この前提で、今から相談したいのは:
<ここにやりたいことを書く>
```

## 8. コピペ用の依頼テンプレート

```text
showdown-suite という monorepo の相談です。

前提:
- Pokémon Showdown のローカル LAN サーバーがある
- Go 製の GUI / HTTP API / AI 学習クライアントがある
- 独自フォーマット [Gen 9] Showdown Suite Studio がある
- validate-team や local-format API はすでにある
- AI 学習は showtrain で自己対戦ベースの最小実装が入っている

今回やりたいこと:
<やりたいこと>

期待する回答:
- まず現状構成を踏まえた実装方針
- どのファイル/レイヤに責務を置くべきか
- 既存機能を壊さずに進めるための注意点
- 可能なら段階的な実装手順
```
