# Showdown Suite

English version: [README.md](./README.md)

`showdown-suite` は、以下をまとめたワークスペースです。

- LAN 用のローカル Pokémon Showdown サーバー
- ローカル配信用の Showdown Web クライアント
- 自動化や運用のための Go ライブラリ、CLI、HTTP API、ブラウザ GUI

このルートはワークスペースの入口です。ビルドや依存管理は各サブディレクトリごとに行います。

## ディレクトリ構成

- `pokemon-showdown-local`
  LAN 向けのローカル Showdown サーバー本体、起動スクリプト、ローカルスタジオフォーマット定義を含みます。
- `pokemon-showdown-client-local`
  LAN スタックから配信する Showdown ブラウザクライアントです。
- `showdown-go-client`
  Go パッケージ、CLI、HTTP API、ブラウザ GUI を含みます。

## 前提条件

- Go `1.25+`
- Node.js `16+`
- ローカル Web クライアント配信用の `python3`

使うサブプロジェクトごとに依存関係を入れてください。

```bash
cd pokemon-showdown-local && npm install
cd ../pokemon-showdown-client-local && npm install
cd ../showdown-go-client && go mod download
```

## クイックスタート

### 1. LAN スタックを起動する

```bash
cd pokemon-showdown-local
./scripts/start-lan-stack.sh
```

関連コマンド:

```bash
./scripts/status-lan-stack.sh
./scripts/stop-lan-stack.sh
./scripts/create-admin.sh yourname
```

既定ポート:

- Showdown サーバー: `8000`
- 静的 Web クライアント: `8080`

`start-lan-server.sh` は `dist/` が無ければ自動でサーバービルドを行います。  
`start-lan-client.sh` は `pokemon-showdown-client-local` を `python3 -m http.server` で配信します。

### 2. ゲームクライアントを開く

次のどちらかを使います。

- サーバーのランディングページ: `http://HOST:8000`
- 直接 LAN クライアントを開く URL: `http://HOST:8080/play.pokemonshowdown.com/lan.html?~~HOST:8000&serverid=koharulocal`

`./scripts/start-lan-stack.sh` は、取得できる場合は現在の LAN IP を使った URL を表示します。
`lan.html` はローカルサーバー内で名前選択を完結させるため、公開 `testclient.html` のログイン回避手順が不要です。

### 3. Go ツールを使う

```bash
cd showdown-go-client
go run ./cmd/showcli ping --server http://127.0.0.1:8000
go run ./cmd/showcli status --server http://127.0.0.1:8000
go run ./cmd/showcli gui --addr 127.0.0.1:8099
```

GUI と API の既定 URL は `http://127.0.0.1:8099/assets/` です。

## Go クライアント概要

`showdown-go-client` の主要ディレクトリ:

- `pkg/showdown`
  埋め込み用 Go クライアントと高水準ヘルパーです。
- `cmd/showcli`
  CLI の入口です。
- `internal/httpapi`
  GUI や他アプリから使うローカル JSON API です。
- `internal/gui`
  ブラウザ GUI のアセットと HTTP ハンドラです。
- `../pokemon-showdown-local/config/showdown-suite-local-format.json`
  ローカルスタジオフォーマット設定のソースです。

### CLI コマンド

`showcli` では現在次を利用できます。

- `ping`
  WebSocket 接続と rename フローを確認します。
- `status`
  接続情報、ルーム情報、利用可能フォーマット、ローカルスタジオフォーマット定義を取得します。
- `mockbattle`
  2 つのローカルクライアントで簡易自動対戦を実行します。
- `serve`
  ブラウザを自動で開かずに HTTP API と GUI を起動します。
- `gui`
  HTTP API と GUI を起動し、ブラウザも自動で開きます。

例:

```bash
go run ./cmd/showcli ping --server http://127.0.0.1:8000 --username koharu
go run ./cmd/showcli status --server http://127.0.0.1:8000 --username koharu
go run ./cmd/showcli mockbattle --server http://127.0.0.1:8000 --format gen9randombattle --timeout 90s
go run ./cmd/showcli serve --addr 127.0.0.1:8099
go run ./cmd/showcli gui --addr 127.0.0.1:8099
```

補足: スタジオフォーマット用のカスタムチーム検証や、そのチームを使った mock battle は、現状では専用 CLI フラグではなく HTTP API と GUI から扱います。

## HTTP API

`showcli serve` または `showcli gui` 実行中は、次のエンドポイントを利用できます。

- `GET /api/healthz`
- `GET /api/local-format`
- `POST /api/local-format`
- `POST /api/ping`
- `POST /api/status`
- `POST /api/validate-team`
- `POST /api/mock-battle`

例:

```bash
curl -s http://127.0.0.1:8099/api/status \
  -H 'Content-Type: application/json' \
  -d '{"server_url":"http://127.0.0.1:8000","username":"api"}'
```

`POST /api/local-format` はスタジオフォーマット設定を更新します。`targetPokemon` と `bannedPokemon` の種族名は Showdown のデータで正規化され、存在しない名前は明示エラーで拒否されます。

## ローカルスタジオフォーマット

ローカルスタジオフォーマット名は `[Gen 9] Showdown Suite Studio` です。

編集可能な設定ファイル:

- `pokemon-showdown-local/config/showdown-suite-local-format.json`

編集方法は 2 つあります。

- Go GUI から編集する
- `POST /api/local-format` を呼ぶ

設定できる主な項目:

- シングルス / ダブルスのプリセット
- レベル、持ち込み数、選出数
- Open Team Sheets
- テラスタル可否
- `targetPokemon`
- `bannedPokemon`
- 追加の Showdown カスタムルール

生成されたルール定義はローカルフォーマットの query/API で配信され、ルール違反のチームは明示的なメッセージ付きで弾かれます。

## 補足ドキュメント

- LAN 固有メモ: `pokemon-showdown-local/LOCAL_LAN_SETUP.md`
- 引き渡しガイド: `HANDOFF.ja.md`
- ChatGPT 向け要約: `CHATGPT_CONTEXT.ja.md`
- Go クライアント詳細: `showdown-go-client/README.md`
- AI 学習ガイド: `showdown-go-client/AI_TRAINING.ja.md`
- upstream Showdown サーバー文書: `pokemon-showdown-local/README.md`
- upstream Showdown クライアント文書: `pokemon-showdown-client-local/README.md`
