package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"k8s.io/klog/v2"
)

// uiLogHook forwards logrus entries to the Bubble Tea log pane. Entries are
// only forwarded while a run is active; the run ID lets the UI discard late
// messages from a previous run.
type uiLogHook struct {
	program   *tea.Program
	level     logrus.Level
	formatter logrus.Formatter
	runID     atomic.Int64
}

func newUILogHook(program *tea.Program, level logrus.Level) *uiLogHook {
	return &uiLogHook{
		program: program,
		level:   level,
		formatter: &logrus.TextFormatter{
			DisableColors:          true,
			DisableTimestamp:       true,
			DisableLevelTruncation: true,
			DisableQuote:           true,
		},
	}
}

func (h *uiLogHook) Levels() []logrus.Level {
	levels := make([]logrus.Level, 0, len(logrus.AllLevels))
	for _, level := range logrus.AllLevels {
		if level <= h.level {
			levels = append(levels, level)
		}
	}
	return levels
}

func (h *uiLogHook) Fire(entry *logrus.Entry) error {
	runID := h.runID.Load()
	if runID == 0 {
		return nil
	}
	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	h.program.Send(logLineMsg{
		runID: int(runID),
		line:  strings.TrimRight(string(line), "\r\n"),
	})
	return nil
}

func (h *uiLogHook) SetRunID(id int) {
	h.runID.Store(int64(id))
}

// fileLogHook appends installer logs to the assets directory's
// .openshift_install.log, mirroring the CLI's file hook.
type fileLogHook struct {
	mu        sync.Mutex
	file      *os.File
	formatter logrus.Formatter
}

func newFileLogHook() *fileLogHook {
	return &fileLogHook{
		formatter: &logrus.TextFormatter{
			DisableColors:          true,
			FullTimestamp:          true,
			DisableLevelTruncation: true,
			DisableQuote:           true,
		},
	}
}

func (h *fileLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *fileLogHook) Fire(entry *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file == nil {
		return nil
	}
	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.file.Write(line)
	return err
}

func (h *fileLogHook) SetFile(file *os.File) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		_ = h.file.Close()
	}
	h.file = file
}

func (h *fileLogHook) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		_ = h.file.Close()
		h.file = nil
	}
}

func (m *Model) installLogging() {
	// Capture debug output in the TUI and let the log filter control what is
	// displayed.
	m.uiLog = newUILogHook(m.program, logrus.DebugLevel)
	m.fileLog = newFileLogHook()

	logrus.SetOutput(io.Discard)
	logrus.SetLevel(logrus.TraceLevel)
	logrus.AddHook(m.uiLog)
	logrus.AddHook(m.fileLog)

	// klog normally writes directly to stderr, which corrupts the alternate
	// screen. Route it through logrus so it lands in the TUI/file logs instead.
	klog.LogToStderr(false)
	klog.SetOutput(logrus.StandardLogger().WriterLevel(logrus.DebugLevel))
}

func (m *Model) closeLogging() {
	if m.fileLog != nil {
		m.fileLog.Close()
	}
}

func (m *Model) openLogFile(dir string) error {
	path := filepath.Join(dir, logFileName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666)
	if err != nil {
		return err
	}
	m.fileLog.SetFile(file)
	return nil
}
