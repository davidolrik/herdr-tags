# herdr-tags

Tag a workspace with where it stands — `#in-review`, `#blocked`, `#wip` — and
see it in [herdr](https://herdr.dev)'s sidebar, right below the branch name,
each tag in its own color:

```text
◐ herdr-titles
  󰙅 fix-login ↑1
  #in-review #blocked
```

## Features

- **A popup picker on a key** — arrows (or `j`/`k`) move, space toggles,
  enter saves, escape cancels. The list previews every tag in its color.
- **Your tags, your colors** — tags and their colors live in a small HCL
  config; add as many as you like (herdr allows 32 tokens per workspace).
- **Survives restarts** — herdr forgets workspace tokens when its server
  restarts; the plugin stores the selection per session and re-reports it
  from its startup hook. Closed workspaces are forgotten.
- **Painless install** — prebuilt, checksum-verified binaries; only `sh` and
  `curl` needed, never Go.

## Install

```sh
herdr plugin install davidolrik/herdr-tags
```

For development, link a local checkout instead:

```sh
herdr plugin link /path/to/herdr-tags
go build -o bin/herdr-tags .   # `plugin link` does not run the build step
```

## Configure

Three steps: the tag list, the sidebar rows, the key.

### 1. Tags

Generate the documented default config, then edit it:

```sh
herdr plugin action invoke davidolrik.tags.init-config
$EDITOR "$(herdr plugin config-dir davidolrik.tags)/config.hcl"
```

One block per tag, in the order the picker and the sidebar show them:

```hcl
tag "in-review" { color = "#f9e2af" }
tag "blocked"   { color = "#f38ba8" }
tag "wip"       { color = "#89b4fa" }
```

Names may use ASCII letters, digits, underscores and hyphens (up to 28
characters); colors are `#RGB` or `#RRGGBB`. Without a config file the
defaults above plus `ready` and `on-hold` apply.

### 2. Sidebar rows

Herdr colors sidebar tokens only from its own `config.toml`, so the colors
have to be copied there once. The plugin renders the section for you:

```sh
herdr plugin action invoke davidolrik.tags.sidebar-rows
cat "$(herdr plugin config-dir davidolrik.tags)/sidebar-rows.toml"
```

Paste it into `~/.config/herdr/config.toml` (replacing any existing
`[ui.sidebar.spaces]` section) and run `herdr server reload-config`. Rerun
both steps whenever you add or recolor a tag.

```toml
[ui.sidebar.spaces]
rows = [
  ["state_icon", "workspace"],
  ["branch", "git_status"],
  [
    { token = "$tag_in-review", fg = "#f9e2af" },
    { token = "$tag_blocked", fg = "#f38ba8" },
    { token = "$tag_wip", fg = "#89b4fa" },
  ],
]
```

Each tag is one workspace token (`$tag_<name>`, value `#<name>`). A row whose
tokens are all unset disappears, so untagged workspaces stay compact.

### 3. Key

```toml
[[keys.command]]
key = "prefix+#"
type = "plugin_action"
command = "davidolrik.tags.pick"
description = "Tag workspace"
```

## Actions

| Action                         | What it does                                                 |
| ------------------------------ | ------------------------------------------------------------ |
| `davidolrik.tags.pick`         | Open the picker popup for the current workspace              |
| `davidolrik.tags.init-config`  | Write the documented default config file (never overwrites)  |
| `davidolrik.tags.sidebar-rows` | Render the `[ui.sidebar.spaces]` snippet from the tag colors |

Invoke with `herdr plugin action invoke <action>`, or bind one to a key in
herdr's `config.toml` (`type = "plugin_action"`).

## How it works

The `pick` action opens a manifest `[[panes]]` entrypoint as a session-modal
popup, sized to the tag list, handing it the workspace ID through its
environment. The popup runs the picker in raw mode; on enter it patches the
workspace's tokens over herdr's socket API (`workspace.report_metadata`,
setting selected tags and clearing the rest) and writes the selection to
`tags.<session>.json` in the plugin's state dir. The `[[startup]]` hook reads
that file, drops workspaces herdr no longer lists, and re-reports the rest.

## Development

```sh
mise install      # Go, prek; installs the pre-commit hooks
mise run test
mise run build
```

## License

MIT
