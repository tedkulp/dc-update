package core

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func TestComposeStatusLine(t *testing.T) {
	entries := []statusEntry{
		{name: "app", phase: "pulling"},
		{name: "db", phase: "checking"},
		{name: "web", phase: "restarting"},
	}

	tests := []struct {
		name    string
		entries []statusEntry
		waiting int
		width   int
		want    string
	}{
		{
			name:    "no entries",
			entries: nil,
			width:   80,
			want:    "",
		},
		{
			name:    "single entry",
			entries: entries[:1],
			width:   80,
			want:    "⠋ app: pulling",
		},
		{
			name:    "single entry with message looks like one spinner",
			entries: []statusEntry{{name: "app", phase: "pulling", message: "Updating app"}},
			width:   80,
			want:    "⠋ Updating app",
		},
		{
			name:    "single entry message is cut to width",
			entries: []statusEntry{{name: "app", phase: "pulling", message: "Updating and restarting app"}},
			width:   12,
			want:    "⠋ Updating ",
		},
		{
			name:    "all entries fit",
			entries: entries,
			width:   80,
			want:    "⠋ app: pulling | db: checking | web: restarting",
		},
		{
			name:    "unknown width does not truncate",
			entries: entries,
			width:   0,
			want:    "⠋ app: pulling | db: checking | web: restarting",
		},
		{
			name:    "overflow shows +N more",
			entries: entries,
			width:   40,
			want:    "⠋ app: pulling | db: checking | +1 more",
		},
		{
			name:    "overflow of all but one",
			entries: entries,
			width:   30,
			want:    "⠋ app: pulling | +2 more",
		},
		{
			name:    "queued services show as waiting",
			entries: entries,
			waiting: 12,
			width:   80,
			want:    "⠋ app: pulling | db: checking | web: restarting | +12 waiting",
		},
		{
			name:    "waiting merges into +N more on overflow",
			entries: entries,
			waiting: 12,
			width:   50,
			want:    "⠋ app: pulling | db: checking | +13 more",
		},
		{
			name:    "lone entry with a queue lists the entry, not its message",
			entries: []statusEntry{{name: "app", phase: "pulling", message: "Updating app"}},
			waiting: 2,
			width:   80,
			want:    "⠋ app: pulling | +2 waiting",
		},
		{
			name:    "nothing fits but the counter",
			entries: entries,
			width:   16,
			want:    "⠋ +3 more",
		},
		{
			name:    "even the counter does not fit",
			entries: entries,
			width:   6,
			want:    "⠋ +3 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := composeStatusLine("⠋", tt.entries, tt.waiting, tt.width)
			if got != tt.want {
				t.Errorf("composeStatusLine() = %q, want %q", got, tt.want)
			}
			if tt.width > 0 && utf8.RuneCountInString(got) >= tt.width {
				t.Errorf("line %q is %d columns, must stay under width %d to avoid wrapping",
					got, utf8.RuneCountInString(got), tt.width)
			}
		})
	}
}

// permanentLines extracts the lines that end in a newline, i.e. the
// permanent output, from the raw terminal stream.
func permanentLines(out string) []string {
	var lines []string
	for _, chunk := range strings.SplitAfter(out, "\n") {
		if !strings.HasSuffix(chunk, "\n") {
			continue
		}
		chunk = strings.TrimSuffix(chunk, "\n")
		if i := strings.LastIndex(chunk, clearLine); i >= 0 {
			chunk = chunk[i+len(clearLine):]
		}
		lines = append(lines, chunk)
	}
	return lines
}

func newTestRenderer(buf *bytes.Buffer) *StatusRenderer {
	r := NewStatusRenderer(buf, func() int { return 80 })
	r.interval = time.Hour // keep the ticker out of the way
	return r
}

func TestStatusRendererFinishedLines(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRenderer(&buf)

	app := r.Add("app", "checking", "Updating app")
	db := r.Add("db", "checking", "Updating db")
	web := r.Add("web", "checking", "Updating web")
	r.SetPhase(db, "pulling", "")

	if got := r.currentLine(); got != "⠋ app: checking | db: pulling | web: checking" {
		t.Fatalf("live line = %q", got)
	}

	r.Finish(db, "✅ Updated db")
	if got := r.currentLine(); got != "⠋ app: checking | web: checking" {
		t.Fatalf("live line after db finished = %q", got)
	}

	r.Finish(web, "") // suppressed warning: drops off silently
	r.SetPhase(app, "restarting", "Updating and restarting app")
	if got := r.currentLine(); got != "⠋ Updating and restarting app" {
		t.Fatalf("live line with one service left = %q", got)
	}
	r.Finish(app, "✅ app is already up to date")

	want := []string{"✅ Updated db", "✅ app is already up to date"}
	got := permanentLines(buf.String())
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("permanent lines = %q, want %q", got, want)
	}

	out := buf.String()
	if tail := out[strings.LastIndex(out, "\n")+1:]; tail != "" && tail != clearLine {
		t.Errorf("live line should be gone once all services finish, got trailing %q", tail)
	}
	if r.running() {
		t.Errorf("ticker should stop once every service is finished")
	}
}

func TestStatusRendererPrintAbove(t *testing.T) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	r := newTestRenderer(&buf)

	app := r.Add("app", "checking", "Updating app")
	fmt.Fprint(r.Above(&errBuf), "Error updating app: boom\n")

	if errBuf.String() != "Error updating app: boom\n" {
		t.Errorf("stderr = %q", errBuf.String())
	}
	out := buf.String()
	if !strings.HasSuffix(out, clearLine+clearLine+"⠋ Updating app") {
		t.Errorf("live line should be cleared then redrawn around stderr output, got %q", out)
	}
	r.Finish(app, "")
}

func TestStatusRendererWaiting(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRenderer(&buf)

	r.AddWaiting(3)
	r.AddWaiting(-1)
	app := r.Add("app", "checking", "Updating app")

	if got := r.currentLine(); got != "⠋ app: checking | +2 waiting" {
		t.Fatalf("live line = %q", got)
	}
	r.Finish(app, "")
}

func TestStatusRendererDuplicateNames(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRenderer(&buf)

	first := r.Add("app", "checking", "")
	second := r.Add("app", "checking", "")
	r.SetPhase(second, "pulling", "")

	if got := r.currentLine(); got != "⠋ app: checking | app: pulling" {
		t.Fatalf("live line = %q", got)
	}

	r.Finish(second, "second")
	if got := r.currentLine(); got != "⠋ app: checking" {
		t.Fatalf("live line after second finished = %q", got)
	}
	r.Finish(first, "first")
}

func TestStatusRendererConcurrent(t *testing.T) {
	var buf bytes.Buffer
	r := NewStatusRenderer(&buf, func() int { return 60 })
	r.interval = time.Millisecond

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("svc%d", i)
			id := r.Add(name, "checking", "Updating "+name)
			r.SetPhase(id, "pulling", "")
			time.Sleep(2 * time.Millisecond)
			r.Finish(id, "✅ Updated "+name)
		}(i)
	}
	wg.Wait()

	lines := permanentLines(buf.String())
	if len(lines) != n {
		t.Fatalf("got %d permanent lines, want %d: %q", len(lines), n, lines)
	}
	seen := map[string]bool{}
	for _, l := range lines {
		if seen[l] {
			t.Errorf("duplicate permanent line %q", l)
		}
		seen[l] = true
	}
	if r.running() {
		t.Errorf("ticker should stop once every service is finished")
	}
}
