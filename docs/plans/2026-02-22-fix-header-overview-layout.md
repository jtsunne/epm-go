# Fix header and overview bar layout issues

## Overview

The TUI has two layout bugs visible at standard terminal widths:

1. **Header wraps to second line** — when cluster name + status + poll interval exceed terminal width, `renderHeader` clamps spacing to 0 but still concatenates all segments at full length. `lipgloss.Style.Width()` only enforces *minimum* width (pads with spaces), not maximum, so the row overflows and wraps.

2. **Overview bar doesn't fill width + inconsistent card heights** — the formula `(width - 14) / 7` subtracts 14 for no visible element (no borders/margins between cards) and integer division loses the remainder. The bar ends up ~14-20 chars short. Additionally, cards have 2/3/4 content lines (Status=2, CPU/JVM=3, Storage=4) but no explicit `.Height()` is set, so `JoinHorizontal` pads shorter cards with empty lines that lack the card's background color.

## Context (from discovery)

- Files/components involved:
  - `internal/tui/header.go` — `renderHeader()` lines 196-216 (spacing/layout logic)
  - `internal/tui/overview.go` — `renderOverview()` lines 28-40 (width calc), 67-141 (card rendering)
  - `internal/tui/styles.go` — `StyleHeader` (Padding 0,1), `StyleOverviewCard` (Padding 0,1, Margin 0, Align Center)
  - `internal/tui/table.go:218` — `truncateName()` helper to reuse for header truncation
- Related patterns found: lipgloss v1.1.0 supports `.MaxWidth()`, `.Height()`, `.AlignVertical()`
- Dependencies: no new dependencies needed

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional - they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - add new test cases for new code paths
  - update existing test cases if behavior changes
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** - no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Testing Strategy

- **Unit tests**: required for every task (see Development Approach above)

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix
- Update plan if implementation deviates from original scope
- Keep plan in sync with actual work done

## Implementation Steps

### Task 1: Fix header overflow with progressive truncation

In `renderHeader()` (`header.go:196-216`), replace the spacing logic with progressive truncation when content exceeds `innerWidth`:

- [x] Implement progressive truncation with priority: center (status) > right (timing) > left (cluster name):
  1. First truncate `left` (cluster name) using existing `truncateName()` from `table.go:218`
  2. If center + right alone exceed width, drop `left` entirely and truncate `right` via `lipgloss.NewStyle().MaxWidth()`
  3. If even center alone overflows, hard-truncate center via `MaxWidth`, drop others
- [x] Add minimum gap constant (`minGap = 1`) to ensure at least 1 space between segments when all present
- [x] Add `MaxWidth(innerWidth)` to the final `StyleHeader` render as safety net against any overflow
- [x] Write test: header stays single line at width=60 with long cluster name
- [x] Write test: header stays single line at width=30 (very narrow terminal)
- [x] Write test: disconnected header doesn't wrap at width=60 (long error + retry text)
- [x] Run `make test` — must pass before task 2

### Task 2: Fix overview bar width distribution

In `renderOverview()` (`overview.go:28-40`), fix the width formula to fill full terminal width:

- [ ] Replace `(width - 14) / 7` with `width / 7` + remainder distribution: first `width % 7` cards get +1 width
- [ ] Use `cardWidths []int` slice (len 7) instead of single `cardWidth` variable
- [ ] Update each card render (card1-card7) to use `cardWidths[i]` instead of `cardWidth`
- [ ] Calculate `barWidth` per-card for cards 5-7 (progress bars) as `cardWidths[i] - 4`
- [ ] For narrow mode: use `width / 2` with remainder to first card in each row; card7 alone gets full `width`
- [ ] Write test: verify total rendered width of overview in wide mode matches `app.width`
- [ ] Run `make test` — must pass before task 3

### Task 3: Fix overview card height consistency

In `renderOverview()` (`overview.go:67-141`), equalize card heights in wide mode:

- [ ] Define `const maxCardHeight = 4` (Storage card has 4 lines: value + bar + used/total + label)
- [ ] In wide mode: add `.Height(maxCardHeight).AlignVertical(lipgloss.Center)` to all 7 card style chains — makes lipgloss fill background for full height and vertically centers shorter content
- [ ] In narrow mode: skip explicit height (rows are stacked, per-row heights equalize naturally via `JoinHorizontal`)
- [ ] Use helper closure `applyHeight := func(s lipgloss.Style) lipgloss.Style` that conditionally applies height only in wide mode
- [ ] Write test: verify all 7 cards in wide mode have equal line count
- [ ] Run `make test` — must pass before task 4

### Task 4: Verify acceptance criteria

- [ ] Verify header stays on single line at widths 60, 80, 120, 200
- [ ] Verify overview fills full terminal width with no trailing gap
- [ ] Verify all 7 overview cards have consistent height in wide mode
- [ ] Run full test suite (`make test`)
- [ ] Run linter (`make lint`) — all issues must be fixed

### Task 5: [Final] Update documentation

- [ ] Update CLAUDE.md if any new patterns discovered
- [ ] Update README.md if needed

## Technical Details

**lipgloss v1.1.0 API used:**
- `.Width(n)` — sets minimum width (pads with spaces), already used everywhere
- `.MaxWidth(n)` — truncates content exceeding width, preserves ANSI — used as safety net on header
- `.Height(n)` — sets minimum height (pads with empty lines using background), used for card equalization
- `.AlignVertical(lipgloss.Center)` — vertically centers content within Height — available since v1.0

**Width formula change:**
- Old: `cardWidth = (width - 14) / 7` — loses 14 + up to 6 chars of remainder
- New: `base = width / 7`, first `width % 7` cards get `base + 1`

**Header truncation priority:** center (status) > right (timing) > left (cluster name)

## Post-Completion

**Manual verification:**
- Run `epm` against a live ES cluster at various terminal widths (resize terminal window)
- Verify header never wraps regardless of cluster name length
- Verify overview bar fills entire width with consistent card heights
