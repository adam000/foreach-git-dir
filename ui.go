package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type status int

const (
	statusStarted status = iota
	statusSuccess
	statusError
	statusComplete
)

type statusEvent struct {
	dir   string
	state status
	err   error
}

type statusItem struct {
	dir        string
	state      status
	err        error
	finishedAt time.Time
}

type model struct {
	entries      []statusItem
	spinnerIndex int
	width        int
	height       int
	done         bool
}

type tickMsg struct{}

type statusMsg statusEvent

type quitMsg struct{}

func newModel() model {
	return model{
		entries: make([]statusItem, 0),
	}
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.spinnerIndex++
		m.pruneEntries()
		if m.done && len(m.entries) == 0 {
			return m, tea.Quit
		}
		return m, tick()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case statusMsg:
		event := statusEvent(msg)
		if event.state == statusComplete {
			m.done = true
			m.entries = nil
			return m, tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return quitMsg{} })
		}
		if event.state == statusStarted {
			m.entries = append(m.entries, statusItem{
				dir:   event.dir,
				state: statusStarted,
			})
		} else {
			m.markFinished(event)
		}
		m.pruneEntries()
		return m, nil
	case quitMsg:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m *model) markFinished(event statusEvent) {
	for i, item := range m.entries {
		if item.dir == event.dir && item.state == statusStarted {
			m.entries[i].state = statusSuccess
			if event.err != nil {
				m.entries[i].state = statusError
				m.entries[i].err = event.err
			}
			m.entries[i].finishedAt = time.Now()
			return
		}
	}

	state := statusSuccess
	if event.err != nil {
		state = statusError
	}
	m.entries = append(m.entries, statusItem{
		dir:        event.dir,
		state:      state,
		err:        event.err,
		finishedAt: time.Now(),
	})
}

func (m *model) pruneEntries() {
	now := time.Now()
	filtered := make([]statusItem, 0, len(m.entries))

	for _, item := range m.entries {
		if item.state == statusSuccess && !item.finishedAt.IsZero() && now.Sub(item.finishedAt) >= 2*time.Second {
			continue
		}
		filtered = append(filtered, item)
	}

	m.entries = filtered
	maxRows := m.height - 2
	if maxRows <= 0 {
		return
	}

	for len(m.entries) > maxRows {
		removed := false
		for i, item := range m.entries {
			if item.state != statusStarted {
				m.entries = append(m.entries[:i], m.entries[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			m.entries = m.entries[1:]
		}
	}
}

func (m model) View() string {
	if m.done && len(m.entries) == 0 {
		return "\nProcessing complete.\n"
	}

	spinnerFrames := []string{"|", "/", "-", "\\"}
	output := strings.Builder{}

	if len(m.entries) == 0 {
		output.WriteString("Waiting to start processing...\n")
	} else {
		for _, item := range m.entries {
			symbol := " "
			switch item.state {
			case statusStarted:
				symbol = spinnerFrames[m.spinnerIndex%len(spinnerFrames)]
			case statusSuccess:
				symbol = "✔"
			case statusError:
				symbol = "✖"
			}
			fmt.Fprintf(&output, "%s %s\n", symbol, item.dir)
		}
	}

	output.WriteString("\nProcessing directories...\n")

	return output.String()
}

func startStatusUI(statusCh chan statusEvent) error {
	ui := tea.NewProgram(newModel(), tea.WithAltScreen())
	go func() {
		for event := range statusCh {
			ui.Send(statusMsg(event))
		}
		ui.Send(statusMsg{state: statusComplete})
	}()

	_, err := ui.Run()
	return err
}
