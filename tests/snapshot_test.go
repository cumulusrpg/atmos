package atmos_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cumulusrpg/atmos"
	"github.com/cumulusrpg/atmos/repository"
)

// =============================================================================
// Test State and Events
// =============================================================================

type GameState struct {
	Score int `json:"score"`
	Level int `json:"level"`
}

func NewGameState() GameState {
	return GameState{
		Score: 0,
		Level: 1,
	}
}

type ScoreEvent struct {
	Points int
}

func (e ScoreEvent) Type() string { return "score" }

func ReduceScore(engine *atmos.Engine, state interface{}, event atmos.Event) interface{} {
	s := state.(GameState)
	e := event.(ScoreEvent)
	s.Score += e.Points
	return s
}

// =============================================================================
// Test Context
// =============================================================================

type snapshotTestContext struct {
	engine       *atmos.Engine
	snapshotRepo *repository.InMemorySnapshot
	lastError    error
}

// =============================================================================
// Step Definitions
// =============================================================================

func (ctx *snapshotTestContext) anEngineWithSnapshotcapableRepository() error {
	ctx.snapshotRepo = repository.NewInMemorySnapshot()
	ctx.engine = atmos.NewEngine(atmos.WithRepository(ctx.snapshotRepo))
	return nil
}

func (ctx *snapshotTestContext) anEngineWithStandardRepository() error {
	ctx.snapshotRepo = nil
	ctx.engine = atmos.NewEngine()
	return nil
}

func (ctx *snapshotTestContext) aStateWithDefaultValues(stateName string, table *godog.Table) error {
	ctx.engine.RegisterState(stateName, NewGameState())
	ctx.engine.When("score").Updates(stateName, ReduceScore)
	return nil
}

func (ctx *snapshotTestContext) iSetASnapshotForStateWith(stateName string, table *godog.Table) error {
	data := tableToMap(table)
	ctx.lastError = ctx.engine.SetSnapshot(stateName, data)
	return nil
}

func (ctx *snapshotTestContext) iClearTheSnapshotFor(stateName string) error {
	ctx.lastError = ctx.engine.ClearSnapshot(stateName)
	return nil
}

func (ctx *snapshotTestContext) iTryToSetASnapshotForState(stateName string) error {
	ctx.lastError = ctx.engine.SetSnapshot(stateName, map[string]interface{}{"score": 100})
	return nil
}

func (ctx *snapshotTestContext) iEmitAEventWithPoints(eventType string, points int) error {
	if eventType != "score" {
		return fmt.Errorf("unknown event type: %s", eventType)
	}
	ctx.engine.Emit(ScoreEvent{Points: points})
	return nil
}

func (ctx *snapshotTestContext) theSnapshotForShouldExist(stateName string) error {
	if !ctx.engine.HasSnapshot(stateName) {
		return fmt.Errorf("expected snapshot for %q to exist", stateName)
	}
	return nil
}

func (ctx *snapshotTestContext) theSnapshotForShouldNotExist(stateName string) error {
	if ctx.engine.HasSnapshot(stateName) {
		return fmt.Errorf("expected snapshot for %q to not exist", stateName)
	}
	return nil
}

func (ctx *snapshotTestContext) theSnapshotForShouldContain(stateName string, table *godog.Table) error {
	if ctx.snapshotRepo == nil {
		return fmt.Errorf("no snapshot repository available")
	}

	data, exists := ctx.snapshotRepo.GetSnapshot(stateName)
	if !exists {
		return fmt.Errorf("snapshot for %q does not exist", stateName)
	}

	var snapshot map[string]interface{}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	expected := tableToMap(table)
	for field, expectedValue := range expected {
		actualValue, exists := snapshot[field]
		if !exists {
			return fmt.Errorf("field %q not found in snapshot", field)
		}

		expectedJSON, _ := json.Marshal(expectedValue)
		actualJSON, _ := json.Marshal(actualValue)
		if string(expectedJSON) != string(actualJSON) {
			return fmt.Errorf("field %q: expected %s, got %s", field, expectedJSON, actualJSON)
		}
	}
	return nil
}

func (ctx *snapshotTestContext) theStateShouldHave(stateName string, table *godog.Table) error {
	state := ctx.engine.GetState(stateName)
	if state == nil {
		return fmt.Errorf("state %q not found", stateName)
	}

	gameState := state.(GameState)
	expected := tableToMap(table)

	for field, expectedValue := range expected {
		var actualValue interface{}
		switch field {
		case "score":
			actualValue = gameState.Score
		case "level":
			actualValue = gameState.Level
		default:
			return fmt.Errorf("unknown field: %s", field)
		}

		expectedJSON, _ := json.Marshal(expectedValue)
		actualJSON, _ := json.Marshal(actualValue)
		if string(expectedJSON) != string(actualJSON) {
			return fmt.Errorf("field %q: expected %s, got %s", field, expectedJSON, actualJSON)
		}
	}
	return nil
}

func (ctx *snapshotTestContext) iShouldReceiveASnapshotError() error {
	if ctx.lastError == nil {
		return fmt.Errorf("expected an error but got none")
	}
	if !strings.Contains(strings.ToLower(ctx.lastError.Error()), "snapshot") {
		return fmt.Errorf("expected error to mention 'snapshot', got: %v", ctx.lastError)
	}
	return nil
}

func (ctx *snapshotTestContext) theEngineShouldReportASnapshotExistsFor(stateName string) error {
	if !ctx.engine.HasSnapshot(stateName) {
		return fmt.Errorf("expected HasSnapshot(%q) to return true", stateName)
	}
	return nil
}

func (ctx *snapshotTestContext) theEngineShouldReportNoSnapshotFor(stateName string) error {
	if ctx.engine.HasSnapshot(stateName) {
		return fmt.Errorf("expected HasSnapshot(%q) to return false", stateName)
	}
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func tableToMap(table *godog.Table) map[string]interface{} {
	result := make(map[string]interface{})
	if len(table.Rows) < 2 {
		return result
	}

	headers := table.Rows[0].Cells
	fieldIdx := -1
	valueIdx := -1
	for i, cell := range headers {
		switch cell.Value {
		case "field":
			fieldIdx = i
		case "value":
			valueIdx = i
		}
	}

	if fieldIdx == -1 || valueIdx == -1 {
		return result
	}

	for _, row := range table.Rows[1:] {
		field := row.Cells[fieldIdx].Value
		value := row.Cells[valueIdx].Value
		result[field] = parseValue(value)
	}

	return result
}

func parseValue(value string) interface{} {
	if value == "" {
		return ""
	}
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	return value
}

// =============================================================================
// Test Runner
// =============================================================================

func InitializeScenario(sc *godog.ScenarioContext) {
	ctx := &snapshotTestContext{}

	sc.Step(`^an engine with snapshot-capable repository$`, ctx.anEngineWithSnapshotcapableRepository)
	sc.Step(`^an engine with standard repository$`, ctx.anEngineWithStandardRepository)
	sc.Step(`^a "([^"]*)" state with default values:$`, ctx.aStateWithDefaultValues)
	sc.Step(`^I set a snapshot for "([^"]*)" state with:$`, ctx.iSetASnapshotForStateWith)
	sc.Step(`^I clear the snapshot for "([^"]*)"$`, ctx.iClearTheSnapshotFor)
	sc.Step(`^I try to set a snapshot for "([^"]*)" state$`, ctx.iTryToSetASnapshotForState)
	sc.Step(`^I emit a "([^"]*)" event with points (\d+)$`, ctx.iEmitAEventWithPoints)
	sc.Step(`^the snapshot for "([^"]*)" should exist$`, ctx.theSnapshotForShouldExist)
	sc.Step(`^the snapshot for "([^"]*)" should not exist$`, ctx.theSnapshotForShouldNotExist)
	sc.Step(`^the snapshot for "([^"]*)" should contain:$`, ctx.theSnapshotForShouldContain)
	sc.Step(`^the "([^"]*)" state should have:$`, ctx.theStateShouldHave)
	sc.Step(`^I should receive a snapshot error$`, ctx.iShouldReceiveASnapshotError)
	sc.Step(`^the engine should report a snapshot exists for "([^"]*)"$`, ctx.theEngineShouldReportASnapshotExistsFor)
	sc.Step(`^the engine should report no snapshot for "([^"]*)"$`, ctx.theEngineShouldReportNoSnapshotFor)
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Strict:   true, // fail on undefined or pending steps
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
