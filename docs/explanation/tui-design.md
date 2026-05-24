# TUI Design

Our TUI design is based on the proven patterns from [gh-dash](https://github.com/dlvhdr/gh-dash), distilled into principles that apply to any bubbletea application with a two-pane layout.

> For rationale on why these specific patterns were chosen over alternatives, see the [Why Temporal](why-temporal.md) approach of weighing tradeoffs for each architectural decision.

## Reference application

gh-dash is a terminal dashboard for GitHub built with bubbletea v2 and lipgloss v2. It has been battle-tested across thousands of users and hundreds of terminal environments. Its layout, pane, and resizing approach is the reference model for this project's TUI.

## Layout architecture

### Top-down space budget

Space is allocated from top to bottom, with each region consuming a fixed or computed height:

```
┌──────────────────────────────────────┐
│ Tabs row                    (2 rows) │  HeaderHeight
├──────────────────────────────────────┤
│ Main content area                    │  MainContentHeight ─┐
│  ┌────────────────┬──────────────┐   │                     │
│  │ Search bar     │              │   │   SearchHeight      │
│  │ Table/list     │   Preview    │   │                     │
│  │                │   pane       │   │                     │
│  │                │              │   │                     │
│  └────────────────┴──────────────┘   │                     │
├──────────────────────────────────────┤  <─ base content height
│ Footer / status bar         (1 row)  │  FooterHeight
└──────────────────────────────────────┘
```

Layout constants are declared once in a shared package and never hardcoded:

```go
var (
    HeaderHeight       = 2
    SearchHeight       = 3
    FooterHeight       = 1
    InputBoxHeight     = 8
    TabsHeight         = TabsBorderHeight + TabsContentHeight
    TableHeaderHeight  = 2
)
```

### Preview pane positioning

The preview pane can be positioned **right** (side-by-side) or **bottom** (stacked), chosen by user config or automatically based on available width:

| Mode | Triggers | Main Content | Preview |
|------|----------|--------------|---------|
| Right | Width >= 80 after preview | Horizontal slice | Vertical full-height |
| Bottom | Width < 80 or user pref | Vertical slice | Horizontal full-width |
| Hidden | User toggle (keybinding) | Full width | Invisible |

**Auto logic**: When the user sets `position: "auto"`, check if `screenWidth - previewWidth >= 80`. If not, fall back to bottom mode. This ensures the table/lists remain usable on narrow terminals.

**Defaults** (from gh-dash, adopt unless domain differences dictate otherwise):
- Preview open: `true`
- Preview width: `0.45` (45% of screen, clamped)
- Preview height (bottom mode): `0.60` (60% of available)
- Position: `"auto"`

### Responsive resize

A single `syncMainContentDimensions()` function recalculates all computed dimensions after every `WindowSizeMsg`. Components never compute their own dimensions — they read precomputed values from the shared context:

```go
type ProgramContext struct {
    ScreenHeight         int
    ScreenWidth          int
    MainContentWidth     int    // width for section tables
    MainContentHeight    int    // height for section tables
    DynamicPreviewWidth  int    // width of preview pane (0 = hidden)
    DynamicPreviewHeight int    // height of preview pane (bottom mode only)
    PreviewPosition      string // "right" or "bottom"
    SidebarOpen          bool
}
```

The resize function:
1. Resolves preview position (auto → right or bottom)
2. If sidebar closed → main content fills entire screen
3. If right mode → subtracts preview width from screen width
4. If bottom mode → subtracts preview height from available content height
5. Calls `syncProgramContext()` to push new dimensions to all child components

### Layers and compositing

Use `lipgloss.Compositor` for overlay elements (search autocomplete, command palettes) rather than trying to bake them into the main layout. This avoids shifting content when overlays appear.

## Component architecture

### Package-per-component

Each UI component lives in its own package under `internal/tui/components/`:

```
components/
├── sidebar/        # Preview pane wrapper (viewport)
├── section/        # Section interface + base implementation
├── table/          # Data table with columns, rendering
├── listviewport/   # Virtual scroll viewport for lists
├── tabs/           # Horizontal tab/carousel navigation
├── search/         # Search bar with autocomplete
├── footer/         # Status bar
├── prview/         # PR detail view (shown in sidebar)
├── issueview/      # Issue detail view (shown in sidebar)
├── prompt/         # Yes/No confirmation prompts
└── ...
```

### Bubbletea component contract

Every component follows the same pattern:

```go
type Model struct { /* component state */ }

func NewModel(ctx *context.ProgramContext) Model { /* init */ }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) { /* handle messages */ }

func (m Model) View() string { /* render */ }

func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) { /* receive new dimensions/themes */ }
```

**No `*Model` receiver for Update/View**: gh-dash uses value receivers for `Update` and `View`, copying all state into local variables and returning a new model. This makes the data flow explicit and avoids shared-state bugs. However, pointer receivers are acceptable when the component is large or contains a viewport (which embeds a mutex).

### Section interface

Multiple view types (PRs, Issues, Notifications, Branches) share a common `Section` interface:

```go
type Section interface {
    // Identity
    GetId() int
    GetType() string

    // Bubbletea
    Update(msg tea.Msg) (Section, tea.Cmd)
    View() string

    // Navigation
    NumRows() int
    GetCurrRow() data.RowData
    CurrRow() int
    NextRow() int
    PrevRow() int

    // Data
    FetchNextPageSectionRows() []tea.Cmd
    ResetRows()

    // Search
    SetIsSearching(val bool) tea.Cmd
    IsSearchFocused() bool

    // Context
    UpdateProgramContext(ctx *context.ProgramContext)
    GetConfig() config.SectionConfig
}
```

A `BaseModel` struct provides default implementations that concrete sections embed. Sections with different behavior (e.g., repo branches vs. PR search) override specific methods.

### Multi-level update delegation

The root model's `Update` dispatches to child components in a specific order, reflecting input priority:

1. **Modal overlays** (prompt confirmation, search) — highest priority
2. **Text input boxes** (comment, label, assign)
3. **Quit confirmation**
4. **Custom keybindings** (user-defined commands from config)
5. **Navigation keys** (up/down/prev-section/next-section)
6. **View-specific actions** (approve, merge, close — varies by active view tab)
7. **Child component updates** (tabs, sidebar, sidebar contents, footer)

After dispatching, the root always finalizes with:

```go
m.syncProgramContext()  // push dimensions to all children
// collect and batch all child commands
return m, tea.Batch(cmds...)
```

## Sidebar (preview pane)

### Viewport-based scrolling

The sidebar wraps a `viewport.Model` from `charm.land/bubbles/v2/viewport`. Content is set as a flat string via `SetContent(data string)`, and the viewport handles scrolling, page up/down, and percent display.

The sidebar does not render content itself — it delegates to type-specific sub-components:

```go
switch row := currRowData.(type) {
case *prrow.Data:
    m.prView.SetRow(row)
    m.prView.SetWidth(width)
    m.sidebar.SetContent(m.prView.View())  // renders the PR detail
case *data.IssueData:
    m.issueSidebar.SetRow(row)
    m.issueSidebar.SetWidth(width)
    m.sidebar.SetContent(m.issueSidebar.View())
}
```

### Scroll behavior

- On row change: scroll to top (`viewport.GotoTop()`)
- On text input focus (comment/assign): scroll to bottom (`viewport.GotoBottom()`)
- On notification content load with a latest comment URL: scroll to bottom
- Page up/down keys are handled by the sidebar's own Update, not the root

### Internal tab switching

Sub-components shown in the sidebar (e.g., PR detail) can have internal tabs. The `prView` component switches between Overview, Activity, Files, and Checks tabs using internal state. The sidebar is unaware of this — it simply renders whatever `prView.View()` returns.

## Table and list viewport

### Virtual scrolling

The `listviewport` component wraps bubbletea's viewport to provide virtual list scrolling:

```go
type Model struct {
    viewport        viewport.Model
    topBoundId      int
    bottomBoundId   int
    currId          int
    ListItemHeight  int
    NumCurrentItems int
}
```

When the cursor reaches the bottom of the visible area, the viewport scrolls by one `ListItemHeight` and adjusts bounds. This keeps rendering bounded regardless of list length.

### Column layout

Tables define columns with explicit or computed widths:

| Column property | Behavior |
|-----------------|----------|
| `Width` set to a value | Fixed width |
| `Width` nil, `Grow` true | Evenly splits remaining width |
| `Width` nil, `Grow` false | Natural width (content-sized) |
| `Hidden` true | Excluded from render |

Column widths are cached on each render pass (via `cacheColumnWidths()`) to avoid recomputing for each row.

### Item height

Item height adapts to the theme config:

- **Compact mode**: 1 row per item
- **Normal mode**: 2 rows per item
- **Separator on**: +1 row for border

The viewport's visible item count is `viewportHeight / itemHeight`.

## Key handling

### Key bindings as data

Key bindings are defined as `key.Binding` structs (from `charm.land/bubbles/v2/key`), not raw strings. They support key sequences (e.g., `ctrl+shift+up`) and help text:

```go
type KeyMap struct {
    Up             key.Binding
    Down           key.Binding
    TogglePreview  key.Binding
    Refresh        key.Binding
    Quit           key.Binding
}
```

Matching uses `key.Matches(msg, keys.Keys.Up)` rather than `msg.String() == "k"`. This enables user-configurable rebinding and documentation generation.

### Input priority

Key dispatch follows a strict priority order in the root Update method:

1. Search bar focused → delegate to section's search
2. PR comment/assign input focused → delegate to prView
3. Issue comment/assign input focused → delegate to issueSidebar
4. Quit confirmation → handle y/enter or dismiss
5. Notification action confirmation → handle or reset
6. Custom keybindings → execute template command
7. Built-in navigation → prev/next section, up/down, home/end
8. Built-in actions → toggle preview, refresh, search, help
9. View-specific actions → approve, merge, close (varies by active tab)
10. Default → pass through to child components

## Context and dependency injection

### Centralized context

A single `ProgramContext` struct is the source of truth for all shared state. Every component receives it via `UpdateProgramContext(ctx)`:

```go
type ProgramContext struct {
    // Dimensions
    ScreenHeight, ScreenWidth int
    MainContentWidth, MainContentHeight int
    DynamicPreviewWidth, DynamicPreviewHeight int
    PreviewPosition string
    SidebarOpen bool

    // Config & state
    Config  *config.Config
    Theme   theme.Theme
    Styles  Styles
    View    ViewType
    User    string
    Error   error

    // Repo
    Repo     repository.Repository
    RepoPath string
    RepoUrl  string

    // Task tracking
    StartTask func(task Task) tea.Cmd
}
```

Components **read** from context but never **write** to it directly. The root model computes context values and pushes them down.

### The sync cascade

When dimensions change (resize, toggle preview, toggle help):

```
WindowSizeMsg / key event
  → m.onWindowSizeChanged()
    → m.syncMainContentDimensions()  // compute new dimensions
      → m.syncSidebar()              // re-render preview
      → m.syncProgramContext()       // push to all children
```

This ensures all components agree on dimensions before the next render frame.

## Theme and styling

### Centralized styles

All lipgloss styles are defined in a `Styles` struct, created from a `Theme`:

```go
type Styles struct {
    Sidebar   SidebarStyles    // Root, BottomRoot, PagerStyle, etc.
    Section   SectionStyles    // ContainerStyle, KeyStyle, etc.
    Table     TableStyles      // HeaderStyle, CellStyle, SelectedCellStyle, etc.
    Tabs      TabsStyles       // Tab, ActiveTab, etc.
    Common    CommonStyles     // MainTextStyle, FooterStyle, etc.
}
```

Components reference these styles rather than creating their own. This enables theme switching without component changes.

### Theme config

Themes are YAML-configurable with a default fallback. gh-dash's pattern: load theme → build Styles → store in context. Components receive styles via `UpdateProgramContext`.

## Mouse support

Mouse scrolling must be enabled on every pane that can overflow. This is framework-supported, not a limitation — it simply needs to be turned on in the right places.

### Why gh-dash doesn't have it

gh-dash only handles `tea.MouseClickMsg` (donate button). It does not enable mouse wheel on any viewport. This was a choice, not a framework gap. We should do better.

### How bubbletea handles the mouse

Bubbletea v2 sends mouse events when the root `View()` requests them:

```go
func (m Model) View() tea.View {
    var v tea.View
    v.MouseMode = tea.MouseModeCellMotion  // enables click + wheel events
    // ...
}
```

The viewport component from `charm.land/bubbles/v2/viewport` has built-in mouse wheel support:

```go
type Model struct {
    // ...
    MouseWheelEnabled bool  // set to true to enable
    MouseWheelDelta   int   // lines per wheel tick (default: 3)
    // ...
}
```

When `MouseWheelEnabled` is true, the viewport's `Update` method handles `tea.MouseWheelMsg` internally — scrolling up/down by `MouseWheelDelta` lines per event. No custom handler is needed in the parent component.

### What to enable

| Component | What to set | Where |
|-----------|------------|-------|
| **Main list viewport** (yak tree) | `MouseWheelEnabled: true` | In `listviewport.NewModel()` or `table.NewModel()` |
| **Sidebar/preview viewport** | `MouseWheelEnabled: true` | In `sidebar.NewModel()` |
| **Root `View()`** | `v.MouseMode = tea.MouseModeCellMotion` | Already set in gh-dash pattern, keep it |

The viewport's `MouseWheelEnabled` only takes effect when the root View enables mouse mode. Both conditions must be true. The root View already enables `MouseModeCellMotion` per the gh-dash pattern — if this gets removed for any reason, mouse scrolling silently breaks everywhere.

### Scroll target resolution

When multiple viewports are visible simultaneously (e.g., list on the left, preview on the right), bubbletea routes mouse events to all components during the update cycle. Each viewport independently processes `tea.MouseWheelMsg` in its `Update` method. This means **both the list and the preview will scroll simultaneously** unless the app resolves a scroll target.

To fix this, use bubblezone (`github.com/lrstanley/bubblezone/v2`) to detect which pane the cursor is over:

```go
case tea.MouseMsg:
    if msg.Action != tea.MouseActionPress && msg.Action != tea.MouseActionRelease {
        // Wheel events
        if zone.Get("main-content").InBounds(msg) {
            // route to main list viewport
            m.listViewport, cmd = m.listViewport.Update(msg)
        } else if zone.Get("sidebar").InBounds(msg) {
            // route to sidebar viewport
            m.sidebar, cmd = m.sidebar.Update(msg)
        }
    }
```

gh-dash already imports bubblezone (for the donate button click target). Extend this pattern to all scrollable panes.

### Default config values

```go
MouseWheelDelta: 3  // sensible default for most terminals
```

Exposing `MouseWheelDelta` as a user-configurable option is low priority. Hardcoded 3 is fine initially. Some terminals send events with varying delta magnitudes (e.g., macOS trackpad sends fractional deltas). The viewport handles this — it scrolls by `MouseWheelDelta` lines regardless of the event magnitude.

## Comparison with current yaketyyak TUI

| Pattern | gh-dash | yaketyyak (current) |
|---------|---------|---------------------|
| **Layout calculation** | Declarative; precomputed dimensions in context, single sync function | Inline manual arithmetic in View() (treeWidth, detailWidth, bodyHeight) |
| **Component structure** | Package per component; each has Update/View | Monolithic model.go with all logic in one struct |
| **Rendering** | lipgloss.JoinHorizontal/Vertical with proper Width constraints | Manual line-by-line string concatenation in a for loop |
| **Scrolling** | bubbles viewport for virtual scroll, bounds tracking | Manual offset with int-based slicing |
| **Key bindings** | bubbles/key bindings, declarative KeyMap, configurable | Raw string matching in switch cases |
| **Preview/sidebar** | Full viewport-based sidebar with sub-components | No preview pane; detail is a static column |
| **Resize** | Full recalc on WindowSizeMsg, all children notified | Stored in width/height fields but tree offset only recalculated |
| **Tabs** | Carousel-based tab navigation | Flat repo list (no tabs) |
| **Overlays** | Compositor for search autocomplete/completions | Status/help inline in statusBar |
| **Styling** | Centralized Styles struct, theme-configurable | Package-level var styles (lipgloss styles as globals) |
| **Mouse scrolling** | Not enabled (only click handled) | Not supported |
| **Context** | Passed explicitly via UpdateProgramContext | Not used; repos/state are fields on Model |

## Domains where we diverge from gh-dash

gh-dash is a **read/view** dashboard (view PRs, approve, merge). yaketyyak is a **workflow orchestrator** (start workflows, send signals, watch CI). This means:

- **We need signal/action dispatch** that gh-dash's "preview detail" doesn't cover. Our preview should show workflow state and offer signal buttons.
- **Our tree view** (repos → yaks hierarchy) is domain-specific. gh-dash has flat lists. We should keep the tree but render it with a proper viewport.
- **Modal dialogs** for confirmation (start workflow, pause, resume) fit gh-dash's prompt pattern.
- **Real-time updates** (workflow state changes) need polling-style comm updates (gh-dash uses periodic refresh too).

## When adding TUI features

Follow this checklist, in order:

1. **Dimensions**: Compute space in `syncMainContentDimensions()`, not inline
2. **Component**: New component gets its own package with `Model`, `NewModel`, `Update`, `View`, `UpdateProgramContext`
3. **Context**: Read dimensions/styles from `ProgramContext`, never hardcode
4. **Styling**: Add to `Styles` struct, theme-configurable; never create inline styles
5. **Key handling**: Add to `KeyMap`, use `key.Matches`, support config rebinding
6. **Scrolling**: Use bubbles `viewport.Model`, not manual offset math
7. **Mouse scrolling**: Set `MouseWheelEnabled = true` on every viewport; wrap panes in bubblezone zones so wheel events route to the pane under the cursor
8. **Layout**: Use `lipgloss.JoinHorizontal`/`JoinVertical` with `Width` constraints
9. **Overlays**: Use `lipgloss.Compositor` for floating elements

> For the workflow architecture, see [Architecture](architecture.md).
> For why Temporal was chosen over polling/scripts, see [Why Temporal](why-temporal.md).
