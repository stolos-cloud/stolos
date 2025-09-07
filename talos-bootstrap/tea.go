package main

import (
	"fmt"
	"strings"

	bspinner "github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* --------------------------------------------------------------------------------
   Styling
-------------------------------------------------------------------------------- */

var (
	styleAppTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleSubtitle   = lipgloss.NewStyle().Faint(true)
	styleStepActive = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1)
	styleStepDone   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleStepTodo   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleKey        = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("240")).Padding(0, 1)
	styleHelp       = lipgloss.NewStyle().Faint(true)
	styleError      = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	styleLabel      = lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Bold(true)
	styleCard       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("241"))
	styleDim        = lipgloss.NewStyle().Faint(true)
	styleOK         = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
)

/* --------------------------------------------------------------------------------
   Root Model impl
-------------------------------------------------------------------------------- */

func (m *model) Init() tea.Cmd {
	if len(m.pages) > 0 {
		return tea.Batch(m.pages[0].Init(), m.pages[0].FocusFirst())
	}
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch tm := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = tm.Width, tm.Height
		return m, nil

	case submittedMsg:
		// Harvest any form values before moving forward
		m.harvestFormValuesIfAny()
		return m.nextStep()

	case tea.KeyMsg:
		switch tm.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "home":
			return m.gotoStep(0)
		case "end":
			return m.gotoStep(len(steps) - 1)
		case "left", "ctrl+p":
			return m.prevStep()
		case "right", "ctrl+n":
			if m.pages[m.activeIdx].CanGoNext() {
				m.harvestFormValuesIfAny()
				return m.nextStep()
			}
		}
	}

	// Let the active page update
	pg, cmd := m.pages[m.activeIdx].Update(msg)
	m.pages[m.activeIdx] = pg
	return m, cmd
}

func (m *model) View() string {
	if m.width <= 0 {
		return "loading..."
	}

	leftW := 36
	if m.width < 80 {
		leftW = 24
	}
	rightW := m.width - leftW - 3
	if rightW < 20 {
		rightW = 20
	}

	stepper := m.renderStepper(leftW)
	content := m.pages[m.activeIdx].View(rightW)

	body := lipgloss.JoinHorizontal(lipgloss.Top, stepper, "   ", content)
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, body, "", help)
}

func (m *model) renderStepper(w int) string {
	var items []string
	for i, meta := range steps {
		icon := "•"
		style := styleStepTodo
		if i == m.activeIdx {
			style = styleStepActive
			icon = "»"
		}
		if m.completed[meta.id] {
			style = styleStepDone
			icon = "✓"
		}
		line := fmt.Sprintf("%s %s", icon, meta.title)
		if i == m.activeIdx && meta.subtitle != "" {
			line = lipgloss.JoinVertical(lipgloss.Left, line, styleSubtitle.Render("   "+meta.subtitle))
		}
		items = append(items, style.Render(line))
	}
	header := styleAppTitle.Render("Talos Bootstrapping Wizard")
	progress := styleDim.Render(fmt.Sprintf("Step %d/%d", m.activeIdx+1, len(steps)))
	sidebar := lipgloss.JoinVertical(lipgloss.Left, header, progress, "", strings.Join(items, "\n\n"))
	return styleCard.Width(w).Render(sidebar)
}

func (m *model) renderHelp() string {
	return styleHelp.Render(
		fmt.Sprintf("%s/%s Next/Prev   %s Go to Start   %s Go to End   %s Quit",
			styleKey.Render("→ / Ctrl+N"),
			styleKey.Render("← / Ctrl+P"),
			styleKey.Render("Home"),
			styleKey.Render("End"),
			styleKey.Render("Q"),
		) + "\n" +
			fmt.Sprintf("On spinners & info: press %s to continue. On forms: %s Next Field, %s Prev Field, %s Submit.",
				styleKey.Render("Enter"),
				styleKey.Render("Tab / Enter"),
				styleKey.Render("Shift+Tab"),
				styleKey.Render("Enter on last field"),
			),
	)
}

func (m *model) nextStep() (tea.Model, tea.Cmd) {
	m.completed[steps[m.activeIdx].id] = true
	if m.activeIdx+1 < len(m.pages) {
		m.activeIdx++
		return m, tea.Batch(m.pages[m.activeIdx].Init(), m.pages[m.activeIdx].FocusFirst())
	}
	// Finished
	done := lipgloss.JoinVertical(lipgloss.Left,
		styleOK.Render("Wizard Complete!"),
		styleDim.Render("This scaffold does not perform any Talos actions."),
	)
	fmt.Println(styleCard.Render(done))
	return m, tea.Quit
}

func (m *model) prevStep() (tea.Model, tea.Cmd) {
	if m.activeIdx-1 >= 0 {
		m.activeIdx--
		return m, tea.Batch(m.pages[m.activeIdx].Init(), m.pages[m.activeIdx].FocusFirst())
	}
	return m, nil
}

func (m *model) gotoStep(i int) (tea.Model, tea.Cmd) {
	if i >= 0 && i < len(m.pages) {
		m.activeIdx = i
		return m, tea.Batch(m.pages[m.activeIdx].Init(), m.pages[m.activeIdx].FocusFirst())
	}
	return m, nil
}

/* --------------------------------------------------------------------------------
   Page interface
-------------------------------------------------------------------------------- */

type page interface {
	Init() tea.Cmd
	Update(tea.Msg) (page, tea.Cmd)
	View(width int) string

	SubmitCmd() tea.Cmd
	CanGoNext() bool
	FocusFirst() tea.Cmd
	Blur() tea.Cmd
}

/* --------------------------------------------------------------------------------
   Helpers: Form Page
-------------------------------------------------------------------------------- */

type formField struct {
	Key        string
	Label      string
	Input      textinput.Model
	Required   bool
	Help       string
	Validation func(string) (ok bool, err string)
}

type formPage struct {
	Title       string
	Subtitle    string
	Description string
	Fields      []formField
	focusIdx    int
	errMsg      string
}

type submittedMsg struct{} // page requests to advance

func newTextInput(placeholder, value string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.Prompt = "> "
	ti.CharLimit = 512
	if width > 0 {
		ti.Width = width
	}
	return ti
}

func newFormPage(title, subtitle, desc string, fields []formField) *formPage {
	fp := &formPage{
		Title:       title,
		Subtitle:    subtitle,
		Description: desc,
		Fields:      fields,
		focusIdx:    0,
	}
	if len(fp.Fields) > 0 {
		fp.Fields[0].Input.Focus()
	}
	return fp
}

func (f *formPage) FocusFirst() tea.Cmd {
	if len(f.Fields) == 0 {
		return nil
	}
	for i := range f.Fields {
		f.Fields[i].Input.Blur()
	}
	f.focusIdx = 0
	return f.Fields[0].Input.Focus()
}

func (f *formPage) Blur() tea.Cmd {
	for i := range f.Fields {
		f.Fields[i].Input.Blur()
	}
	return nil
}

func (f *formPage) CanGoNext() bool {
	for _, fld := range f.Fields {
		val := strings.TrimSpace(fld.Input.Value())
		if fld.Required && val == "" {
			return false
		}
		if fld.Validation != nil {
			ok, _ := fld.Validation(val)
			if !ok {
				return false
			}
		}
	}
	return true
}

func (f *formPage) SubmitCmd() tea.Cmd { return func() tea.Msg { return submittedMsg{} } }
func (f *formPage) Init() tea.Cmd      { return nil }

func (f *formPage) Update(msg tea.Msg) (page, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "ctrl+n", "right":
			if f.CanGoNext() {
				return f, f.SubmitCmd()
			}
			f.errMsg = "Please complete required fields and fix any validation errors."
			return f, nil
		case "ctrl+p", "left":
			return f, nil
		case "tab", "shift+tab", "backtab", "enter":
			if len(f.Fields) == 0 {
				if m.String() == "enter" && f.CanGoNext() {
					return f, f.SubmitCmd()
				}
				return f, nil
			}
			f.Fields[f.focusIdx].Input.Blur()

			forward := m.String() != "shift+tab" && m.String() != "backtab"
			if forward {
				f.focusIdx++
			} else {
				f.focusIdx--
			}

			if f.focusIdx >= len(f.Fields) {
				f.focusIdx = len(f.Fields) - 1
				if m.String() == "enter" {
					if f.CanGoNext() {
						return f, f.SubmitCmd()
					}
					f.errMsg = "Please complete required fields and fix any validation errors."
				}
			} else if f.focusIdx < 0 {
				f.focusIdx = 0
			}
			return f, f.Fields[f.focusIdx].Input.Focus()
		}
	}

	// Update active input
	if len(f.Fields) > 0 && f.focusIdx >= 0 && f.focusIdx < len(f.Fields) {
		var cmd tea.Cmd
		f.Fields[clamp(f.focusIdx, len(f.Fields))].Input, cmd = f.Fields[f.focusIdx].Input.Update(msg)
		return f, cmd
	}

	return f, nil
}

func (f *formPage) View(width int) string {
	var b strings.Builder
	header := lipgloss.JoinVertical(lipgloss.Left,
		styleAppTitle.Render(f.Title),
		styleSubtitle.Render(f.Subtitle),
	)
	if f.Description != "" {
		header = lipgloss.JoinVertical(lipgloss.Left, header, f.Description)
	}
	b.WriteString(styleCard.Width(width).Render(header))

	// fields
	var rows []string
	for i := range f.Fields {
		fi := &f.Fields[i]
		lbl := styleLabel.Render(fi.Label)
		line := lipgloss.JoinHorizontal(lipgloss.Left, lbl, "  ", fi.Input.View())
		if fi.Help != "" {
			line = lipgloss.JoinVertical(lipgloss.Left, line, styleDim.Render("  "+fi.Help))
		}
		if fi.Required {
			line = lipgloss.JoinVertical(lipgloss.Left, line, styleDim.Render("  (required)"))
		}
		rows = append(rows, line)
	}
	if len(rows) > 0 {
		b.WriteString("\n\n")
		b.WriteString(styleCard.Width(width).Render(strings.Join(rows, "\n\n")))
	}

	if f.errMsg != "" {
		b.WriteString("\n\n" + styleError.Render("Error: "+f.errMsg))
	}

	return b.String()
}

func clamp(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

/* --------------------------------------------------------------------------------
   Helpers: Info Page (for Step 2 note)
-------------------------------------------------------------------------------- */

type infoPage struct {
	Title    string
	Subtitle string
	Content  string
}

func newInfoPage(title, subtitle, content string) *infoPage {
	return &infoPage{Title: title, Subtitle: subtitle, Content: content}
}

func (p *infoPage) Init() tea.Cmd       { return nil }
func (p *infoPage) FocusFirst() tea.Cmd { return nil }
func (p *infoPage) Blur() tea.Cmd       { return nil }
func (p *infoPage) CanGoNext() bool     { return true }
func (p *infoPage) SubmitCmd() tea.Cmd  { return func() tea.Msg { return submittedMsg{} } }

func (p *infoPage) Update(msg tea.Msg) (page, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter", "ctrl+n", "right":
			return p, p.SubmitCmd()
		}
	}
	return p, nil
}

func (p *infoPage) View(width int) string {
	header := lipgloss.JoinVertical(lipgloss.Left,
		styleAppTitle.Render(p.Title),
		styleSubtitle.Render(p.Subtitle),
	)
	body := p.Content
	if strings.TrimSpace(body) == "" {
		body = styleDim.Render("(No implementation; press Enter to continue)")
	}
	card := styleCard.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
	return card
}

/* --------------------------------------------------------------------------------
   Helpers: Spinner Page (with simple $VAR interpolation)
-------------------------------------------------------------------------------- */

type spinnerPage struct {
	Title    string
	Subtitle string
	Lines    []string
	spin     bspinner.Model
	varsFn   func() map[string]string // returns dynamic vars to fill like $NAME, $ENDPOINT, $IP, $FILE
}

func newSpinnerPage(title, subtitle string, lines []string, varsFn func() map[string]string) *spinnerPage {
	s := bspinner.New()
	s.Spinner = bspinner.Dot
	return &spinnerPage{
		Title:    title,
		Subtitle: subtitle,
		Lines:    lines,
		spin:     s,
		varsFn:   varsFn,
	}
}

func (p *spinnerPage) Init() tea.Cmd       { return p.spin.Tick }
func (p *spinnerPage) FocusFirst() tea.Cmd { return nil }
func (p *spinnerPage) Blur() tea.Cmd       { return nil }
func (p *spinnerPage) CanGoNext() bool     { return true }
func (p *spinnerPage) SubmitCmd() tea.Cmd  { return func() tea.Msg { return submittedMsg{} } }

func (p *spinnerPage) Update(msg tea.Msg) (page, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "enter", "ctrl+n", "right":
			return p, p.SubmitCmd()
		}
	}
	var cmd tea.Cmd
	p.spin, cmd = p.spin.Update(msg)
	return p, cmd
}

func (p *spinnerPage) View(width int) string {
	header := lipgloss.JoinVertical(lipgloss.Left,
		styleAppTitle.Render(p.Title),
		styleSubtitle.Render(p.Subtitle),
	)

	lines := p.Lines
	// Interpolate simple $VARS if provided
	if p.varsFn != nil {
		vars := p.varsFn()
		for i := range lines {
			for k, v := range vars {
				lines[i] = strings.ReplaceAll(lines[i], "$"+k, v)
			}
		}
	}

	spinLine := lipgloss.JoinHorizontal(lipgloss.Left, p.spin.View(), " ", "Working…")
	body := strings.Join(lines, "\n")
	if strings.TrimSpace(body) == "" {
		body = styleDim.Render("(No implementation; press Enter to continue)")
	}
	card := styleCard.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, header, spinLine, "", body))
	return card
}

/* --------------------------------------------------------------------------------
   Helpers: pull values from Step 1 form into state
-------------------------------------------------------------------------------- */

func (m *model) harvestFormValuesIfAny() {
	fp, ok := m.pages[m.activeIdx].(*formPage)
	if !ok {
		// Try to harvest from previous form (common when navigating with arrows)
		if m.activeIdx > 0 {
			if prev, ok2 := m.pages[m.activeIdx-1].(*formPage); ok2 {
				m.applyForm(prev)
			}
		}
		return
	}
	m.applyForm(fp)
}

func (m *model) applyForm(fp *formPage) {
	for _, fld := range fp.Fields {
		val := strings.TrimSpace(fld.Input.Value())
		switch fld.Key {
		case "cluster_name":
			m.wizState.ClusterName = val
		case "talos_version":
			m.wizState.TalosVersion = val
		case "image_overlay":
			m.wizState.ImageFactoryOverlayPath = val
		case "mc_overlay":
			m.wizState.MachineConfigOverlayPath = val
		case "http_enabled":
			m.wizState.HTTPEnabled = strings.EqualFold(val, "true")
		case "http_port":
			m.wizState.HTTPPort = val
		case "pxe_enabled":
			m.wizState.PXEEnabled = strings.EqualFold(val, "true")
		case "pxe_port":
			m.wizState.PXEPort = val
		}
	}

	// Keep endpoint placeholder unless you later compute it from real IPs.
	if m.wizState.Endpoint == "" {
		m.wizState.Endpoint = "$ENDPOINT"
	}
}
