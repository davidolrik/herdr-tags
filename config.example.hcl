# herdr-tags configuration
#
# One block per tag, in the order the picker lists them and the sidebar
# shows them. Add, remove, or recolor freely; the plugin reads this file on
# every invocation, so changes apply the next time you open the picker.
#
# Names become herdr sidebar tokens ("in-review" is shown as "#in-review"
# through the $t1_in-review token, $t2_in-review on the second line, and so
# on), so they may use ASCII letters, digits, underscores and hyphens only,
# up to 28 characters.
#
# Colors are strict #RGB or #RRGGBB hex, the only form herdr's sidebar
# accepts. After editing, regenerate the sidebar rows for config.toml:
#
#   herdr plugin action invoke davidolrik.tags.sidebar-rows
#
# The defaults are Catppuccin Mocha accents.
#
# Tags that do not fit on one sidebar line wrap onto the next. The plugin
# reads the sidebar width from the herdr client on this machine; a client
# attached with "herdr --remote" keeps its width where the plugin cannot see
# it. If such a client has a narrower sidebar, give its width here and tags
# are wrapped to fit the narrower of the two:
#
# narrowest_sidebar_width = 24

tag "in-review" { color = "#f9e2af" }
tag "blocked"   { color = "#f38ba8" }
tag "wip"       { color = "#89b4fa" }
tag "ready"     { color = "#a6e3a1" }
tag "on-hold"   { color = "#9399b2" }
