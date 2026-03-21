package showdown

import "time"

const (
	LocalStudioFormatName = "[Gen 9] Showdown Suite Studio"
	LocalStudioFormatID   = "gen9showdownsuitestudio"
)

type ServerConfig struct {
	BaseURL  string
	Username string
}

type ConnectionInfo struct {
	ServerURL string    `json:"server_url"`
	Connected bool      `json:"connected"`
	Username  string    `json:"username"`
	Named     bool      `json:"named"`
	CheckedAt time.Time `json:"checked_at"`
}

type RoomSummary struct {
	Title     string `json:"title"`
	UserCount int    `json:"user_count,omitempty"`
	Desc      string `json:"desc,omitempty"`
}

type RoomsResponse struct {
	BattleCount int                    `json:"battleCount"`
	UserCount   int                    `json:"userCount"`
	Official    map[string]RoomSummary `json:"official"`
	Chat        map[string]RoomSummary `json:"chat"`
}

type LocalFormatPreset struct {
	ID                    string `json:"id"`
	Label                 string `json:"label"`
	Mode                  string `json:"mode"`
	Summary               string `json:"summary"`
	DefaultLevel          int    `json:"defaultLevel"`
	DefaultMaxTeamSize    int    `json:"defaultMaxTeamSize"`
	DefaultPickedTeamSize int    `json:"defaultPickedTeamSize"`
}

type LocalFormatConfig struct {
	Preset         string   `json:"preset"`
	Level          int      `json:"level"`
	MaxTeamSize    int      `json:"maxTeamSize"`
	PickedTeamSize int      `json:"pickedTeamSize"`
	OpenTeamSheets bool     `json:"openTeamSheets"`
	AllowTerastal  bool     `json:"allowTerastal"`
	CustomRules    []string `json:"customRules"`
	TargetPokemon  []string `json:"targetPokemon"`
	BannedPokemon  []string `json:"bannedPokemon"`
}

type LocalFormatDefinition struct {
	FormatID      string              `json:"formatId"`
	Name          string              `json:"name"`
	Config        LocalFormatConfig   `json:"config"`
	Preset        LocalFormatPreset   `json:"preset"`
	Ruleset       []string            `json:"ruleset"`
	Summary       []string            `json:"summary"`
	TargetPokemon []string            `json:"targetPokemon"`
	BannedPokemon []string            `json:"bannedPokemon"`
	Presets       []LocalFormatPreset `json:"presets,omitempty"`
}

type LocalFormatState struct {
	Config     LocalFormatConfig     `json:"config"`
	Presets    []LocalFormatPreset   `json:"presets"`
	Definition LocalFormatDefinition `json:"definition"`
}

type ServerStatus struct {
	Connection  ConnectionInfo         `json:"connection"`
	Rooms       RoomsResponse          `json:"rooms"`
	Formats     []string               `json:"formats"`
	LocalFormat *LocalFormatDefinition `json:"local_format,omitempty"`
}

type MockBattleRequest struct {
	ServerURL   string        `json:"server_url"`
	Format      string        `json:"format"`
	Timeout     time.Duration `json:"timeout"`
	TeamA       string        `json:"team_a,omitempty"`
	TeamB       string        `json:"team_b,omitempty"`
	PackedTeamA string        `json:"-"`
	PackedTeamB string        `json:"-"`
}

type MockBattleResult struct {
	ServerURL  string        `json:"server_url"`
	Format     string        `json:"format"`
	PlayerA    string        `json:"player_a"`
	PlayerB    string        `json:"player_b"`
	BattleID   string        `json:"battle_id"`
	Winner     string        `json:"winner"`
	Completed  bool          `json:"completed"`
	Turns      int           `json:"turns"`
	Duration   time.Duration `json:"duration"`
	LogLines   []string      `json:"log_lines"`
	FinishedAt time.Time     `json:"finished_at"`
}

type TeamValidationResult struct {
	Valid      bool      `json:"valid"`
	FormatID   string    `json:"format_id"`
	PackedTeam string    `json:"packed_team,omitempty"`
	Errors     []string  `json:"errors,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}
