package showdown

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	aiStateFeatureSize  = 64
	aiActionFeatureSize = 16
	aiModelInputSize    = aiStateFeatureSize + aiActionFeatureSize
)

type AITrainingConfig struct {
	ServerURL         string        `json:"server_url"`
	APIBaseURL        string        `json:"api_base_url,omitempty"`
	Format            string        `json:"format"`
	Timeout           time.Duration `json:"timeout"`
	Battles           int           `json:"battles"`
	Temperature       float64       `json:"temperature"`
	LearningRate      float64       `json:"learning_rate"`
	HiddenSize        int           `json:"hidden_size"`
	Seed              int64         `json:"seed"`
	ModelPath         string        `json:"model_path"`
	MetricsPath       string        `json:"metrics_path,omitempty"`
	TeamAPath         string        `json:"team_a_path,omitempty"`
	TeamBPath         string        `json:"team_b_path,omitempty"`
	EvaluationBattles int           `json:"evaluation_battles,omitempty"`
}

type AITrainingSummary struct {
	ServerURL        string               `json:"server_url"`
	Format           string               `json:"format"`
	BattlesRequested int                  `json:"battles_requested"`
	BattlesCompleted int                  `json:"battles_completed"`
	AverageTurns     float64              `json:"average_turns"`
	ModelPath        string               `json:"model_path,omitempty"`
	MetricsPath      string               `json:"metrics_path,omitempty"`
	Baseline         float64              `json:"baseline"`
	StartedAt        time.Time            `json:"started_at"`
	FinishedAt       time.Time            `json:"finished_at"`
	LastBattle       *AIBattleRecord      `json:"last_battle,omitempty"`
	Evaluation       *AIEvaluationSummary `json:"evaluation,omitempty"`
}

type AIEvaluationConfig struct {
	ServerURL    string        `json:"server_url"`
	APIBaseURL   string        `json:"api_base_url,omitempty"`
	Format       string        `json:"format"`
	Timeout      time.Duration `json:"timeout"`
	Battles      int           `json:"battles"`
	Temperature  float64       `json:"temperature"`
	Seed         int64         `json:"seed"`
	ModelPath    string        `json:"model_path"`
	OpponentPath string        `json:"opponent_path,omitempty"`
	TeamAPath    string        `json:"team_a_path,omitempty"`
	TeamBPath    string        `json:"team_b_path,omitempty"`
}

type AIEvaluationSummary struct {
	ServerURL        string          `json:"server_url"`
	Format           string          `json:"format"`
	BattlesRequested int             `json:"battles_requested"`
	BattlesCompleted int             `json:"battles_completed"`
	Wins             int             `json:"wins"`
	Losses           int             `json:"losses"`
	Draws            int             `json:"draws"`
	WinRate          float64         `json:"win_rate"`
	AverageTurns     float64         `json:"average_turns"`
	ModelPath        string          `json:"model_path"`
	Opponent         string          `json:"opponent"`
	StartedAt        time.Time       `json:"started_at"`
	FinishedAt       time.Time       `json:"finished_at"`
	LastBattle       *AIBattleRecord `json:"last_battle,omitempty"`
}

type AIBattleRecord struct {
	BattleNumber int           `json:"battle_number"`
	Format       string        `json:"format"`
	BattleID     string        `json:"battle_id"`
	PlayerA      string        `json:"player_a"`
	PlayerB      string        `json:"player_b"`
	Winner       string        `json:"winner"`
	Completed    bool          `json:"completed"`
	Turns        int           `json:"turns"`
	Duration     time.Duration `json:"duration"`
	RewardA      float64       `json:"reward_a"`
	RewardB      float64       `json:"reward_b"`
	DecisionsA   int           `json:"decisions_a"`
	DecisionsB   int           `json:"decisions_b"`
	RecordedAt   time.Time     `json:"recorded_at"`
}

type aiCheckpoint struct {
	InputSize      int       `json:"input_size"`
	HiddenSize     int       `json:"hidden_size"`
	W1             []float64 `json:"w1"`
	B1             []float64 `json:"b1"`
	W2             []float64 `json:"w2"`
	B2             float64   `json:"b2"`
	Baseline       float64   `json:"baseline"`
	BattlesTrained int       `json:"battles_trained"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type aiGradient struct {
	W1 []float64
	B1 []float64
	W2 []float64
	B2 float64
}

type aiForwardCache struct {
	Input  []float64
	Hidden []float64
}

type aiDecisionTrace struct {
	Inputs [][]float64
	Chosen int
}

type aiPlayerController interface {
	selectAction(rng *rand.Rand, request map[string]any, turn int) (string, *aiDecisionTrace, error)
}

type aiModelController struct {
	checkpoint  *aiCheckpoint
	temperature float64
}

type aiRandomController struct{}

type aiBattleEpisode struct {
	result     MockBattleResult
	decisionsA []aiDecisionTrace
	decisionsB []aiDecisionTrace
}

type aiCandidate struct {
	Command string
	Input   []float64
}

type aiAtomicChoice struct {
	Command    string
	Kind       int
	MoveIndex  int
	TargetCode int
	SwitchSlot int
	TeamLead   int
}

func TrainSelfPlay(ctx context.Context, cfg AITrainingConfig) (AITrainingSummary, error) {
	cfg = normalizeAITrainingConfig(cfg)
	started := time.Now()

	checkpoint, err := loadOrCreateCheckpoint(cfg.ModelPath, cfg.HiddenSize, cfg.Seed)
	if err != nil {
		return AITrainingSummary{}, err
	}

	packedA, packedB, err := loadPackedTeams(ctx, cfg.APIBaseURL, cfg.Format, cfg.TeamAPath, cfg.TeamBPath)
	if err != nil {
		return AITrainingSummary{}, err
	}

	rng := rand.New(rand.NewSource(cfg.Seed))
	writer, closer, err := openMetricsWriter(cfg.MetricsPath)
	if err != nil {
		return AITrainingSummary{}, err
	}
	if closer != nil {
		defer closer()
	}

	var (
		totalTurns int
		lastBattle *AIBattleRecord
	)
	for battle := 1; battle <= cfg.Battles; battle++ {
		episode, err := runAIBattleEpisode(ctx, rng, aiBattleConfig{
			ServerURL:   cfg.ServerURL,
			Format:      cfg.Format,
			Timeout:     cfg.Timeout,
			PackedTeamA: packedA,
			PackedTeamB: packedB,
			ControllerA: &aiModelController{checkpoint: checkpoint, temperature: cfg.Temperature},
			ControllerB: &aiModelController{checkpoint: checkpoint, temperature: cfg.Temperature},
		})
		if err != nil {
			return AITrainingSummary{}, err
		}

		rewardA, rewardB := aiRewards(episode.result, episode.result.PlayerA, episode.result.PlayerB)
		checkpoint.trainEpisode(episode.decisionsA, rewardA, cfg.LearningRate)
		checkpoint.trainEpisode(episode.decisionsB, rewardB, cfg.LearningRate)
		checkpoint.UpdatedAt = time.Now()
		checkpoint.BattlesTrained++
		totalTurns += episode.result.Turns

		record := &AIBattleRecord{
			BattleNumber: battle,
			Format:       episode.result.Format,
			BattleID:     episode.result.BattleID,
			PlayerA:      episode.result.PlayerA,
			PlayerB:      episode.result.PlayerB,
			Winner:       episode.result.Winner,
			Completed:    episode.result.Completed,
			Turns:        episode.result.Turns,
			Duration:     episode.result.Duration,
			RewardA:      rewardA,
			RewardB:      rewardB,
			DecisionsA:   len(episode.decisionsA),
			DecisionsB:   len(episode.decisionsB),
			RecordedAt:   time.Now(),
		}
		lastBattle = record
		if writer != nil {
			if err := json.NewEncoder(writer).Encode(record); err != nil {
				return AITrainingSummary{}, err
			}
		}
		if cfg.ModelPath != "" {
			if err := checkpoint.save(cfg.ModelPath); err != nil {
				return AITrainingSummary{}, err
			}
		}
	}

	var evaluation *AIEvaluationSummary
	if cfg.EvaluationBattles > 0 {
		summary, err := EvaluateModel(ctx, AIEvaluationConfig{
			ServerURL:   cfg.ServerURL,
			APIBaseURL:  cfg.APIBaseURL,
			Format:      cfg.Format,
			Timeout:     cfg.Timeout,
			Battles:     cfg.EvaluationBattles,
			Temperature: 0,
			Seed:        cfg.Seed + 7919,
			ModelPath:   cfg.ModelPath,
			TeamAPath:   cfg.TeamAPath,
			TeamBPath:   cfg.TeamBPath,
		})
		if err != nil {
			return AITrainingSummary{}, err
		}
		evaluation = &summary
	}

	finished := time.Now()
	summary := AITrainingSummary{
		ServerURL:        cfg.ServerURL,
		Format:           cfg.Format,
		BattlesRequested: cfg.Battles,
		BattlesCompleted: cfg.Battles,
		AverageTurns:     safeAverageTurns(totalTurns, cfg.Battles),
		ModelPath:        cfg.ModelPath,
		MetricsPath:      cfg.MetricsPath,
		Baseline:         checkpoint.Baseline,
		StartedAt:        started,
		FinishedAt:       finished,
		LastBattle:       lastBattle,
		Evaluation:       evaluation,
	}
	return summary, nil
}

func EvaluateModel(ctx context.Context, cfg AIEvaluationConfig) (AIEvaluationSummary, error) {
	cfg = normalizeAIEvaluationConfig(cfg)
	started := time.Now()

	model, err := loadCheckpoint(cfg.ModelPath)
	if err != nil {
		return AIEvaluationSummary{}, err
	}
	var opponent aiPlayerController = &aiRandomController{}
	opponentLabel := "random"
	if strings.TrimSpace(cfg.OpponentPath) != "" {
		opponentModel, err := loadCheckpoint(cfg.OpponentPath)
		if err != nil {
			return AIEvaluationSummary{}, err
		}
		opponent = &aiModelController{checkpoint: opponentModel, temperature: cfg.Temperature}
		opponentLabel = cfg.OpponentPath
	}

	packedA, packedB, err := loadPackedTeams(ctx, cfg.APIBaseURL, cfg.Format, cfg.TeamAPath, cfg.TeamBPath)
	if err != nil {
		return AIEvaluationSummary{}, err
	}

	rng := rand.New(rand.NewSource(cfg.Seed))
	var (
		wins, losses, draws int
		totalTurns          int
		lastBattle          *AIBattleRecord
	)
	for battle := 1; battle <= cfg.Battles; battle++ {
		episode, err := runAIBattleEpisode(ctx, rng, aiBattleConfig{
			ServerURL:   cfg.ServerURL,
			Format:      cfg.Format,
			Timeout:     cfg.Timeout,
			PackedTeamA: packedA,
			PackedTeamB: packedB,
			ControllerA: &aiModelController{checkpoint: model, temperature: cfg.Temperature},
			ControllerB: opponent,
		})
		if err != nil {
			return AIEvaluationSummary{}, err
		}

		rewardA, rewardB := aiRewards(episode.result, episode.result.PlayerA, episode.result.PlayerB)
		switch {
		case rewardA > rewardB:
			wins++
		case rewardB > rewardA:
			losses++
		default:
			draws++
		}
		totalTurns += episode.result.Turns
		lastBattle = &AIBattleRecord{
			BattleNumber: battle,
			Format:       episode.result.Format,
			BattleID:     episode.result.BattleID,
			PlayerA:      episode.result.PlayerA,
			PlayerB:      episode.result.PlayerB,
			Winner:       episode.result.Winner,
			Completed:    episode.result.Completed,
			Turns:        episode.result.Turns,
			Duration:     episode.result.Duration,
			RewardA:      rewardA,
			RewardB:      rewardB,
			DecisionsA:   len(episode.decisionsA),
			DecisionsB:   len(episode.decisionsB),
			RecordedAt:   time.Now(),
		}
	}

	finished := time.Now()
	completed := wins + losses + draws
	return AIEvaluationSummary{
		ServerURL:        cfg.ServerURL,
		Format:           cfg.Format,
		BattlesRequested: cfg.Battles,
		BattlesCompleted: completed,
		Wins:             wins,
		Losses:           losses,
		Draws:            draws,
		WinRate:          safeWinRate(wins, completed),
		AverageTurns:     safeAverageTurns(totalTurns, completed),
		ModelPath:        cfg.ModelPath,
		Opponent:         opponentLabel,
		StartedAt:        started,
		FinishedAt:       finished,
		LastBattle:       lastBattle,
	}, nil
}

func ProbeTrainingTarget(ctx context.Context, serverURL, username string) (ServerStatus, error) {
	if strings.TrimSpace(username) == "" {
		username = "showtrain-probe"
	}
	return FetchStatus(ctx, serverURL, username)
}

type aiBattleConfig struct {
	ServerURL   string
	Format      string
	Timeout     time.Duration
	PackedTeamA string
	PackedTeamB string
	ControllerA aiPlayerController
	ControllerB aiPlayerController
}

func runAIBattleEpisode(ctx context.Context, rng *rand.Rand, cfg aiBattleConfig) (aiBattleEpisode, error) {
	format := cfg.Format
	if strings.TrimSpace(format) == "" {
		format = "gen9randombattle"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	playerA, err := NewClient(cfg.ServerURL, randomUsername("ailearn-a"))
	if err != nil {
		return aiBattleEpisode{}, err
	}
	defer playerA.Close()
	playerB, err := NewClient(cfg.ServerURL, randomUsername("ailearn-b"))
	if err != nil {
		return aiBattleEpisode{}, err
	}
	defer playerB.Close()

	for _, client := range []*Client{playerA, playerB} {
		if err := client.Connect(ctx); err != nil {
			return aiBattleEpisode{}, err
		}
		if err := client.Rename(ctx); err != nil {
			return aiBattleEpisode{}, err
		}
		if err := waitForNamed(ctx, client); err != nil {
			return aiBattleEpisode{}, err
		}
	}
	if err := applyBattleTeam(ctx, playerA, cfg.PackedTeamA); err != nil {
		return aiBattleEpisode{}, err
	}
	if err := applyBattleTeam(ctx, playerB, cfg.PackedTeamB); err != nil {
		return aiBattleEpisode{}, err
	}
	if err := playerA.Send(ctx, "", fmt.Sprintf("/challenge %s, %s", playerB.Username(), format)); err != nil {
		return aiBattleEpisode{}, err
	}
	if err := waitForIncomingChallenge(ctx, playerB, playerA.Username()); err != nil {
		return aiBattleEpisode{}, err
	}
	if err := playerB.Send(ctx, "", fmt.Sprintf("/accept %s", playerA.Username())); err != nil {
		return aiBattleEpisode{}, err
	}

	started := time.Now()
	result := MockBattleResult{
		ServerURL: cfg.ServerURL,
		Format:    format,
		PlayerA:   playerA.Username(),
		PlayerB:   playerB.Username(),
		LogLines:  make([]string, 0, 64),
	}

	var (
		turnA   int
		turnB   int
		episode aiBattleEpisode
	)

	for {
		select {
		case err := <-playerA.Errors():
			return aiBattleEpisode{}, err
		case err := <-playerB.Errors():
			return aiBattleEpisode{}, err
		case <-ctx.Done():
			result.Duration = time.Since(started)
			result.FinishedAt = time.Now()
			result.LogLines = append(result.LogLines, "ai battle timed out before completion")
			episode.result = result
			return episode, nil
		case line := <-playerA.Lines():
			done, trace, nextTurn, err := processAIPlayerLine(ctx, rng, cfg.ControllerA, playerA, line, &result, started, turnA)
			if err != nil {
				return aiBattleEpisode{}, err
			}
			turnA = nextTurn
			if trace != nil {
				episode.decisionsA = append(episode.decisionsA, *trace)
			}
			if done {
				episode.result = result
				return episode, nil
			}
		case line := <-playerB.Lines():
			done, trace, nextTurn, err := processAIPlayerLine(ctx, rng, cfg.ControllerB, playerB, line, &result, started, turnB)
			if err != nil {
				return aiBattleEpisode{}, err
			}
			turnB = nextTurn
			if trace != nil {
				episode.decisionsB = append(episode.decisionsB, *trace)
			}
			if done {
				episode.result = result
				return episode, nil
			}
		}
	}
}

func processAIPlayerLine(
	ctx context.Context,
	rng *rand.Rand,
	controller aiPlayerController,
	client *Client,
	line Line,
	result *MockBattleResult,
	started time.Time,
	turn int,
) (bool, *aiDecisionTrace, int, error) {
	if line.Raw == "" {
		return false, nil, turn, nil
	}
	if strings.HasPrefix(line.RoomID, "battle-") && result.BattleID == "" {
		result.BattleID = line.RoomID
	}
	if strings.HasPrefix(line.RoomID, "battle-") && len(result.LogLines) < 120 {
		result.LogLines = append(result.LogLines, fmt.Sprintf("%s %s", line.RoomID, line.Raw))
	}
	if line.RoomID == result.BattleID && strings.HasPrefix(line.Raw, "|turn|") {
		fmt.Sscanf(strings.TrimPrefix(line.Raw, "|turn|"), "%d", &result.Turns)
		turn = result.Turns
	}
	if strings.HasPrefix(line.Raw, "|request|") && strings.HasPrefix(line.RoomID, "battle-") {
		raw := strings.TrimPrefix(line.Raw, "|request|")
		if raw != "" {
			var request map[string]any
			if err := json.Unmarshal([]byte(raw), &request); err != nil {
				return false, nil, turn, nil
			}
			command, trace, err := controller.selectAction(rng, request, turn)
			if err != nil {
				return false, nil, turn, err
			}
			if command != "" {
				if err := client.Send(ctx, line.RoomID, "/choose "+command); err != nil {
					return false, nil, turn, err
				}
			}
			return false, trace, turn, nil
		}
	}
	if strings.HasPrefix(line.Raw, "|win|") && line.RoomID == result.BattleID {
		result.Winner = strings.TrimPrefix(line.Raw, "|win|")
		result.Completed = true
		result.Duration = time.Since(started)
		result.FinishedAt = time.Now()
		return true, nil, turn, nil
	}
	return false, nil, turn, nil
}

func (c *aiModelController) selectAction(rng *rand.Rand, request map[string]any, turn int) (string, *aiDecisionTrace, error) {
	candidates, err := buildAICandidates(request, turn)
	if err != nil {
		return "", nil, err
	}
	if len(candidates) == 0 {
		return "", nil, nil
	}
	if len(candidates) == 1 {
		return candidates[0].Command, nil, nil
	}
	chosen, err := c.checkpoint.chooseIndex(rng, candidates, c.temperature)
	if err != nil {
		return "", nil, err
	}
	inputs := make([][]float64, 0, len(candidates))
	for _, candidate := range candidates {
		inputs = append(inputs, candidate.Input)
	}
	return candidates[chosen].Command, &aiDecisionTrace{
		Inputs: inputs,
		Chosen: chosen,
	}, nil
}

func (c *aiRandomController) selectAction(rng *rand.Rand, request map[string]any, turn int) (string, *aiDecisionTrace, error) {
	candidates, err := buildAICandidates(request, turn)
	if err != nil {
		return "", nil, err
	}
	if len(candidates) == 0 {
		return "", nil, nil
	}
	return candidates[rng.Intn(len(candidates))].Command, nil, nil
}

func normalizeAITrainingConfig(cfg AITrainingConfig) AITrainingConfig {
	if strings.TrimSpace(cfg.ServerURL) == "" {
		cfg.ServerURL = "http://127.0.0.1:8000"
	}
	if strings.TrimSpace(cfg.Format) == "" {
		cfg.Format = "gen9randombattle"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 90 * time.Second
	}
	if cfg.Battles <= 0 {
		cfg.Battles = 20
	}
	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	if cfg.LearningRate <= 0 {
		cfg.LearningRate = 0.01
	}
	if cfg.HiddenSize <= 0 {
		cfg.HiddenSize = 64
	}
	if cfg.Seed == 0 {
		cfg.Seed = time.Now().UnixNano()
	}
	return cfg
}

func normalizeAIEvaluationConfig(cfg AIEvaluationConfig) AIEvaluationConfig {
	if strings.TrimSpace(cfg.ServerURL) == "" {
		cfg.ServerURL = "http://127.0.0.1:8000"
	}
	if strings.TrimSpace(cfg.Format) == "" {
		cfg.Format = "gen9randombattle"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 90 * time.Second
	}
	if cfg.Battles <= 0 {
		cfg.Battles = 10
	}
	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	if cfg.Seed == 0 {
		cfg.Seed = time.Now().UnixNano()
	}
	return cfg
}

func loadOrCreateCheckpoint(path string, hiddenSize int, seed int64) (*aiCheckpoint, error) {
	if strings.TrimSpace(path) != "" {
		if checkpoint, err := loadCheckpoint(path); err == nil {
			return checkpoint, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	rng := rand.New(rand.NewSource(seed))
	return newCheckpoint(aiModelInputSize, hiddenSize, rng), nil
}

func loadCheckpoint(path string) (*aiCheckpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var checkpoint aiCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, err
	}
	if checkpoint.InputSize != aiModelInputSize {
		return nil, fmt.Errorf("checkpoint input size %d does not match current input size %d", checkpoint.InputSize, aiModelInputSize)
	}
	if len(checkpoint.W1) != checkpoint.InputSize*checkpoint.HiddenSize ||
		len(checkpoint.B1) != checkpoint.HiddenSize ||
		len(checkpoint.W2) != checkpoint.HiddenSize {
		return nil, errors.New("checkpoint shape is invalid")
	}
	return &checkpoint, nil
}

func newCheckpoint(inputSize, hiddenSize int, rng *rand.Rand) *aiCheckpoint {
	checkpoint := &aiCheckpoint{
		InputSize:  inputSize,
		HiddenSize: hiddenSize,
		W1:         make([]float64, inputSize*hiddenSize),
		B1:         make([]float64, hiddenSize),
		W2:         make([]float64, hiddenSize),
	}
	for i := range checkpoint.W1 {
		checkpoint.W1[i] = (rng.Float64()*2 - 1) * 0.05
	}
	for i := range checkpoint.W2 {
		checkpoint.W2[i] = (rng.Float64()*2 - 1) * 0.05
	}
	checkpoint.UpdatedAt = time.Now()
	return checkpoint
}

func (c *aiCheckpoint) save(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (c *aiCheckpoint) trainEpisode(decisions []aiDecisionTrace, reward, learningRate float64) {
	if len(decisions) == 0 {
		c.updateBaseline(reward)
		return
	}
	advantage := reward - c.Baseline
	grad := aiGradient{
		W1: make([]float64, len(c.W1)),
		B1: make([]float64, len(c.B1)),
		W2: make([]float64, len(c.W2)),
	}
	scale := 1 / float64(len(decisions))
	for _, decision := range decisions {
		scores := make([]float64, len(decision.Inputs))
		caches := make([]aiForwardCache, len(decision.Inputs))
		for idx, input := range decision.Inputs {
			score, cache := c.forward(input)
			scores[idx] = score
			caches[idx] = cache
		}
		probs := softmax(scores)
		for idx := range probs {
			gradScalar := advantage * probs[idx] * scale
			if idx == decision.Chosen {
				gradScalar -= advantage * scale
			}
			c.accumulateGradient(caches[idx], gradScalar, &grad)
		}
	}
	c.applyGradient(grad, learningRate)
	c.updateBaseline(reward)
}

func (c *aiCheckpoint) updateBaseline(reward float64) {
	if c.BattlesTrained == 0 && c.Baseline == 0 {
		c.Baseline = reward
		return
	}
	c.Baseline = c.Baseline*0.95 + reward*0.05
}

func (c *aiCheckpoint) chooseIndex(rng *rand.Rand, candidates []aiCandidate, temperature float64) (int, error) {
	if len(candidates) == 0 {
		return 0, errors.New("no candidates")
	}
	if len(candidates) == 1 {
		return 0, nil
	}
	scores := make([]float64, len(candidates))
	for idx, candidate := range candidates {
		score, _ := c.forward(candidate.Input)
		scores[idx] = score
	}
	if temperature <= 0 {
		bestIndex := 0
		bestScore := scores[0]
		for idx := 1; idx < len(scores); idx++ {
			if scores[idx] > bestScore {
				bestIndex = idx
				bestScore = scores[idx]
			}
		}
		return bestIndex, nil
	}
	for idx := range scores {
		scores[idx] /= temperature
	}
	probs := softmax(scores)
	roll := rng.Float64()
	acc := 0.0
	for idx, prob := range probs {
		acc += prob
		if roll <= acc {
			return idx, nil
		}
	}
	return len(probs) - 1, nil
}

func (c *aiCheckpoint) forward(input []float64) (float64, aiForwardCache) {
	hidden := make([]float64, c.HiddenSize)
	for h := 0; h < c.HiddenSize; h++ {
		sum := c.B1[h]
		row := h * c.InputSize
		for i := 0; i < c.InputSize; i++ {
			sum += c.W1[row+i] * input[i]
		}
		hidden[h] = math.Tanh(sum)
	}
	score := c.B2
	for h := 0; h < c.HiddenSize; h++ {
		score += c.W2[h] * hidden[h]
	}
	cache := aiForwardCache{
		Input:  append([]float64(nil), input...),
		Hidden: hidden,
	}
	return score, cache
}

func (c *aiCheckpoint) accumulateGradient(cache aiForwardCache, scalar float64, grad *aiGradient) {
	grad.B2 += scalar
	for h := 0; h < c.HiddenSize; h++ {
		grad.W2[h] += scalar * cache.Hidden[h]
		hiddenGrad := scalar * c.W2[h] * (1 - cache.Hidden[h]*cache.Hidden[h])
		grad.B1[h] += hiddenGrad
		row := h * c.InputSize
		for i := 0; i < c.InputSize; i++ {
			grad.W1[row+i] += hiddenGrad * cache.Input[i]
		}
	}
}

func (c *aiCheckpoint) applyGradient(grad aiGradient, learningRate float64) {
	for i := range c.W1 {
		c.W1[i] -= learningRate * grad.W1[i]
	}
	for i := range c.B1 {
		c.B1[i] -= learningRate * grad.B1[i]
		c.W2[i] -= learningRate * grad.W2[i]
	}
	c.B2 -= learningRate * grad.B2
}

func buildAICandidates(request map[string]any, turn int) ([]aiCandidate, error) {
	if wait, _ := request["wait"].(bool); wait {
		return nil, nil
	}
	side, _ := request["side"].(map[string]any)
	pokemon, _ := side["pokemon"].([]any)
	stateFeatures := encodeAIState(request, pokemon, turn)

	if request["teamPreview"] != nil {
		order := make([]int, 0, len(pokemon))
		for idx := range pokemon {
			order = append(order, idx+1)
		}
		parts := make([]string, 0, len(order))
		for _, idx := range order {
			parts = append(parts, fmt.Sprintf("%d", idx))
		}
		features := encodeAIActionFeatures([]aiAtomicChoice{{
			Command:  "team " + strings.Join(parts, ""),
			Kind:     3,
			TeamLead: 1,
		}})
		return []aiCandidate{{
			Command: "team " + strings.Join(parts, ""),
			Input:   append(stateFeatures, features...),
		}}, nil
	}

	if rawForce, ok := request["forceSwitch"].([]any); ok && len(rawForce) > 0 {
		force := make([]bool, len(rawForce))
		for idx, item := range rawForce {
			force[idx], _ = item.(bool)
		}
		return combineAIChoices(stateFeatures, buildForceSwitchChoices(pokemon, force), nil), nil
	}

	rawActive, ok := request["active"].([]any)
	if !ok || len(rawActive) == 0 {
		return nil, nil
	}
	perSlot := make([][]aiAtomicChoice, 0, len(rawActive))
	for idx, raw := range rawActive {
		active, _ := raw.(map[string]any)
		choices := buildTurnChoicesForSlot(active, pokemon, idx, len(rawActive))
		perSlot = append(perSlot, choices)
	}
	return combineAIChoices(stateFeatures, perSlot, nil), nil
}

func combineAIChoices(state []float64, perSlot [][]aiAtomicChoice, prefix []aiAtomicChoice) []aiCandidate {
	if len(perSlot) == 0 {
		commandParts := make([]string, 0, len(prefix))
		for _, item := range prefix {
			commandParts = append(commandParts, item.Command)
		}
		command := strings.Join(commandParts, ", ")
		if strings.TrimSpace(command) == "" {
			return nil
		}
		actionFeatures := encodeAIActionFeatures(prefix)
		return []aiCandidate{{
			Command: command,
			Input:   append(append([]float64(nil), state...), actionFeatures...),
		}}
	}

	candidates := make([]aiCandidate, 0, 16)
	for _, choice := range perSlot[0] {
		if aiHasSwitchConflict(prefix, choice) {
			continue
		}
		next := append(append([]aiAtomicChoice(nil), prefix...), choice)
		candidates = append(candidates, combineAIChoices(state, perSlot[1:], next)...)
	}
	return candidates
}

func aiHasSwitchConflict(prefix []aiAtomicChoice, candidate aiAtomicChoice) bool {
	if candidate.Kind != 1 || candidate.SwitchSlot == 0 {
		return false
	}
	for _, item := range prefix {
		if item.Kind == 1 && item.SwitchSlot == candidate.SwitchSlot {
			return true
		}
	}
	return false
}

func buildForceSwitchChoices(pokemon []any, force []bool) [][]aiAtomicChoice {
	perSlot := make([][]aiAtomicChoice, 0, len(force))
	for slot, mustSwitch := range force {
		if !mustSwitch {
			perSlot = append(perSlot, []aiAtomicChoice{{Command: "pass", Kind: 2}})
			continue
		}
		switches := availableSwitches(pokemon, len(force), nil)
		choices := make([]aiAtomicChoice, 0, len(switches))
		for _, switchSlot := range switches {
			if slot >= 0 {
				choices = append(choices, aiAtomicChoice{
					Command:    fmt.Sprintf("switch %d", switchSlot),
					Kind:       1,
					SwitchSlot: switchSlot,
				})
			}
		}
		if len(choices) == 0 {
			choices = append(choices, aiAtomicChoice{Command: "pass", Kind: 2})
		}
		perSlot = append(perSlot, choices)
	}
	return perSlot
}

func buildTurnChoicesForSlot(active map[string]any, pokemon []any, index, activeSlots int) []aiAtomicChoice {
	if shouldPassTurnSlot(pokemon, index) {
		return []aiAtomicChoice{{Command: "pass", Kind: 2}}
	}
	choices := make([]aiAtomicChoice, 0, 12)
	for _, move := range buildMoveChoices(active, index) {
		choices = append(choices, move)
	}
	trapped, _ := active["trapped"].(bool)
	if !trapped {
		for _, switchSlot := range availableSwitches(pokemon, activeSlots, nil) {
			choices = append(choices, aiAtomicChoice{
				Command:    fmt.Sprintf("switch %d", switchSlot),
				Kind:       1,
				SwitchSlot: switchSlot,
			})
		}
	}
	if len(choices) == 0 {
		return []aiAtomicChoice{{Command: "move 1", Kind: 0, MoveIndex: 1}}
	}
	return choices
}

func buildMoveChoices(active map[string]any, activeIndex int) []aiAtomicChoice {
	rawMoves, _ := active["moves"].([]any)
	choices := make([]aiAtomicChoice, 0, len(rawMoves))
	for idx, rawMove := range rawMoves {
		move, _ := rawMove.(map[string]any)
		disabled, _ := move["disabled"].(bool)
		if disabled {
			continue
		}
		command := fmt.Sprintf("move %d", idx+1)
		target, _ := move["target"].(string)
		targetCode := 0
		if target != "" {
			suffix, code := chooseAITargetSuffix(target, activeIndex)
			command += suffix
			targetCode = code
		}
		choices = append(choices, aiAtomicChoice{
			Command:    command,
			Kind:       0,
			MoveIndex:  idx + 1,
			TargetCode: targetCode,
		})
	}
	return choices
}

func chooseAITargetSuffix(target string, index int) (string, int) {
	switch target {
	case "normal", "any", "adjacentFoe":
		return " 1", 1
	case "adjacentAlly":
		return fmt.Sprintf(" -%d", (index^1)+1), -((index ^ 1) + 1)
	case "adjacentAllyOrSelf":
		return fmt.Sprintf(" -%d", index+1), -(index + 1)
	default:
		return "", 0
	}
}

func encodeAIState(request map[string]any, pokemon []any, turn int) []float64 {
	features := make([]float64, 0, aiStateFeatureSize)
	activeList, _ := request["active"].([]any)
	forceList, _ := request["forceSwitch"].([]any)
	wait, _ := request["wait"].(bool)
	teamPreview := request["teamPreview"] != nil

	alive := 0
	bench := 0
	for _, raw := range pokemon {
		poke, _ := raw.(map[string]any)
		condition, _ := poke["condition"].(string)
		active, _ := poke["active"].(bool)
		_, fainted, _ := parseAICondition(condition)
		if !fainted {
			alive++
		}
		if !active && !fainted {
			bench++
		}
	}

	features = append(features,
		boolToFloat(wait),
		boolToFloat(teamPreview),
		normalizeCount(len(activeList), 2),
		normalizeCount(len(forceList), 2),
		normalizeCount(bench, 6),
		normalizeCount(alive, 6),
		normalizeCount(turn, 100),
		0,
	)

	for idx := 0; idx < 6; idx++ {
		if idx < len(pokemon) {
			poke, _ := pokemon[idx].(map[string]any)
			condition, _ := poke["condition"].(string)
			active, _ := poke["active"].(bool)
			hpRatio, fainted, status := parseAICondition(condition)
			features = append(features,
				hpRatio,
				boolToFloat(fainted),
				boolToFloat(active),
				boolToFloat(status == "brn"),
				boolToFloat(status == "par"),
				boolToFloat(status == "slp"),
				boolToFloat(status == "psn" || status == "tox"),
				boolToFloat(status == "frz"),
			)
		} else {
			features = append(features, 0, 0, 0, 0, 0, 0, 0, 0)
		}
	}

	for idx := 0; idx < 2; idx++ {
		if idx < len(activeList) {
			active, _ := activeList[idx].(map[string]any)
			trapped, _ := active["trapped"].(bool)
			rawMoves, _ := active["moves"].([]any)
			enabled := 0
			for _, rawMove := range rawMoves {
				move, _ := rawMove.(map[string]any)
				disabled, _ := move["disabled"].(bool)
				if !disabled {
					enabled++
				}
			}
			forceSwitch := false
			if idx < len(forceList) {
				forceSwitch, _ = forceList[idx].(bool)
			}
			canSwitch := boolToFloat(len(availableSwitches(pokemon, len(activeList), nil)) > 0 && !trapped)
			features = append(features,
				normalizeCount(enabled, 4),
				canSwitch,
				boolToFloat(trapped),
				boolToFloat(forceSwitch),
			)
		} else {
			features = append(features, 0, 0, 0, 0)
		}
	}

	return features
}

func encodeAIActionFeatures(choices []aiAtomicChoice) []float64 {
	features := make([]float64, aiActionFeatureSize)
	for idx := 0; idx < 2 && idx < len(choices); idx++ {
		base := idx * 8
		switch choices[idx].Kind {
		case 0:
			features[base+0] = 1
		case 1:
			features[base+1] = 1
		case 2:
			features[base+2] = 1
		case 3:
			features[base+3] = 1
		}
		features[base+4] = normalizeCount(choices[idx].MoveIndex, 4)
		features[base+5] = float64(choices[idx].TargetCode) / 4
		features[base+6] = normalizeCount(choices[idx].SwitchSlot, 6)
		features[base+7] = normalizeCount(choices[idx].TeamLead, 6)
	}
	return features
}

func parseAICondition(condition string) (float64, bool, string) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return 0, false, ""
	}
	if strings.Contains(condition, "fnt") {
		return 0, true, ""
	}
	parts := strings.Fields(condition)
	hpRatio := 0.0
	if len(parts) > 0 && strings.Contains(parts[0], "/") {
		split := strings.SplitN(parts[0], "/", 2)
		if len(split) == 2 {
			var current, max float64
			fmt.Sscanf(split[0], "%f", &current)
			fmt.Sscanf(split[1], "%f", &max)
			if max > 0 {
				hpRatio = current / max
			}
		}
	}
	status := ""
	if len(parts) > 1 {
		status = strings.ToLower(parts[1])
	}
	return clamp01(hpRatio), false, status
}

func aiRewards(result MockBattleResult, playerA, playerB string) (float64, float64) {
	if !result.Completed || strings.TrimSpace(result.Winner) == "" {
		return 0, 0
	}
	switch showdownID(result.Winner) {
	case showdownID(playerA):
		return 1, -1
	case showdownID(playerB):
		return -1, 1
	default:
		return 0, 0
	}
}

func safeAverageTurns(totalTurns, battles int) float64 {
	if battles <= 0 {
		return 0
	}
	return float64(totalTurns) / float64(battles)
}

func safeWinRate(wins, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(wins) / float64(total)
}

func softmax(scores []float64) []float64 {
	if len(scores) == 0 {
		return nil
	}
	maxScore := scores[0]
	for _, score := range scores[1:] {
		if score > maxScore {
			maxScore = score
		}
	}
	total := 0.0
	out := make([]float64, len(scores))
	for idx, score := range scores {
		out[idx] = math.Exp(score - maxScore)
		total += out[idx]
	}
	if total == 0 {
		uniform := 1 / float64(len(out))
		for idx := range out {
			out[idx] = uniform
		}
		return out
	}
	for idx := range out {
		out[idx] /= total
	}
	return out
}

func normalizeCount(value, max int) float64 {
	if max <= 0 {
		return 0
	}
	return clamp01(float64(value) / float64(max))
}

func boolToFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func loadPackedTeams(ctx context.Context, apiBaseURL, formatID, teamAPath, teamBPath string) (string, string, error) {
	if strings.TrimSpace(teamAPath) == "" && strings.TrimSpace(teamBPath) == "" {
		return "", "", nil
	}
	if strings.TrimSpace(teamAPath) == "" {
		teamAPath = teamBPath
	}
	if strings.TrimSpace(teamBPath) == "" {
		teamBPath = teamAPath
	}
	if strings.TrimSpace(apiBaseURL) == "" {
		return "", "", errors.New("team files require --api-base so the remote client can call /api/validate-team")
	}
	teamAText, err := os.ReadFile(teamAPath)
	if err != nil {
		return "", "", err
	}
	teamBText, err := os.ReadFile(teamBPath)
	if err != nil {
		return "", "", err
	}
	resultA, err := remoteValidateTeam(ctx, apiBaseURL, formatID, string(teamAText))
	if err != nil {
		return "", "", err
	}
	resultB, err := remoteValidateTeam(ctx, apiBaseURL, formatID, string(teamBText))
	if err != nil {
		return "", "", err
	}
	return resultA.PackedTeam, resultB.PackedTeam, nil
}

func remoteValidateTeam(ctx context.Context, apiBaseURL, formatID, team string) (TeamValidationResult, error) {
	base, err := url.Parse(apiBaseURL)
	if err != nil {
		return TeamValidationResult{}, err
	}
	base.Path = "/api/validate-team"
	payload, err := json.Marshal(map[string]any{
		"format_id": formatID,
		"team":      team,
	})
	if err != nil {
		return TeamValidationResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base.String(), bytes.NewReader(payload))
	if err != nil {
		return TeamValidationResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TeamValidationResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TeamValidationResult{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return TeamValidationResult{}, fmt.Errorf("validate-team returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var result TeamValidationResult
	if err := json.Unmarshal(body, &result); err != nil {
		return TeamValidationResult{}, err
	}
	if !result.Valid {
		return TeamValidationResult{}, fmt.Errorf("team is invalid: %s", strings.Join(result.Errors, "; "))
	}
	return result, nil
}

func openMetricsWriter(path string) (*os.File, func() error, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, file.Close, nil
}
