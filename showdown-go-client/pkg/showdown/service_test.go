package showdown

import (
	"math/rand"
	"strings"
	"testing"
)

func TestChooseActionDoublesProducesTwoChoices(t *testing.T) {
	request := map[string]any{
		"active": []any{
			map[string]any{
				"moves": []any{
					map[string]any{"disabled": false, "target": "adjacentFoe"},
				},
			},
			map[string]any{
				"moves": []any{
					map[string]any{"disabled": false, "target": "adjacentAlly"},
				},
			},
		},
		"side": map[string]any{
			"pokemon": []any{
				map[string]any{"condition": "100/100", "active": true},
				map[string]any{"condition": "100/100", "active": true},
				map[string]any{"condition": "100/100", "active": false},
			},
		},
	}

	got := chooseAction(rand.New(rand.NewSource(1)), request)
	parts := strings.Split(got, ", ")
	if len(parts) != 2 {
		t.Fatalf("chooseAction() should emit two choices for doubles, got %q", got)
	}
}

func TestChooseActionForceSwitchUsesCommaSeparatedChoices(t *testing.T) {
	request := map[string]any{
		"forceSwitch": []any{true, false},
		"side": map[string]any{
			"pokemon": []any{
				map[string]any{"condition": "0 fnt", "active": true},
				map[string]any{"condition": "100/100", "active": true},
				map[string]any{"condition": "100/100", "active": false},
				map[string]any{"condition": "100/100", "active": false},
			},
		},
	}

	got := chooseAction(rand.New(rand.NewSource(1)), request)
	parts := strings.Split(got, ", ")
	if len(parts) != 2 {
		t.Fatalf("chooseAction() should emit two switch slots, got %q", got)
	}
	if !strings.HasPrefix(parts[0], "switch ") || parts[1] != "pass" {
		t.Fatalf("unexpected force switch choice %q", got)
	}
}
