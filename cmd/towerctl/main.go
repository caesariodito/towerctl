package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	Projects  []Project  `json:"projects"`
	Snapshots []Snapshot `json:"snapshots"`
}

type ProjectWithLatestSnapshot struct {
	Project
	LatestSnapshot *Snapshot `json:"latest_snapshot,omitempty"`
}

type CheckQuestion struct {
	ProjectID string   `json:"project_id"`
	Questions []string `json:"questions"`
}

type Snapshot struct {
	ID           int    `json:"id"`
	ProjectID    string `json:"project_id"`
	StoppedWhere string `json:"stopped_where"`
	WhatChanged  string `json:"what_changed"`
	NextAction   string `json:"next_action"`
	Blockers     string `json:"blockers"`
	FilesLinks   string `json:"files_links"`
	NeedHelp     string `json:"need_help"`
	CreatedAt    string `json:"created_at"`
}

type Project struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Status            string `json:"status"`
	Priority          int    `json:"priority"`
	PriorityReason    string `json:"priority_reason"`
	LastPrioritizedAt string `json:"last_prioritized_at,omitempty"`
	Deadline          string `json:"deadline,omitempty"`
	Notes             string `json:"notes"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		printHelp()
		return nil
	}
	s, err := loadStore()
	if err != nil {
		return err
	}
	if args[0] == "park" {
		return park(s, args[1:])
	}
	if args[0] == "morning" {
		return morning(s, args)
	}
	if args[0] == "next" {
		return next(s, args)
	}
	if args[0] == "check" {
		return check(s, args)
	}
	if args[0] == "summary" {
		return morning(s, args)
	}
	if args[0] == "export" {
		return exportMarkdown(s)
	}
	if args[0] == "park-day" {
		return parkDay(s)
	}
	if args[0] == "serve-mcp" {
		return serveMCP(s)
	}
	if len(args) < 2 || args[0] != "project" {
		return fmt.Errorf("unknown command: %s", args[0])
	}
	switch args[1] {
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("project name required")
		}
		p := Project{Name: args[2], ID: slug(args[2]), Type: "personal", Status: "active", Priority: 3, Notes: ""}
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--type":
				i++
				p.Type = args[i]
			case "--priority":
				i++
				n, _ := strconv.Atoi(args[i])
				p.Priority = n
			case "--priority-reason":
				i++
				p.PriorityReason = args[i]
			case "--deadline":
				i++
				p.Deadline = args[i]
			}
		}
		if err := validateProjectType(p.Type); err != nil {
			return err
		}
		if err := validateProjectStatus(p.Status); err != nil {
			return err
		}
		if _, ok := findProject(s, p.ID); ok {
			return fmt.Errorf("project already exists: %s", p.ID)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		p.CreatedAt, p.UpdatedAt = now, now
		if p.PriorityReason != "" {
			p.LastPrioritizedAt = now
		}
		s.Projects = append(s.Projects, p)
		return saveStore(s)
	case "list":
		statusFilter := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "--status" && i+1 < len(args) {
				statusFilter = args[i+1]
			}
		}
		var filtered []Project
		for _, p := range s.Projects {
			if statusFilter == "" || p.Status == statusFilter {
				filtered = append(filtered, p)
			}
		}
		if hasJSONFormat(args) {
			b, _ := json.Marshal(filtered)
			fmt.Println(string(b))
			return nil
		}
		for _, p := range filtered {
			fmt.Printf("%s\t%s\t%s\tpriority:%d\n", p.ID, p.Name, p.Status, p.Priority)
		}
		return nil
	case "show":
		if len(args) < 3 {
			return fmt.Errorf("project id required")
		}
		p, ok := findProject(s, args[2])
		if !ok {
			return fmt.Errorf("project not found: %s", args[2])
		}
		if hasJSONFormat(args) {
			b, _ := json.Marshal(p)
			fmt.Println(string(b))
			return nil
		}
		printProjectHuman(p)
		return nil
	case "update":
		if len(args) < 3 {
			return fmt.Errorf("project id required")
		}
		idx := -1
		for i := range s.Projects {
			if s.Projects[i].ID == args[2] {
				idx = i
			}
		}
		if idx < 0 {
			return fmt.Errorf("project not found: %s", args[2])
		}
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--status":
				i++
				s.Projects[idx].Status = args[i]
			case "--type":
				i++
				s.Projects[idx].Type = args[i]
			case "--priority":
				i++
				n, _ := strconv.Atoi(args[i])
				s.Projects[idx].Priority = n
				s.Projects[idx].LastPrioritizedAt = time.Now().UTC().Format(time.RFC3339)
			case "--priority-reason":
				i++
				s.Projects[idx].PriorityReason = args[i]
				s.Projects[idx].LastPrioritizedAt = time.Now().UTC().Format(time.RFC3339)
			case "--deadline":
				i++
				s.Projects[idx].Deadline = args[i]
			case "--notes":
				i++
				s.Projects[idx].Notes = args[i]
			}
		}
		if err := validateProjectType(s.Projects[idx].Type); err != nil {
			return err
		}
		if err := validateProjectStatus(s.Projects[idx].Status); err != nil {
			return err
		}
		s.Projects[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return saveStore(s)
	default:
		return fmt.Errorf("unknown project command %q", args[1])
	}
}

func storePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".towerctl", "tower.db")
}

func loadStore() (Store, error) {
	db, err := openDB()
	if err != nil {
		return Store{}, err
	}
	defer db.Close()
	return readStore(db)
}

func openDB() (*sql.DB, error) {
	path := storePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT OR IGNORE INTO schema_version(version) VALUES (1);
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT DEFAULT 'personal' CHECK(type IN ('work','side_hustle','personal')),
    status TEXT DEFAULT 'active' CHECK(status IN ('active','blocked','paused','done')),
    priority INTEGER DEFAULT 3 CHECK(priority BETWEEN 1 AND 5),
    priority_reason TEXT DEFAULT '',
    last_prioritized_at TEXT,
    deadline TEXT,
    notes TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stopped_where TEXT DEFAULT '',
    what_changed TEXT DEFAULT '',
    next_action TEXT DEFAULT '',
    blockers TEXT DEFAULT '',
    files_links TEXT DEFAULT '',
    need_help TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_snapshots_project ON snapshots(project_id, created_at DESC);
`

func readStore(db *sql.DB) (Store, error) {
	var s Store
	projects, err := db.Query(`SELECT id, name, type, status, priority, priority_reason, COALESCE(last_prioritized_at,''), COALESCE(deadline,''), notes, created_at, updated_at FROM projects ORDER BY created_at, id`)
	if err != nil {
		return s, err
	}
	defer projects.Close()
	for projects.Next() {
		var p Project
		if err := projects.Scan(&p.ID, &p.Name, &p.Type, &p.Status, &p.Priority, &p.PriorityReason, &p.LastPrioritizedAt, &p.Deadline, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return s, err
		}
		s.Projects = append(s.Projects, p)
	}
	if err := projects.Err(); err != nil {
		return s, err
	}

	snapshots, err := db.Query(`SELECT id, project_id, stopped_where, what_changed, next_action, blockers, files_links, need_help, created_at FROM snapshots ORDER BY id`)
	if err != nil {
		return s, err
	}
	defer snapshots.Close()
	for snapshots.Next() {
		var snap Snapshot
		if err := snapshots.Scan(&snap.ID, &snap.ProjectID, &snap.StoppedWhere, &snap.WhatChanged, &snap.NextAction, &snap.Blockers, &snap.FilesLinks, &snap.NeedHelp, &snap.CreatedAt); err != nil {
			return s, err
		}
		s.Snapshots = append(s.Snapshots, snap)
	}
	return s, snapshots.Err()
}

func park(s Store, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project id required")
	}
	idx := -1
	for i := range s.Projects {
		if s.Projects[i].ID == args[0] {
			idx = i
		}
	}
	if idx < 0 {
		return fmt.Errorf("project not found: %s", args[0])
	}
	snap := Snapshot{ID: len(s.Snapshots) + 1, ProjectID: args[0], CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--stopped":
			i++
			snap.StoppedWhere = args[i]
		case "--changed":
			i++
			snap.WhatChanged = args[i]
		case "--next":
			i++
			snap.NextAction = args[i]
		case "--blocker":
			i++
			snap.Blockers = args[i]
		case "--files":
			i++
			snap.FilesLinks = args[i]
		case "--help":
			i++
			snap.NeedHelp = args[i]
		case "--status":
			i++
			s.Projects[idx].Status = args[i]
		case "--priority":
			i++
			n, _ := strconv.Atoi(args[i])
			s.Projects[idx].Priority = n
			s.Projects[idx].LastPrioritizedAt = time.Now().UTC().Format(time.RFC3339)
		case "--priority-reason":
			i++
			s.Projects[idx].PriorityReason = args[i]
			s.Projects[idx].LastPrioritizedAt = time.Now().UTC().Format(time.RFC3339)
		case "--deadline":
			i++
			s.Projects[idx].Deadline = args[i]
		}
	}
	s.Projects[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	s.Snapshots = append(s.Snapshots, snap)
	return saveStore(s)
}

type MCPRequest struct {
	ID     int `json:"id"`
	Params struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	} `json:"params"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
}

func serveMCP(s Store) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req MCPRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			return err
		}
		var result interface{}
		switch req.Params.Name {
		case "get_active_project_context":
			result = activeProjectContext(s)
		case "get_project_list":
			result = s.Projects
		case "get_project_detail":
			p, _ := findProject(s, req.Params.Arguments["project_id"])
			result = ProjectWithLatestSnapshot{Project: p, LatestSnapshot: latestSnapshot(s, p.ID)}
		case "park_context":
			if err := park(s, mcpParkArgs(req.Params.Arguments)); err != nil {
				return err
			}
			var err error
			s, err = loadStore()
			if err != nil {
				return err
			}
			result = map[string]string{"status": "ok"}
		case "start_park_day":
			result = activeProjectContext(s)
		case "park_day_update":
			args := []string{req.Params.Arguments["project_id"]}
			if flag := parkingFieldFlag(req.Params.Arguments["field_name"]); flag != "" {
				args = append(args, flag, req.Params.Arguments["value"])
			}
			if err := park(s, args); err != nil {
				return err
			}
			var err error
			s, err = loadStore()
			if err != nil {
				return err
			}
			result = map[string]string{"status": "ok"}
		default:
			result = map[string]string{"error": "unknown tool"}
		}
		b, _ := json.Marshal(MCPResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
		fmt.Println(string(b))
	}
	return scanner.Err()
}

func printHelp() {
	fmt.Println(`towerctl - local context management for Hermes-agent

Usage:
  towerctl project add <name> [--type work|side_hustle|personal] [--priority <1-5>] [--priority-reason <text>] [--deadline <date>]
  towerctl project list [--status active|blocked|paused|done] [--format json]
  towerctl project show <id> [--format json]
  towerctl project update <id> [--status active|blocked|paused|done] [--priority <1-5>] [--priority-reason <text>] [--deadline <date>] [--notes <text>]
  towerctl park <project-id> --stopped "..." --changed "..." --next "..."
  towerctl morning --format json
  towerctl next --format json
  towerctl check --format json
  towerctl park-day
  towerctl export --format markdown
  towerctl serve-mcp`)
}

func mcpParkArgs(arguments map[string]string) []string {
	args := []string{arguments["project_id"]}
	mapping := []struct{ key, flag string }{{"stopped_where", "--stopped"}, {"what_changed", "--changed"}, {"next_action", "--next"}, {"blockers", "--blocker"}, {"files_links", "--files"}, {"need_help", "--help"}, {"priority", "--priority"}, {"priority_reason", "--priority-reason"}, {"deadline", "--deadline"}, {"status", "--status"}}
	for _, m := range mapping {
		if v := arguments[m.key]; v != "" {
			args = append(args, m.flag, v)
		}
	}
	return args
}

func parkingFieldFlag(field string) string {
	switch field {
	case "stopped_where":
		return "--stopped"
	case "what_changed":
		return "--changed"
	case "next_action":
		return "--next"
	case "blockers":
		return "--blocker"
	case "files_links":
		return "--files"
	case "need_help":
		return "--help"
	case "priority":
		return "--priority"
	case "priority_reason":
		return "--priority-reason"
	case "deadline":
		return "--deadline"
	case "status":
		return "--status"
	default:
		return ""
	}
}

func parkDay(s Store) error {
	scanner := bufio.NewScanner(os.Stdin)
	read := func(prompt string) string {
		fmt.Fprintln(os.Stderr, prompt)
		if scanner.Scan() {
			return scanner.Text()
		}
		return ""
	}
	projectID := read("Project ID:")
	return park(s, []string{
		projectID,
		"--stopped", read("Where I stopped:"),
		"--changed", read("What changed since last session:"),
		"--next", read("Next exact action:"),
		"--blocker", read("Blockers:"),
		"--files", read("Files/links:"),
		"--help", read("Need help from whom:"),
	})
}

func exportMarkdown(s Store) error {
	var b strings.Builder
	b.WriteString("# towerctl Export\n\n")
	for _, p := range s.Projects {
		b.WriteString("## " + p.Name + "\n\n")
		b.WriteString("- ID: " + p.ID + "\n")
		b.WriteString("- Status: " + p.Status + "\n")
		b.WriteString("- Priority: " + strconv.Itoa(p.Priority) + "\n")
		if p.PriorityReason != "" {
			b.WriteString("- Priority reason: " + p.PriorityReason + "\n")
		}
		if snap := latestSnapshot(s, p.ID); snap != nil {
			b.WriteString("\n### Latest Snapshot\n\n")
			b.WriteString("- Stopped where: " + snap.StoppedWhere + "\n")
			b.WriteString("- What changed: " + snap.WhatChanged + "\n")
			b.WriteString("- Next action: " + snap.NextAction + "\n")
			b.WriteString("- Blockers: " + snap.Blockers + "\n")
		}
		b.WriteString("\n")
	}
	path := filepath.Join(filepath.Dir(storePath()), "exports", "towerctl.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func check(s Store, args []string) error {
	var out []CheckQuestion
	for _, p := range s.Projects {
		if p.Status != "active" {
			continue
		}
		latest := latestSnapshot(s, p.ID)
		nextAction := "this project"
		if latest != nil && latest.NextAction != "" {
			nextAction = latest.NextAction
		}
		out = append(out, CheckQuestion{ProjectID: p.ID, Questions: []string{
			"Did you make progress on " + nextAction + "?",
			"What changed?",
			"Are there blockers?",
			"Should priority context change?",
		}})
	}
	if hasJSONFormat(args) {
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return nil
	}
	for _, item := range out {
		fmt.Println(item.ProjectID)
		for _, q := range item.Questions {
			fmt.Printf("- %s\n", q)
		}
	}
	return nil
}

func next(s Store, args []string) error {
	var out []Snapshot
	for _, snap := range s.Snapshots {
		out = append(out, snap)
	}
	if hasJSONFormat(args) {
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return nil
	}
	for _, snap := range out {
		fmt.Printf("%s: %s\n", snap.ProjectID, snap.NextAction)
	}
	return nil
}

func morning(s Store, args []string) error {
	out := activeProjectContext(s)
	if hasJSONFormat(args) {
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return nil
	}
	for _, item := range out {
		fmt.Printf("%s (%s, priority:%d)\n", item.Name, item.Status, item.Priority)
		if item.PriorityReason != "" {
			fmt.Printf("  priority reason: %s\n", item.PriorityReason)
		}
		if item.LatestSnapshot != nil && item.LatestSnapshot.NextAction != "" {
			fmt.Printf("  next: %s\n", item.LatestSnapshot.NextAction)
		}
	}
	return nil
}

func activeProjectContext(s Store) []ProjectWithLatestSnapshot {
	var out []ProjectWithLatestSnapshot
	for _, p := range s.Projects {
		if p.Status != "active" {
			continue
		}
		item := ProjectWithLatestSnapshot{Project: p}
		for i := range s.Snapshots {
			if s.Snapshots[i].ProjectID == p.ID {
				item.LatestSnapshot = &s.Snapshots[i]
			}
		}
		out = append(out, item)
	}
	return out
}

func latestSnapshot(s Store, projectID string) *Snapshot {
	var latest *Snapshot
	for i := range s.Snapshots {
		if s.Snapshots[i].ProjectID == projectID {
			latest = &s.Snapshots[i]
		}
	}
	return latest
}

func findProject(s Store, id string) (Project, bool) {
	for _, p := range s.Projects {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

func saveStore(s Store) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM snapshots`); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM projects`); err != nil {
		tx.Rollback()
		return err
	}
	for _, p := range s.Projects {
		if _, err := tx.Exec(`INSERT INTO projects(id, name, type, status, priority, priority_reason, last_prioritized_at, deadline, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`, p.ID, p.Name, p.Type, p.Status, p.Priority, p.PriorityReason, p.LastPrioritizedAt, p.Deadline, p.Notes, p.CreatedAt, p.UpdatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}
	for _, snap := range s.Snapshots {
		if _, err := tx.Exec(`INSERT INTO snapshots(id, project_id, stopped_where, what_changed, next_action, blockers, files_links, need_help, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, snap.ID, snap.ProjectID, snap.StoppedWhere, snap.WhatChanged, snap.NextAction, snap.Blockers, snap.FilesLinks, snap.NeedHelp, snap.CreatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func printProjectHuman(p Project) {
	fmt.Printf("%s\n", p.Name)
	fmt.Printf("id: %s\n", p.ID)
	fmt.Printf("type: %s\n", p.Type)
	fmt.Printf("status: %s\n", p.Status)
	fmt.Printf("priority: %d\n", p.Priority)
	if p.PriorityReason != "" {
		fmt.Printf("priority reason: %s\n", p.PriorityReason)
	}
	if p.Deadline != "" {
		fmt.Printf("deadline: %s\n", p.Deadline)
	}
	if p.Notes != "" {
		fmt.Printf("notes: %s\n", p.Notes)
	}
}

func validateProjectStatus(status string) error {
	switch status {
	case "active", "blocked", "paused", "done":
		return nil
	default:
		return fmt.Errorf("invalid project status: %s", status)
	}
}

func hasJSONFormat(args []string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--format" && args[i+1] == "json" {
			return true
		}
	}
	return false
}

func validateProjectType(t string) error {
	switch t {
	case "work", "side_hustle", "personal":
		return nil
	default:
		return fmt.Errorf("invalid project type: %s", t)
	}
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
