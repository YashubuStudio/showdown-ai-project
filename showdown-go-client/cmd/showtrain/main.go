package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ysu/showdown-go-client/pkg/showdown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "probe":
		runProbe(os.Args[2:])
	case "train":
		runTrain(os.Args[2:])
	case "evaluate":
		runEvaluate(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`showtrain <command>

Commands:
  probe      Check the remote Showdown server and list available formats
  train      Run self-play reinforcement learning against the server
  evaluate   Evaluate a trained checkpoint against random or another checkpoint`)
}

func runProbe(args []string) {
	fs := flag.NewFlagSet("probe", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8000", "Showdown server base URL")
	username := fs.String("username", "showtrain-probe", "Probe username")
	timeout := fs.Duration("timeout", 12*time.Second, "Probe timeout")
	fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	status, err := showdown.ProbeTrainingTarget(ctx, *server, *username)
	exitIf(err)
	printJSON(status)
}

func runTrain(args []string) {
	fs := flag.NewFlagSet("train", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8000", "Showdown server base URL")
	apiBase := fs.String("api-base", "", "Optional Go API base URL for remote team validation")
	format := fs.String("format", "gen9randombattle", "Battle format")
	battles := fs.Int("battles", 20, "Number of self-play battles")
	timeout := fs.Duration("timeout", 90*time.Second, "Per-battle timeout")
	temperature := fs.Float64("temperature", 1.0, "Policy sampling temperature; 0 uses greedy actions")
	learningRate := fs.Float64("learning-rate", 0.01, "Policy learning rate")
	hiddenSize := fs.Int("hidden-size", 64, "Hidden layer size")
	seed := fs.Int64("seed", time.Now().UnixNano(), "Random seed")
	modelPath := fs.String("model", "models/selfplay-latest.json", "Checkpoint path")
	metricsPath := fs.String("metrics", "models/selfplay-metrics.jsonl", "Optional JSONL metrics path")
	evalBattles := fs.Int("eval-battles", 10, "Evaluation battles after training")
	teamAPath := fs.String("team-a-file", "", "Optional exported team text file for player A")
	teamBPath := fs.String("team-b-file", "", "Optional exported team text file for player B")
	fs.Parse(args)

	ctx := context.Background()
	summary, err := showdown.TrainSelfPlay(ctx, showdown.AITrainingConfig{
		ServerURL:         *server,
		APIBaseURL:        *apiBase,
		Format:            *format,
		Timeout:           *timeout,
		Battles:           *battles,
		Temperature:       *temperature,
		LearningRate:      *learningRate,
		HiddenSize:        *hiddenSize,
		Seed:              *seed,
		ModelPath:         *modelPath,
		MetricsPath:       *metricsPath,
		TeamAPath:         *teamAPath,
		TeamBPath:         *teamBPath,
		EvaluationBattles: *evalBattles,
	})
	exitIf(err)
	printJSON(summary)
}

func runEvaluate(args []string) {
	fs := flag.NewFlagSet("evaluate", flag.ExitOnError)
	server := fs.String("server", "http://127.0.0.1:8000", "Showdown server base URL")
	apiBase := fs.String("api-base", "", "Optional Go API base URL for remote team validation")
	format := fs.String("format", "gen9randombattle", "Battle format")
	battles := fs.Int("battles", 20, "Number of evaluation battles")
	timeout := fs.Duration("timeout", 90*time.Second, "Per-battle timeout")
	temperature := fs.Float64("temperature", 0, "Policy temperature for evaluation; 0 uses greedy actions")
	seed := fs.Int64("seed", time.Now().UnixNano(), "Random seed")
	modelPath := fs.String("model", "models/selfplay-latest.json", "Checkpoint path")
	opponentPath := fs.String("opponent-model", "", "Optional opponent checkpoint; empty means random policy")
	teamAPath := fs.String("team-a-file", "", "Optional exported team text file for player A")
	teamBPath := fs.String("team-b-file", "", "Optional exported team text file for player B")
	fs.Parse(args)

	ctx := context.Background()
	summary, err := showdown.EvaluateModel(ctx, showdown.AIEvaluationConfig{
		ServerURL:    *server,
		APIBaseURL:   *apiBase,
		Format:       *format,
		Timeout:      *timeout,
		Battles:      *battles,
		Temperature:  *temperature,
		Seed:         *seed,
		ModelPath:    *modelPath,
		OpponentPath: *opponentPath,
		TeamAPath:    *teamAPath,
		TeamBPath:    *teamBPath,
	})
	exitIf(err)
	printJSON(summary)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Printf("failed to encode JSON output: %v", err)
	}
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
