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
	"regexp"
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
func linksIn(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var targets []string
	for _, match := range linkPattern.FindAllStringSubmatch(string(data), -1) {
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
	return targets, nil
}

// linkResolves reports whether target, interpreted relative to the
// directory of source, points at an existing file.
func linkResolves(source, target string) bool {
	dir := filepath.Dir(source)
	resolved := filepath.Clean(filepath.Join(dir, target))
	_, err := os.Stat(resolved)
	return err == nil
}
