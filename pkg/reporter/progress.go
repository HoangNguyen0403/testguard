package reporter

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	unicodeFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	asciiFrames   = []rune{'-', '\\', '|', '/'}
)

// ProgressConfig holds configuration options for terminal progress indication.
type ProgressConfig struct {
	IsTerminal bool
	NoColor    bool
	Disabled   bool
}

// ProgressReporter coordinates live terminal feedback across audit execution phases.
type ProgressReporter struct {
	out         io.Writer
	config      ProgressConfig
	mu          sync.Mutex
	frameIdx    int
	phase       string
	current     int
	total       int
	currentFile string
	isFinished    bool
	renderedPhase string
	lastRender    time.Time
	ticker        *time.Ticker
	done          chan struct{}
	doneAck       chan struct{}
}

// isTerminalFile inspects whether the target writer is an interactive character device.
func isTerminalFile(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// isInteractive determines if the writer supports dynamic cursor rewrites.
func isInteractive(w io.Writer) bool {
	ci := os.Getenv("CI")
	if ci != "" && ci != "false" && ci != "0" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminalFile(w)
}

// NewProgressReporter initializes an interactive terminal progress reporter attached to w.
func NewProgressReporter(w io.Writer) *ProgressReporter {
	isTerm := isInteractive(w)
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	return NewProgressReporterWithOptions(w, ProgressConfig{
		IsTerminal: isTerm,
		NoColor:    noColor,
		Disabled:   false,
	})
}

// NewProgressReporterWithOptions creates a customized progress reporter instance.
func NewProgressReporterWithOptions(w io.Writer, cfg ProgressConfig) *ProgressReporter {
	reporter := &ProgressReporter{
		out:    w,
		config: cfg,
	}

	if cfg.IsTerminal && !cfg.Disabled {
		reporter.ticker = time.NewTicker(80 * time.Millisecond)
		reporter.done = make(chan struct{})
		reporter.doneAck = make(chan struct{})

		go func() {
			defer close(reporter.doneAck)
			for {
				select {
				case <-reporter.done:
					return
				case <-reporter.ticker.C:
					reporter.mu.Lock()
					if !reporter.isFinished {
						reporter.frameIdx++
						reporter.renderLocked()
					}
					reporter.mu.Unlock()
				}
			}
		}()
	}

	return reporter
}

// Update records current scan metrics and triggers throttled display synchronization.
func (p *ProgressReporter) Update(phase string, current, total int, currentFile string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.Disabled || p.isFinished {
		return
	}

	p.phase = phase
	p.current = current
	p.total = total
	p.currentFile = currentFile
	now := time.Now()
	if phase != p.renderedPhase || now.Sub(p.lastRender) >= 80*time.Millisecond {
		p.renderedPhase = phase
		p.renderLocked()
		p.lastRender = now
	}
}

// renderLocked writes formatted progress lines to the configured writer under lock.
func (p *ProgressReporter) renderLocked() {
	if p.config.Disabled {
		return
	}

	if !p.config.IsTerminal {
		return
	}

	frames := unicodeFrames
	if p.config.NoColor {
		frames = asciiFrames
	}
	frame := frames[p.frameIdx%len(frames)]

	switch p.phase {
	case "discovering":
		_, _ = fmt.Fprintf(p.out, "\r\033[2K%c Discovering test files...", frame)
	case "discovered":
		_, _ = fmt.Fprintf(p.out, "\r\033[2K%c Discovered %d test files. Beginning audit...", frame, p.total)
	case "scanning":
		truncated := TruncatePath(p.currentFile, 32)
		_, _ = fmt.Fprintf(p.out, "\r\033[2K%c Scanning test files [%d/%d] • %s", frame, p.current, p.total, truncated)
	}
}

// Finish cleanly clears the active progress line and releases background resources.
func (p *ProgressReporter) Finish() {
	p.mu.Lock()
	if p.isFinished || p.config.Disabled {
		p.mu.Unlock()
		return
	}

	p.isFinished = true
	if p.ticker != nil {
		p.ticker.Stop()
		close(p.done)
	}
	p.mu.Unlock()

	if p.doneAck != nil {
		<-p.doneAck
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.config.IsTerminal {
		_, _ = fmt.Fprintf(p.out, "\r\033[2K")
	}
}

// TruncatePath shortens long file paths for single-line terminal viewports, rune-safe.
func TruncatePath(path string, maxLen int) string {
	runes := []rune(path)
	if maxLen <= 0 || len(runes) <= maxLen {
		return path
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return "..." + string(runes[len(runes)-(maxLen-3):])
}
