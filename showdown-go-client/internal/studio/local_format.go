package studio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ysu/showdown-go-client/pkg/showdown"
)

type presetMeta struct {
	showdown.LocalFormatPreset
	ruleset []string
}

var presetTable = map[string]presetMeta{
	"standard-singles": {
		LocalFormatPreset: showdown.LocalFormatPreset{
			ID:                    "standard-singles",
			Label:                 "Standard Singles",
			Mode:                  "singles",
			Summary:               "Smogon-style singles baseline with local target-pool enforcement.",
			DefaultLevel:          50,
			DefaultMaxTeamSize:    3,
			DefaultPickedTeamSize: 3,
		},
		ruleset: []string{"Standard"},
	},
	"standard-doubles": {
		LocalFormatPreset: showdown.LocalFormatPreset{
			ID:                    "standard-doubles",
			Label:                 "Standard Doubles",
			Mode:                  "doubles",
			Summary:               "Smogon-style doubles baseline with local target-pool enforcement.",
			DefaultLevel:          50,
			DefaultMaxTeamSize:    4,
			DefaultPickedTeamSize: 4,
		},
		ruleset: []string{"Standard Doubles"},
	},
	"custom-singles": {
		LocalFormatPreset: showdown.LocalFormatPreset{
			ID:                    "custom-singles",
			Label:                 "Custom Singles",
			Mode:                  "singles",
			Summary:               "Lightweight custom singles baseline with team preview and local validation.",
			DefaultLevel:          50,
			DefaultMaxTeamSize:    3,
			DefaultPickedTeamSize: 3,
		},
		ruleset: []string{"Standard AG"},
	},
	"custom-doubles": {
		LocalFormatPreset: showdown.LocalFormatPreset{
			ID:                    "custom-doubles",
			Label:                 "Custom Doubles",
			Mode:                  "doubles",
			Summary:               "Lightweight custom doubles baseline with team preview and local validation.",
			DefaultLevel:          50,
			DefaultMaxTeamSize:    4,
			DefaultPickedTeamSize: 4,
		},
		ruleset: []string{"Standard AG"},
	},
}

var presetOrder = []string{
	"standard-singles",
	"standard-doubles",
	"custom-singles",
	"custom-doubles",
}

var defaultConfig = showdown.LocalFormatConfig{
	Preset:         "standard-singles",
	Level:          50,
	MaxTeamSize:    3,
	PickedTeamSize: 3,
	OpenTeamSheets: false,
	AllowTerastal:  true,
	CustomRules:    []string{},
	TargetPokemon:  []string{"Pikachu", "Charizard", "Meowscarada"},
	BannedPokemon:  []string{},
}

var (
	serverRootOnce sync.Once
	cachedRoot     string
	cachedRootErr  error

	configMu sync.Mutex // protects concurrent config file read/write operations
)

func LoadLocalFormatState() (showdown.LocalFormatState, error) {
	cfg, err := ReadLocalFormatConfig()
	if err != nil {
		return showdown.LocalFormatState{}, err
	}
	return showdown.LocalFormatState{
		Config:     cfg,
		Presets:    AvailablePresets(),
		Definition: BuildLocalFormatDefinition(cfg),
	}, nil
}

func AvailablePresets() []showdown.LocalFormatPreset {
	out := make([]showdown.LocalFormatPreset, 0, len(presetOrder))
	for _, id := range presetOrder {
		preset := presetTable[id].LocalFormatPreset
		out = append(out, preset)
	}
	return out
}

func ReadLocalFormatConfig() (showdown.LocalFormatConfig, error) {
	root, err := serverRoot()
	if err != nil {
		return showdown.LocalFormatConfig{}, err
	}

	path := filepath.Join(root, "config", "showdown-suite-local-format.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return normalizeConfig(defaultConfig), nil
		}
		return showdown.LocalFormatConfig{}, err
	}

	cfg := defaultConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return showdown.LocalFormatConfig{}, err
	}
	cfg = normalizeConfig(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return normalizeSpeciesLists(ctx, root, cfg)
}

func SaveLocalFormatConfig(ctx context.Context, cfg showdown.LocalFormatConfig, restartServer bool) (showdown.LocalFormatState, error) {
	configMu.Lock()
	defer configMu.Unlock()

	root, err := serverRoot()
	if err != nil {
		return showdown.LocalFormatState{}, err
	}

	cfg = normalizeConfig(cfg)
	cfg, err = normalizeSpeciesLists(ctx, root, cfg)
	if err != nil {
		return showdown.LocalFormatState{}, err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return showdown.LocalFormatState{}, err
	}
	data = append(data, '\n')

	path := filepath.Join(root, "config", "showdown-suite-local-format.json")
	previousData, readErr := os.ReadFile(path)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return showdown.LocalFormatState{}, readErr
	}
	if err := writeConfigAtomically(path, data); err != nil {
		return showdown.LocalFormatState{}, err
	}

	if err := validateLocalFormatConfig(ctx, root); err != nil {
		_ = restoreLocalFormatConfig(path, previousData, readErr)
		return showdown.LocalFormatState{}, err
	}

	if restartServer {
		if err := RestartServer(ctx); err != nil {
			recoverErr := restartPreviousServer(root, path, previousData, readErr)
			if recoverErr != nil {
				return showdown.LocalFormatState{}, fmt.Errorf("restart LAN server: %w (previous config restored but server recovery failed: %v)", err, recoverErr)
			}
			return showdown.LocalFormatState{}, err
		}
	}

	return showdown.LocalFormatState{
		Config:     cfg,
		Presets:    AvailablePresets(),
		Definition: BuildLocalFormatDefinition(cfg),
	}, nil
}

func BuildLocalFormatDefinition(cfg showdown.LocalFormatConfig) showdown.LocalFormatDefinition {
	cfg = normalizeConfig(cfg)
	preset := presetTable[cfg.Preset]
	presets := AvailablePresets()
	return showdown.LocalFormatDefinition{
		FormatID:      showdown.LocalStudioFormatID,
		Name:          showdown.LocalStudioFormatName,
		Config:        cfg,
		Preset:        preset.LocalFormatPreset,
		Ruleset:       buildRuleset(cfg),
		Summary:       buildSummary(cfg),
		TargetPokemon: append([]string{}, cfg.TargetPokemon...),
		BannedPokemon: append([]string{}, cfg.BannedPokemon...),
		Presets:       presets,
	}
}

func ValidateTeam(ctx context.Context, formatID, teamText string) (showdown.TeamValidationResult, error) {
	if strings.TrimSpace(teamText) == "" {
		return showdown.TeamValidationResult{
			Valid:     false,
			FormatID:  defaultFormatID(formatID),
			Errors:    []string{"team is empty"},
			CheckedAt: time.Now(),
		}, nil
	}

	root, err := serverRoot()
	if err != nil {
		return showdown.TeamValidationResult{}, err
	}

	formatID = defaultFormatID(formatID)
	_, stderr, err := runCommand(ctx, root, teamText, "./pokemon-showdown", "validate-team", formatID)
	if err != nil {
		validationErrors := splitLines(stderr)
		if len(validationErrors) == 0 {
			validationErrors = []string{err.Error()}
		}
		return showdown.TeamValidationResult{
			Valid:     false,
			FormatID:  formatID,
			Errors:    validationErrors,
			CheckedAt: time.Now(),
		}, nil
	}

	stdout, _, err := runCommand(ctx, root, teamText, "./pokemon-showdown", "pack-team")
	if err != nil {
		return showdown.TeamValidationResult{}, err
	}

	return showdown.TeamValidationResult{
		Valid:      true,
		FormatID:   formatID,
		PackedTeam: strings.TrimSpace(stdout),
		CheckedAt:  time.Now(),
	}, nil
}

func RestartServer(ctx context.Context) error {
	root, err := serverRoot()
	if err != nil {
		return err
	}
	_, stderr, err := runCommand(ctx, root, "", "./scripts/restart-lan-server.sh")
	if err != nil {
		return fmt.Errorf("restart LAN server: %w: %s", err, strings.TrimSpace(stderr))
	}
	return nil
}

func validateLocalFormatConfig(ctx context.Context, root string) error {
	_, stderr, err := runCommand(
		ctx,
		root,
		"",
		"node",
		"-e",
		"const {Dex}=require('./dist/sim/dex'); const format=Dex.formats.get('gen9showdownsuitestudio'); Dex.formats.getRuleTable(format);",
	)
	if err == nil {
		return nil
	}

	for _, line := range splitLines(stderr) {
		if strings.HasPrefix(line, "Error: ") {
			return fmt.Errorf("invalid local format config: %s", strings.TrimPrefix(line, "Error: "))
		}
	}
	if message := strings.TrimSpace(stderr); message != "" {
		return fmt.Errorf("invalid local format config: %s", message)
	}
	return fmt.Errorf("invalid local format config: %w", err)
}

func normalizeSpeciesLists(ctx context.Context, root string, cfg showdown.LocalFormatConfig) (showdown.LocalFormatConfig, error) {
	input := struct {
		TargetPokemon []string `json:"targetPokemon"`
		BannedPokemon []string `json:"bannedPokemon"`
	}{
		TargetPokemon: cfg.TargetPokemon,
		BannedPokemon: cfg.BannedPokemon,
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return showdown.LocalFormatConfig{}, err
	}

	script := `
const fs = require('fs');
const {Pokedex} = require('./dist/data/pokedex');

function toID(value) {
	return String(value ?? '').toLowerCase().replace(/[^a-z0-9]+/g, '');
}

try {
	const input = JSON.parse(fs.readFileSync(0, 'utf8'));
	function normalize(field) {
		const seen = new Set();
		const out = [];
		for (const raw of input[field] || []) {
			const text = String(raw ?? '').trim();
			if (!text) continue;
			const id = toID(text);
			const species = Pokedex[id];
			if (!species || !species.name) throw new Error('unknown species in ' + field + ': ' + text);
			const canonicalID = toID(species.name);
			if (!canonicalID || seen.has(canonicalID)) continue;
			seen.add(canonicalID);
			out.push(species.name);
		}
		return out;
	}
	process.stdout.write(JSON.stringify({
		targetPokemon: normalize('targetPokemon'),
		bannedPokemon: normalize('bannedPokemon'),
	}));
} catch (error) {
	console.error(error && error.message ? error.message : String(error));
	process.exit(1);
}
`

	stdout, stderr, err := runCommand(ctx, root, string(payload), "node", "-e", script)
	if err != nil {
		message := strings.TrimSpace(stderr)
		if message == "" {
			message = err.Error()
		}
		return showdown.LocalFormatConfig{}, fmt.Errorf("invalid species list: %s", message)
	}

	var normalized struct {
		TargetPokemon []string `json:"targetPokemon"`
		BannedPokemon []string `json:"bannedPokemon"`
	}
	if err := json.Unmarshal([]byte(stdout), &normalized); err != nil {
		return showdown.LocalFormatConfig{}, err
	}

	cfg.TargetPokemon = normalized.TargetPokemon
	cfg.BannedPokemon = normalized.BannedPokemon
	return cfg, nil
}

func restoreLocalFormatConfig(path string, previousData []byte, readErr error) error {
	if errors.Is(readErr, os.ErrNotExist) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeConfigAtomically(path, previousData)
}

func restartPreviousServer(root, path string, previousData []byte, readErr error) error {
	if err := restoreLocalFormatConfig(path, previousData, readErr); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return RestartServer(ctx)
}

func normalizeConfig(cfg showdown.LocalFormatConfig) showdown.LocalFormatConfig {
	preset, ok := presetTable[cfg.Preset]
	if !ok {
		preset = presetTable[defaultConfig.Preset]
		cfg.Preset = defaultConfig.Preset
	}

	minTeamSize := 1
	if preset.Mode == "doubles" {
		minTeamSize = 2
	}

	cfg.Level = clampInt(cfg.Level, preset.DefaultLevel, 1, 100)
	cfg.MaxTeamSize = clampInt(cfg.MaxTeamSize, preset.DefaultMaxTeamSize, minTeamSize, 6)
	cfg.PickedTeamSize = clampInt(cfg.PickedTeamSize, preset.DefaultPickedTeamSize, minTeamSize, cfg.MaxTeamSize)
	cfg.CustomRules = normalizeRules(cfg.CustomRules)
	cfg.CustomRules = filterCustomRules(cfg, preset.ruleset)
	cfg.TargetPokemon = normalizeList(cfg.TargetPokemon)
	cfg.BannedPokemon = normalizeList(cfg.BannedPokemon)

	return cfg
}

func buildRuleset(cfg showdown.LocalFormatConfig) []string {
	preset := presetTable[cfg.Preset]
	ruleset := append([]string(nil), preset.ruleset...)
	ruleset = append(ruleset, fmt.Sprintf("Adjust Level = %d", cfg.Level))
	ruleset = append(ruleset, fmt.Sprintf("Max Team Size = %d", cfg.MaxTeamSize))
	ruleset = append(ruleset, fmt.Sprintf("Picked Team Size = %d", cfg.PickedTeamSize))
	if cfg.OpenTeamSheets {
		ruleset = append(ruleset, "Open Team Sheets")
	}
	if !cfg.AllowTerastal {
		ruleset = append(ruleset, "Terastal Clause")
	}
	ruleset = append(ruleset, cfg.CustomRules...)
	return ruleset
}

func buildSummary(cfg showdown.LocalFormatConfig) []string {
	preset := presetTable[cfg.Preset]
	summary := []string{
		fmt.Sprintf("%s preset", preset.Label),
		fmt.Sprintf("Level: %d", cfg.Level),
		fmt.Sprintf("Bring %d, choose %d", cfg.MaxTeamSize, cfg.PickedTeamSize),
	}
	if cfg.OpenTeamSheets {
		summary = append(summary, "Open Team Sheets enabled")
	} else {
		summary = append(summary, "Open Team Sheets disabled")
	}
	if cfg.AllowTerastal {
		summary = append(summary, "Terastallization allowed")
	} else {
		summary = append(summary, "Terastallization disabled")
	}
	if len(cfg.CustomRules) > 0 {
		summary = append(summary, "Custom rules: "+strings.Join(cfg.CustomRules, ", "))
	} else {
		summary = append(summary, "Custom rules: none")
	}
	if len(cfg.TargetPokemon) > 0 {
		summary = append(summary, "Target Pokemon: "+strings.Join(cfg.TargetPokemon, ", "))
	} else {
		summary = append(summary, "Target Pokemon: unrestricted")
	}
	if len(cfg.BannedPokemon) > 0 {
		summary = append(summary, "Additional banned Pokemon: "+strings.Join(cfg.BannedPokemon, ", "))
	} else {
		summary = append(summary, "Additional banned Pokemon: none")
	}
	return summary
}

func normalizeList(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		id := listID(item)
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, item)
	}
	return out
}

func normalizeRules(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := normalizeRuleKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func filterCustomRules(cfg showdown.LocalFormatConfig, presetRules []string) []string {
	seen := map[string]bool{}
	for _, rule := range presetRules {
		seen[normalizeRuleKey(rule)] = true
	}
	seen[normalizeRuleKey(fmt.Sprintf("Adjust Level = %d", cfg.Level))] = true
	seen[normalizeRuleKey(fmt.Sprintf("Max Team Size = %d", cfg.MaxTeamSize))] = true
	seen[normalizeRuleKey(fmt.Sprintf("Picked Team Size = %d", cfg.PickedTeamSize))] = true
	if cfg.OpenTeamSheets {
		seen[normalizeRuleKey("Open Team Sheets")] = true
	}
	if !cfg.AllowTerastal {
		seen[normalizeRuleKey("Terastal Clause")] = true
	}

	out := make([]string, 0, len(cfg.CustomRules))
	for _, rule := range cfg.CustomRules {
		key := normalizeRuleKey(rule)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, rule)
	}
	return out
}

func normalizeRuleKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), ""))
}

func splitLines(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func writeConfigAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0o644); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func defaultFormatID(value string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return showdown.LocalStudioFormatID
}

func clampInt(value, fallback, min, max int) int {
	if value < min || value > max {
		value = fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func listID(value string) string {
	value = strings.ToLower(value)
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func serverRoot() (string, error) {
	serverRootOnce.Do(func() {
		cachedRoot, cachedRootErr = findServerRoot()
	})
	return cachedRoot, cachedRootErr
}

func findServerRoot() (string, error) {
	candidates := make([]string, 0, 6)
	if env := os.Getenv("SHOWDOWN_SERVER_ROOT"); env != "" {
		candidates = append(candidates, env)
	}
	if env := os.Getenv("SHOWDOWN_SUITE_ROOT"); env != "" {
		candidates = append(candidates, filepath.Join(env, "pokemon-showdown-local"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "..", "pokemon-showdown-local"),
			filepath.Join(cwd, "pokemon-showdown-local"),
			cwd,
		)
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "pokemon-showdown-local"),
			filepath.Join(exeDir, "..", "..", "pokemon-showdown-local"),
		)
	}

	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, err := os.Stat(filepath.Join(candidate, "pokemon-showdown")); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("could not locate pokemon-showdown-local; set SHOWDOWN_SERVER_ROOT to override")
}

func runCommand(ctx context.Context, workdir, stdin string, args ...string) (string, string, error) {
	if len(args) == 0 {
		return "", "", errors.New("no command specified")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = workdir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
