// Package orthlib imports texts from a local copy of orthlib.ru (HIP files,
// extracted from the site's RAR archives) into the corpus.
package orthlib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/n0where3an/orthodoxy-liturgical-service/internal/hip"
)

// Doc is a parsed HIP file with a plain-text index for anchor search.
type Doc struct {
	Path  string
	Paras []hip.Paragraph
	text  []rune // paragraphs joined by '\n'
	plain string
	pos   []int // plain rune index -> text rune index
	prs   []rune
}

var pageRef = regexp.MustCompile(`\s*\(с\.\s*\d+\)`)

// CleanText removes page references like "(с. 767)" and tidies spaces.
func CleanText(s string) string {
	s = pageRef.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(l), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// LoadDoc reads and parses a .hip file.
func LoadDoc(path string) (*Doc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	paras, err := hip.ParseFile(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	d := &Doc{Path: path, Paras: paras}
	lines := make([]string, len(paras))
	for i, p := range paras {
		lines[i] = pageRef.ReplaceAllString(p.Text(), "")
	}
	d.text = []rune(strings.Join(lines, "\n"))
	d.plain, d.pos = hip.Plain(string(d.text))
	d.prs = []rune(d.plain)
	return d, nil
}

func (d *Doc) indexFrom(needle string, from int) int {
	if from > len(d.prs) {
		return -1
	}
	n := []rune(needle)
	for i := from; i+len(n) <= len(d.prs); i++ {
		match := true
		for k := range n {
			if d.prs[i+k] != n[k] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// Slice returns the text from the first occurrence of anchor `from` (searched
// after anchor `after`, if given) to the end of the following occurrence of
// `to`. Anchors are civil-script strings compared after folding.
func (d *Doc) Slice(after, from, to string) (string, error) {
	start := 0
	if after != "" {
		i := d.indexFrom(hip.Fold(after), 0)
		if i < 0 {
			return "", fmt.Errorf("%s: anchor 'after' %q not found", filepath.Base(d.Path), after)
		}
		start = i + len([]rune(hip.Fold(after)))
	}
	f := hip.Fold(from)
	s := d.indexFrom(f, start)
	if s < 0 {
		return "", fmt.Errorf("%s: anchor 'from' %q not found", filepath.Base(d.Path), from)
	}
	t := hip.Fold(to)
	e := d.indexFrom(t, s)
	if e < 0 {
		return "", fmt.Errorf("%s: anchor 'to' %q not found after %q", filepath.Base(d.Path), to, from)
	}
	e += len([]rune(t))
	rs, re := d.pos[s], d.pos[e-1]+1
	for re < len(d.text) && unicode.Is(unicode.Mn, d.text[re]) {
		re++
	}
	return CleanText(string(d.text[rs:re])), nil
}
