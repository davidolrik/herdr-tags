# herdr-tags configuration
#
# One block per tag, in the order the picker lists them and the sidebar
# shows them. Add, remove, or recolor freely; the plugin reads this file on
# every invocation, so changes apply the next time you open the picker.
#
# Names become herdr sidebar tokens ("in-review" is shown as "#in-review"
# through the $tag_in-review token), so they may use ASCII letters, digits,
# underscores and hyphens only, up to 28 characters.
#
# Colors are strict #RGB or #RRGGBB hex, the only form herdr's sidebar
# accepts. After editing, regenerate the sidebar rows for config.toml:
#
#   herdr plugin action invoke davidolrik.tags.sidebar-rows
#
# The defaults are Catppuccin Mocha accents.

tag "in-review" { color = "#f9e2af" }
tag "blocked"   { color = "#f38ba8" }
tag "wip"       { color = "#89b4fa" }
tag "ready"     { color = "#a6e3a1" }
tag "on-hold"   { color = "#9399b2" }
