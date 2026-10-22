package core

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// clearLine returns the cursor to column 0 and erases the current line
const clearLine = "\r\033[K"

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// statusEntry is one in-flight service shown on the live line
type statusEntry struct {
	name  string
	phase string
}

// composeStatusLine builds the live line for the in-flight entries. When the
// entries don't fit in width columns it shows as many as fit followed by
// "+N more". The result is always shorter than width so the terminal never
// wraps it. A width of 0 or less means unknown and disables truncation.
func composeStatusLine(frame string, entries []statusEntry, width int) string {
	if len(entries) == 0 {
		return ""
	}

	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%s: %s", e.name, e.phase)
	}

	for shown := len(entries); shown >= 0; shown-- {
		visible := parts[:shown]
		if hidden := len(entries) - shown; hidden > 0 {
			visible = append(visible[:shown:shown], fmt.Sprintf("+%d more", hidden))
		}
		line := frame + " " + strings.Join(visible, " | ")
		if width <= 0 || utf8.RuneCountInString(line) < width {
			return line
		}
	}

	// Not even "+N more" fits: cut it down to the available columns
	line := frame + " " + fmt.Sprintf("+%d more", len(entries))
	return string([]rune(line)[:width-1])
}

// StatusRenderer draws a single live status line shared by every in-flight
// service, and prints each finished service's message as a permanent line
// above it. It is safe for concurrent use.
type StatusRenderer struct {
	mu       sync.Mutex
	out      io.Writer
	width    func() int
	interval time.Duration
	entries  []statusEntry
	frame    int
	drawn    bool
	stop     chan struct{}
}

// NewStatusRenderer creates a renderer writing to out. width reports the
// current terminal width in columns, or 0 if unknown.
func NewStatusRenderer(out io.Writer, width func() int) *StatusRenderer {
	return &StatusRenderer{
		out:      out,
		width:    width,
		interval: 100 * time.Millisecond,
	}
}

// Add puts a service on the live line with its initial phase
func (r *StatusRenderer) Add(name, phase string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = append(r.entries, statusEntry{name: name, phase: phase})
	if r.stop == nil {
		r.stop = make(chan struct{})
		go r.tick(r.stop)
	}
	r.draw()
}

// SetPhase updates the phase shown for an in-flight service
func (r *StatusRenderer) SetPhase(name, phase string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if i := r.index(name); i >= 0 {
		r.entries[i].phase = phase
		r.draw()
	}
}

// Finish removes a service from the live line. A non-empty message is printed
// as a permanent line above the live line; an empty one prints nothing.
func (r *StatusRenderer) Finish(name, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if i := r.index(name); i >= 0 {
		r.entries = append(r.entries[:i], r.entries[i+1:]...)
	}

	r.clear()
	if message != "" {
		fmt.Fprintf(r.out, "%s\n", message)
	}

	if len(r.entries) == 0 && r.stop != nil {
		close(r.stop)
		r.stop = nil
		return
	}
	r.draw()
}

// PrintAbove writes s to w (e.g. stderr) without corrupting the live line:
// the line is cleared first and redrawn afterwards.
func (r *StatusRenderer) PrintAbove(w io.Writer, s string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clear()
	fmt.Fprint(w, s)
	r.draw()
}

func (r *StatusRenderer) tick(stop chan struct{}) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			r.mu.Lock()
			select {
			case <-stop:
			default:
				r.frame = (r.frame + 1) % len(spinnerFrames)
				r.draw()
			}
			r.mu.Unlock()
		}
	}
}

// currentLine returns the live line as it would be drawn now. Callers hold r.mu
// or are single-threaded tests.
func (r *StatusRenderer) currentLine() string {
	return composeStatusLine(spinnerFrames[r.frame], r.entries, r.width())
}

// running reports whether the spinner ticker is active
func (r *StatusRenderer) running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stop != nil
}

func (r *StatusRenderer) index(name string) int {
	for i, e := range r.entries {
		if e.name == name {
			return i
		}
	}
	return -1
}

// draw redraws the live line in place. Callers must hold r.mu.
func (r *StatusRenderer) draw() {
	if len(r.entries) == 0 {
		return
	}
	fmt.Fprint(r.out, clearLine+r.currentLine())
	r.drawn = true
}

// clear erases the live line if it is drawn. Callers must hold r.mu.
func (r *StatusRenderer) clear() {
	if r.drawn {
		fmt.Fprint(r.out, clearLine)
		r.drawn = false
	}
}
