# skitz CLI Roadmap

## Current Features

- [x] List and select resources interactively
- [x] Compact quick-reference view (`skitz docker`)
- [x] Browse detail sections (`skitz docker -i`)
- [x] Glamour markdown rendering

---

## v0.2 - Search & Navigation ✓

- [x] **Fuzzy search** - filter commands within a resource (`-s` flag)
- [x] **Breadcrumb navigation** - resource → action → view drill-down
- [x] **Back navigation** - "← Back" option at each level

## v0.3 - Clipboard & Execution

- [ ] **Copy to clipboard** - yank selected command
- [x] **Execute command** - run with confirmation prompt
- [ ] **Command preview** - show full command before action

## v0.4 - Personalization

- [ ] **Recently used** - track and surface frequent commands
- [ ] **Favorites** - star commands for quick access
- [ ] **Custom themes** - light/dark/custom color schemes

## v0.5 - Authoring

- [ ] **Quick add** - `skitz add <resource> "command" "description"`
- [ ] **Edit in $EDITOR** - `skitz edit docker`
- [ ] **Sync** - pull community resources from a registry

---

## Ideas / Backlog

- Tags/categories for filtering (e.g., `skitz docker --tag cleanup`)
- Shell completions (zsh, bash, fish)
- Export to different formats (PDF, HTML)
- TUI dashboard mode with multiple panes
- Integration with tldr pages
- `skitz init` to scaffold a new resource
