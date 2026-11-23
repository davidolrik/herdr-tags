# herdr-tags

Tag a workspace with where it stands — `#in-review`, `#blocked`, `#wip` — and
see it in [herdr](https://herdr.dev)'s sidebar, right below the branch name,
each tag in its own color, with a short note underneath if you want one:

```text
◐ herdr-titles
  󰙅 fix-login ↑1
  #in-review #blocked
  Waiting on Søren, second
  pass of the review
```

## Features

- **A popup picker on a key** — arrows (or `j`/`k`) move, space toggles,
  enter saves, escape cancels. The list previews every tag in its color.
- **A note per workspace** — a free-text field below the tags in the picker
  (tab jumps to it); the sidebar shows it under the tags, wrapped onto at
  most two lines.
- **Your tags, your colors** — tags and their colors live in a small HCL
  config; add as many as you like (herdr allows 32 tokens per workspace).
- **Wraps instead of truncating** — tags that do not fit the sidebar's width
  continue on the next line.
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

Tags wrap to the sidebar width of the herdr client on the same machine as
the server. A client attached with `herdr --remote` keeps its width on its
own machine, out of the plugin's sight; if its sidebar is narrower, state
that width and tags wrap to fit the narrower of the two:

```hcl
narrowest_sidebar_width = 24
```

The note wraps at word boundaries onto at most `note_lines` sidebar lines; a
longer note ends in `…`. Both settings are optional, and `note_lines = 0`
turns the note off, field and all:

```hcl
note_lines = 2
note_color = "#9399b2"
```

In the picker, move down past the last tag or press tab to reach the note
field, then type. Left and right move the cursor, home and end (or `ctrl+a`
and `ctrl+e`) jump to either end, and backspace and delete erase. `ctrl+u`
erases everything before the cursor, which clears the whole note when the
cursor is at the end, where it starts; `ctrl+k` erases everything after it.
A note holds up to 200 characters.

### 2. Sidebar rows

Herdr colors sidebar tokens only from its own `config.toml`, so the colors
have to be copied there once. The plugin renders the section for you:

```sh
herdr plugin action invoke davidolrik.tags.sidebar-rows
cat "$(herdr plugin config-dir davidolrik.tags)/sidebar-rows.toml"
```

Paste it into `~/.config/herdr/config.toml` (replacing any existing
`[ui.sidebar.spaces]` section) and run `herdr server reload-config`. Rerun
both steps whenever you add or recolor a tag, or change `note_lines` or
`note_color`.

```toml
[ui.sidebar.spaces]
rows = [
  ["state_icon", "workspace"],
  ["branch", "git_status"],
  [
    { token = "$t1_in-review", fg = "#f9e2af" },
    { token = "$t1_blocked", fg = "#f38ba8" },
    { token = "$t1_wip", fg = "#89b4fa" },
  ],
  [
    { token = "$t2_in-review", fg = "#f9e2af" },
    { token = "$t2_blocked", fg = "#f38ba8" },
    { token = "$t2_wip", fg = "#89b4fa" },
  ],
  [
    { token = "$t3_in-review", fg = "#f9e2af" },
    { token = "$t3_blocked", fg = "#f38ba8" },
    { token = "$t3_wip", fg = "#89b4fa" },
  ],
  [{ token = "$note1", fg = "#9399b2" }],
  [{ token = "$note2", fg = "#9399b2" }],
]
```

Herdr lays tokens out in fixed rows and truncates a row that is too long, so
there is one row per line the tags may wrap onto (one per tag, at most 14
less the note's lines), and each tag has a token per line
(`$t<line>_<name>`, value `#<name>`). The plugin sets only the token on the
line where the tag fits. The note follows with one row and one token
(`$note<line>`) per line. A row whose tokens are all unset disappears, so a
workspace takes no more lines than its tags and note need.

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
setting selected tags and the note's lines and clearing the rest) and writes
the selection and the note to `tags.<session>.json` in the plugin's state
dir. The `[[startup]]` hook reads
that file, drops workspaces herdr no longer lists, and re-reports the rest.

Herdr does not expose its sidebar width, so the plugin derives it the way the
herdr client does: the width the sidebar was dragged to, saved in the client's
preferences file under herdr's state dir, or else `ui.sidebar_width` from
`config.toml`, clamped to `sidebar_min_width`/`sidebar_max_width`. Tags and
the note are wrapped when they are reported, so after resizing the sidebar,
save the picker once more (or restart herdr) to wrap them again.

## Development

```sh
mise install      # Go, prek; installs the pre-commit hooks
mise run test
mise run build
```

## License

MIT
