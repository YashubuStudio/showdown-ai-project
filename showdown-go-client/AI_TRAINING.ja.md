# AI 学習クライアント

`showtrain` は、別 PC から Showdown サーバーへ接続して自己対戦学習と評価を行うためのベースです。

この実装は、教師データを前提にしない自己対戦型の強化学習ベースです。  
ニューラルネットワークは Go だけで動く小さな MLP を使っており、追加の ML フレームワークは不要です。

## できること

- リモート Showdown サーバーへの疎通確認
- 2 クライアントを自動で立ち上げて自己対戦
- 戦闘ログ由来の報酬で方策ネットワークを更新
- 学習済みモデルの保存
- ランダム方策または別モデルとの評価戦
- 必要なら既存 Go API を通じたカスタムチーム検証

## 1. サーバー PC 側の起動

### 対戦サーバーだけ使う場合

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/start-lan-stack.sh
```

既定では次が立ち上がります。

- Showdown サーバー: `8000`
- 静的クライアント: `8080`

別 PC から学習させる場合は、学習 PC から見える IP を使って接続します。

例:

- `http://192.168.1.50:8000`

### カスタムチーム検証も別 PC から使う場合

Go API も外から見えるように起動します。

```bash
cd /home/ysu/projects/showdown-suite/showdown-go-client
go run ./cmd/showcli serve --addr 0.0.0.0:8099
```

この場合、学習 PC からは例えば次を使えます。

- Showdown サーバー: `http://192.168.1.50:8000`
- Go API: `http://192.168.1.50:8099`

注意:

- API には認証がありません
- LAN 内だけで使う前提です

## 2. 学習 PC 側の使い方

### サーバー確認

```bash
cd /home/ysu/projects/showdown-suite/showdown-go-client
go run ./cmd/showtrain probe --server http://192.168.1.50:8000
```

### ランダムバトルで自己対戦学習

```bash
go run ./cmd/showtrain train \
  --server http://192.168.1.50:8000 \
  --format gen9randombattle \
  --battles 50 \
  --model models/selfplay-latest.json \
  --metrics models/selfplay-metrics.jsonl
```

### 学習済みモデルをランダム方策と評価

```bash
go run ./cmd/showtrain evaluate \
  --server http://192.168.1.50:8000 \
  --format gen9randombattle \
  --battles 20 \
  --model models/selfplay-latest.json
```

### 学習済みモデル同士を評価

```bash
go run ./cmd/showtrain evaluate \
  --server http://192.168.1.50:8000 \
  --format gen9randombattle \
  --battles 20 \
  --model models/selfplay-latest.json \
  --opponent-model models/older-checkpoint.json
```

## 3. ローカルスタジオフォーマットを使う場合

`[Gen 9] Showdown Suite Studio` で学習または評価したい場合、ランダムバトルのように自動チーム生成はできません。  
そのため、エクスポート済みチームテキストを 2 つ渡し、Go API による `validate-team` を使います。

```bash
go run ./cmd/showtrain train \
  --server http://192.168.1.50:8000 \
  --api-base http://192.168.1.50:8099 \
  --format gen9showdownsuitestudio \
  --team-a-file /path/to/team-a.txt \
  --team-b-file /path/to/team-b.txt \
  --battles 20 \
  --model models/studio-selfplay.json
```

## 4. 学習の中身

現在のベースラインは次の仕様です。

- 状態特徴:
  - 自軍の HP、状態異常、行動可能数、ベンチ数、ターン数など
- 行動特徴:
  - move / switch / pass の種類
  - move 番号
  - target 情報
  - switch 先
- モデル:
  - 1 隠れ層の小さな MLP
- 学習:
  - 自己対戦の勝敗報酬による policy gradient

強い AI にはまだ遠いですが、別 PC から回せる「戦わせる」「評価する」「学習済み重みを残す」基盤としては使えます。

## 5. 出力物

- `--model`
  - 学習済みモデルの JSON
- `--metrics`
  - 各バトルの記録を JSONL で保存

## 6. 限界

- 現状の特徴量は簡易版です
- ダブルスでも動作しますが、target 選択はまだ単純です
- policy gradient のみなので学習効率は高くありません
- 真面目に強化したいなら、状態表現、報酬設計、探索、対戦相手プールを拡張する必要があります
