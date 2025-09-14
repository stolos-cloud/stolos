// tea.go
package main

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StepKind describes what the step renders.]
type StepKind int

const (
	StepForm StepKind = iota
	StepSpinner
	StepPlain
)

// Field describes a single text input line in a form step.
type Field struct {
	Label       string
	Optional    bool
	Placeholder string
	Input       textinput.Model
}

// Step defines one wizard step.
type Step struct {
	Title       string
	Kind        StepKind
	Fields      []Field // used when Kind == StepForm
	Body        string  // used when Kind == StepPlain or StepSpinner
	IsDone      bool
	AutoAdvance bool
	OnEnter     func(*Model) tea.Cmd // hook called when step is entered
}

// NewTextField constructs a text input field
func NewTextField(label, placeholder string, optional bool) Field {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = placeholder
	//ti.CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	return Field{
		Label:       label,
		Placeholder: placeholder,
		Optional:    optional,
		Input:       ti,
	}
}

// UILogger lets external goroutines (e.g., HTTP handlers) push log messages to the TUI.
// It uses program.Send under the hood
type UILogger struct {
	send func(msg tea.Msg)
}

// Info logs an info line (non-blocking).
func (l *UILogger) Info(s string)    { l.emit(logMsg{Level: levelInfo, Text: s, At: time.Now()}) }
func (l *UILogger) Warn(s string)    { l.emit(logMsg{Level: levelWarn, Text: s, At: time.Now()}) }
func (l *UILogger) Error(s string)   { l.emit(logMsg{Level: levelError, Text: s, At: time.Now()}) }
func (l *UILogger) Success(s string) { l.emit(logMsg{Level: levelSuccess, Text: s, At: time.Now()}) }

func (l *UILogger) Infof(f string, a ...any)    { l.Info(fmt.Sprintf(f, a...)) }
func (l *UILogger) Warnf(f string, a ...any)    { l.Warn(fmt.Sprintf(f, a...)) }
func (l *UILogger) Errorf(f string, a ...any)   { l.Error(fmt.Sprintf(f, a...)) }
func (l *UILogger) Successf(f string, a ...any) { l.Success(fmt.Sprintf(f, a...)) }

// emit always spawns a goroutine so the caller can't ever block on the UI thread.
func (l *UILogger) emit(m tea.Msg) {
	go func() { l.send(m) }()
}

// NewWizard constructs the Bubble Tea program + UILogger from the provided steps.
func NewWizard(steps []Step) (*tea.Program, *UILogger) {
	m := newModel(steps)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	m.program = p
	l := &UILogger{send: p.Send}
	return p, l
}

// Log levels & message type injected via UILogger.
type logLevel int

const (
	levelInfo logLevel = iota
	levelWarn
	levelError
	levelSuccess
)

type logMsg struct {
	Level logLevel
	Text  string
	At    time.Time
}

type stepEnteredMsg struct{ idx int }
type advanceMsg struct{}
type tickMsg struct{}

// Model holds UI state.
type Model struct {
	steps             []Step
	currentStepIndex  int
	width             int
	height            int
	spinner           spinner.Model
	focusedFieldIndex int
	logs              []logMsg
	maxLogs           int
	program           *tea.Program // Backref for internal Cmds that may need Send
}

func newModel(steps []Step) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Model{
		steps:            steps,
		currentStepIndex: 0,
		spinner:          s,
		maxLogs:          500,
	}
}

func (m *Model) Init() tea.Cmd {
	// Enter the first step; we also kick the spinner in case first step needs it.
	return tea.Batch(m.enterStepCmd(0), m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// AutoAdvance when step IsDone
	if m.getCurrentStep().AutoAdvance && m.getCurrentStep().IsDone {
		return m, m.advanceCmd()
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.getCurrentStep().IsDone {
				return m, m.advanceCmd()
			}
			return m, nil

		case "tab", "down":
			if m.getCurrentStep().Kind == StepForm && len(m.getCurrentStep().Fields) > 0 {
				m.steps[m.currentStepIndex].Fields[m.focusedFieldIndex].Input.Blur()
				m.focusedFieldIndex = (m.focusedFieldIndex + 1) % len(m.getCurrentStep().Fields)
				m.steps[m.currentStepIndex].Fields[m.focusedFieldIndex].Input.Focus()
			}
			return m, nil

		case "shift+tab", "up":
			if m.getCurrentStep().Kind == StepForm && len(m.getCurrentStep().Fields) > 0 {
				m.steps[m.currentStepIndex].Fields[m.focusedFieldIndex].Input.Blur()
				m.focusedFieldIndex--
				if m.focusedFieldIndex < 0 {
					m.focusedFieldIndex = len(m.getCurrentStep().Fields) - 1
				}
				m.steps[m.currentStepIndex].Fields[m.focusedFieldIndex].Input.Focus()
			}
			return m, nil

		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case logMsg:
		m.appendLog(msg)
		return m, nil

	case stepEnteredMsg:
		cmds := []tea.Cmd{}
		if m.getCurrentStep().Kind == StepSpinner {
			cmds = append(cmds, m.spinner.Tick)
		}
		if m.getCurrentStep().OnEnter != nil {
			cmds = append(cmds, m.getCurrentStep().OnEnter(m))
		}
		return m, tea.Batch(cmds...)

	case advanceMsg:
		return m, m.advanceCmd()
	}

	// If we're on a form step, keep updating focused input.
	if m.getCurrentStep().Kind == StepForm {
		return m.updateForm(msg)
	}
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "initializing..."
	}
	var b strings.Builder
	// Breadcrumb header
	b.WriteString(m.renderBreadcrumbs())
	b.WriteString("\n\n")

	// Step title
	title := lipgloss.NewStyle().Bold(true).Render(m.getCurrentStep().Title)
	b.WriteString(title + "\n\n")

	// Step body by kind
	switch m.getCurrentStep().Kind {
	case StepForm:
		b.WriteString(m.renderForm())
	case StepSpinner:
		b.WriteString(m.renderSpinnerBody())
	default:
		// Plain
		b.WriteString(m.wrap(m.getCurrentStep().Body, m.width))
	}

	// Logs
	b.WriteString("\n\n")
	b.WriteString(m.renderLogsPane())

	return b.String()
}

// Steps helper / processing

func (m *Model) getCurrentStep() Step { return m.steps[m.currentStepIndex] }

func (m *Model) enterStepCmd(i int) tea.Cmd {
	m.currentStepIndex = i
	if m.getCurrentStep().Kind == StepForm && len(m.getCurrentStep().Fields) > 0 {
		for j := range m.getCurrentStep().Fields {
			m.steps[m.currentStepIndex].Fields[j].Input.Blur()
		}
		m.focusedFieldIndex = 0
		m.steps[m.currentStepIndex].Fields[0].Input.Focus()
	}
	return func() tea.Msg { return stepEnteredMsg{idx: i} }
}

func (m *Model) advanceCmd() tea.Cmd {
	if m.currentStepIndex >= len(m.steps)-1 {
		return tea.Quit
	}
	next := m.currentStepIndex + 1
	return m.enterStepCmd(next)
}

// Forms

func (m *Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.getCurrentStep().Fields) == 0 {
		return m, nil
	}
	// Route to the focused input
	curField := &m.steps[m.currentStepIndex].Fields[m.focusedFieldIndex]
	var cmd tea.Cmd
	curField.Input, cmd = curField.Input.Update(msg)
	return m, cmd
}

func (m *Model) renderForm() string {
	var out []string
	for idx := range m.getCurrentStep().Fields {
		f := m.getCurrentStep().Fields[idx]
		label := lipgloss.NewStyle().Bold(true).Render(f.Label)
		line := fmt.Sprintf("%s\n%s", label, f.Input.View())
		if idx < len(m.getCurrentStep().Fields)-1 {
			line += "\n"
		}
		out = append(out, line)
	}
	footer := lipgloss.NewStyle().Bold(true).Faint(false).Render("\n\n*** Press Enter to continue")
	return strings.Join(out, "\n") + footer
}

// Spinner

func (m *Model) renderSpinnerBody() string {
	sp := m.spinner.View()
	body := m.getCurrentStep().Body
	if body == "" {
		body = "Working..."
	}
	return fmt.Sprintf("%s %s", sp, body)
}

// Custom Logging

func (m *Model) appendLog(l logMsg) {
	m.logs = append(m.logs, l)
	if len(m.logs) > m.maxLogs {
		// Drop oldest
		m.logs = m.logs[len(m.logs)-m.maxLogs:]
	}
}

func (m *Model) renderLogsPane() string {
	title := lipgloss.NewStyle().Faint(true).Render("Logs")
	var lines []string
	maxLines := max(10, m.height/4) // adaptive height-ish
	start := 0
	if len(m.logs) > maxLines {
		start = len(m.logs) - maxLines
	}
	for i := start; i < len(m.logs); i++ {
		lines = append(lines, renderLogLine(m.logs[i], m.width))
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width).
		Render(title + "\n" + strings.Join(lines, "\n"))
	return box
}

func renderLogLine(l logMsg, width int) string {
	ts := l.At.Format("15:04:05")
	level := ""
	switch l.Level {
	case levelInfo:
		level = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("INFO")
	case levelWarn:
		level = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("WARN")
	case levelError:
		level = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("ERROR")
	case levelSuccess:
		level = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("OK")
	}
	msg := l.Text
	line := fmt.Sprintf("%s %-5s %s", ts, level, msg)
	if width > 0 {
		return truncate(line, width-4) // keep inside the box
	}
	return line
}

// Nav breadcrumbs

func (m *Model) renderBreadcrumbs() string {
	var parts []string
	for idx, s := range m.steps {
		label := s.Title
		if idx == m.currentStepIndex {
			label = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render(label)
		} else {
			label = lipgloss.NewStyle().Faint(true).Render(label)
		}
		parts = append(parts, label)
	}
	sep := lipgloss.NewStyle().Faint(true).Render("  ›  ")
	line := strings.Join(parts, sep)
	return truncate(line, m.width)
}

// Utils

func (m *Model) wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	if len(s) <= width {
		return s
	}
	words := strings.Fields(s)
	var out, line string
	for _, w := range words {
		if len(line)+1+len(w) > width {
			out += line + "\n"
			line = w
			continue
		}
		if line == "" {
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		out += line
	}
	return out
}

func truncate(s string, width int) string {
	if width <= 0 || len([]rune(s)) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	r := []rune(s)
	return string(r[:width-1]) + "…"
}

// parseBool fuzzy
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	case "0", "f", "false", "no", "n", "off":
		return false
	default:
		// Try to parse as int in case someone types a port number accidentally :)
		if i, err := strconv.Atoi(s); err == nil {
			return i != 0
		}
		return false
	}
}

// DefaultFunc is a no-arg function that returns the default string value.
type DefaultFunc func() string

// DefaultFuncRegistry holds default providers keyed by name.
var DefaultFuncRegistry = map[string]DefaultFunc{}

// RegisterDefaultFunc adds a function to the registry.
func RegisterDefaultFunc(name string, fn DefaultFunc) {
	if name == "" || fn == nil {
		//log.Printf("RegisterDefaultFunc: ignored empty name or nil fn")
		return
	}
	DefaultFuncRegistry[name] = fn
}

func safeCallDefault(fn DefaultFunc) (val string, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("field_default_func panic: %v", r)
			ok = false
		}
	}()
	return fn(), true
}

func createFieldsForStruct[T any]() []Field {
	formFields := []Field{}

	t := reflect.TypeFor[T]()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		// 1) Try field_default_func (by name in registry)
		def := ""
		if fnName := sf.Tag.Get("field_default_func"); fnName != "" {
			if fn, ok := DefaultFuncRegistry[fnName]; ok && fn != nil {
				if v, ok2 := safeCallDefault(fn); ok2 {
					def = v
				}
			}
		}

		// 2) Fallback to literal field_default if no value yet
		if def == "" {
			def = sf.Tag.Get("field_default")
		}

		input := textinput.New()
		input.Prompt = "? "
		input.SetValue(def) // use your SetDefault if you have one

		// Note: field_required=true -> Optional=false
		required := strings.EqualFold(sf.Tag.Get("field_required"), "true")

		formFields = append(formFields, Field{
			Label:    sf.Tag.Get("field_label"),
			Optional: !required,
			Input:    input,
		})
	}
	return formFields
}

func retrieveStructFromFields[T any](fields []Field) (*T, error) {
	result := reflect.New(reflect.TypeFor[T]())
	numFields := result.Elem().Type().NumField()
	for i := 0; i < numFields; i++ {
		value := strings.TrimSpace(fields[i].Input.Value())
		val := reflect.ValueOf(value)
		structField := result.Elem().Field(i)

		if val.Type().AssignableTo(structField.Type()) {
			structField.Set(val)
		} else {
			switch structField.Kind() {
			case reflect.Int:
				intVal, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("failed to convert string to int: %v", err)
				}
				structField.SetInt(int64(intVal))
				break
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("failed to convert string to bool: %v", err)
				}
				structField.SetBool(boolVal)
				break
			default:
				return nil, fmt.Errorf("type mismatch: cannot assign %v to %v", val.Type(), structField.Type())
			}
		}
	}
	return result.Interface().(*T), nil
}
