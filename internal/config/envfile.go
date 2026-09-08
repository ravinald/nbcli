package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// envfile.go reads simple KEY=value env files. Format:
//
//	# comment line (ignored)
//	KEY=value
//	KEY="quoted value"
//	KEY='single quotes'
//	export KEY=value     # leading "export " stripped for shell-script compat
//	KEY=trailing # comment   # stripped when preceded by whitespace
//
// Intentionally not supported:
//
//   - shell variable expansion ($VAR, $(cmd), backticks)
//   - multi-line values
//   - backslash escapes
//
// Skipping those keeps the parser predictable and safe for credential files
// — there's no surprise behavior between what `cat` shows and what nbcli sees.

// ErrEnvFileFormat is returned by ParseEnvFile when a line can't be parsed.
// Wrap-checkable with errors.Is.
var ErrEnvFileFormat = errors.New("envfile: bad line format")

// LoadEnvFile reads path and returns the parsed key/value map. A non-existent
// file is intentionally NOT an error — returns (nil, nil) so callers can
// probe candidate paths (`~/.config/nbcli/secrets.env`, `~/.env.netbox`)
// without checking os.IsNotExist themselves.
func LoadEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from config search or --env-file
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("envfile: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return ParseEnvFile(f)
}

// ParseEnvFile parses an env-file stream into a map. Returns the parse error
// at the first malformed line; any keys read before the error are still in
// the returned map so callers can inspect them for diagnostics.
func ParseEnvFile(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	sc := bufio.NewScanner(r)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip optional "export " prefix so shell-style env files work.
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return out, fmt.Errorf("%w: line %d: missing '='", ErrEnvFileFormat, lineNum)
		}
		key := strings.TrimSpace(line[:eq])
		if !validEnvKey(key) {
			return out, fmt.Errorf("%w: line %d: invalid key %q", ErrEnvFileFormat, lineNum, key)
		}
		val := strings.TrimSpace(line[eq+1:])
		// Trailing "  # comment" only stripped when value isn't quoted (so
		// values containing '#' — URL fragments, password specials — survive).
		if !isQuoted(val) {
			if i := indexUnquotedComment(val); i >= 0 {
				val = strings.TrimSpace(val[:i])
			}
		}
		val = stripQuotes(val)
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return out, fmt.Errorf("envfile: read: %w", err)
	}
	return out, nil
}

// validEnvKey enforces the POSIX-ish env-var name rule: leading letter or
// underscore, then letters/digits/underscores.
func validEnvKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		if i == 0 && (unicode.IsDigit(r) || r == ' ') {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func isQuoted(s string) bool {
	if len(s) < 2 {
		return false
	}
	return (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')
}

func stripQuotes(s string) string {
	if isQuoted(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// indexUnquotedComment returns the index of the first '#' character that is
// preceded by whitespace. Caller already checked the value isn't fully
// quoted, so this is "best-effort trailing comment stripping" rather than a
// full inline-quote-aware parser.
func indexUnquotedComment(s string) int {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}
	return -1
}
