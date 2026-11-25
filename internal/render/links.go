package render

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Hyperlinks is the process-wide link setting. When true, a markdown link
// [text](url) renders as its text alone, wrapped in an OSC 8 hyperlink
// sequence. When false (the default, and always for -check and -dump) the
// text is followed by its URL in the dim color and no escape sequence is
// emitted, so the URL is still readable and copyable.
var Hyperlinks bool

// A link's destination is rewritten to a placeholder URL before glamour
// sees the source. Glamour prints a link as its text, a space, then its URL;
// the Link style's template (linkFormat) turns the printed URL into a
// zero-width sentinel, so the text wraps as if the URL were not there.
// finishLinks then swaps the real URL into the OSC 8 sequence around the
// text and drops the sentinel with the space before it.
const (
	linkScheme = "tp-link:"
	sentinel   = "\u2060" // word joiner: zero width
)

// linkFormat is the Link style's Format: it hides placeholder URLs and
// leaves any other (an autolink's, which is its text) alone.
var linkFormat = `{{if Matches .text "` + linkScheme + `"}}` + sentinel + `{{else}}{{.text}}{{end}}`

func linkMarker(n int) string { return linkScheme + strconv.Itoa(n) }

// markerLink decodes the link number from an OSC 8 URL.
func markerLink(url string) (n int, ok bool) {
	rest, ok := strings.CutPrefix(url, linkScheme)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

var schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// hideLinkURLs rewrites every inline link [text](dest) in src outside code
// to [text](marker) and returns the real destinations by marker number; a
// destination that is not an absolute URL (a relative path) maps to "" and
// renders as plain text. Images, references, autolinks and anchors are left
// alone.
func hideLinkURLs(src string) (string, []string) {
	var urls []string
	var out strings.Builder
	fence := ""
	for li, line := range strings.Split(src, "\n") {
		if li > 0 {
			out.WriteByte('\n')
		}
		if m := topFenceRe.FindStringSubmatch(line); m != nil {
			switch {
			case fence == "":
				fence = m[1]
			case strings.HasPrefix(m[1], fence) && strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), fence[:1])) == "":
				fence = ""
			}
			out.WriteString(line)
			continue
		}
		if fence != "" {
			out.WriteString(line)
			continue
		}
		out.WriteString(hideLineLinks(line, &urls))
	}
	return out.String(), urls
}

// hideLineLinks is hideLinkURLs for one prose line.
func hideLineLinks(line string, urls *[]string) string {
	if !strings.Contains(line, "](") {
		return line
	}
	var out strings.Builder
	for i := 0; i < len(line); {
		c := line[i]
		switch {
		case c == '\\' && i+1 < len(line):
			out.WriteString(line[i : i+2])
			i += 2
		case c == '`':
			j := i
			for j < len(line) && line[j] == '`' {
				j++
			}
			run := line[i:j]
			end := strings.Index(line[j:], run)
			if end < 0 {
				out.WriteString(run)
				i = j
				continue
			}
			out.WriteString(line[i : j+end+len(run)])
			i = j + end + len(run)
		case c == '!' && i+1 < len(line) && line[i+1] == '[':
			// An image: copy its [alt] and (src) untouched.
			if text, ok := bracketEnd(line, i+1); ok {
				if dest, ok := parenEnd(line, text); ok {
					out.WriteString(line[i:dest])
					i = dest
					continue
				}
			}
			out.WriteByte(c)
			i++
		case c == '[':
			if text, ok := bracketEnd(line, i); ok {
				if dest, ok := parenEnd(line, text); ok {
					d := destination(line[text+2 : dest-1])
					if d != "" && !strings.HasPrefix(d, "#") && !strings.ContainsAny(d, "<>") {
						u := ""
						if schemeRe.MatchString(d) {
							u = d
						}
						*urls = append(*urls, u)
						out.WriteString(line[i:text+1] + "(" + linkMarker(len(*urls)-1) + ")")
						i = dest
						continue
					}
				}
			}
			out.WriteByte(c)
			i++
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String()
}

// bracketEnd is the index of the ] matching the [ at line[i].
func bracketEnd(line string, i int) (int, bool) {
	depth := 0
	for j := i; j < len(line); j++ {
		switch line[j] {
		case '\\':
			j++
		case '`':
			return 0, false
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return j, true
			}
		}
	}
	return 0, false
}

// parenEnd is the index just past the ")" closing the "(" that follows the
// ] at line[i], or false if no "(...)" follows.
func parenEnd(line string, i int) (int, bool) {
	if i+1 >= len(line) || line[i+1] != '(' {
		return 0, false
	}
	depth := 0
	for j := i + 1; j < len(line); j++ {
		switch line[j] {
		case '\\':
			j++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return j + 1, true
			}
		}
	}
	return 0, false
}

// destination is the URL of a link's (...) contents: the text up to the
// first space, which drops an optional "title".
func destination(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		s = s[:i]
	}
	return s
}

// hyperlink is the OSC 8 sequence that opens a link to url.
func hyperlink(url string, n int) string {
	h := fnv.New32a()
	h.Write([]byte(url))
	return ansi.SetHyperlink(url, fmt.Sprintf("id=%d-%d", h.Sum32(), n))
}

// finishLinks completes the lines of one glamour render after hideLinkURLs:
// it points each link's OSC 8 sequences at its real URL (or removes them for
// a link with none), and removes the zero-width URL glamour printed after
// the link text together with the space before it.
func finishLinks(lines []string, urls []string) []string {
	out := make([]string, len(lines))
	p := ansi.NewParser()
	for i, l := range lines {
		if !strings.Contains(l, "\x1b]8;") {
			out[i] = l
			continue
		}
		out[i] = finishLine(p, l, urls)
	}
	return out
}

type linkPiece struct {
	s     string
	width int
}

func finishLine(p *ansi.Parser, line string, urls []string) string {
	var pieces []linkPiece
	var state byte
	skipClose := false // inside a link with no URL
	for len(line) > 0 {
		seq, width, n, ns := ansi.DecodeSequence(line, state, p)
		state = ns
		line = line[n:]
		switch {
		case seq == sentinel:
			// Drop the space glamour put between the text and its URL.
			for j := len(pieces) - 1; j >= 0; j-- {
				if pieces[j].width > 0 {
					if pieces[j].s == " " {
						pieces = append(pieces[:j], pieces[j+1:]...)
					}
					break
				}
			}
		case !strings.HasPrefix(seq, "\x1b]8;"):
			pieces = append(pieces, linkPiece{seq, width})
		case hyperlinkURL(seq) == "": // close
			if skipClose {
				skipClose = false
			} else {
				pieces = append(pieces, linkPiece{seq, 0})
			}
		default:
			num, ok := markerLink(hyperlinkURL(seq))
			switch {
			case !ok:
				pieces = append(pieces, linkPiece{seq, 0})
			case urls[num] != "":
				pieces = append(pieces, linkPiece{hyperlink(urls[num], num), 0})
			default:
				skipClose = true
			}
		}
	}
	var sb strings.Builder
	for _, pc := range pieces {
		sb.WriteString(pc.s)
	}
	return sb.String()
}

// hyperlinkURL is the URL of an OSC 8 sequence ("" for one that closes a link).
func hyperlinkURL(seq string) string {
	s := strings.TrimPrefix(seq, "\x1b]8;")
	s = strings.TrimSuffix(strings.TrimSuffix(s, "\x07"), "\x1b\\")
	if i := strings.Index(s, ";"); i >= 0 {
		return s[i+1:]
	}
	return ""
}

// StripLinks removes every OSC 8 hyperlink sequence from s, leaving the
// text they wrap.
func StripLinks(s string) string {
	if !strings.Contains(s, "\x1b]8;") {
		return s
	}
	var sb strings.Builder
	p := ansi.NewParser()
	var state byte
	for len(s) > 0 {
		seq, _, n, ns := ansi.DecodeSequence(s, state, p)
		state = ns
		s = s[n:]
		if !strings.HasPrefix(seq, "\x1b]8;") {
			sb.WriteString(seq)
		}
	}
	return sb.String()
}
