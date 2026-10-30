// fzt — interactive fuzzy picker over stdin or a YAML tree.
//
// Restores the fzf-style CLI that lived in the engine repo until the
// 2026-04-07 engine/terminal split. Shell scripts pipe a list in and read the
// selection back:
//
//	$entries | fzt --layout reverse --border --header "Remove which repo?"
//	fzt --yaml menu.yaml --title "What would you like to do?"
//	ls | fzt --filter "query"          # non-interactive: ranked matches
//	$rows | fzt --input-col 3          # rows with a 3rd field ask for text
//
// Exit codes follow fzf: 0 selected, 1 no input, 130 cancelled.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/romaine-life/fzt-terminal/tui"
	"github.com/romaine-life/fzt/core"
	"github.com/romaine-life/fzt/render"
)

func main() {
	var (
		layout       = flag.String("layout", "default", "Layout: 'default' (prompt at bottom) or 'reverse' (prompt at top)")
		border       = flag.Bool("border", false, "Draw border around the finder")
		headerLines  = flag.Int("header-lines", 0, "Number of leading input lines pinned as a header (not filtered)")
		header       = flag.String("header", "", "Header line (delimiter-separated for columns)")
		nth          = flag.String("nth", "", "Comma-separated field indices to search (1-based)")
		acceptNth    = flag.String("accept-nth", "", "Comma-separated field indices to output (1-based)")
		prompt       = flag.String("prompt", "> ", "Prompt string")
		delimiter    = flag.String("delimiter", "\t", "Field delimiter")
		tiered       = flag.Bool("tiered", false, "Input is a tree: the first column is depth, and a row holds the deeper rows after it (Enter opens it, Esc or Backspace goes back)")
		depthPenalty = flag.Int("depth-penalty", 5, "Score penalty per depth level (with --tiered)")
		searchCols   = flag.String("search-cols", "", "Comma-separated field indices qualifying for the name tier (overrides --nth)")
		searchDepth  = flag.Int("search-depth", 0, "Limit search to N levels below the current scope (0 = unlimited)")
		filter       = flag.String("filter", "", "Non-interactive: print matches for this query, best first, and exit")
		showScores   = flag.Bool("show-scores", false, "Annotate --filter output with scores")
		ansi         = flag.Bool("ansi", false, "Parse and preserve ANSI color codes from input")
		yamlPath     = flag.String("yaml", "", "Load a hierarchical tree from a YAML file (implies tree mode)")
		title        = flag.String("title", "", "Title shown on the border")
		titlePos     = flag.String("title-pos", "left", "Title position: 'left', 'center' or 'right'")
		inputCol     = flag.Int("input-col", 0, "Field index (1-based) that makes a row ask for text: a row with this field set prompts for text on Enter, the field as its hint, and prints the selection, a tab and the text")
		version      = flag.Bool("version", false, "Print version and exit")
	)
	flag.StringVar(delimiter, "d", "\t", "Field delimiter (shorthand)")
	// Accepted for fzf compatibility; the picker always takes the full screen.
	flag.String("height", "", "Ignored (fzf compatibility)")
	flag.Parse()

	if *version {
		fmt.Println(render.Version)
		return
	}

	cfg := tui.Config{
		Layout:       *layout,
		Border:       *border,
		HeaderLines:  *headerLines,
		Nth:          parseIntList(*nth),
		AcceptNth:    parseIntList(*acceptNth),
		Prompt:       *prompt,
		Delimiter:    *delimiter,
		Tiered:       *tiered,
		DepthPenalty: *depthPenalty,
		SearchCols:   parseIntList(*searchCols),
		SearchDepth:  *searchDepth,
		ShowScores:   *showScores,
		ANSI:         *ansi,
		Title:        *title,
		TitlePos:     *titlePos,
		// A plain picker has no commands to offer; keep the `:` palette out.
		HidePalette: true,
		EnvTags:     []string{"terminal"},
	}

	var items []core.Item
	if *yamlPath != "" {
		var err error
		items, err = core.LoadYAML(*yamlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fzt: loading yaml: %v\n", err)
			os.Exit(2)
		}
		cfg.Tiered = true
		cfg.TreeMode = true
		if *header != "" {
			headerItem := core.Item{Fields: strings.Split(*header, *delimiter), Depth: -1}
			items = append([]core.Item{headerItem}, items...)
			if cfg.HeaderLines == 0 {
				cfg.HeaderLines = 1
			}
		}
	} else {
		lines, err := readStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fzt: %v\n", err)
			os.Exit(2)
		}
		if len(lines) == 0 {
			os.Exit(1)
		}
		items = stdinItems(lines, &cfg, *header, *inputCol)
	}

	if *filter != "" {
		data := items
		if cfg.HeaderLines > 0 && cfg.HeaderLines <= len(items) {
			data = items[cfg.HeaderLines:]
		}
		tui.RunFilter(data, *filter, cfg)
		return
	}

	pick, err := tui.Run(items, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fzt: %v\n", err)
		os.Exit(2)
	}
	if pick.Kind == "cancel" {
		os.Exit(130)
	}
	fmt.Println(pick.Output)
}

func readStdin() ([]string, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf("no input: pipe a list in or pass --yaml")
	}

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if line := strings.TrimRight(scanner.Text(), "\r"); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// isDir reports whether a line names a directory, relative to the working
// directory. Surrounding quotes are stripped: PowerShell quotes completions
// that contain spaces ('.\My Folder').
func isDir(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return false
	}
	info, err := os.Stat(s)
	return err == nil && info.IsDir()
}

func parseIntList(s string) []int {
	if s == "" {
		return nil
	}
	var result []int
	for _, p := range strings.Split(s, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			result = append(result, n)
		}
	}
	return result
}

// markInput makes a row an input leaf when its col-th field (1-based) is set:
// picking it asks for text, hinted by that field. The renderer draws only the
// first two fields, so the hint stays out of the list.
func markInput(item *core.Item, col int) {
	if col < 1 || len(item.Fields) < col || item.Fields[col-1] == "" {
		return
	}
	item.Action = &core.ItemAction{Type: "input", Target: item.Fields[col-1]}
}

// linkTree makes depth-prefixed rows a tree, as LoadYAML does for a YAML menu:
// a row's parent is the nearest earlier row one level shallower.
func linkTree(rows []core.Item) {
	var stack []int // indices into rows, shallowest first
	for i := range rows {
		for len(stack) > 0 && rows[stack[len(stack)-1]].Depth >= rows[i].Depth {
			stack = stack[:len(stack)-1]
		}
		rows[i].ParentIdx = -1
		rows[i].Path = rows[i].Fields[0]
		if len(stack) > 0 {
			p := stack[len(stack)-1]
			rows[i].ParentIdx = p
			rows[i].Path = rows[p].Path + " › " + rows[i].Fields[0]
			rows[p].Children = append(rows[p].Children, i)
			rows[p].HasChildren = true
		}
		stack = append(stack, i)
	}
}

// stdinItems builds the picker's items from piped lines: header first, then the
// rows, linked into a tree when cfg.Tiered is set.
func stdinItems(lines []string, cfg *tui.Config, header string, inputCol int) []core.Item {
	tiered := cfg.Tiered
	if header != "" {
		h := header
		if tiered {
			// the header has no depth column of its own
			h = "0" + cfg.Delimiter + h
		}
		lines = append([]string{h}, lines...)
		if cfg.HeaderLines == 0 {
			cfg.HeaderLines = 1
		}
	}
	items := core.ParseLines(lines, cfg.Delimiter, tiered, cfg.ANSI)
	// Render lists with the tree renderer fzt-automate uses (search box
	// panel, icon column), so every fzt picker looks the same.
	cfg.Tiered = true
	cfg.TreeMode = true
	start := cfg.HeaderLines
	if start > len(items) {
		start = len(items)
	}
	if header != "" {
		items[0].Depth = -1
	}
	if tiered {
		// The state sets header lines apart, so a parent index counts from the
		// first row after them.
		linkTree(items[start:])
	}
	for i := start; i < len(items); i++ {
		items[i].IsDir = isDir(items[i].Fields[0])
		if !items[i].HasChildren {
			markInput(&items[i], inputCol)
		}
	}
	return items
}
