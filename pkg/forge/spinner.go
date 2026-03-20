package forge

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type spinnerDoneMsg struct{ err error }

type spinnerModel struct {
	spinner spinner.Model
	title   string
	done    bool
	err     error
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("interrupted")
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return m.spinner.View() + " " + m.title
}

// isTerminal checks if stdout is connected to a terminal.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Spin shows a spinner while fn runs. In JSON mode or non-TTY environments,
// fn runs without a spinner.
func (o *Output) Spin(title string, fn func() error) error {
	if o.mode != Interactive || !isTerminal() {
		return fn()
	}

	s := spinner.New(
		spinner.WithSpinner(o.theme.Spinner),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(o.theme.Primary)),
	)

	m := spinnerModel{spinner: s, title: title}
	p := tea.NewProgram(m)

	go func() {
		err := fn()
		p.Send(spinnerDoneMsg{err: err})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	return finalModel.(spinnerModel).err
}
