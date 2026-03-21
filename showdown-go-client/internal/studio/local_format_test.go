package studio

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeSpeciesListsCanonicalizesNames(t *testing.T) {
	root, err := serverRoot()
	if err != nil {
		t.Skipf("server root unavailable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "data", "pokedex.js")); err != nil {
		t.Skip("built server bundle unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := defaultConfig
	cfg.TargetPokemon = []string{"pikachu", "Charizard", "pikachu"}
	cfg.BannedPokemon = []string{"meowscarada"}

	got, err := normalizeSpeciesLists(ctx, root, cfg)
	if err != nil {
		t.Fatalf("normalizeSpeciesLists() error = %v", err)
	}

	if strings.Join(got.TargetPokemon, ",") != "Pikachu,Charizard" {
		t.Fatalf("TargetPokemon = %#v, want canonical species names", got.TargetPokemon)
	}
	if strings.Join(got.BannedPokemon, ",") != "Meowscarada" {
		t.Fatalf("BannedPokemon = %#v, want canonical species names", got.BannedPokemon)
	}
}

func TestNormalizeSpeciesListsRejectsUnknownSpecies(t *testing.T) {
	root, err := serverRoot()
	if err != nil {
		t.Skipf("server root unavailable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "data", "pokedex.js")); err != nil {
		t.Skip("built server bundle unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := defaultConfig
	cfg.TargetPokemon = []string{"Pikcahu"}

	_, err = normalizeSpeciesLists(ctx, root, cfg)
	if err == nil {
		t.Fatal("normalizeSpeciesLists() should reject unknown species")
	}
	if !strings.Contains(err.Error(), "unknown species") {
		t.Fatalf("normalizeSpeciesLists() error = %q, want unknown species message", err.Error())
	}
}
