// Package pager is a less-like interactive pager for nbcli's CLI table
// output. Each `show <resource>` command can opt in with --interactive,
// passing a Fetcher closure that loads one page of rows. The pager owns:
//
//   - Terminal control (raw mode for single-key input, ANSI screen clear)
//   - The pagination state machine (offset / limit / total)
//   - The committed search query (sent to Netbox as `?q=` via FetchOpts.Query)
//   - The key map: n/space/enter (next), p/backspace (prev), </> (first/last),
//     g (goto page), / (search), ? (help), q/Esc (quit)
//
// Resource-specific concerns (which Netbox endpoint, which columns, which
// positional filters) live in the caller's fetcher closure. The pager
// itself is generic over row type via `any` — output.Renderer already takes
// `any` for the rows slice, so no further plumbing is needed.
package pager

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/ravinald/nbcli/internal/output"
)

// FetchOpts is what the pager hands to the caller's fetcher each iteration.
// Caller merges these with its own positional filters before hitting Netbox.
type FetchOpts struct {
	Offset int
	Limit  int
	Query  string // free-text search, "" when no search committed
}

// FetchResult is what the caller's fetcher returns. Rows is the typed slice
// from the API (e.g. []netbox.Site); the pager hands it back to output.Renderer
// which type-asserts inside its Extract closures.
type FetchResult struct {
	Rows  any
	Total int // total matching across all pages (from Netbox Page.Count)
}

// Fetcher is the closure each show command supplies.
type Fetcher func(ctx context.Context, opts FetchOpts) (FetchResult, error)

// Config parameterizes one Run invocation.
type Config struct {
	Title    string          // e.g. "Sites" — shown at the top of each page
	PageSize int             // rows per page; 0 = compute from terminal height
	Columns  []output.Column // column set passed to output.Renderer
	Out      *os.File        // must be a TTY (typically os.Stdout)
	In       *os.File        // must be a TTY (typically os.Stdin)
}

// ANSI escape sequences. Each is a small set so we don't pull a deps for them.
const (
	escClear = "\x1b[H\x1b[2J" // cursor to (1,1) + erase screen
	escHide  = "\x1b[?25l"     // hide cursor
	escShow  = "\x1b[?25h"     // show cursor
)

// Run drives the interactive loop until the user quits with q/Esc/Ctrl+C.
// Returns nil on clean exit, error on terminal-control failure or a Read
// error from stdin. Fetcher errors are shown as a notice on the next render
// and don't abort the session.
func Run(cfg Config, fetch Fetcher) error {
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.In == nil {
		cfg.In = os.Stdin
	}
	outFd := int(cfg.Out.Fd())
	inFd := int(cfg.In.Fd())
	if !term.IsTerminal(outFd) || !term.IsTerminal(inFd) {
		return errors.New("pager: --interactive requires a TTY on stdin and stdout")
	}
	if cfg.PageSize <= 0 {
		_, h, err := term.GetSize(outFd)
		if err == nil {
			// Reserve: 1 title + 1 blank + 2 table header + 1 prompt + 1 safety.
			cfg.PageSize = h - 6
		}
		if cfg.PageSize < 5 {
			cfg.PageSize = 5
		}
	}

	renderer, err := output.New(output.FormatTable)
	if err != nil {
		return err
	}

	state, err := term.MakeRaw(inFd)
	if err != nil {
		return fmt.Errorf("pager: raw mode: %w", err)
	}
	defer func() {
		_ = term.Restore(inFd, state)
		_, _ = fmt.Fprint(cfg.Out, escShow)
	}()
	_, _ = fmt.Fprint(cfg.Out, escHide)

	offset, query, notice := 0, "", ""

	for {
		result, ferr := fetch(context.Background(), FetchOpts{
			Offset: offset, Limit: cfg.PageSize, Query: query,
		})
		if ferr != nil {
			notice = "error: " + ferr.Error()
		}

		render(cfg, renderer, result, offset, query, notice)
		notice = ""

		key, kerr := readKey(cfg.In)
		if kerr != nil {
			return kerr
		}
		switch key {
		case 'n', ' ', '\r', '\n':
			if offset+cfg.PageSize < result.Total {
				offset += cfg.PageSize
			}
		case 'p', 0x7f: // 0x7f == DEL/backspace on most terminals
			offset -= cfg.PageSize
			if offset < 0 {
				offset = 0
			}
		case '<':
			offset = 0
		case '>':
			if result.Total > 0 {
				offset = ((result.Total - 1) / cfg.PageSize) * cfg.PageSize
			}
		case 'g':
			line, err := readLine(cfg, inFd, state, "Page (1-based): ")
			if err == nil && strings.TrimSpace(line) != "" {
				if p, perr := strconv.Atoi(strings.TrimSpace(line)); perr == nil && p > 0 {
					offset = (p - 1) * cfg.PageSize
					if result.Total > 0 && offset > result.Total-1 {
						offset = ((result.Total - 1) / cfg.PageSize) * cfg.PageSize
					}
				} else {
					notice = "not a number: " + line
				}
			}
		case '/':
			line, err := readLine(cfg, inFd, state, "Search: ")
			if err == nil {
				query = strings.TrimSpace(line)
				offset = 0
			}
		case 'c':
			// 'c'lear search; useful shortcut.
			query = ""
			offset = 0
		case '?':
			notice = "n/space next · p/⌫ prev · </> first/last · g goto · / search · c clear · q quit"
		case 'q', 'Q', 0x1b, 0x03: // Esc or Ctrl+C
			return nil
		}
	}
}

// render clears the screen and draws title + table + prompt for the current page.
func render(cfg Config, renderer output.Renderer, result FetchResult, offset int, query, notice string) {
	_, _ = fmt.Fprint(cfg.Out, escClear)
	_, _ = fmt.Fprintf(cfg.Out, "%s\r\n\r\n", cfg.Title)
	if result.Rows != nil {
		var buf strings.Builder
		_ = renderer.Render(&buf, cfg.Columns, result.Rows)
		// Raw mode discards \n→\r\n translation, so emit explicit \r\n per line.
		for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
			_, _ = fmt.Fprintf(cfg.Out, "%s\r\n", line)
		}
	}
	_, _ = fmt.Fprint(cfg.Out, "\r\n", promptLine(cfg.PageSize, offset, result.Total, query, notice))
}

// promptLine formats the bottom-of-screen status + key hints.
func promptLine(pageSize, offset, total int, query, notice string) string {
	page := 1
	totalPages := 1
	if pageSize > 0 && total > 0 {
		page = offset/pageSize + 1
		totalPages = (total + pageSize - 1) / pageSize
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	info := fmt.Sprintf("page %d/%d · rows %d-%d of %d", page, totalPages, offset+1, end, total)
	if query != "" {
		info = "search " + strconv.Quote(query) + " · " + info
	}
	if notice != "" {
		return "--more-- " + notice + "\r\n         " + info + " · n/p/g//? · q quit"
	}
	return "--more-- " + info + " · n=next p=prev g=goto /=search ?=help q=quit"
}

// readKey reads a single byte from in. Returns io.EOF if the terminal closes.
func readKey(in io.Reader) (byte, error) {
	buf := make([]byte, 1)
	n, err := in.Read(buf)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return buf[0], nil
}

// readLine temporarily drops out of raw mode so the user can type a value
// with normal line-editing (backspace, etc.), then restores raw mode.
func readLine(cfg Config, inFd int, state *term.State, prompt string) (string, error) {
	_ = term.Restore(inFd, state)
	_, _ = fmt.Fprint(cfg.Out, escShow)
	defer func() {
		_, _ = term.MakeRaw(inFd)
		_, _ = fmt.Fprint(cfg.Out, escHide)
	}()
	_, _ = fmt.Fprintf(cfg.Out, "\r\n%s", prompt)
	scanner := bufio.NewScanner(cfg.In)
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	return scanner.Text(), nil
}
