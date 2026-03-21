package showdown

import (
	"math/rand"
	"path/filepath"
	"testing"
)

func TestParseAICondition(t *testing.T) {
	hp, fainted, status := parseAICondition("73/100 brn")
	if fainted {
		t.Fatal("expected non-fainted pokemon")
	}
	if status != "brn" {
		t.Fatalf("status = %q, want brn", status)
	}
	if hp <= 0.72 || hp >= 0.74 {
		t.Fatalf("hp ratio = %v, want about 0.73", hp)
	}
}

func TestBuildAICandidatesIncludesSwitches(t *testing.T) {
	request := map[string]any{
		"active": []any{
			map[string]any{
				"moves": []any{
					map[string]any{"disabled": false, "target": "adjacentFoe"},
					map[string]any{"disabled": false, "target": "self"},
				},
			},
		},
		"side": map[string]any{
			"pokemon": []any{
				map[string]any{"condition": "100/100", "active": true},
				map[string]any{"condition": "100/100", "active": false},
				map[string]any{"condition": "100/100", "active": false},
			},
		},
	}

	candidates, err := buildAICandidates(request, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) < 4 {
		t.Fatalf("expected moves and switches, got %d candidates", len(candidates))
	}

	foundSwitch := false
	for _, candidate := range candidates {
		if candidate.Command == "switch 2" || candidate.Command == "switch 3" {
			foundSwitch = true
			break
		}
	}
	if !foundSwitch {
		t.Fatal("expected at least one switch candidate")
	}
}

func TestCheckpointRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	checkpoint := newCheckpoint(aiModelInputSize, 8, rng)
	path := filepath.Join(t.TempDir(), "model.json")
	if err := checkpoint.save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.HiddenSize != checkpoint.HiddenSize || loaded.InputSize != checkpoint.InputSize {
		t.Fatal("checkpoint shape mismatch after round trip")
	}
	if len(loaded.W1) != len(checkpoint.W1) || len(loaded.W2) != len(checkpoint.W2) {
		t.Fatal("checkpoint weights mismatch after round trip")
	}
}
