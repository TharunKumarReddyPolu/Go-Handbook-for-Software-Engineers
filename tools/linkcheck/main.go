// Command linkcheck validates internal, relative Markdown links in the
// handbook. It is intentionally dependency-free and deliberately does not
// check external http(s) links: external link rot is better handled by a
// scheduled job, not the PR gate.
//
// Usage:
//
//	go run ./tools/linkcheck [directory ...]
//
// With no arguments it walks the repository root, skipping hidden
// directories, the Go module cache, and node_modules.
package main

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// linkPattern matches inline Markdown links [text](target).
// Reference-style links are rare in this handbook; add support if that
// changes.
var linkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

func main() {
	roots := os.Args[1:]
	if len(roots) == 0 {
		roots = []string{"."}
	}

	broken := 0
	checked := 0

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" {
					if path != "." {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}
			targets, err := linksIn(path)
			if err != nil {
				return err
			}
			for _, target := range targets {
				checked++
				if !linkResolves(path, target) {
					broken++
					fmt.Printf("BROKEN %s -> %s\n", path, target)
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "linkcheck: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("linkcheck: %d internal links checked, %d broken\n", checked, broken)
	if broken > 0 {
		os.Exit(1)
	}
}

// linksIn reads a Markdown file and returns every relative link target.
// Lines inside fenced code blocks (``` or ~~~) are skipped, and inline
// code spans are stripped first: Go code legitimately contains [i](h)
// shapes (indexing followed by a call) that the inline-link regex
// would otherwise misread.
func linksIn(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var active []string
	inFence := false
	var fenceMarker string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := strings.Repeat(string(trimmed[0]), 3)
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if strings.HasPrefix(trimmed, fenceMarker) {
				inFence = false
			}
			continue
		}
		if inFence {
			continue
		}
		active = append(active, line)
	}
	// Strip inline code spans before link extraction: `f(x)(y)` or
	// generics prose like `TopK[T any](...)` contain ](...)-shaped text
	// that is code, not a link. Spans may span lines. (Windows made
	// violations invisible for years: os.Stat on a trailing-dot path
	// resolves to the parent directory.)
	targets := []string{}
	for _, line := range stripInlineCode(active) {
		for _, match := range linkPattern.FindAllStringSubmatch(line, -1) {
			raw := match[1]
			// Strip anchors and resolve URL escapes such as %20.
			if idx := strings.IndexByte(raw, '#'); idx >= 0 {
				raw = raw[:idx]
			}
			decoded, err := url.PathUnescape(raw)
			if err != nil {
				decoded = raw
			}
			// Skip empty anchors (#), external links, and mailto links.
			if decoded == "" ||
				strings.HasPrefix(decoded, "http://") ||
				strings.HasPrefix(decoded, "https://") ||
				strings.HasPrefix(decoded, "mailto:") {
				continue
			}
			targets = append(targets, decoded)
		}
	}
	return targets, nil
}

// stripInlineCode removes inline code spans (`...`) from the given
// lines, correctly handling spans that open on one line and close on a
// later one (“ ` “ runs are matched by length, as CommonMark does).
// It returns fresh lines so the input slice is not modified.
func stripInlineCode(lines []string) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	open := 0 // backtick count of a span awaiting its closer; 0 = none
	for i, line := range out {
		var b strings.Builder
		j := 0
		for j < len(line) {
			if line[j] == '`' {
				n := j
				for n < len(line) && line[n] == '`' {
					n++
				}
				run := n - j
				switch {
				case open == 0:
					open = run
					b.Reset()
					b.WriteString(out[i][:j])
				case run == open:
					open = 0
				}
				j = n
				continue
			}
			if open == 0 {
				b.WriteByte(line[j])
			}
			j++
		}
		// Assign even when a span is still open: b holds the stripped
		// prefix (or nothing, when the whole line is inside a span).
		out[i] = b.String()
	}
	return out
}

// linkResolves reports whether target, interpreted relative to the
// directory of source, points at an existing file. Existence is checked
// case-EXACTLY: os.Stat is case-insensitive on Windows and default macOS,
// which would let links like [x](Ring.go) pass locally and fail on the
// case-sensitive CI filesystem. The final path component is verified
// against the parent directory's real entries by exact name.
func linkResolves(source, target string) bool {
	dir := filepath.Dir(source)
	resolved := filepath.Clean(filepath.Join(dir, target))
	parent := filepath.Dir(resolved) // never empty; "." for bare names
	base := filepath.Base(resolved)
	entries, err := os.ReadDir(parent)
	if err != nil {
		// On case-sensitive systems this also catches a case-mismatched
		// ancestor directory; on case-insensitive systems the ancestor
		// opens and the final component is still checked exactly.
		return false
	}
	for _, e := range entries {
		if e.Name() == base {
			return true
		}
	}
	return false
}
