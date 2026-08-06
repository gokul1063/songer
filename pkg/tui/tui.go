package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"

	"songer/pkg/autoplay"
	"songer/pkg/config"
	"songer/pkg/library"
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

type page int

const (
	pageMain page = iota
	pagePlaylist
	pageCount = 2
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
	lib          *library.Library
	fav          bool
	liked        bool
	page         page
}

const maxQueue = 60

func Run(ctx context.Context, player *mpv.Player, current search.Video, theme config.Theme, perNode, depth int, lib *library.Library) error {
	m := Model{
		player:  player,
		theme:   theme,
		ctx:     ctx,
		current: current,
		status:  "▶ " + current.Title,
		perNode: perNode,
		depth:   depth,
		focus:   focusQueue,
		state:   player.State(),
		lib:     lib,
	}
	if lib != nil {
		_ = lib.RecordPlay(current)
		m.fav = lib.IsFavorite(current.ID)
		m.liked = lib.IsLiked(current.ID)
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
		m.wave = nextWave(waveCount(msg.Width*70/100))
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height-2), len(m.upcoming))
		return m, nil
	case waveTick:
		m.wave = nextWave(waveCount(m.width*70/100))
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
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height-2), len(m.upcoming))
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
		if msg.String() == "?" || msg.String() == "/" || msg.Type == tea.KeyEsc {
			m.showHelp = false
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyTab:
		if msg.Alt {
			m.page = (m.page - 1 + pageCount) % pageCount
		} else {
			m.page = (m.page + 1) % pageCount
		}
		return m, nil
	case tea.KeyShiftTab:
		m.page = (m.page - 1 + pageCount) % pageCount
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
	case "?", "/":
		m.showHelp = !m.showHelp
		return m, nil
	case "f":
		return m.toggleFavorite()
	case "g":
		return m.toggleLiked()
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
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height-2), len(m.upcoming))
		return m, nil
	case "k":
		if m.queueIdx > 0 {
			m.queueIdx--
		}
		m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height-2), len(m.upcoming))
		return m, nil
	case "d":
		return m.deleteFocused()
	}
	return m, nil
}

// onPlayed records a play in history and refreshes the fav/liked flags.
func (m Model) onPlayed(v search.Video) Model {
	if m.lib != nil {
		_ = m.lib.RecordPlay(v)
		m.fav = m.lib.IsFavorite(v.ID)
		m.liked = m.lib.IsLiked(v.ID)
	}
	return m
}

func (m Model) advanceNext() (tea.Model, tea.Cmd) {
	if len(m.upcoming) == 0 {
		m.status = "end of queue"
		return m, nil
	}
	v := m.upcoming[0]
	m.upcoming = m.upcoming[1:]
	m.current = v
	m = m.onPlayed(v)
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

// playFrom plays the song at index i, removing it from the queue entirely.
func (m Model) playFrom(i int) (tea.Model, tea.Cmd) {
	if i < 0 || i >= len(m.upcoming) {
		return m, nil
	}
	v := m.upcoming[i]
	m.upcoming = append(append([]search.Video{}, m.upcoming[:i]...), m.upcoming[i+1:]...)
	m.current = v
	m = m.onPlayed(v)
	m.state.Ended = false
	m.status = "▶ " + v.Title
	if i >= len(m.upcoming) {
		i = len(m.upcoming) - 1
	}
	if i < 0 {
		i = 0
	}
	m.queueIdx = i
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
	m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height-2), len(m.upcoming))
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
	m.listScroll = clampScroll(m.listScroll, m.queueIdx, listVisible(m.height-2), len(m.upcoming))
	return m, nil
}

func (m Model) toggleFavorite() (tea.Model, tea.Cmd) {
	if m.lib == nil {
		return m, nil
	}
	if m.fav {
		_ = m.lib.RemoveFavorite(m.current.ID)
		m.fav = false
		m.status = "removed from favorites"
	} else {
		_ = m.lib.AddFavorite(m.current)
		m.fav = true
		m.status = "♥ added to favorites"
	}
	return m, nil
}

func (m Model) toggleLiked() (tea.Model, tea.Cmd) {
	if m.lib == nil {
		return m, nil
	}
	if m.liked {
		_ = m.lib.RemoveLiked(m.current.ID)
		m.liked = false
		m.status = "removed from liked"
	} else {
		_ = m.lib.AddLiked(m.current)
		m.liked = true
		m.status = "★ added to liked"
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "songer — loading…"
	}
	base := m.baseView()
	if m.showHelp {
		box := m.helpBox()
		boxLines := strings.Split(box, "\n")
		boxW := 0
		for _, l := range boxLines {
			if w := ansi.PrintableRuneWidth(l); w > boxW {
				boxW = w
			}
		}
		boxH := len(boxLines)
		x := (m.width - boxW) / 2
		if x < 0 {
			x = 0
		}
		y := (m.height - boxH) / 2
		if y < 1 {
			y = 1
		}
		return overlayWindow(base, box, x, y)
	}
	return base
}

func (m Model) baseView() string {
	bodyH := m.height - 2
	if bodyH < 4 {
		bodyH = 4
	}
	return lipgloss.JoinVertical(lipgloss.Top,
		m.headerView(m.width),
		m.bodyView(m.width, bodyH),
		m.footerView(m.width),
	)
}

func (m Model) bodyView(w, h int) string {
	mainW := w*70/100 - 2
	if mainW < 10 {
		mainW = 10
	}
	sideW := w - mainW - 4
	if sideW < 10 {
		sideW = 10
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, m.mainView(mainW, h), m.sideView(sideW, h))
}

func (m Model) headerView(w int) string {
	th := m.theme
	state := "playing"
	if m.state.Paused {
		state = "paused"
	}
	focus := "list"
	if m.focus == focusMain {
		focus = "main"
	}
	left := "SONGER"
	right := fmt.Sprintf("▸ %s • vol %d%% • [%s]", state, m.state.Volume, focus)
	pad := w - runewidth.StringWidth(left) - runewidth.StringWidth(right)
	if pad < 1 {
		pad = 1
	}
	line := left + strings.Repeat(" ", pad) + right
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(th.Header)).
		Background(lipgloss.Color(th.HeaderBg)).
		Width(w).
		Render(line)
}

func (m Model) footerView(w int) string {
	th := m.theme
	hints := []string{"q quit", "space pause", "n next", "] +5s", "[ -5s", "f fav", "g like"}
	if m.focus == focusQueue {
		hints = append(hints, "j/k nav", "ctrl+j/k move", "d delete", "enter play", "ctrl+h main")
	} else {
		hints = append(hints, "h/l seek", "ctrl+l list")
	}
	hints = append(hints, "? help")
	line := truncate(strings.Join(hints, "   "), w)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Footer)).
		Background(lipgloss.Color(th.FooterBg)).
		Width(w).
		Render(line)
}

func (m Model) mainView(w, h int) string {
	th := m.theme
	var content string
	if m.page == pagePlaylist {
		content = m.playlistView(w)
	} else {
		content = m.nowPlayingContent(w, h)
	}

	border := lipgloss.RoundedBorder()
	borderColor := th.Border
	if m.focus == focusMain {
		border = lipgloss.DoubleBorder()
		borderColor = th.Selection
	}
	return lipgloss.NewStyle().
		Width(w).
		Height(h - 2).
		Border(border).
		BorderForeground(lipgloss.Color(borderColor)).
		Render(content)
}

func (m Model) nowPlayingContent(w, h int) string {
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

	contentH := h - 2
	waveRows := contentH * 45 / 100
	if waveRows < 3 {
		waveRows = 3
	}
	if waveRows > 22 {
		waveRows = 22
	}
	wave := lipgloss.NewStyle().
		Width(w - 2).
		Align(lipgloss.Center).
		Render(waveView(m.wave, waveRows, th.Wave, th.WaveAlt))

	pct := 0.0
	if m.state.Duration > 0 {
		pct = m.state.Position.Seconds() / m.state.Duration.Seconds()
	}
	pctStr := fmt.Sprintf("%3.0f%%", pct*100)
	barW := w - 30
	if barW < 10 {
		barW = 10
	}
	bar := progressBar(m.state.Position, m.state.Duration, barW, th.Progress, th.Track, th.Thumb)
	progressLine := pctStr + "  " + bar + "  " + fmtDur(m.state.Position) + " / " + fmtDur(m.state.Duration)
	if m.state.Duration > 0 {
		progressLine += "  -" + fmtDur(m.state.Duration-m.state.Position)
	}
	progressLine = lipgloss.NewStyle().Width(w).Render(progressLine)

	// song name under the bar
	marks := ""
	if m.fav {
		marks += "♥ "
	}
	if m.liked {
		marks += "★ "
	}
	songLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Primary)).
		Bold(true).
		Render(sym + "  " + marks + truncate(title, w-8))

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

	used := waveRows + 6 // bar + name + details + 3 blank spacers
	topPad := (contentH - used) / 2
	if topPad < 0 {
		topPad = 0
	}
	topSpacer := lipgloss.NewStyle().Height(topPad).Render("")
	return lipgloss.JoinVertical(lipgloss.Center,
		topSpacer,
		wave,
		"",
		progressLine,
		"",
		songLine,
		"",
		detailsLine,
	)
}

// playlistView renders the playlist page: favorite/liked boxes, custom
// playlists, and a trailing "new playlist" (+) box.
func (m Model) playlistView(w int) string {
	th := m.theme
	var boxes []string
	boxes = append(boxes, playlistBox("★", "Favorites", th))
	boxes = append(boxes, playlistBox("♥", "Liked", th))
	if m.lib != nil {
		for _, name := range m.lib.PlaylistNames() {
			boxes = append(boxes, playlistBox("♺", name, th))
		}
	}
	boxes = append(boxes, playlistBox("＋", "New", th))

	const boxW = 20
	perRow := w / boxW
	if perRow < 1 {
		perRow = 1
	}
	var rows []string
	for i := 0; i < len(boxes); i += perRow {
		end := i + perRow
		if end > len(boxes) {
			end = len(boxes)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, boxes[i:end]...))
	}
	grid := lipgloss.JoinVertical(lipgloss.Center, rows...)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Secondary)).Render("PLAYLISTS")
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("tab switches view • + adds a playlist")
	return lipgloss.NewStyle().
		Width(w).
		Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Center, title, "\n", grid, "\n", hint))
}

func playlistBox(symbol, label string, th config.Theme) string {
	body := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(th.Primary)).
		Width(14).
		Align(lipgloss.Center).
		Render(symbol + "\n" + truncate(label, 14))
	return lipgloss.NewStyle().
		Width(16).
		Height(4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.Border)).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(body)
}

func (m Model) sideView(w, h int) string {
	th := m.theme
	total := len(m.upcoming)
	maxVis := listVisible(h)
	scroll := clampScroll(m.listScroll, m.queueIdx, maxVis, total)
	end := scroll + maxVis
	if end > total {
		end = total
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Secondary)).
		Bold(true).
		Render(fmt.Sprintf("UP NEXT (%d)", total))
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Border)).Render(strings.Repeat("─", w-2))

	var items []string
	if total == 0 {
		items = append(items, lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("nothing queued"))
	}
	for i := scroll; i < end; i++ {
		v := m.upcoming[i]
		num := fmt.Sprintf("%2d.", i+1)
		title := truncate(v.Title, w-8)
		if v.Duration != "" {
			title = truncate(v.Title, w-14) + "  " + lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render(v.Duration)
		}
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

	list := strings.Join(items, "\n\n")
	content := lipgloss.JoinVertical(lipgloss.Left, header, sep, list)

	border := lipgloss.RoundedBorder()
	borderColor := th.Border
	if m.focus == focusQueue {
		border = lipgloss.DoubleBorder()
		borderColor = th.Selection
	}
	return lipgloss.NewStyle().
		Width(w).
		Height(h - 2).
		Border(border).
		BorderForeground(lipgloss.Color(borderColor)).
		Render(content)
}

func (m Model) helpBox() string {
	th := m.theme
	rows := [][2]string{
		{"q / ctrl+c", "quit"},
		{"space / p", "play / pause"},
		{"n", "next song (plays list head)"},
		{"] / [", "+5s / -5s"},
		{"} / {", "+10s / -10s"},
		{"h / l", "-5s / +5s (main area)"},
		{"j / k", "navigate queue"},
		{"ctrl+j / ctrl+k", "move song up / down"},
		{"d", "delete focused song"},
		{"enter", "play selected"},
		{"ctrl+h / ctrl+l", "focus main / focus list"},
		{"+ / -", "volume"},
		{"f", "add current to favorites"},
		{"g", "add current to liked"},
		{"tab / shift+tab", "switch main / playlist page"},
		{"? / /", "this help"},
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(th.Primary)).Render("CONTROLS\n"))
	for _, r := range rows {
		key := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Accent)).Bold(true).Render(fmt.Sprintf("%-16s", r[0]))
		b.WriteString(key + r[1] + "\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(th.Muted)).Render("? toggles this window"))
	return lipgloss.NewStyle().
		Width(52).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.Selection)).
		Padding(1, 2).
		Render(b.String())
}

// waveCount is how many wave bars fit the main panel width, with small side padding.
func waveCount(mainW int) int {
	n := mainW - 8
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
