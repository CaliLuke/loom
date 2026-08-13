// Command docscheck validates Loom's maintained Markdown documentation.
package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	markdownLinkPattern = regexp.MustCompile(`!?\[[^]]*\]\(([^)\s]+)(?:\s+[^)]*)?\)`)
	headingPattern      = regexp.MustCompile(`^#{1,6}\s+(.+?)\s*#*\s*$`)
	localCommandPattern = regexp.MustCompile(`\bloom\s+(?:gen|example)\s+\./`)
	legacyNamingPattern = regexp.MustCompile(`(?:github\.com/goadesign/goa|goa\.design/goa|\bgoa\s+(?:gen|example)\b|\bGOA_[A-Z0-9_]+\b)`)
	reasonPattern       = regexp.MustCompile(`\bReason\w+\s+Reason\s*=\s*"([^"]+)"`)
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	documents, err := maintainedDocuments(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	issues := checkMarkdown(root, documents)
	issues = append(issues, checkDuplicateSkillGuides(root)...)
	for _, guide := range []string{".agents/skills/loom/SKILL.md", "docs/production.md"} {
		issues = append(issues, checkObserverReasons(
			root,
			"observability/transport/events.go",
			guide,
		)...)
	}
	if len(issues) == 0 {
		return
	}
	for _, issue := range issues {
		fmt.Fprintln(os.Stderr, "docs lint:", issue)
	}
	os.Exit(1)
}

func maintainedDocuments(root string) ([]string, error) {
	documentSet := map[string]struct{}{
		"AGENTS.md": {}, "CONTRIBUTING.md": {}, "README.md": {},
	}
	for _, directory := range []string{
		".agents/skills/framework-capability",
		".agents/skills/loom",
		".agents/skills/loom-framework",
		"docs",
		"jsonrpc",
	} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			documentSet[filepath.ToSlash(rel)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("discover documentation in %s: %w", directory, err)
		}
	}
	documents := make([]string, 0, len(documentSet))
	for document := range documentSet {
		documents = append(documents, document)
	}
	sort.Strings(documents)
	return documents, nil
}

func checkMarkdown(root string, documents []string) []string {
	anchors := make(map[string]map[string]struct{}, len(documents))
	contents := make(map[string]string, len(documents))
	var issues []string
	for _, document := range documents {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(document)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: read document: %v", document, err))
			continue
		}
		contents[document] = string(content)
		anchors[document] = markdownAnchors(string(content))
	}

	for _, document := range documents {
		content, ok := contents[document]
		if !ok {
			continue
		}
		for _, match := range localCommandPattern.FindAllStringIndex(content, -1) {
			issues = append(issues, fmt.Sprintf(
				"%s:%d: use a Go import path with loom gen/example, not a filesystem path",
				document,
				lineNumber(content, match[0]),
			))
		}
		for _, match := range legacyNamingPattern.FindAllStringIndex(content, -1) {
			issues = append(issues, fmt.Sprintf(
				"%s:%d: legacy upstream naming %q is not allowed in Loom documentation",
				document,
				lineNumber(content, match[0]),
				content[match[0]:match[1]],
			))
		}
		linkContent := maskMarkdownCode(content)
		for _, match := range markdownLinkPattern.FindAllStringSubmatchIndex(linkContent, -1) {
			target := linkContent[match[2]:match[3]]
			if issue := checkLink(root, document, target, anchors); issue != "" {
				issues = append(issues, fmt.Sprintf("%s:%d: %s", document, lineNumber(content, match[0]), issue))
			}
		}
	}
	sort.Strings(issues)
	return issues
}

func checkDuplicateSkillGuides(root string) []string {
	canonical := make(map[string]string)
	docsRoot := filepath.Join(root, "docs")
	entries, err := os.ReadDir(docsRoot)
	if err != nil {
		return []string{fmt.Sprintf("docs: discover canonical guides: %v", err)}
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		canonical[strings.ToLower(entry.Name())] = filepath.ToSlash(filepath.Join("docs", entry.Name()))
	}

	var issues []string
	for _, skill := range []string{"loom", "loom-framework"} {
		skillRoot := filepath.Join(root, ".agents", "skills", skill)
		err = filepath.WalkDir(skillRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			canonicalPath, duplicate := canonical[strings.ToLower(filepath.Base(path))]
			if !duplicate {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			issues = append(issues, fmt.Sprintf(
				"%s: duplicates canonical guide name %s",
				filepath.ToSlash(rel),
				canonicalPath,
			))
			return nil
		})
		if err != nil {
			issues = append(issues, fmt.Sprintf(".agents/skills/%s: discover skill guides: %v", skill, err))
		}
	}
	sort.Strings(issues)
	return issues
}

func maskMarkdownCode(content string) string {
	masked := []byte(content)
	inFence := false
	lineStart := 0
	for lineStart < len(masked) {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(masked)
		} else {
			lineEnd += lineStart
		}
		trimmed := strings.TrimSpace(content[lineStart:lineEnd])
		switch {
		case strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~"):
			inFence = !inFence
			maskBytes(masked[lineStart:lineEnd])
		case inFence:
			maskBytes(masked[lineStart:lineEnd])
		default:
			maskInlineCode(masked[lineStart:lineEnd])
		}
		lineStart = lineEnd + 1
	}
	return string(masked)
}

func maskInlineCode(line []byte) {
	for offset := 0; offset < len(line); {
		start := bytesIndexByte(line[offset:], '`')
		if start < 0 {
			return
		}
		start += offset
		end := bytesIndexByte(line[start+1:], '`')
		if end < 0 {
			return
		}
		end += start + 1
		maskBytes(line[start : end+1])
		offset = end + 1
	}
}

func bytesIndexByte(value []byte, target byte) int {
	for index, candidate := range value {
		if candidate == target {
			return index
		}
	}
	return -1
}

func maskBytes(value []byte) {
	for index := range value {
		value[index] = ' '
	}
}

func checkLink(root, document, rawTarget string, anchors map[string]map[string]struct{}) string {
	target := strings.Trim(rawTarget, "<>")
	if strings.HasPrefix(target, "/Users/") {
		return fmt.Sprintf("machine-local absolute path %q", target)
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Sprintf("invalid link %q: %v", target, err)
	}
	if parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(target, "//") {
		return ""
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return fmt.Sprintf("invalid escaped link path %q: %v", parsed.Path, err)
	}
	linkedDocument := document
	if path != "" {
		linkedPath := filepath.Clean(filepath.Join(filepath.Dir(document), filepath.FromSlash(path)))
		if _, err := os.Stat(filepath.Join(root, linkedPath)); err != nil {
			if os.IsNotExist(err) {
				return fmt.Sprintf("link target does not exist: %q", target)
			}
			return fmt.Sprintf("inspect link target %q: %v", target, err)
		}
		linkedDocument = filepath.ToSlash(linkedPath)
	}
	if parsed.Fragment == "" || !strings.EqualFold(filepath.Ext(linkedDocument), ".md") {
		return ""
	}
	fragment, err := url.PathUnescape(parsed.Fragment)
	if err != nil {
		return fmt.Sprintf("invalid escaped anchor %q: %v", parsed.Fragment, err)
	}
	known, ok := anchors[linkedDocument]
	if !ok {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(linkedDocument)))
		if err != nil {
			return fmt.Sprintf("read anchor target %q: %v", target, err)
		}
		known = markdownAnchors(string(content))
	}
	if _, ok := known[strings.ToLower(fragment)]; !ok {
		return fmt.Sprintf("anchor does not exist: %q", target)
	}
	return ""
}

func markdownAnchors(content string) map[string]struct{} {
	anchors := make(map[string]struct{})
	counts := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFence := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		match := headingPattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		slug := githubSlug(match[1])
		if count := counts[slug]; count > 0 {
			anchors[fmt.Sprintf("%s-%d", slug, count)] = struct{}{}
		} else {
			anchors[slug] = struct{}{}
		}
		counts[slug]++
	}
	return anchors
}

func githubSlug(heading string) string {
	heading = strings.ToLower(strings.TrimSpace(heading))
	var slug strings.Builder
	lastHyphen := false
	for _, r := range heading {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), r == '_':
			slug.WriteRune(r)
			lastHyphen = false
		case unicode.IsSpace(r), r == '-':
			if slug.Len() > 0 && !lastHyphen {
				slug.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.TrimSuffix(slug.String(), "-")
}

func lineNumber(content string, offset int) int {
	return strings.Count(content[:offset], "\n") + 1
}

func checkObserverReasons(root, sourcePath, guidePath string) []string {
	source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		return []string{fmt.Sprintf("%s: read observer reasons: %v", sourcePath, err)}
	}
	guide, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(guidePath)))
	if err != nil {
		return []string{fmt.Sprintf("%s: read observer guide: %v", guidePath, err)}
	}
	var issues []string
	for _, match := range reasonPattern.FindAllStringSubmatch(string(source), -1) {
		reason := match[1]
		if !strings.Contains(string(guide), "`"+reason+"`") {
			issues = append(issues, fmt.Sprintf("%s: missing documented observer reason `%s`", guidePath, reason))
		}
	}
	sort.Strings(issues)
	return issues
}
