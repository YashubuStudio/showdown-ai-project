# Showdown Suite 引き渡しガイド

このドキュメントは、`showdown-suite` を「GUI 付きアプリケーション + API を持つローカルサービス」として他者へ渡すときの基準です。

## 推奨する渡し方

一番安全なのは、次の 3 つをまとめて渡す形です。

- ソース一式
- 起動・停止・設定変更の手順書
- API の利用方法と制約

このワークスペースは単一バイナリではなく、次の 3 要素で構成されています。

- `pokemon-showdown-local`
  対戦サーバー本体
- `pokemon-showdown-client-local`
  ブラウザクライアント
- `showdown-go-client`
  GUI、HTTP API、CLI

そのため、「アプリ本体」としては `showdown-go-client` だけでは不十分です。サーバーとクライアントもセットで渡す必要があります。

## 引き渡し物の最小構成

最低限、次を含めてください。

- `README.md`
- `README.ja.md`
- この `HANDOFF.ja.md`
- `pokemon-showdown-local/`
- `pokemon-showdown-client-local/`
- `showdown-go-client/`

合わせて、次の情報を別紙またはチケットに明記してください。

- 使用ポート
  - Showdown サーバー: `8000`
  - 静的クライアント: `8080`
  - Go GUI / API: `8099`
- 起動コマンド
- 停止コマンド
- 管理者作成コマンド
- ローカルスタジオフォーマット設定ファイルの場所
- API のベース URL
- 外部公開しない前提かどうか

## 推奨する受け渡しパターン

### 1. 開発者向けに渡す場合

次をそのまま共有するのが基本です。

- ワークスペース全体
- `npm install` / `go mod download` 済みであるかどうか
- `start-lan-stack.sh` と `showcli gui` の使い方

この場合、相手はソースを読みながら運用できます。

### 2. 操作者向けに渡す場合

コード理解を前提にしないなら、次を明示してください。

- 起動は `pokemon-showdown-local/scripts/start-lan-stack.sh`
- 停止は `pokemon-showdown-local/scripts/stop-lan-stack.sh`
- GUI/API は `showdown-go-client` で `go run ./cmd/showcli gui --addr 127.0.0.1:8099`
- 管理者作成は `pokemon-showdown-local/scripts/create-admin.sh <username>`

この用途なら、セットアップ済みマシンか VM ごと渡す方が安定します。

### 3. API 利用者向けに渡す場合

API 利用者には、GUI ではなく次の契約を渡してください。

- API ベース URL
  - 例: `http://127.0.0.1:8099`
- 利用可能エンドポイント
- HTTP メソッド
- 代表的なリクエスト例
- エラー時の戻り方
- ローカル専用 API であること

## API 利用者へ伝えるべき内容

現在の API は次です。

- `GET /api/healthz`
  - 生存確認
- `GET /api/local-format`
  - ローカルスタジオフォーマット定義の取得
- `POST /api/local-format`
  - ローカルスタジオフォーマット設定の更新
- `POST /api/ping`
  - Showdown サーバー疎通確認
- `POST /api/status`
  - 接続情報、ルーム情報、利用可能フォーマット、ローカルフォーマット定義の取得
- `POST /api/validate-team`
  - チーム妥当性確認
- `POST /api/mock-battle`
  - 簡易自動対戦

最低限、次の例を利用者へ渡してください。

```bash
curl -s http://127.0.0.1:8099/api/status \
  -H 'Content-Type: application/json' \
  -d '{"server_url":"http://127.0.0.1:8000","username":"api"}'
```

```bash
curl -s http://127.0.0.1:8099/api/local-format
```

```bash
curl -s http://127.0.0.1:8099/api/validate-team \
  -H 'Content-Type: application/json' \
  -d '{"format_id":"gen9showdownsuitestudio","team":"<exported showdown team text>"}'
```

## 情報取得アプリとして渡す場合の説明

このワークスペースは、単に対戦するだけでなく、次の情報取得用途にも使えます。

- サーバー稼働確認
- ルーム一覧取得
- 利用可能フォーマット取得
- ローカルスタジオフォーマット定義取得
- チーム検証結果取得
- mock battle の結果ログ取得

そのため、引き渡し時には「ゲームアプリ」としてではなく、次の 2 面を分けて説明した方が伝わりやすいです。

- 人間向け UI
  - ブラウザクライアント
  - Go GUI
- システム連携向け I/F
  - HTTP API

## 設定変更の渡し方

ルール変更や対象ポケモン変更を運用で行うなら、次を必ず伝えてください。

- 設定ファイルは `pokemon-showdown-local/config/showdown-suite-local-format.json`
- 変更は GUI または `POST /api/local-format` から行える
- `targetPokemon` と `bannedPokemon` は実在種族名へ正規化される
- 存在しない種族名はエラーになる
- ルール違反チームは検証時に明示メッセージ付きで拒否される

## セキュリティ上の注意

この API は、現状ではローカル運用前提です。

- `showcli serve` / `showcli gui` の既定 bind は `127.0.0.1`
- 認証機構は入っていない
- TLS も入っていない

そのため、別マシンや外部ネットワークへ公開するなら、そのまま渡さないでください。少なくとも次が必要です。

- リバースプロキシ
- TLS
- 認証
- IP 制限

## 引き渡し前チェックリスト

- `pokemon-showdown-local/scripts/start-lan-stack.sh` でサーバーと静的クライアントが起動する
- `pokemon-showdown-local/scripts/status-lan-stack.sh` で状態確認できる
- `showdown-go-client` で `go run ./cmd/showcli gui --addr 127.0.0.1:8099` が起動する
- ブラウザで `http://127.0.0.1:8099/assets/` が開く
- `GET /api/healthz` が成功する
- `POST /api/status` が成功する
- `GET /api/local-format` が成功する
- `POST /api/local-format` で設定変更できる
- typo を含む種族名が `POST /api/local-format` で拒否される
- `POST /api/validate-team` で合法 / 非合法チームの判定が返る
- `POST /api/mock-battle` が動く
- `create-admin.sh` で管理者を作成できる

## 受け渡し時の一言での説明例

「この一式は、LAN 上で動く Pokémon Showdown サーバーと、その Web クライアント、さらに状態取得・チーム検証・ローカルルール編集・簡易自動対戦を行う GUI/API をまとめたワークスペースです。人はブラウザ GUI を使い、システム連携は HTTP API を使ってください。」
