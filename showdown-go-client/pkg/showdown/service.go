package showdown

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"slices"
	"strings"
	"time"
)

func Ping(ctx context.Context, serverURL, username string) (ConnectionInfo, error) {
	client, err := NewClient(serverURL, username)
	if err != nil {
		return ConnectionInfo{}, err
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		return ConnectionInfo{}, err
	}
	if err := client.Rename(ctx); err != nil {
		return ConnectionInfo{}, err
	}

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-deadline.C:
			return ConnectionInfo{}, errors.New("timed out waiting for updateuser")
		case err := <-client.Errors():
			return ConnectionInfo{}, err
		case <-client.Lines():
			if client.Named() {
				return ConnectionInfo{
					ServerURL: serverURL,
					Connected: true,
					Username:  client.Username(),
					Named:     client.Named(),
					CheckedAt: time.Now(),
				}, nil
			}
		}
	}
}

func FetchStatus(ctx context.Context, serverURL, username string) (ServerStatus, error) {
	info, err := Ping(ctx, serverURL, username)
	if err != nil {
		return ServerStatus{}, err
	}

	client, err := NewClient(serverURL, username+"-status")
	if err != nil {
		return ServerStatus{}, err
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		return ServerStatus{}, err
	}
	if err := client.Rename(ctx); err != nil {
		return ServerStatus{}, err
	}
	waitForFormats(ctx, client)

	raw, err := client.Query(ctx, "roomlist")
	if err != nil {
		return ServerStatus{}, err
	}

	var rooms RoomsResponse
	if err := json.Unmarshal(raw, &rooms); err != nil {
		return ServerStatus{}, err
	}

	var localFormat *LocalFormatDefinition
	if rawLocal, err := client.Query(ctx, "localformat"); err == nil && len(rawLocal) > 0 && string(rawLocal) != "null" {
		var parsed LocalFormatDefinition
		if err := json.Unmarshal(rawLocal, &parsed); err == nil {
			localFormat = &parsed
		}
	}

	return ServerStatus{
		Connection:  info,
		Rooms:       rooms,
		Formats:     client.Formats(),
		LocalFormat: localFormat,
	}, nil
}

func waitForFormats(ctx context.Context, client *Client) {
	deadline := time.NewTimer(1500 * time.Millisecond)
	defer deadline.Stop()
	for len(client.Formats()) == 0 {
		select {
		case <-ctx.Done():
			return
		case <-client.Done():
			return
		case <-deadline.C:
			return
		case <-client.Lines():
		}
	}
}

func waitForNamed(ctx context.Context, client *Client) error {
	if client.Named() {
		return nil
	}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-client.Done():
			if err := client.connectionErr(); err != nil {
				return err
			}
			return errors.New("connection closed while waiting for rename")
		case <-deadline.C:
			return errors.New("timed out waiting for rename")
		case <-client.Lines():
			if client.Named() {
				return nil
			}
		}
	}
}

func RunMockBattle(ctx context.Context, req MockBattleRequest) (MockBattleResult, error) {
	format := req.Format
	if format == "" {
		format = "gen9randombattle"
	}

	if req.Timeout <= 0 {
		req.Timeout = 90 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	playerAName := randomUsername("mocka")
	playerBName := randomUsername("mockb")

	playerA, err := NewClient(req.ServerURL, playerAName)
	if err != nil {
		return MockBattleResult{}, err
	}
	defer playerA.Close()

	playerB, err := NewClient(req.ServerURL, playerBName)
	if err != nil {
		return MockBattleResult{}, err
	}
	defer playerB.Close()

	for _, client := range []*Client{playerA, playerB} {
		if err := client.Connect(ctx); err != nil {
			return MockBattleResult{}, err
		}
		if err := client.Rename(ctx); err != nil {
			return MockBattleResult{}, err
		}
		if err := waitForNamed(ctx, client); err != nil {
			return MockBattleResult{}, err
		}
	}

	if err := applyBattleTeam(ctx, playerA, req.PackedTeamA); err != nil {
		return MockBattleResult{}, err
	}
	if err := applyBattleTeam(ctx, playerB, req.PackedTeamB); err != nil {
		return MockBattleResult{}, err
	}
	if err := playerA.Send(ctx, "", fmt.Sprintf("/challenge %s, %s", playerB.Username(), format)); err != nil {
		return MockBattleResult{}, err
	}
	if err := waitForIncomingChallenge(ctx, playerB, playerA.Username()); err != nil {
		return MockBattleResult{}, err
	}
	if err := playerB.Send(ctx, "", fmt.Sprintf("/accept %s", playerA.Username())); err != nil {
		return MockBattleResult{}, err
	}

	started := time.Now()
	result := MockBattleResult{
		ServerURL: req.ServerURL,
		Format:    format,
		PlayerA:   playerA.Username(),
		PlayerB:   playerB.Username(),
		LogLines:  make([]string, 0, 128),
	}

	requestRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case err := <-playerA.Errors():
			return result, err
		case err := <-playerB.Errors():
			return result, err
		case <-ctx.Done():
			result.Duration = time.Since(started)
			result.FinishedAt = time.Now()
			result.LogLines = append(result.LogLines, "mock battle timed out before completion")
			return result, nil
		case line := <-playerA.Lines():
			done, err := processBattleLine(ctx, requestRand, playerA, line, &result, started)
			if err != nil {
				return result, err
			}
			if done {
				return result, nil
			}
		case line := <-playerB.Lines():
			done, err := processBattleLine(ctx, requestRand, playerB, line, &result, started)
			if err != nil {
				return result, err
			}
			if done {
				return result, nil
			}
		}
	}
}

func processBattleLine(ctx context.Context, rng *rand.Rand, client *Client, line Line, result *MockBattleResult, started time.Time) (bool, error) {
	if line.Raw == "" {
		return false, nil
	}
	if strings.HasPrefix(line.RoomID, "battle-") && result.BattleID == "" {
		result.BattleID = line.RoomID
	}
	if strings.HasPrefix(line.RoomID, "battle-") && len(result.LogLines) < 120 {
		result.LogLines = append(result.LogLines, fmt.Sprintf("%s %s", line.RoomID, line.Raw))
	}
	if line.RoomID == result.BattleID && strings.HasPrefix(line.Raw, "|turn|") {
		fmt.Sscanf(strings.TrimPrefix(line.Raw, "|turn|"), "%d", &result.Turns)
	}
	if strings.HasPrefix(line.Raw, "|request|") && strings.HasPrefix(line.RoomID, "battle-") {
		if err := handleBattleRequest(ctx, rng, client, line); err != nil {
			return false, err
		}
	}
	if strings.HasPrefix(line.Raw, "|win|") && line.RoomID == result.BattleID {
		result.Winner = strings.TrimPrefix(line.Raw, "|win|")
		result.Completed = true
		result.Duration = time.Since(started)
		result.FinishedAt = time.Now()
		return true, nil
	}
	return false, nil
}

func waitForIncomingChallenge(ctx context.Context, client *Client, challenger string) error {
	challengerID := showdownID(challenger)
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-client.Done():
			if err := client.connectionErr(); err != nil {
				return err
			}
			return errors.New("connection closed while waiting for incoming challenge")
		case err := <-client.Errors():
			return err
		case <-deadline.C:
			return fmt.Errorf("timed out waiting for challenge from %s", challenger)
		case line := <-client.Lines():
			if line.Raw == "" {
				continue
			}
			if strings.HasPrefix(line.Raw, "|popup|") {
				return errors.New(strings.TrimPrefix(line.Raw, "|popup|"))
			}
			if !strings.HasPrefix(line.Raw, "|updatechallenges|") {
				continue
			}

			raw := strings.TrimPrefix(line.Raw, "|updatechallenges|")
			var state struct {
				ChallengesFrom map[string]string `json:"challengesFrom"`
			}
			if err := json.Unmarshal([]byte(raw), &state); err != nil {
				continue
			}
			if _, ok := state.ChallengesFrom[challengerID]; ok {
				return nil
			}
		}
	}
}

func handleBattleRequest(ctx context.Context, rng *rand.Rand, client *Client, line Line) error {
	raw := strings.TrimPrefix(line.Raw, "|request|")
	if raw == "" {
		return nil
	}

	var request map[string]any
	if err := json.Unmarshal([]byte(raw), &request); err != nil {
		log.Printf("showdown: failed to parse battle request JSON: %v", err)
		return nil
	}

	action := chooseAction(rng, request)
	if action == "" {
		return nil
	}
	return client.Send(ctx, line.RoomID, "/choose "+action)
}

func showdownID(value string) string {
	value = strings.ToLower(value)
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func chooseAction(rng *rand.Rand, request map[string]any) string {
	if wait, _ := request["wait"].(bool); wait {
		return ""
	}

	side, _ := request["side"].(map[string]any)
	pokemon, _ := side["pokemon"].([]any)

	if request["teamPreview"] != nil {
		order := make([]int, 0, len(pokemon))
		for idx := range pokemon {
			order = append(order, idx+1)
		}
		rng.Shuffle(len(order), func(i, j int) {
			order[i], order[j] = order[j], order[i]
		})
		parts := make([]string, 0, len(order))
		for _, idx := range order {
			parts = append(parts, fmt.Sprintf("%d", idx))
		}
		return "team " + strings.Join(parts, "")
	}

	if rawForce, ok := request["forceSwitch"].([]any); ok && len(rawForce) > 0 {
		chosen := map[int]bool{}
		choices := make([]string, 0, len(rawForce))
		for slot, rawMustSwitch := range rawForce {
			mustSwitch, _ := rawMustSwitch.(bool)
			if !mustSwitch {
				choices = append(choices, "pass")
				continue
			}

			switchChoices := availableSwitches(pokemon, len(rawForce), chosen)
			if len(switchChoices) == 0 {
				choices = append(choices, "pass")
				continue
			}

			chosenSlot := switchChoices[rng.Intn(len(switchChoices))]
			chosen[chosenSlot] = true
			if slot >= 0 {
				choices = append(choices, fmt.Sprintf("switch %d", chosenSlot))
			}
		}
		return strings.Join(choices, ", ")
	}

	activeList, ok := request["active"].([]any)
	if !ok || len(activeList) == 0 {
		return ""
	}

	switchChoices := map[int]bool{}
	choices := make([]string, 0, len(activeList))
	for index, rawActive := range activeList {
		active, _ := rawActive.(map[string]any)
		if shouldPassTurnSlot(pokemon, index) {
			choices = append(choices, "pass")
			continue
		}

		moveChoices := availableMoves(active, index)
		if len(moveChoices) == 0 {
			if trapped, _ := active["trapped"].(bool); !trapped {
				canSwitch := availableSwitches(pokemon, len(activeList), switchChoices)
				if len(canSwitch) > 0 {
					chosenSlot := canSwitch[rng.Intn(len(canSwitch))]
					switchChoices[chosenSlot] = true
					choices = append(choices, fmt.Sprintf("switch %d", chosenSlot))
					continue
				}
			}
			choices = append(choices, "move 1")
			continue
		}

		chosenMove := moveChoices[rng.Intn(len(moveChoices))]
		choices = append(choices, chosenMove)
	}
	return strings.Join(choices, ", ")
}

func SortRooms(rooms map[string]RoomSummary) []string {
	keys := make([]string, 0, len(rooms))
	for key := range rooms {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func applyBattleTeam(ctx context.Context, client *Client, packedTeam string) error {
	if strings.TrimSpace(packedTeam) == "" {
		return client.Send(ctx, "", "/utm null")
	}
	return client.Send(ctx, "", "/utm "+packedTeam)
}

func availableSwitches(pokemon []any, activeSlots int, chosen map[int]bool) []int {
	choices := make([]int, 0, len(pokemon))
	for idx, raw := range pokemon {
		poke, _ := raw.(map[string]any)
		condition, _ := poke["condition"].(string)
		active, _ := poke["active"].(bool)
		if active || strings.Contains(condition, "fnt") {
			continue
		}
		slot := idx + 1
		if slot <= activeSlots || chosen[slot] {
			continue
		}
		choices = append(choices, slot)
	}
	return choices
}

func shouldPassTurnSlot(pokemon []any, index int) bool {
	if index >= len(pokemon) {
		return false
	}
	poke, _ := pokemon[index].(map[string]any)
	condition, _ := poke["condition"].(string)
	commanding, _ := poke["commanding"].(bool)
	return strings.Contains(condition, "fnt") || commanding
}

func availableMoves(active map[string]any, activeIndex int) []string {
	rawMoves, _ := active["moves"].([]any)
	moveChoices := make([]string, 0, len(rawMoves))
	for idx, rawMove := range rawMoves {
		move, _ := rawMove.(map[string]any)
		disabled, _ := move["disabled"].(bool)
		if disabled {
			continue
		}

		choice := fmt.Sprintf("move %d", idx+1)
		target, _ := move["target"].(string)
		if target != "" {
			choice += chooseTargetSuffix(target, activeIndex)
		}
		moveChoices = append(moveChoices, choice)
	}
	return moveChoices
}

func chooseTargetSuffix(target string, index int) string {
	switch target {
	case "normal", "any", "adjacentFoe":
		return " 1"
	case "adjacentAlly":
		return fmt.Sprintf(" -%d", (index^1)+1)
	case "adjacentAllyOrSelf":
		return fmt.Sprintf(" -%d", index+1)
	default:
		return ""
	}
}
