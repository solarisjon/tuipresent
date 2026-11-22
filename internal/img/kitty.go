package img

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math/rand/v2"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi/kitty"
)

// chunkSize is the kitty protocol's maximum base64 payload per escape sequence.
const chunkSize = 4096

// Transmit returns the escape sequences that upload a PNG as image id with a
// virtual placement of cols x rows cells (a=T,U=1), chunked per the protocol.
// q=2 suppresses terminal replies so they never reach the key reader.
func Transmit(id uint32, pngData []byte, cols, rows int) string {
	data := base64.StdEncoding.EncodeToString(pngData)
	var sb strings.Builder
	first := true
	for first || len(data) > 0 {
		n := min(chunkSize, len(data))
		chunk := data[:n]
		data = data[n:]
		more := 0
		if len(data) > 0 {
			more = 1
		}
		if first {
			fmt.Fprintf(&sb, "\x1b_Ga=T,U=1,f=100,t=d,i=%d,c=%d,r=%d,q=2,m=%d;%s\x1b\\", id, cols, rows, more, chunk)
			first = false
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d,q=2;%s\x1b\\", more, chunk)
		}
	}
	return sb.String()
}

// Delete returns the escape sequence that frees image id and its placements.
func Delete(id uint32) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}

// Placeholder returns rows lines of cols kitty Unicode placeholder cells for
// image id. Every cell carries explicit row and column diacritics, so a line
// sliced at any cell (as transitions do) still addresses the right tile.
func Placeholder(id uint32, cols, rows int) []string {
	cols, rows = min(cols, MaxCells), min(rows, MaxCells)
	color := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", id>>16&0xff, id>>8&0xff, id&0xff)
	lines := make([]string, rows)
	for y := range rows {
		var sb strings.Builder
		sb.WriteString(color)
		for x := range cols {
			sb.WriteRune(kitty.Placeholder)
			sb.WriteRune(kitty.Diacritic(y))
			sb.WriteRune(kitty.Diacritic(x))
		}
		sb.WriteString("\x1b[39m")
		lines[y] = sb.String()
	}
	return lines
}

// Transmitter hands out kitty image ids, one per (image, cell box), and
// queues each upload once until Flush collects it.
type Transmitter struct {
	mu      sync.Mutex
	next    uint32
	ids     map[string]uint32
	pending strings.Builder
}

func NewTransmitter() *Transmitter {
	// A random 24-bit base keeps ids from colliding with other programs'.
	// Ids stay within 24 bits so the foreground colour alone encodes them.
	return &Transmitter{next: 1<<16 + rand.Uint32N(1<<23), ids: map[string]uint32{}}
}

// Place returns placeholder lines for src shown at cols x rows, queueing the
// upload the first time this key and size are seen. key should change when
// the image content does (e.g. path plus modification time).
func (t *Transmitter) Place(key string, src image.Image, cols, rows int) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	k := fmt.Sprintf("%s\x00%dx%d", key, cols, rows)
	id, ok := t.ids[k]
	if !ok {
		id = t.next & 0xffffff
		if id == 0 {
			id = 1
		}
		t.next = id + 1
		t.ids[k] = id
		// Upload at roughly cell resolution (assuming ~10x20 px cells) so big
		// photos don't cost megabytes of escape sequences.
		w, h := src.Bounds().Dx(), src.Bounds().Dy()
		if tw := cols * 12; w > tw {
			h, w = max(h*tw/w, 1), tw
			src = Scale(src, w, h)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, src); err == nil {
			t.pending.WriteString(Transmit(id, buf.Bytes(), cols, rows))
		}
	}
	return Placeholder(id, cols, rows)
}

// Flush returns and clears the queued uploads.
func (t *Transmitter) Flush() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.pending.String()
	t.pending.Reset()
	return s
}

// Cleanup returns escape sequences that delete every image this transmitter uploaded.
func (t *Transmitter) Cleanup() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var sb strings.Builder
	for _, id := range t.ids {
		sb.WriteString(Delete(id))
	}
	return sb.String()
}
