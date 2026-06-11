package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var towerctlBin string

func TestMain(m *testing.M) {
	bin := filepath.Join(os.TempDir(), "towerctl-test-bin")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(".")
	if out, err := cmd.CombinedOutput(); err != nil {
		_, _ = os.Stderr.WriteString("build failed: " + err.Error() + "\n" + string(out))
		os.Exit(1)
	}
	towerctlBin = bin
	code := m.Run()
	_ = os.Remove(bin)
	os.Exit(code)
}

func TestUserCanAddAndListProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	bin := buildTowerctl(t)

	add := exec.Command(bin, "project", "add", "Tower Control", "--type", "side_hustle", "--priority", "4", "--priority-reason", "Need Hermes-agent context layer", "--deadline", "2026-06-30")
	add.Env = os.Environ()
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}

	list := exec.Command(bin, "project", "list", "--format", "json")
	list.Env = os.Environ()
	out, err := list.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"id":"tower-control"`) {
		t.Fatalf("list output missing project id:\n%s", out)
	}
	if !strings.Contains(string(out), `"priority_reason":"Need Hermes-agent context layer"`) {
		t.Fatalf("list output missing priority reason:\n%s", out)
	}
}

func TestUserCanShowAndUpdateProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control", "--priority", "2")
	runCmd(t, bin, "project", "update", "tower-control", "--status", "blocked", "--priority", "5", "--priority-reason", "Blocked until MCP contract is done")

	out := runCmd(t, bin, "project", "show", "tower-control", "--format", "json")
	if !strings.Contains(out, `"status":"blocked"`) {
		t.Fatalf("show output missing updated status:\n%s", out)
	}
	if !strings.Contains(out, `"priority":5`) {
		t.Fatalf("show output missing updated priority:\n%s", out)
	}
	if !strings.Contains(out, `"priority_reason":"Blocked until MCP contract is done"`) {
		t.Fatalf("show output missing updated priority reason:\n%s", out)
	}
}

func runCmd(t *testing.T, bin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestUserCanParkContextAndGetMorningContext(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control", "--priority", "4", "--priority-reason", "Need context layer")
	runCmd(t, bin, "park", "tower-control", "--stopped", "Implemented project commands", "--changed", "CLI persists state", "--next", "Add MCP tools", "--blocker", "Need tool schema", "--files", "cmd/towerctl/main.go", "--help", "none", "--status", "active", "--priority", "5", "--priority-reason", "MCP adoption is next")

	out := runCmd(t, bin, "morning", "--format", "json")
	if !strings.Contains(out, `"next_action":"Add MCP tools"`) {
		t.Fatalf("morning output missing latest next action:\n%s", out)
	}
	if !strings.Contains(out, `"priority_reason":"MCP adoption is next"`) {
		t.Fatalf("morning output missing refreshed priority context:\n%s", out)
	}
}

func TestNextReturnsLatestNextActions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control", "--priority", "4")
	runCmd(t, bin, "park", "tower-control", "--next", "Wire MCP server")

	out := runCmd(t, bin, "next", "--format", "json")
	if !strings.Contains(out, `"project_id":"tower-control"`) || !strings.Contains(out, `"next_action":"Wire MCP server"`) {
		t.Fatalf("next output missing action:\n%s", out)
	}
}

func TestCheckReturnsFixedQuestionsForActiveProjects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	runCmd(t, bin, "park", "tower-control", "--next", "Wire MCP server")

	out := runCmd(t, bin, "check", "--format", "json")
	if !strings.Contains(out, `"project_id":"tower-control"`) || !strings.Contains(out, "Did you make progress on Wire MCP server?") {
		t.Fatalf("check output missing project question:\n%s", out)
	}
}

func TestSummaryAndExportReturnStoredContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	runCmd(t, bin, "park", "tower-control", "--next", "Use it with Hermes-agent")

	summary := runCmd(t, bin, "summary", "--format", "json")
	if !strings.Contains(summary, `"id":"tower-control"`) {
		t.Fatalf("summary missing project:\n%s", summary)
	}
	runCmd(t, bin, "export", "--format", "markdown")
	exported, err := os.ReadFile(filepath.Join(home, ".towerctl", "exports", "towerctl.md"))
	if err != nil {
		t.Fatalf("export file missing: %v", err)
	}
	if !strings.Contains(string(exported), "Tower Control") || !strings.Contains(string(exported), "Use it with Hermes-agent") {
		t.Fatalf("export missing context:\n%s", exported)
	}
}

func TestStateIsStoredInSQLiteDatabasePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	path := filepath.Join(home, ".towerctl", "tower.db")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("tower database missing: %v", err)
	}
	if !strings.HasPrefix(string(b), "SQLite format 3") {
		t.Fatalf("tower database is not sqlite, header: %q", string(b[:min(len(b), 20)]))
	}
}

func TestParkDayStoresSnapshotFromStdin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	cmd := exec.Command(bin, "park-day")
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader("tower-control\nStopped after CLI\nAdded park-day\nImplement MCP\nNo blockers\ncmd/towerctl/main.go\nnone\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("park-day failed: %v\n%s", err, out)
	}

	out := runCmd(t, bin, "morning", "--format", "json")
	if !strings.Contains(out, `"next_action":"Implement MCP"`) {
		t.Fatalf("morning missing park-day snapshot:\n%s", out)
	}
}

func TestMCPGetActiveProjectContext(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	runCmd(t, bin, "park", "tower-control", "--next", "Use MCP")

	cmd := exec.Command(bin, "serve-mcp")
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_active_project_context","arguments":{}}}` + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("serve-mcp failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"id":1`) || !strings.Contains(string(out), `"next_action":"Use MCP"`) {
		t.Fatalf("mcp response missing active context:\n%s", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestMCPHandlesMultipleRequestsInOneSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	cmd := exec.Command(bin, "serve-mcp")
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_project_list","arguments":{}}}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_active_project_context","arguments":{}}}` + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("serve-mcp failed: %v\n%s", err, out)
	}
	if strings.Count(string(out), `"jsonrpc":"2.0"`) != 2 {
		t.Fatalf("expected two mcp responses, got:\n%s", out)
	}
}

func TestMCPUnknownToolReturnsStructuredError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	out := callMCP(t, bin, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)
	if !strings.Contains(out, `"id":7`) || !strings.Contains(out, `"error":"unknown tool"`) {
		t.Fatalf("expected structured unknown tool error, got:\n%s", out)
	}
}

func TestMCPCanParkContextAndReadProjectDetail(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	callMCP(t, bin, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"park_context","arguments":{"project_id":"tower-control","next_action":"Finish MCP","priority":"5","priority_reason":"Agent integration ready"}}}`)
	out := callMCP(t, bin, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_project_detail","arguments":{"project_id":"tower-control"}}}`)
	if !strings.Contains(out, `"next_action":"Finish MCP"`) || !strings.Contains(out, `"priority_reason":"Agent integration ready"`) {
		t.Fatalf("mcp detail missing parked context:\n%s", out)
	}
}

func callMCP(t *testing.T, bin, request string) string {
	t.Helper()
	cmd := exec.Command(bin, "serve-mcp")
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(request + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("serve-mcp failed: %v\n%s", err, out)
	}
	return string(out)
}

func TestMCPStartParkDayAndParkDayUpdate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	start := callMCP(t, bin, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"start_park_day","arguments":{}}}`)
	if !strings.Contains(start, `"id":"tower-control"`) {
		t.Fatalf("start_park_day missing active project:\n%s", start)
	}
	callMCP(t, bin, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"park_day_update","arguments":{"project_id":"tower-control","field_name":"next_action","value":"Review docs"}}}`)
	out := runCmd(t, bin, "morning", "--format", "json")
	if !strings.Contains(out, `"next_action":"Review docs"`) {
		t.Fatalf("park_day_update did not store context:\n%s", out)
	}
}

func TestProjectAddRejectsInvalidType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	cmd := exec.Command(bin, "project", "add", "Tower Control", "--type", "client")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid type to fail")
	}
	if !strings.Contains(string(out), "invalid project type") {
		t.Fatalf("expected helpful validation error, got:\n%s", out)
	}
}

func TestProjectListCanFilterByStatus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Active Project")
	runCmd(t, bin, "project", "add", "Blocked Project")
	runCmd(t, bin, "project", "update", "blocked-project", "--status", "blocked")

	out := runCmd(t, bin, "project", "list", "--status", "blocked", "--format", "json")
	if !strings.Contains(out, `"id":"blocked-project"`) {
		t.Fatalf("blocked project missing from filtered list:\n%s", out)
	}
	if strings.Contains(out, `"id":"active-project"`) {
		t.Fatalf("active project included in blocked filtered list:\n%s", out)
	}
}

func TestProjectListDefaultsToHumanReadableOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	out := runCmd(t, bin, "project", "list")
	if strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Fatalf("default list should be human-readable, got json:\n%s", out)
	}
	if !strings.Contains(out, "Tower Control") || !strings.Contains(out, "active") {
		t.Fatalf("human list missing project data:\n%s", out)
	}
}

func TestProjectAddRejectsDuplicateSlug(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	cmd := exec.Command(bin, "project", "add", "Tower Control")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected duplicate project to fail")
	}
	if !strings.Contains(string(out), "project already exists: tower-control") {
		t.Fatalf("expected duplicate project error, got:\n%s", out)
	}
}

func TestProjectUpdateRejectsInvalidStatus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	cmd := exec.Command(bin, "project", "update", "tower-control", "--status", "waiting")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected invalid status to fail")
	}
	if !strings.Contains(string(out), "invalid project status") {
		t.Fatalf("expected helpful status error, got:\n%s", out)
	}
}

func TestRootHelpShowsCoreCommands(t *testing.T) {
	bin := buildTowerctl(t)
	cmd := exec.Command(bin, "--help")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}
	for _, want := range []string{"project", "morning", "check", "park-day", "serve-mcp"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("help missing %s:\n%s", want, out)
		}
	}
}

func TestProjectShowDefaultsToHumanAndSupportsJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control", "--priority", "4")
	human := runCmd(t, bin, "project", "show", "tower-control")
	if strings.HasPrefix(strings.TrimSpace(human), "{") {
		t.Fatalf("default show should be human-readable, got json:\n%s", human)
	}
	if !strings.Contains(human, "Tower Control") || !strings.Contains(human, "priority: 4") {
		t.Fatalf("human show missing project data:\n%s", human)
	}
	jsonOut := runCmd(t, bin, "project", "show", "tower-control", "--format", "json")
	if !strings.Contains(jsonOut, `"id":"tower-control"`) || !strings.Contains(jsonOut, `"priority":4`) {
		t.Fatalf("json show missing project data:\n%s", jsonOut)
	}
}

func TestMorningDefaultsToHumanAndSupportsJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	runCmd(t, bin, "park", "tower-control", "--next", "Wire Hermes")
	human := runCmd(t, bin, "morning")
	if strings.HasPrefix(strings.TrimSpace(human), "[") {
		t.Fatalf("default morning should be human-readable, got json:\n%s", human)
	}
	if !strings.Contains(human, "Tower Control") || !strings.Contains(human, "Wire Hermes") {
		t.Fatalf("human morning missing context:\n%s", human)
	}
	jsonOut := runCmd(t, bin, "morning", "--format", "json")
	if !strings.Contains(jsonOut, `"latest_snapshot"`) || !strings.Contains(jsonOut, `"next_action":"Wire Hermes"`) {
		t.Fatalf("json morning missing context:\n%s", jsonOut)
	}
}

func TestNextAndCheckDefaultToHumanAndSupportJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	bin := buildTowerctl(t)

	runCmd(t, bin, "project", "add", "Tower Control")
	runCmd(t, bin, "park", "tower-control", "--next", "Wire Hermes")
	nextHuman := runCmd(t, bin, "next")
	if strings.HasPrefix(strings.TrimSpace(nextHuman), "[") || !strings.Contains(nextHuman, "Wire Hermes") {
		t.Fatalf("human next wrong:\n%s", nextHuman)
	}
	checkHuman := runCmd(t, bin, "check")
	if strings.HasPrefix(strings.TrimSpace(checkHuman), "[") || !strings.Contains(checkHuman, "Did you make progress") {
		t.Fatalf("human check wrong:\n%s", checkHuman)
	}
	nextJSON := runCmd(t, bin, "next", "--format", "json")
	if !strings.Contains(nextJSON, `"next_action":"Wire Hermes"`) {
		t.Fatalf("json next missing action:\n%s", nextJSON)
	}
	checkJSON := runCmd(t, bin, "check", "--format", "json")
	if !strings.Contains(checkJSON, `"questions"`) {
		t.Fatalf("json check missing questions:\n%s", checkJSON)
	}
}

func buildTowerctl(t *testing.T) string {
	t.Helper()
	return towerctlBin
}
