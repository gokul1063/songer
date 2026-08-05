package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"songer/pkg/autoplay"
	"songer/pkg/config"
	"songer/pkg/mpv"
	"songer/pkg/search"
)

type waveTick struct{}

type queueLoadedMsg struct {
	videos []search.Video
}

type queueFailedMsg struct {
	err error
}

type focus int

const (
	focusQueue focus = iota
	focusMain
)

// playerController is the slice of *mpv.Player the UI needs.
type playerController interface {
	Load(url string)
	TogglePause()
	Seek(d time.Duration)
	SetVolume(v int)
}

type Model struct {
	player       playerController
	theme        config.Theme
	ctx          context.Context
	current      search.Video
	upcoming     []search.Video
	queueIdx     int
	state        mpv.State
	wave         []float64
	width        int
	height       int
	showHelp     bool
	status       string
	perNode      int
	depth        int
	queuePending bool
	pendingSeed  string
	focus        focus
	listScroll   int
}

const maxQueue = 60

func Run(ctx context.Context, player *mpv.Player, current search.Video, theme config.Theme, perNode, depth int) error {
	m := Model{
		player:   player,
		theme:    theme,
		ctx:      ctx,
		current:  current,
		upcoming: []search.Video{current},
		status:   "▶ " + current.Title,
		perNode:  perNode,
		depth:    depth,
		focus:    focusQueue,
	}
	m.wave = nextWave(waveCount(80))

	p := tea.NewProgram(m, tea.WithAltScreen())

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case st, ok := <-player.Events():
				if !ok {
					return
				}
				p.Send(st)
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	_, err := p.Run()
	close(stop)
	<-done
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waveCmd(), m.queueCmd())
}

func waveCmd() tea.Cmd {
	return tea.Tick(waveInterval, func(time.Time) tea.Msg { return waveTick{} })
}

// queueCmd fetches the suggested queue for the current song in a goroutine
// and hands the result back to the event loop.
func (m Model) queueCmd() tea.Cmd {
	seed, perNode, depth := m.current.ID, m.perNode, m.depth
	return func() tea.Msg {
		videos, err := autoplay.NewClient().BuildQueue(m.ctx, seed, perNode, depth)
		if err != nil {
			return queueFailedMsg{err: err}
		}
		return queueLoadedMsg{videos: videos}
	}
}

func (m Model) startQueueFetch() (Model, tea.Cmd) {
	if m.current.ID == "" {
		return m, nil
	}
	if m.queuePending && m.pendingSeed == m.current.ID {
		return m, nil
	}
	m.queuePending = true
	m.pendingSeed = m.current.ID
	return m, m.queueCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.wave = nextWave(waveCount(msg.Width))
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height), len(m.upcoming))
		return m, nil
	case waveTick:
		m.wave = nextWave(waveCount(m.width))
		return m, waveCmd()
	case queueLoadedMsg:
		m.queuePending = false
		m.pendingSeed = ""
		seen := map[string]bool{m.current.ID: true}
		for _, v := range m.upcoming {
			seen[v.ID] = true
		}
		for _, v := range msg.videos {
			if v.ID == "" || seen[v.ID] {
				continue
			}
			seen[v.ID] = true
			m.upcoming = append(m.upcoming, v)
		}
		if len(m.upcoming) > maxQueue {
			m.upcoming = m.upcoming[:maxQueue]
		}
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height), len(m.upcoming))
		return m, nil
	case queueFailedMsg:
		m.queuePending = false
		m.pendingSeed = ""
		m.status = "autoplay: " + msg.err.Error()
		return m, nil
	case mpv.State:
		m.state = msg
		if msg.Ended {
			return m.advanceNext()
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.Type == tea.KeyEsc {
			m.showHelp = false
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlH:
		m.focus = focusMain
		return m, nil
	case tea.KeyCtrlL:
		m.focus = focusQueue
		return m, nil
	case tea.KeySpace:
		m.player.TogglePause()
		return m, nil
	case tea.KeyEnter:
		if m.focus == focusQueue {
			return m.playSelected()
		}
		return m, nil
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "p":
		m.player.TogglePause()
		return m, nil
	case "n":
		return m.advanceNext()
	case "]":
		m.player.Seek(5 * time.Second)
		return m, nil
	case "[":
		m.player.Seek(-5 * time.Second)
		return m, nil
	case "}":
		m.player.Seek(10 * time.Second)
		return m, nil
	case "{":
		m.player.Seek(-10 * time.Second)
		return m, nil
	case "+", "=":
		m.player.SetVolume(m.state.Volume + 5)
		return m, nil
	case "-", "_":
		m.player.SetVolume(m.state.Volume - 5)
		return m, nil
	case "?":
		m.showHelp = true
		return m, nil
	}

	if m.focus != focusQueue {
		switch msg.String() {
		case "h":
			m.player.Seek(-5 * time.Second)
			return m, nil
		case "l":
			m.player.Seek(5 * time.Second)
			return m, nil
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlJ:
		return m.moveQueue(+1)
	case tea.KeyCtrlK:
		return m.moveQueue(-1)
	}

	switch msg.String() {
	case "j":
		if m.queueIdx < len(m.upcoming)-1 {
			m.queueIdx++
		}
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height), len(m.upcoming))
		return m, nil
	case "k":
		if m.queueIdx > 0 {
			m.queueIdx--
		}
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height), len(m.upcoming))
		return m, nil
	case "d":
		return m.deleteFocused()
	}
	return m, nil
}

func (m Model) advanceNext() (tea.Model, tea.Cmd) {
	if len(m.upcoming) == 0 {
		m.status = "end of queue"
		return m, nil
	}
	v := m.upcoming[0]
	m.current = v
	m.state.Ended = false
	m.status = "▶ " + v.Title
	m.queueIdx = 0
	m.listScroll = 0
	m.player.Load(v.URL)
	var fetch tea.Cmd
	m, fetch = m.startQueueFetch()
	return m, fetch
}

func (m Model) playSelected() (tea.Model, tea.Cmd) {
	return m.playFrom(m.queueIdx)
}

// playFrom plays the song at index i, moving it to the front of the list.
func (m Model) playFrom(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.upcoming) {
		return m, nil
	}
	v := m.upcoming[i]
	m.upcoming = append(append([]search.Video{v}, m.upcoming[:i]...), m.upcoming[i+1:]...)
	m.current = v
	m.state.Ended = false
	m.status = "▶ " + v.Title
	m.queueIdx = 0
	m.listScroll = 0
	m.player.Load(v.URL)
	var fetch tea.Cmd
	m, fetch = m.startQueueFetch()
	return m, fetch
}

// deleteFocused removes the selected song from the upcoming queue.
func (m Model) deleteFocused() (tea.Model, tea.Cmd) {
	if m.queueIdx < 0 || m.queueIdx >= len(m.upcoming) {
		return m, nil
	}
	m.upcoming = append(append([]search.Video{}, m.upcoming[:m.queueIdx]...), m.upcoming[m.queueIdx+1:]...)
	if m.queueIdx >= len(m.upcoming) && m.queueIdx > 0 {
		m.queueIdx--
	}
	m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height), len(m.upcoming))
	return m, nil
}

// moveQueue shifts the selected song up (dir=-1) or down (dir=+1) in the queue.
func (m Model) moveQueue(dir int) (tea.Model, tea.Cmd) {
	if len(m.upcoming) < 2 {
		return m, nil
	}
	swap := m.queueIdx + dir
	if swap < 0 || swap >= len(m.upcoming) {
		return m, nil
	}
	m.upcoming[m.queueIdx], m.upcoming[swap] = m.upcoming[swap], m.upcoming[m.queueIdx]
	m.queueIdx = swap
	m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height), len(m.upcoming))
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "songer — loading…"
	}
	if m.showHelp {
		return m.helpView()
	}

	mainW := m.width*70/100 - 2
	if mainW < 10 {
		mainW = 10
	}
	sideW := m.width - mainW - 4
	if sideW < 10 {
		sideW = 10
	}

	main := m.mainView(mainW)
	side := m.sideView(sideW)
	return lipgloss.JoinHorizontal(lipgloss.Top, main, side)
}

func (m Model) mainView(w int) string {
	th := m.theme
	title := m.current.Title
	if title == "" {
		title = m.state.Title
	}
	if title == "" {
		title = "—"
	}

	sym := "▶"
	if m.state.Paused {
		sym = "⏸"
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Primary)).
		Bold(true).
		Render(sym + "  " + truncate(title, w-6))

	waveRows := m.height - 6
	if waveRows < 3 {
		waveRows = 3
	}
	if waveRows > 20 {
		waveRows = 20
	}
	wave := lipgloss.NewStyle().
		Width(w - 2).
		Align(lipgloss.Center).
		Render(waveView(m.wave, waveRows, th.Wave))

	barW := w - 16
	if barW < 10 {
		barW = 10
	}
	bar := progressBar(m.state.Position, m.state.Duration, barW, th.Progress, th.Track)
	times := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render(fmtDur(m.state.Position) + " / " + fmtDur(m.state.Duration))
	progressLine := lipgloss.NewStyle().Width(w).Render(bar + "  " + times)

	statusWord := "playing"
	if m.state.Paused {
		statusWord = "paused"
	}
	detailParts := []string{m.current.Channel, m.current.Views, statusWord}
	nonEmpty := []string{}
	for _, p := range detailParts {
		if strings.TrimSpace(p) != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	details := strings.Join(nonEmpty, " • ")
	if m.status != "" {
		details += "   " + m.status
	}
	detailsLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Muted)).
		Render(truncate(details, w-2))

	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		wave,
		progressLine,
		detailsLine,
	)
	return lipgloss.NewStyle().
		Width(w).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.borderColor(focusMain))).
		Render(content)
}

func (m Model) borderColor(f focus) string {
	if m.focus == f {
		return m.theme.Selection
	}
	return m.theme.Border
}

func (m Model) sideView(w int) string {
	th := m.theme
	total := len(m.upcoming)
	maxVis := listVisible(m.height)
	scroll := clampScroll(m.listScroll, m.queueIdx, maxVis, total)
	end := scroll + maxVis
	if end > total {
		end = total
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Secondary)).
		Bold(true).
		Render(fmt.Sprintf("UP NEXT (%d)", total))

	var items []string
	if total == 0 {
		items = append(items, lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("nothing queued"))
	}
	for i := scroll; i < end; i++ {
		v := m.upcoming[i]
		num := fmt.Sprintf("%2d.", i+1)
		title := truncate(v.Title, w-8)
		ch := truncate(v.Channel, w-8)
		line := fmt.Sprintf("%s %s\n   %s", num, title, ch)
		if i == m.queueIdx {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(th.Selection)).
				Bold(true).
				Render("▸ " + line)
		} else {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(th.Text)).
				Render(line)
		}
		items = append(items, line)
	}
	if end < total {
		items = append(items, lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render(fmt.Sprintf("▾ %d more…", total-end)))
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Muted)).
		Render("j/k • ctrl+j/k • d • enter")

	list := strings.Join(items, "\n\n")
	content := lipgloss.JoinVertical(lipgloss.Left, header, list, "\n", hint)
	return lipgloss.NewStyle().
		Width(w).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.borderColor(focusQueue))).
		Render(content)
}

func (m Model) helpView() string {
	th := m.theme
	rows := [][2]string{
		{"q", "quit"},
		{"space / p", "play / pause"},
		{"n", "next song"},
		{"]", "+5s"},
		{"[", "-5s"},
		{"}", "+10s"},
		{"{", "-10s"},
		{"h / l", "-5s / +5s (main area)"},
		{"j / k", "navigate queue"},
		{"ctrl+j / ctrl+k", "move song up / down"},
		{"d", "delete focused song"},
		{"enter", "play selected"},
		{"ctrl+h / ctrl+l", "focus 70 / 30 (main / queue)"},
		{"+ / -", "volume"},
		{"?", "this help"},
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Primary)).Render("SONGER — KEYS\n\n"))
	for _, r := range rows {
		key := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true).Render(fmt.Sprintf("%-10s", r[0]))
		b.WriteString(key + " " + r[1] + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("no mouse — everything is keys") + "\n")
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2).
		Render(b.String())
}

func waveCount(width int) int {
	n := width * 70 / 100 / 2
	if n < 4 {
		n = 4
	}
	if n > 140 {
		n = 140
	}
	return n
}

// listVisible is how many queue entries fit in the list panel at height h.
// Each entry is 2 lines + a blank separator.
func listVisible(h int) int {
	v := (h - 8) / 3
	if v < 1 {
		v = 1
	}
	if v > 24 {
		v = 24
	}
	return v
}

// clampScroll keeps the selection visible within the scroll window.
func clampScroll(scroll, idx, vis, total int) int {
	if vis < 1 {
		vis = 1
	}
	if idx < scroll {
		scroll = idx
	}
	if idx >= scroll+vis {
		scroll = idx - vis + 1
	}
	if max := total - vis; scroll > max {
		scroll = max
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}
