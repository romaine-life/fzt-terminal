package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
	"github.com/romaine-life/fzt/render"
)

// -- Unified renderer --

// revealMargin is how many rows show above a row placed by TreeReveal.
const revealMargin = 2

// drawUnified renders the prompt bar and tree. The tree is the single
// navigation surface -- no separate results section.
// Within this function, ctx.NavMode affects ONLY the prompt icon.
// All other rendering is mode-independent.
func drawUnified(c render.Canvas, s *core.State, cfg Config, w, startY, h int) {
	ctx := s.TopCtx()

	borderOffset := 0
	y := startY

	if cfg.Border {
		title := cfg.Title
		titleStyle := 0
		// StatesBannerOn overrides everything — the inspector is the user's
		// active tool, and a stale title would obscure the state readout.
		if s.StatesBannerOn {
			title = s.Describe()
			titleStyle = 0
		} else if s.TitleOverride != "" {
			// TitleOverride always takes priority — it's an explicit message
			title = s.TitleOverride
			titleStyle = s.TitleStyle
		} else if s.SyncTimerShown && s.SyncNextCheck > 0 {
			remaining := s.SyncNextCheck - time.Now().Unix()
			if remaining < 0 {
				remaining = 0
			}
			m := remaining / 60
			sec := remaining % 60
			title = fmt.Sprintf("next sync check: %dm %02ds", m, sec)
		}
		// Dirty indicator takes priority over sync icon
		icon := s.SyncIcon
		if s.Dirty {
			icon = "\u25cf" // ●
		}
		drawBorderTopWithTitle(c, w, y, title, cfg.TitlePos, titleStyle, s.IsPulsing(), icon, cfg.Label)
		y++
		borderOffset = 1
	}

	if cfg.ShowPanel {
		drawPanel(c, s.Panel, borderOffset, y, w)
		y += core.PanelRows
	}

	hasQuery := len(ctx.Query) > 0
	visible := core.TreeVisibleItems(s)

	// Compute effective name column width from visible items.
	// Each row needs: indent + name + gap. We find the max absolute width
	// (indent + max(nameLen+1, NameColWidth+ColGap-indent)) so all descriptions align.
	effectiveNameCol := ctx.NameColWidth + ctx.ColGap
	if effectiveNameCol < 12 {
		effectiveNameCol = 12 // minimum so headers don't collapse
	}
	for _, row := range visible {
		indent := rowIndent(row, cfg)
		nameW := ctx.NameColWidth + ctx.ColGap - indent
		if len(row.Item.Fields) > 0 {
			nameLen := len([]rune(row.Item.Fields[0])) + 1
			if nameLen > nameW {
				nameW = nameLen
			}
		}
		if nameW+indent > effectiveNameCol {
			effectiveNameCol = nameW + indent
		}
	}

	// Prompt bar -- bordered input field, the primary UI element
	promptBg := PromptSurfaceBg
	borderStyle := tcell.StyleDefault.Foreground(BorderFg)

	// Mode indicator: search (magnifying glass) vs nav (arrow) vs inspect (gear)
	var promptIcon rune
	var promptIconStyle tcell.Style
	if s.PromptMode != "" {
		promptIcon = s.PromptIcon
		if promptIcon == 0 {
			promptIcon = '\uF002'
		}
		promptIconStyle = tcell.StyleDefault.Foreground(SearchModeFg).Bold(true).Background(promptBg)
	} else if s.InspectTargetIdx >= 0 {
		promptIcon = '\u2699' // ⚙ gear for inspect mode
		promptIconStyle = tcell.StyleDefault.Foreground(HintFg).Background(promptBg)
	} else if ctx.NavMode {
		promptIcon = '\uF0A9' //
		promptIconStyle = tcell.StyleDefault.Foreground(NavModeFg).Background(promptBg)
	} else {
		promptIcon = '\uF002' //
		promptIconStyle = tcell.StyleDefault.Foreground(SearchModeFg).Bold(true).Background(promptBg)
	}
	promptLen := 2 // icon + space

	if !cfg.HidePromptBar {
		// Top border of prompt bar
		c.SetContent(borderOffset, y, '\u250c', nil, borderStyle) // +
		for x := borderOffset + 1; x < w-borderOffset-1; x++ {
			c.SetContent(x, y, '\u2500', nil, borderStyle) // -
		}
		c.SetContent(w-borderOffset-1, y, '\u2510', nil, borderStyle) // +
		y++

		// Prompt content line with background
		c.SetContent(borderOffset, y, '\u2502', nil, borderStyle) // |
		for x := borderOffset + 1; x < w-borderOffset-1; x++ {
			c.SetContent(x, y, ' ', nil, tcell.StyleDefault.Background(promptBg))
		}
		c.SetContent(w-borderOffset-1, y, '\u2502', nil, borderStyle) // |

		px := borderOffset + 1       // content starts inside the border
		pw := w - borderOffset*2 - 2 // content width inside borders
		// Prompt: [icon] [locked breadcrumb >] [query or nav preview]
		c.SetContent(px, y, promptIcon, nil, promptIconStyle)
		c.SetContent(px+1, y, ' ', nil, tcell.StyleDefault.Background(promptBg))
		tx := px + promptLen // text position after icon + space

		// Context breadcrumb -- ':' when in a pushed context (command mode)
		scopeLen := 0
		if len(s.Contexts) > 1 && ctx.PromptIcon != 0 {
			lockedStyle := tcell.StyleDefault.Foreground(PathFg).Background(promptBg)
			c.SetContent(tx+scopeLen, y, ctx.PromptIcon, nil, lockedStyle)
			scopeLen++
			c.SetContent(tx+scopeLen, y, ' ', nil, tcell.StyleDefault.Background(promptBg))
			scopeLen++
		}

		// Scope breadcrumb -- scoped view shows the folder's full path; otherwise
		// just the word greyed out with a space after it.
		if cfg.ScopedView && len(ctx.Scope) > 1 {
			lockedStyle := tcell.StyleDefault.Foreground(PathFg).Background(promptBg)
			path := scopedPath(ctx, (pw-promptLen-scopeLen)/2)
			drawText(c, tx+scopeLen, y, path, lockedStyle, pw-promptLen-scopeLen)
			scopeLen += len([]rune(path)) + 1
		} else if len(ctx.Scope) > 1 {
			lockedStyle := tcell.StyleDefault.Foreground(PathFg).Background(promptBg)
			for si := 1; si < len(ctx.Scope); si++ {
				level := ctx.Scope[si]
				if level.ParentIdx >= 0 && level.ParentIdx < len(ctx.AllItems) {
					name := ctx.AllItems[level.ParentIdx].Fields[0]
					drawText(c, tx+scopeLen, y, name, lockedStyle, pw-promptLen-scopeLen)
					scopeLen += len([]rune(name))
					c.SetContent(tx+scopeLen, y, ' ', nil, lockedStyle)
					scopeLen++
				}
			}
		}

		qx := tx + scopeLen // where editable query starts

		contentX := qx // where query or nav preview starts
		contentW := pw - promptLen - scopeLen

		if s.PromptMode != "" {
			queryStyle := tcell.StyleDefault.Foreground(QueryFg).Background(promptBg)
			if len(s.PromptQuery) > 0 {
				drawText(c, contentX, y, string(s.PromptQuery), queryStyle, contentW)
			} else {
				hint := s.PromptPlaceholder
				if hint == "" {
					hint = "search\u2026"
				}
				hintStyle := tcell.StyleDefault.Foreground(HintFg).Italic(true).Background(promptBg)
				drawText(c, contentX, y, hint, hintStyle, contentW)
			}
			c.ShowCursor(contentX+s.PromptCursor, y)
		} else if s.EditMode == "rename" {
			// Rename mode — show edit buffer as editable text
			renameStyle := tcell.StyleDefault.Foreground(QueryFg).Background(promptBg)
			drawText(c, contentX, y, string(s.EditBuffer), renameStyle, contentW)
			c.ShowCursor(contentX+len(s.EditBuffer), y)
		} else if hasQuery {
			queryStyle := tcell.StyleDefault.Foreground(QueryFg).Background(promptBg)
			drawText(c, contentX, y, string(ctx.Query), queryStyle, contentW)
			c.ShowCursor(contentX+ctx.Cursor, y)

			if core.TagSearchActive(ctx) {
				drawTagPrompt(c, ctx, contentX, y, contentW, promptBg)
			} else if ctx.Cursor == len(ctx.Query) && len(ctx.Filtered) > 0 && len(ctx.Filtered[0].Fields) > 0 {
				name := ctx.Filtered[0].Fields[0]
				nameRunes := []rune(name)
				if len(nameRunes) > len(ctx.Query) && strings.EqualFold(string(nameRunes[:len(ctx.Query)]), string(ctx.Query)) {
					ghost := string(nameRunes[len(ctx.Query):])
					ghostStyle := tcell.StyleDefault.Foreground(GhostFg).Background(promptBg)
					drawText(c, contentX+len(ctx.Query), y, ghost, ghostStyle, contentW-len(ctx.Query))
				}
			}
		} else if ctx.SearchActive || len(ctx.Scope) > 1 {
			hint := "search\u2026"
			if cfg.ScopedView && cfg.FoldersOnly && len(ctx.Scope) > 1 && ctx.TreeCursor < 0 {
				hint = "Enter picks this folder, Right opens one, or type to search\u2026"
			}
			hintStyle := tcell.StyleDefault.Foreground(HintFg).Italic(true).Background(promptBg)
			drawText(c, qx, y, hint, hintStyle, pw-promptLen-scopeLen)
			c.ShowCursor(qx, y)
		} else {
			hintStyle := tcell.StyleDefault.Foreground(HintFg).Italic(true).Background(promptBg)
			drawText(c, qx, y, "type to search\u2026", hintStyle, pw-promptLen-scopeLen)
			c.ShowCursor(qx, y)
		}
		y++

		// Bottom border of prompt bar
		c.SetContent(borderOffset, y, '\u2514', nil, borderStyle) // +
		for x := borderOffset + 1; x < w-borderOffset-1; x++ {
			c.SetContent(x, y, '\u2500', nil, borderStyle) // -
		}
		c.SetContent(w-borderOffset-1, y, '\u2518', nil, borderStyle) // +
		y++
	}

	// Headers
	if len(ctx.Headers) > 0 {
		hdrStyle := tcell.StyleDefault.Foreground(HeaderFg).Bold(true)
		x := borderOffset + 5 // match tree row layout: 2 selection + 2 icon + 1 buffer
		for fi, hdr := range ctx.Headers[0].Fields {
			colW := effectiveNameCol
			if fi > 0 {
				colW = 0
			}
			if fi == 1 && core.TagSearchActive(ctx) {
				hdr = "tags" // the column shows tags during a tag search
			}
			drawText(c, x, y, hdr, hdrStyle, w-x-borderOffset)
			x += colW
		}
		y++

		divStyle := tcell.StyleDefault.Foreground(SeparatorFg)
		for x := borderOffset + 1; x < w-borderOffset; x++ {
			c.SetContent(x, y, '\u2500', nil, divStyle)
		}
		y++
	}

	// Tree section -- the single navigation surface
	totalSpace := h - (y - startY) - borderOffset
	treeSpace := totalSpace

	// When query active, find top match in tree for highlighting
	topMatchIdx := -1
	if hasQuery && len(ctx.Filtered) > 0 {
		topMatchIdx = core.FindInAll(ctx.AllItems, ctx.Filtered[0])
	}

	// A row a search or an opened folder picked sits revealMargin rows down,
	// with what follows it below, even when that leaves the end of the list
	// empty; a list that fits is not scrolled.
	if ctx.TreeReveal && ctx.TreeCursor >= 0 {
		ctx.TreeOffset = 0
		if len(visible) > treeSpace {
			ctx.TreeOffset = max(ctx.TreeCursor-revealMargin, 0)
		}
	}
	ctx.TreeReveal = false

	// Scroll tree to keep cursor visible
	if ctx.TreeCursor >= 0 {
		if ctx.TreeCursor < ctx.TreeOffset {
			ctx.TreeOffset = ctx.TreeCursor
		}
		if ctx.TreeCursor >= ctx.TreeOffset+treeSpace {
			ctx.TreeOffset = ctx.TreeCursor - treeSpace + 1
		}
	}
	if ctx.TreeOffset < 0 {
		ctx.TreeOffset = 0
	}

	for i := 0; i < treeSpace; i++ {
		vi := ctx.TreeOffset + i
		if vi >= len(visible) {
			break
		}
		row := visible[vi]
		isSelected := vi == ctx.TreeCursor
		isTopMatch := hasQuery && row.ItemIdx == topMatchIdx && !isSelected
		drawTreeRow(c, row, isSelected, isTopMatch, ctx, cfg, borderOffset, y+i, w, effectiveNameCol)
		if s.Overruns[row.Item.Path] {
			// Its last run printed more than the panel holds.
			c.SetContent(w-borderOffset-2, y+i, '!', nil, tcell.StyleDefault.Foreground(TitleErrorFg).Bold(true))
		}
	}

	if cfg.Border {
		drawBorderBottom(c, w, startY+h-1)
		drawBorderSides(c, w, startY, startY+h-1)
	}
}

// drawTreeRow renders a single tree item row.
func drawTreeRow(c render.Canvas, row core.TreeRow, isSelected, isTopMatch bool, ctx *core.TreeContext, cfg Config, borderOffset, y, w, effectiveNameCol int) {
	// Fill background
	if isSelected || isTopMatch {
		bg := tcell.StyleDefault.Background(SelectionBg)
		for x := borderOffset; x < w-borderOffset; x++ {
			c.SetContent(x, y, ' ', nil, bg)
		}
	}

	x := borderOffset
	hasBg := isSelected || isTopMatch

	// Selection indicator
	if isSelected {
		indStyle := tcell.StyleDefault.Foreground(FolderIconFg).Bold(true).Background(SelectionBg)
		drawText(c, x, y, "\uf054 ", indStyle, 2)
	} else {
		style := tcell.StyleDefault
		if hasBg {
			style = style.Background(SelectionBg)
		}
		drawText(c, x, y, "  ", style, 2)
	}
	x += 2

	// Indentation
	indent := rowIndent(row, cfg)
	for i := 0; i < indent; i++ {
		style := tcell.StyleDefault
		if hasBg {
			style = style.Background(SelectionBg)
		}
		c.SetContent(x+i, y, ' ', nil, style)
	}
	x += indent

	// Icon
	var iconRune rune
	var iconStyle tcell.Style
	if row.Item.IsProperty {
		iconRune = '\u2699' // ⚙ gear icon for property items
		iconStyle = tcell.StyleDefault.Foreground(HintFg)
	} else if row.Item.HasChildren || row.Item.IsDir {
		iconRune = '\U000F024B'
		iconStyle = tcell.StyleDefault.Foreground(FolderIconFg).Bold(true)
	} else {
		iconRune = '\uF15B'
		iconStyle = tcell.StyleDefault.Foreground(FileIconFg)
	}
	if hasBg {
		iconStyle = iconStyle.Background(SelectionBg)
	}
	c.SetContent(x, y, iconRune, nil, iconStyle)
	x += 2 // wide icon occupies 2 cells
	bufStyle := tcell.StyleDefault
	if hasBg {
		bufStyle = bufStyle.Background(SelectionBg)
	}
	c.SetContent(x, y, ' ', nil, bufStyle)
	x++

	// Name
	name := ""
	if len(row.Item.Fields) > 0 {
		name = row.Item.Fields[0]
	}
	var nameStyle tcell.Style
	if row.Item.HasChildren {
		nameStyle = tcell.StyleDefault.Foreground(FolderNameFg).Bold(true)
	} else if isSelected {
		nameStyle = tcell.StyleDefault.Foreground(QueryFg)
	} else {
		nameStyle = tcell.StyleDefault
	}
	if hasBg {
		nameStyle = nameStyle.Background(SelectionBg)
	}

	nameWidth := effectiveNameCol - indent
	nameRunes := []rune(name)
	if nameWidth < len(nameRunes)+1 {
		nameWidth = len(nameRunes) + 1
	}

	// Highlight matched characters for top match
	if isTopMatch && len(row.Item.MatchIndices) > 0 && len(row.Item.MatchIndices[0]) > 0 {
		drawHighlightedText(c, x, y, name, nameStyle, nameWidth, row.Item.MatchIndices[0], hasBg)
	} else {
		drawText(c, x, y, name, nameStyle, nameWidth)
	}
	x += nameWidth

	// Description, or the row's tags during a tag search
	if core.TagSearchActive(ctx) {
		if remaining := w - x - borderOffset; remaining > 0 {
			drawTags(c, x, y, row.Item, ctx.TagTerms, hasBg, remaining)
		}
	} else if len(row.Item.Fields) > 1 {
		desc := row.Item.Fields[1]
		descStyle := tcell.StyleDefault
		if hasBg {
			descStyle = descStyle.Background(SelectionBg)
		}
		remaining := w - x - borderOffset
		if remaining > 0 {
			drawText(c, x, y, desc, descStyle, remaining)
		}
	}
}

// drawHighlightedText draws text with certain character indices highlighted in green.
func drawHighlightedText(c render.Canvas, x, y int, text string, baseStyle tcell.Style, maxW int, matchIndices []int, hasBg bool) {
	runes := []rune(text)
	matchSet := make(map[int]bool, len(matchIndices))
	for _, idx := range matchIndices {
		matchSet[idx] = true
	}

	for i, r := range runes {
		if i >= maxW {
			break
		}
		style := baseStyle
		if matchSet[i] {
			style = tcell.StyleDefault.Foreground(HighlightFg).Bold(true)
			if hasBg {
				style = style.Background(SelectionBg)
			}
		}
		c.SetContent(x+i, y, r, nil, style)
	}
}

// rowIndent is a row's indentation in cells. The scoped view lists one
// folder's children, so nothing is indented.
func rowIndent(row core.TreeRow, cfg Config) int {
	if cfg.ScopedView {
		return 0
	}
	return row.Item.Depth * 2
}

// scopedPath is the current scope's full path with a trailing separator,
// cut from the left to fit maxW.
func scopedPath(ctx *core.TreeContext, maxW int) string {
	path := core.ItemFullPath(ctx, ctx.Scope[len(ctx.Scope)-1].ParentIdx)
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path += string(filepath.Separator)
	}
	runes := []rune(path)
	if maxW > 1 && len(runes) > maxW {
		runes = append([]rune{'…'}, runes[len(runes)-maxW+1:]...)
	}
	return string(runes)
}

// drawPanel draws the latest command result as a box of core.PanelRows rows
// at y: a border colored by how the command ended, around core.PanelLines
// lines. Empty lines stay blank so the layout never moves.
func drawPanel(c render.Canvas, p core.Panel, borderOffset, y, w int) {
	fg := TitleNeutralFg
	switch p.Style {
	case core.PanelOK:
		fg = TitleSuccessFg
	case core.PanelFailed:
		fg = TitleErrorFg
	}
	style := tcell.StyleDefault.Foreground(fg)
	edge := tcell.StyleDefault.Foreground(BorderFg)
	if len(p.Lines) > 0 {
		edge = style
	}
	left, right := borderOffset, w-borderOffset-1
	bottom := y + core.PanelRows - 1
	c.SetContent(left, y, '┌', nil, edge)
	c.SetContent(right, y, '┐', nil, edge)
	c.SetContent(left, bottom, '└', nil, edge)
	c.SetContent(right, bottom, '┘', nil, edge)
	for x := left + 1; x < right; x++ {
		c.SetContent(x, y, '─', nil, edge)
		c.SetContent(x, bottom, '─', nil, edge)
	}
	for row := y + 1; row < bottom; row++ {
		c.SetContent(left, row, '│', nil, edge)
		c.SetContent(right, row, '│', nil, edge)
	}
	drawText(c, left+2, y, " command output ", tcell.StyleDefault.Foreground(BorderFg), right-left-3)

	x := left + 2 // a space of padding inside the left border
	width := right - x - 1
	if len(p.Lines) == 0 {
		drawText(c, x, y+1, "nothing run yet", tcell.StyleDefault.Foreground(panelHintFg).Italic(true), width)
		return
	}
	for i := 0; i < core.PanelLines; i++ {
		if i >= len(p.Lines) {
			continue
		}
		line := []rune(p.Lines[i])
		if p.Clipped && i == core.PanelLines-1 {
			// The ellipsis always shows, even on a line cut at the edge.
			if len(line) > width-2 {
				line = line[:max(width-2, 0)]
			}
			line = append(line, ' ', '…')
		}
		for j, r := range line {
			if j >= width {
				break
			}
			c.SetContent(x+j, y+1+i, r, nil, style)
		}
	}
}

// panelHintFg is the empty panel's placeholder: dimmer than its border and label.
var panelHintFg = tcell.NewRGBColor(0x55, 0x55, 0x55)

// drawTagPrompt follows a tag-search query with the rest of the tag its last
// term resolved to (the ghost Tab accepts) and, past that, the terms that
// matched no tag.
func drawTagPrompt(c render.Canvas, ctx *core.TreeContext, x, y, w int, bg tcell.Color) {
	used := len(ctx.Query)
	if ctx.Cursor == len(ctx.Query) {
		if ghost := core.TagGhost(ctx); ghost != "" {
			drawText(c, x+used, y, ghost, tcell.StyleDefault.Foreground(GhostFg).Background(bg), w-used)
			used += len([]rune(ghost))
		}
	}
	var missing []string
	for _, t := range ctx.TagTerms {
		if t.Tag == "" {
			missing = append(missing, t.Term)
		}
	}
	if len(missing) > 0 && w-used > 2 {
		note := "  no tag: " + strings.Join(missing, ", ")
		drawText(c, x+used, y, note, tcell.StyleDefault.Foreground(TitleErrorFg).Background(bg), w-used)
	}
}

// drawTags draws an item's tags in the description column, the ones the
// search resolved to highlighted.
func drawTags(c render.Canvas, x, y int, item core.Item, terms []core.TagTerm, hasBg bool, maxW int) {
	matched := map[string]bool{}
	for _, t := range terms {
		matched[t.Tag] = true
	}
	end := x + maxW
	for i, tag := range item.Tags {
		style := tcell.StyleDefault.Foreground(GhostFg)
		if matched[tag] {
			style = tcell.StyleDefault.Foreground(HighlightFg).Bold(true)
		}
		if hasBg {
			style = style.Background(SelectionBg)
		}
		if i > 0 {
			x++
		}
		if x >= end {
			return
		}
		drawText(c, x, y, tag, style, end-x)
		x += len([]rune(tag))
	}
}
