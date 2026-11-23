// herdr-tags lets a workspace carry tags ("#in-review", "#blocked", ...)
// and a free-text note, shown in herdr's sidebar below the branch name. Each
// tag is one workspace metadata token, and so is each line of the note; both
// are edited in a popup picker and persisted per session so a startup hook
// can re-report them after a server restart.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const pluginID = "davidolrik.tags"

// popupWorkspaceEnv hands the workspace the action was invoked in over to the
// popup process, which herdr starts without a tab or pane of its own.
const popupWorkspaceEnv = "HERDR_TAGS_WORKSPACE_ID"

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultDir(kind, name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, kind, name)
}

// Herdr passes its managed per-plugin config and state paths to every plugin
// command; the fallbacks name the same directories for a run by hand.
func pluginConfigDir() string {
	return envOr("HERDR_PLUGIN_CONFIG_DIR",
		defaultDir(".config", filepath.Join("herdr", "plugins", "config", pluginID)))
}

func pluginStateDir() string {
	return envOr("HERDR_PLUGIN_STATE_DIR",
		defaultDir(".local/state", filepath.Join("herdr", "plugins", pluginID)))
}

func configPath() string {
	return filepath.Join(pluginConfigDir(), "config.hcl")
}

func statePath() string {
	return StatePath(pluginStateDir(), envOr("HERDR_SESSION", "default"))
}

// herdrConfigPath and herdrStateDir locate herdr's own files the way herdr
// does.
func herdrConfigPath() string {
	if path := os.Getenv("HERDR_CONFIG_PATH"); path != "" {
		return path
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "herdr", "config.toml")
	}
	return filepath.Join(defaultDir(".config", "herdr"), "config.toml")
}

func herdrStateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "herdr")
	}
	return defaultDir(".local/state", "herdr")
}

// sidebarWidth is the sidebar width of the herdr client on this machine.
func sidebarWidth() int {
	return SidebarWidth(herdrConfigPath(), ClientPreferencesPath(herdrStateDir(), sessionSocketPath()))
}

// workspaceIDFromEnv is the workspace an invocation is about: the popup
// hand-off, then herdr's own variable, then the action context JSON.
func workspaceIDFromEnv() string {
	if ws := os.Getenv(popupWorkspaceEnv); ws != "" {
		return ws
	}
	if ws := os.Getenv("HERDR_WORKSPACE_ID"); ws != "" {
		return ws
	}
	var ctx struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.Unmarshal([]byte(os.Getenv("HERDR_PLUGIN_CONTEXT_JSON")), &ctx); err != nil {
		return ""
	}
	return ctx.WorkspaceID
}

// workspace is the slice of a herdr workspace record the plugin uses.
type workspace struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	Worktree    *struct {
		IsLinkedWorktree bool `json:"is_linked_worktree"`
	} `json:"worktree"`
}

// indented reports whether the sidebar may draw the workspace as a child of
// its repository. Herdr only does so while the repository's own workspace is
// open too; assuming it always is keeps the tags readable either way.
func (ws workspace) indented() bool {
	return ws.Worktree != nil && ws.Worktree.IsLinkedWorktree
}

func listWorkspaces(sock string) ([]workspace, error) {
	result, err := apiRequest(sock, "workspace.list", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Workspaces []workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return nil, fmt.Errorf("decode workspace list: %w", err)
	}
	return payload.Workspaces, nil
}

func getWorkspace(sock, id string) (workspace, error) {
	result, err := apiRequest(sock, "workspace.get", map[string]string{"workspace_id": id})
	if err != nil {
		return workspace{}, err
	}
	var payload struct {
		Workspace workspace `json:"workspace"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return workspace{}, fmt.Errorf("decode workspace: %w", err)
	}
	return payload.Workspace, nil
}

// pruneState keeps only the workspaces herdr still has.
func pruneState(st State, live []string) State {
	alive := map[string]bool{}
	for _, id := range live {
		alive[id] = true
	}
	pruned := State{}
	for id, entry := range st {
		if alive[id] {
			pruned[id] = entry
		}
	}
	return pruned
}

// reportEntry pushes a workspace's tags and note to herdr, both wrapped to
// columns cells per sidebar line.
func reportEntry(sock, workspaceID string, cfg *Config, entry Entry, columns int) error {
	if err := ReportTags(sock, workspaceID, cfg, entry.Tags, columns); err != nil {
		return err
	}
	return ReportNote(sock, workspaceID, cfg, entry.Note, columns)
}

// runStartup re-reports every stored selection and note after a server
// restart, forgetting workspaces that no longer exist.
func runStartup() error {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		return err
	}
	st, err := LoadState(statePath())
	if err != nil {
		return err
	}
	sock := sessionSocketPath()
	workspaces, err := listWorkspaces(sock)
	if err != nil {
		return err
	}
	var live []string
	for _, ws := range workspaces {
		live = append(live, ws.WorkspaceID)
	}
	st = pruneState(st, live)
	width := sidebarWidth()
	var errs []error
	for _, ws := range workspaces {
		entry, ok := st[ws.WorkspaceID]
		if !ok {
			continue
		}
		if err := reportEntry(sock, ws.WorkspaceID, cfg, entry, TagColumns(cfg, width, ws.indented())); err != nil {
			errs = append(errs, err)
		}
	}
	if err := os.MkdirAll(pluginStateDir(), 0o755); err != nil {
		return err
	}
	if err := SaveState(statePath(), st); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// popupChromeRows is what Render draws besides the tag lines: header, blank,
// blank, hints, plus the popup's own border.
const popupChromeRows = 6

// noteFieldRows is what the note field adds to the popup: a blank line and
// the field itself.
const noteFieldRows = 2

// popupSize fits the popup to the tag list and, when the note is on, the
// note field with its cursor.
func popupSize(cfg *Config) (width, height int) {
	width = utf8.RuneCountInString(listHints) + 4
	for _, tag := range cfg.Tags {
		if w := len("> [x] ") + len(TagLabel(tag.Name)) + 4; w > width {
			width = w
		}
	}
	height = len(cfg.Tags) + popupChromeRows
	if cfg.NoteLines > 0 {
		width = max(width, len("> Note: ")+noteFieldWidth+1+4, utf8.RuneCountInString(noteHints)+4)
		height += noteFieldRows
	}
	return width, height
}

// runOpen is the action behind the keybinding: open the picker popup for the
// workspace the key was pressed in.
func runOpen() error {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		return err
	}
	ws := workspaceIDFromEnv()
	if ws == "" {
		return errors.New("open: no workspace in the invocation context")
	}
	width, height := popupSize(cfg)
	params := map[string]any{
		"plugin_id":  pluginID,
		"entrypoint": "picker",
		"placement":  "popup",
		"width":      width,
		"height":     height,
		"env":        map[string]string{popupWorkspaceEnv: ws},
	}
	_, err = apiRequest(sessionSocketPath(), "plugin.pane.open", params)
	return err
}

// runPick is the popup process: edit the workspace's tags and note
// interactively, then report and persist them when the picker saved.
func runPick() error {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		return err
	}
	ws := workspaceIDFromEnv()
	if ws == "" {
		return errors.New("pick: no workspace to tag")
	}
	sock := sessionSocketPath()
	info, err := getWorkspace(sock, ws)
	if err != nil {
		return err
	}
	st, err := LoadState(statePath())
	if err != nil {
		return err
	}

	picker := NewPicker(cfg, st[ws].Tags)
	picker.SetNote(st[ws].Note)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer term.Restore(fd, oldState)
	}
	saved, err := RunPicker(os.Stdin, os.Stdout, picker, info.Label)
	if err != nil || !saved {
		return err
	}

	st[ws] = Entry{Tags: picker.SelectedNames(), Note: strings.TrimSpace(picker.Note)}
	if err := reportEntry(sock, ws, cfg, st[ws], TagColumns(cfg, sidebarWidth(), info.indented())); err != nil {
		return err
	}
	if err := os.MkdirAll(pluginStateDir(), 0o755); err != nil {
		return err
	}
	return SaveState(statePath(), st)
}

// initConfig writes the documented default config, refusing to overwrite.
func initConfig(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.hcl")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("init-config: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(DefaultConfigHCL)
	return err
}

// runSidebarRows prints the config.toml snippet and keeps a copy next to
// config.hcl, since an action's stdout only reaches the plugin log.
func runSidebarRows() error {
	cfg, err := LoadConfig(configPath())
	if err != nil {
		return err
	}
	snippet := SidebarRowsTOML(cfg)
	fmt.Print(snippet)
	if err := os.MkdirAll(pluginConfigDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pluginConfigDir(), "sidebar-rows.toml"), []byte(snippet), 0o644)
}

func usage() {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(`
usage: herdr-tags <command>
  open          open the tag picker popup for the current workspace (action)
  pick          the picker itself; runs inside the popup
  startup       re-report every stored tag selection and note (startup hook)
  init-config   write the documented default config.hcl (never overwrites)
  sidebar-rows  print the [ui.sidebar.spaces] snippet for herdr's config.toml`))
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "open":
		err = runOpen()
	case "pick":
		err = runPick()
	case "startup":
		err = runStartup()
	case "init-config":
		err = initConfig(pluginConfigDir())
	case "sidebar-rows":
		err = runSidebarRows()
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "herdr-tags %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}
