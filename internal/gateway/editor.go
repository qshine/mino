package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

// The kernel's line editing erases one column per character, so a deleted
// Chinese character (two columns) stays half on screen and the display stops
// matching the input. During chat Mino turns off the kernel's line editing and
// echo, keeps Ctrl+C as a signal, and edits the line here, erasing each
// character by its display width.
type editor struct {
	mu     sync.Mutex
	out    io.Writer
	cols   func() int // terminal width, read at each prompt; 0 means unknown
	width  int
	line   []rune
	starts []cell // where each character of line begins on screen
	at     cell   // cursor, relative to the row where the line began
}

type cell struct{ row, col int }

const maxLineBytes = 1 << 20

func newEditor(out io.Writer, cols func() int) *editor {
	return &editor{out: out, cols: cols}
}

// prompt prints text and starts the line after it. Text typed ahead while the
// model was answering was echoed in the middle of that output, so it is shown
// again after the prompt.
func (e *editor) prompt(text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, err := io.WriteString(e.out, text); err != nil {
		return err
	}
	e.width = e.cols()
	e.at = cell{col: displayWidth(text[strings.LastIndexByte(text, '\n')+1:])}
	typed := e.line
	e.line, e.starts = nil, nil
	for _, r := range typed {
		e.insert(r)
	}
	return nil
}

// key applies one input character. It returns the finished line on Enter and
// eof for Ctrl+D on an empty line.
func (e *editor) key(r rune) (line string, done, eof bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch r {
	case '\r', '\n':
		line = string(e.line)
		io.WriteString(e.out, "\r\n")
		e.line, e.starts, e.at = nil, nil, cell{}
		return line, true, false
	case 0x04: // Ctrl+D
		return "", false, len(e.line) == 0
	case 0x7f, 0x08: // Delete, Backspace
		e.erase(len(e.line) - 1)
	case 0x15: // Ctrl+U
		e.erase(0)
	case 0x17: // Ctrl+W
		n := len(e.line)
		for n > 0 && unicode.IsSpace(e.line[n-1]) {
			n--
		}
		for n > 0 && !unicode.IsSpace(e.line[n-1]) {
			n--
		}
		e.erase(n)
	default:
		if r == '\t' || unicode.IsPrint(r) {
			e.insert(r)
		}
	}
	return "", false, false
}

func (e *editor) insert(r rune) {
	w, echo := runeWidth(r), string(r)
	if r == '\t' {
		w = 8 - e.at.col%8
		echo = strings.Repeat(" ", w)
	}
	start := e.at
	// A character that does not fit wraps to the next row; a wide character
	// leaves the last column empty.
	if e.width > 0 && start.col+w > e.width {
		start = cell{row: start.row + 1}
	}
	io.WriteString(e.out, echo)
	e.line = append(e.line, r)
	e.starts = append(e.starts, start)
	e.at = cell{start.row, start.col + w}
}

// erase removes line[n:] by moving to where line[n] began and clearing the
// rest of the screen, which also covers characters that wrapped.
func (e *editor) erase(n int) {
	if n < 0 || n >= len(e.line) {
		return
	}
	to := e.starts[n]
	var b strings.Builder
	if up := e.at.row - to.row; up > 0 {
		b.WriteString("\x1b[" + strconv.Itoa(up) + "A")
	}
	b.WriteString("\r")
	if to.col > 0 {
		b.WriteString("\x1b[" + strconv.Itoa(to.col) + "C")
	}
	b.WriteString("\x1b[J")
	io.WriteString(e.out, b.String())
	e.line, e.starts, e.at = e.line[:n], e.starts[:n], to
}

// editLines is scanLines for a terminal in non-canonical mode. Escape
// sequences such as arrow keys are discarded rather than typed into the line.
func editLines(ctx context.Context, input *bufio.Reader, e *editor) <-chan inputLine {
	lines := make(chan inputLine)
	go func() {
		defer close(lines)
		send := func(line inputLine) bool {
			select {
			case lines <- line:
				return true
			case <-ctx.Done():
				return false
			}
		}
		size := 0
		for {
			r, n, err := input.ReadRune()
			if err != nil {
				if err != io.EOF {
					send(inputLine{err: err})
				}
				return
			}
			if r == 0x1b {
				skipEscape(input)
				continue
			}
			if r == unicode.ReplacementChar && n == 1 {
				continue
			}
			if size += n; size > maxLineBytes {
				send(inputLine{err: bufio.ErrTooLong})
				return
			}
			line, done, eof := e.key(r)
			if eof {
				return
			}
			if done {
				size = 0
				if !send(inputLine{text: line}) {
					return
				}
			}
		}
	}()
	return lines
}

// skipEscape consumes the rest of an escape sequence: CSI (ESC [ ... final)
// or SS3 (ESC O x). A lone Escape key is followed by nothing and is dropped.
func skipEscape(input *bufio.Reader) {
	if input.Buffered() == 0 {
		return
	}
	next, err := input.ReadByte()
	if err != nil {
		return
	}
	switch next {
	case '[':
		for {
			b, err := input.ReadByte()
			if err != nil || (b >= 0x40 && b <= 0x7e) {
				return
			}
		}
	case 'O':
		input.ReadByte()
	}
}

func displayWidth(text string) int {
	width := 0
	for _, r := range text {
		width += runeWidth(r)
	}
	return width
}

// runeWidth returns the terminal columns a character occupies: two for East
// Asian wide and fullwidth characters and common emoji, none for combining
// marks, one otherwise.
func runeWidth(r rune) int {
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	for _, span := range wideRunes {
		if r >= span[0] && r <= span[1] {
			return 2
		}
	}
	return 1
}

var wideRunes = [][2]rune{
	{0x1100, 0x115f},   // Hangul Jamo initials
	{0x2e80, 0x303e},   // CJK radicals, punctuation, ideographic space
	{0x3041, 0x33ff},   // kana, bopomofo, CJK compatibility
	{0x3400, 0x4dbf},   // CJK extension A
	{0x4e00, 0x9fff},   // CJK unified ideographs
	{0xa000, 0xa4cf},   // Yi
	{0xa960, 0xa97f},   // Hangul Jamo extended A
	{0xac00, 0xd7a3},   // Hangul syllables
	{0xf900, 0xfaff},   // CJK compatibility ideographs
	{0xfe10, 0xfe19},   // vertical forms
	{0xfe30, 0xfe6f},   // CJK compatibility and small forms
	{0xff00, 0xff60},   // fullwidth forms
	{0xffe0, 0xffe6},   // fullwidth signs
	{0x1f300, 0x1f64f}, // pictographs and emoticons
	{0x1f900, 0x1f9ff}, // supplemental pictographs
	{0x20000, 0x3fffd}, // CJK extensions B and later
}

// terminalColumns reads the width with stty, like the other terminal settings.
func terminalColumns(read func(...string) (string, error)) func() int {
	return func() int {
		size, err := read("size")
		if err != nil {
			return 0
		}
		var rows, cols int
		if _, err := fmt.Sscan(size, &rows, &cols); err != nil {
			return 0
		}
		return cols
	}
}
