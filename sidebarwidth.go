package main

// Herdr does not tell plugins how wide its sidebar is, and it truncates a
// row that does not fit, so the plugin works the width out the way the herdr
// client does: the width the user dragged the sidebar to, which the client
// saves in its preferences file, or else ui.sidebar_width from config.toml,
// clamped to the configured bounds.

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Herdr's defaults for the [ui] sidebar keys.
const (
	defaultSidebarWidth    = 26
	defaultSidebarMinWidth = 18
	defaultSidebarMaxWidth = 36
)

// Sidebar cells a tag line cannot use: three of indent, two of right
// margin, the sidebar's border and the scrollbar it grows when the
// workspaces outnumber its rows.
const sidebarLineChrome = 7

// A linked worktree's workspace is drawn as a child of its repository, with
// a tree guide that indents it five cells further.
const worktreeIndent = 5

// ClientPreferencesPath is where the herdr client attached to the session
// behind apiSockPath keeps its preferences: a file in herdr's state dir named
// after the FNV-1a hash of the client socket path, which sits next to the
// API socket.
func ClientPreferencesPath(herdrStateDir, apiSockPath string) string {
	stem := strings.TrimSuffix(filepath.Base(apiSockPath), filepath.Ext(apiSockPath))
	clientSock := filepath.Join(filepath.Dir(apiSockPath), stem+"-client.sock")
	hash := fnv.New64a()
	hash.Write([]byte(clientSock))
	return filepath.Join(herdrStateDir, "client-shell", fmt.Sprintf("local-%016x.json", hash.Sum64()))
}

// SidebarWidth is the expanded sidebar's width in cells for the client whose
// preferences are at clientPrefsPath. Either file may be missing or
// unreadable; herdr's defaults stand in for whatever cannot be read.
func SidebarWidth(herdrConfigPath, clientPrefsPath string) int {
	var config struct {
		UI struct {
			SidebarWidth    *int `toml:"sidebar_width"`
			SidebarMinWidth *int `toml:"sidebar_min_width"`
			SidebarMaxWidth *int `toml:"sidebar_max_width"`
		} `toml:"ui"`
	}
	// A config.toml herdr cannot parse leaves herdr on its defaults too.
	_, _ = toml.DecodeFile(herdrConfigPath, &config)

	var prefs struct {
		SidebarWidth *int `json:"sidebar_width"`
	}
	if data, err := os.ReadFile(clientPrefsPath); err == nil {
		_ = json.Unmarshal(data, &prefs)
	}

	width := defaultSidebarWidth
	if config.UI.SidebarWidth != nil {
		width = *config.UI.SidebarWidth
	}
	if prefs.SidebarWidth != nil {
		width = *prefs.SidebarWidth
	}

	lo, hi := defaultSidebarMinWidth, defaultSidebarMaxWidth
	if config.UI.SidebarMinWidth != nil {
		lo = *config.UI.SidebarMinWidth
	}
	if config.UI.SidebarMaxWidth != nil {
		hi = *config.UI.SidebarMaxWidth
	}
	if lo > hi {
		lo, hi = defaultSidebarMinWidth, defaultSidebarMaxWidth
	}
	return min(max(width, lo), hi)
}

// TagColumns is how many cells a line of tags gets in a sidebar of the given
// width, on a workspace that is or is not indented. narrowest_sidebar_width
// caps the width first, so the lines also fit the clients it stands for.
func TagColumns(cfg *Config, sidebarWidth int, indented bool) int {
	if cfg.NarrowestSidebarWidth > 0 {
		sidebarWidth = min(sidebarWidth, cfg.NarrowestSidebarWidth)
	}
	columns := sidebarWidth - sidebarLineChrome
	if indented {
		columns -= worktreeIndent
	}
	return columns
}
