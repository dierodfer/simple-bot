package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	keystore "simple-bot/internal/database"
	"simple-bot/internal/models"
	"simple-bot/internal/utils"
	"simple-bot/internal/version"
)

// Dark theme colors (Tokyo Night palette).
var (
	cCyan      = lipgloss.Color("#7dcfff")
	cGreen     = lipgloss.Color("#9ece6a")
	cRed       = lipgloss.Color("#f7768e")
	cYellow    = lipgloss.Color("#e0af68")
	cMagenta   = lipgloss.Color("#bb9af7")
	cDim       = lipgloss.Color("#565f89")
	cFg        = lipgloss.Color("#c0caf5")
	cHighlight = lipgloss.Color("#33467c")
)

// Styles.
var (
	sTitle    = lipgloss.NewStyle().Foreground(cCyan).Bold(true)
	sDim      = lipgloss.NewStyle().Foreground(cDim)
	sHeader   = lipgloss.NewStyle().Foreground(cFg).Bold(true)
	sRow      = lipgloss.NewStyle().Foreground(cFg)
	sSelected = lipgloss.NewStyle().Background(cHighlight).Foreground(cFg).Bold(true)
	sProfit   = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	sLoss     = lipgloss.NewStyle().Foreground(cRed)
	sNeutral  = lipgloss.NewStyle().Foreground(cYellow)
	sCelest   = lipgloss.NewStyle().Foreground(cMagenta).Bold(true)
	sStatus   = lipgloss.NewStyle().Foreground(cCyan)
	sHelp     = lipgloss.NewStyle().Foreground(cDim)
	sOk       = lipgloss.NewStyle().Foreground(cGreen)
	sErr      = lipgloss.NewStyle().Foreground(cRed)
	sInputLbl = lipgloss.NewStyle().Foreground(cYellow).Bold(true)
	sInputBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(cYellow).
			Foreground(cFg).
			Padding(0, 1)
	sWeapon   = lipgloss.NewStyle().Foreground(cCyan).Bold(true)
	sCelBadge = lipgloss.NewStyle().Foreground(cMagenta).Bold(true)
	sBox      = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(cDim).
			Padding(1, 3)
)

// Messages.
type (
	itemMsg      models.MarketItem
	scanDoneMsg  struct{}
	buyResultMsg struct {
		index   int
		success bool
		message string
	}
	dbUpdateResultMsg struct {
		key   string
		value string
		err   error
	}
	dbRangeProgressMsg struct {
		startID int
		endID   int
		current int
		total   int
		failed  int
	}
	dbRangeDoneMsg struct {
		startID int
		endID   int
		updated int
		failed  int
		err     error
	}
)

type appState int
type dbInputMode int

const (
	stateMenu appState = iota
	stateScanning
	stateDB
)

const (
	dbInputNone dbInputMode = iota
	dbInputSearch
	dbInputRange
)

// Model is the bubbletea model for the interactive market UI.
type Model struct {
	state      appState
	menuIdx    int
	items      []models.MarketItem
	scanned    int
	cursor     int
	offset     int
	bought     map[int]string
	scanning   bool
	scanDone   bool
	scanStop   bool
	scanCancel context.CancelFunc
	itemCh     chan models.MarketItem
	spinner    spinner.Model
	width      int
	height     int
	httpClient *utils.HTTPClient
	store      keystore.KeyValueStore
	baseURL    string
	opts       utils.MarketOptions
	dbEntries  []keystore.Entry
	dbCursor   int
	dbOffset   int
	dbLimit    int
	dbPage     int
	dbTotal    int
	dbAllTotal int
	dbInput    dbInputMode
	dbInputVal string
	dbQuery    string
	dbMessage  string
	dbAction   string
	dbRangeCh  chan tea.Msg
}

// shouldShow returns true if an item passes the display filter:
// positive profit, or Celestial rarity with profit >= CelestialMaxLoss.
func shouldShow(item models.MarketItem) bool {
	diff := item.Diff()
	if diff > 0 {
		return true
	}
	return item.Rarity == "Celestial" && diff >= float64(utils.CelestialMaxLoss)
}

func newModel(httpClient *utils.HTTPClient, store keystore.KeyValueStore, baseURL string, opts utils.MarketOptions) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(cCyan)
	return Model{
		bought:     make(map[int]string),
		spinner:    sp,
		httpClient: httpClient,
		store:      store,
		baseURL:    baseURL,
		opts:       opts,
		dbLimit:    500,
	}
}

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

// --- Update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case itemMsg:
		m.scanned++
		item := models.MarketItem(msg)
		if shouldShow(item) {
			m.items = append(m.items, item)
		}
		return m, waitForItem(m.itemCh)
	case scanDoneMsg:
		m.scanCancel = nil
		m.scanning, m.scanDone = false, true
		return m, nil
	case buyResultMsg:
		if msg.success {
			m.bought[msg.index] = sOk.Render("✓ " + msg.message)
		} else {
			m.bought[msg.index] = sErr.Render("✗ " + msg.message)
		}
		return m, nil
	case dbUpdateResultMsg:
		if msg.err != nil {
			m.dbAction = sErr.Render("Update error: " + msg.err.Error())
			return m, nil
		}
		m.dbAction = sOk.Render(fmt.Sprintf("Updated %s -> %s", msg.key, msg.value))
		for i := range m.dbEntries {
			if m.dbEntries[i].Key == msg.key {
				m.dbEntries[i].Value = msg.value
				break
			}
		}
		return m, nil
	case dbRangeProgressMsg:
		progressPct := 0.0
		if msg.total > 0 {
			progressPct = (float64(msg.current) / float64(msg.total)) * 100
		}
		m.dbAction = sStatus.Render(fmt.Sprintf("Updating range %d-%d... %d/%d (%.0f%%) fail:%d", msg.startID, msg.endID, msg.current, msg.total, progressPct, msg.failed))
		return m, waitForDBRangeEvent(m.dbRangeCh)
	case dbRangeDoneMsg:
		m.dbRangeCh = nil
		if msg.err != nil {
			m.dbAction = sErr.Render("Range update error: " + msg.err.Error())
			return m, nil
		}
		m.loadDBEntries()
		m.dbAction = sOk.Render(fmt.Sprintf("Range %d-%d updated: %d ok, %d failed", msg.startID, msg.endID, msg.updated, msg.failed))
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.scanCancel != nil {
			m.scanCancel()
			m.scanCancel = nil
		}
		return m, tea.Quit
	}

	switch m.state {
	case stateMenu:
		return m.handleMenuKey(msg)
	case stateScanning:
		return m.handleScanKey(msg)
	case stateDB:
		return m.handleDBKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.menuIdx > 0 {
			m.menuIdx--
		}
	case "down", "j":
		if m.menuIdx < 2 {
			m.menuIdx++
		}
	case "enter":
		switch m.menuIdx {
		case 0:
			m.startScan(true)
			return m, waitForItem(m.itemCh)
		case 1:
			m.openDBView()
			return m, nil
		default:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) handleScanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.fixScroll()
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			m.fixScroll()
		}
	case "s":
		if m.scanning && m.scanCancel != nil {
			m.scanCancel()
			m.scanCancel = nil
			m.scanning = false
			m.scanStop = true
		}
	case "r":
		if !m.scanning {
			// Resume/restart scan from this view.
			m.startScan(false)
			return m, waitForItem(m.itemCh)
		}
	case "b":
		if len(m.items) > 0 {
			if _, done := m.bought[m.cursor]; !done {
				idx := m.cursor
				item := m.items[idx]
				return m, func() tea.Msg {
					ok, msg := utils.BuyItem(m.httpClient, m.baseURL, item)
					return buyResultMsg{index: idx, success: ok, message: msg}
				}
			}
		}
	case "esc":
		if m.scanCancel != nil {
			m.scanCancel()
			m.scanCancel = nil
		}
		m.state = stateMenu
	}

	return m, nil
}

func (m Model) handleDBKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dbInput != dbInputNone {
		switch msg.Type {
		case tea.KeyEnter:
			if m.dbInput == dbInputSearch {
				m.dbQuery = strings.TrimSpace(m.dbInputVal)
				m.searchDBByID()
				m.dbInputVal = ""
				m.dbInput = dbInputNone
				return m, nil
			}

			startID, endID, err := parseIDRange(strings.TrimSpace(m.dbInputVal))
			if err != nil {
				m.dbMessage = sErr.Render("Invalid range. Use format: 100-200")
				return m, nil
			}
			m.dbInput = dbInputNone
			m.dbInputVal = ""
			m.dbAction = sStatus.Render(fmt.Sprintf("Updating range %d-%d...", startID, endID))
			m.dbRangeCh = make(chan tea.Msg, 32)
			go runDBRangeUpdate(m.httpClient, m.store, m.baseURL, startID, endID, m.dbRangeCh)
			return m, waitForDBRangeEvent(m.dbRangeCh)
		case tea.KeyEsc:
			m.dbInput = dbInputNone
			m.dbInputVal = ""
		case tea.KeyBackspace, tea.KeyCtrlH:
			if len(m.dbInputVal) > 0 {
				m.dbInputVal = m.dbInputVal[:len(m.dbInputVal)-1]
			}
		default:
			if len(msg.Runes) == 1 && unicode.IsPrint(msg.Runes[0]) {
				m.dbInputVal += string(msg.Runes)
			}
		}
		return m, nil
	}

	if isSearchTrigger(msg) {
		m.dbInput = dbInputSearch
		m.dbInputVal = ""
		m.dbMessage = sStatus.Render("Search by ID fragment: type and press Enter")
		return m, nil
	}

	if keyHasRune(msg, 'u') {
		if len(m.dbEntries) == 0 {
			if strings.TrimSpace(m.dbQuery) != "" {
				m.dbQuery = ""
				m.dbPage = 0
				m.loadDBEntries()
				m.dbAction = sStatus.Render("No rows with current filter. Filter cleared automatically; press u again.")
				return m, nil
			}
			m.dbAction = sErr.Render("No entries available to update")
			return m, nil
		}
		entry := m.dbEntries[m.dbCursor]
		m.dbAction = sStatus.Render("Updating " + entry.Key + "...")
		return m, func() tea.Msg {
			value, err := utils.RefreshItemValue(m.httpClient, m.store, m.baseURL, entry.Key)
			if err != nil {
				return dbUpdateResultMsg{key: entry.Key, err: err}
			}
			return dbUpdateResultMsg{key: entry.Key, value: fmt.Sprintf("%.0f", value)}
		}
	}

	if keyHasRune(msg, 'x') {
		m.dbInput = dbInputRange
		m.dbInputVal = ""
		m.dbAction = sStatus.Render("Range update mode: type 100-200 and press Enter")
		return m, nil
	}

	if keyHasRune(msg, 'r') {
		m.loadDBEntries()
		return m, nil
	}

	if keyHasRune(msg, 'c') {
		m.dbQuery = ""
		m.dbPage = 0
		m.loadDBEntries()
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.dbCursor > 0 {
			m.dbCursor--
			m.fixDBScroll()
		}
	case "down", "j":
		if m.dbCursor < len(m.dbEntries)-1 {
			m.dbCursor++
			m.fixDBScroll()
		}
	case "left", "h":
		if m.dbPage > 0 {
			m.dbPage--
			m.loadDBEntries()
		}
	case "right", "l":
		if (m.dbPage+1)*m.dbLimit < m.dbTotal {
			m.dbPage++
			m.loadDBEntries()
		}
	case "esc":
		m.state = stateMenu
	}

	return m, nil
}

func (m *Model) openDBView() {
	m.state = stateDB
	m.dbPage = 0
	m.dbTotal = 0
	m.dbAllTotal = 0
	m.dbCursor = 0
	m.dbOffset = 0
	m.dbInput = dbInputNone
	m.dbInputVal = ""
	m.dbQuery = ""
	m.dbAction = ""
	m.loadDBEntries()
}

func (m *Model) loadDBEntries() {
	offset := m.dbPage * m.dbLimit
	query := strings.TrimSpace(m.dbQuery)

	var (
		entries []keystore.Entry
		err     error
		total   int
	)

	allTotal, err := m.store.Count()
	if err != nil {
		m.dbEntries = nil
		m.dbTotal = 0
		m.dbAllTotal = 0
		m.dbMessage = sErr.Render("Error loading local DB: " + err.Error())
		return
	}
	m.dbAllTotal = allTotal

	if query == "" {
		total = allTotal
		entries, err = m.store.ListPage(offset, m.dbLimit)
	} else {
		total, err = m.store.CountSearch(query)
		if err == nil {
			entries, err = m.store.SearchPage(query, offset, m.dbLimit)
		}
	}

	if err != nil {
		m.dbEntries = nil
		m.dbTotal = 0
		m.dbMessage = sErr.Render("Error loading local DB: " + err.Error())
		return
	}

	if offset > 0 && len(entries) == 0 {
		m.dbPage = max(0, m.dbPage-1)
		offset = m.dbPage * m.dbLimit
		if query == "" {
			entries, err = m.store.ListPage(offset, m.dbLimit)
		} else {
			entries, err = m.store.SearchPage(query, offset, m.dbLimit)
		}
		if err != nil {
			m.dbEntries = nil
			m.dbTotal = 0
			m.dbMessage = sErr.Render("Error loading local DB: " + err.Error())
			return
		}
	}

	m.dbEntries = entries
	m.dbTotal = total
	totalPages := 1
	if total > 0 {
		totalPages = (total + m.dbLimit - 1) / m.dbLimit
	}
	from := 0
	to := 0
	if total > 0 {
		from = offset + 1
		to = offset + len(entries)
	}
	if query == "" {
		m.dbMessage = sStatus.Render(fmt.Sprintf("%d total • page %d/%d • showing %d-%d", total, m.dbPage+1, totalPages, from, to))
	} else {
		m.dbMessage = sStatus.Render(fmt.Sprintf("Matches for %q: %d • page %d/%d • showing %d-%d", query, total, m.dbPage+1, totalPages, from, to))
	}
	if m.dbCursor >= len(m.dbEntries) {
		m.dbCursor = max(0, len(m.dbEntries)-1)
	}
	m.fixDBScroll()
}

func (m *Model) searchDBByID() {
	query := strings.TrimSpace(m.dbQuery)
	if query == "" {
		m.dbPage = 0
		m.loadDBEntries()
		return
	}
	m.dbPage = 0
	m.dbCursor = 0
	m.dbOffset = 0
	m.loadDBEntries()
}

func (m *Model) fixScroll() {
	vis := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
}

func (m *Model) fixDBScroll() {
	vis := m.visibleRows()
	if m.dbCursor < m.dbOffset {
		m.dbOffset = m.dbCursor
	} else if m.dbCursor >= m.dbOffset+vis {
		m.dbOffset = m.dbCursor - vis + 1
	}
}

func (m Model) visibleRows() int {
	// Keep headroom for headers/help so status lines are not clipped on short terminals.
	r := m.height - 14
	if r < 3 {
		r = 3
	}
	return r
}

func waitForItem(ch <-chan models.MarketItem) tea.Cmd {
	return func() tea.Msg {
		item, ok := <-ch
		if !ok {
			return scanDoneMsg{}
		}
		return itemMsg(item)
	}
}

func waitForDBRangeEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func runDBRangeUpdate(httpClient *utils.HTTPClient, store keystore.KeyValueStore, baseURL string, startID, endID int, out chan<- tea.Msg) {
	defer close(out)

	entries, err := store.ListNumericRange(startID, endID)
	if err != nil {
		out <- dbRangeDoneMsg{startID: startID, endID: endID, err: err}
		return
	}

	total := len(entries)
	if total == 0 {
		out <- dbRangeDoneMsg{startID: startID, endID: endID, updated: 0, failed: 0}
		return
	}

	const rangeUpdateMaxConcurrent = 12

	updated := 0
	failed := 0

	type rangeResult struct {
		key string
		err error
	}

	results := make(chan rangeResult, total)
	sem := make(chan struct{}, rangeUpdateMaxConcurrent)

	var wg sync.WaitGroup
	for _, entry := range entries {
		entry := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_, refreshErr := utils.RefreshItemValue(httpClient, store, baseURL, entry.Key)
			results <- rangeResult{key: entry.Key, err: refreshErr}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	current := 0
	for result := range results {
		current++
		if result.err != nil {
			failed++
		} else {
			updated++
		}

		out <- dbRangeProgressMsg{
			startID: startID,
			endID:   endID,
			current: current,
			total:   total,
			failed:  failed,
		}
	}

	out <- dbRangeDoneMsg{startID: startID, endID: endID, updated: updated, failed: failed}
}

// --- View ---

func (m Model) View() string {
	switch m.state {
	case stateMenu:
		return m.viewMenu()
	case stateDB:
		return m.viewDB()
	default:
		return m.viewScan()
	}
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(sTitle.Render("🤖 Simple Bot v"+version.AppVersion) + "\n")
	b.WriteString(sDim.Render("   Market Analyzer") + "\n\n")

	opts := []string{"⚔  Analyze Market", "🗄  Local DB", "✖  Quit"}
	for i, o := range opts {
		if i == m.menuIdx {
			b.WriteString("  " + sProfit.Render("> "+o) + "\n")
		} else {
			b.WriteString("    " + sDim.Render(o) + "\n")
		}
	}
	b.WriteString("\n" + sHelp.Render("  ↑↓ navigate • enter select • q quit"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, sBox.Render(b.String()))
}

func (m Model) viewScan() string {
	var b strings.Builder

	// Title + scan status
	title := sTitle.Render("🤖 Simple Bot v" + version.AppVersion)
	if m.scanning {
		title += "  " + m.spinner.View() + sStatus.Render(fmt.Sprintf(" Scanning... %d scanned • %d shown", m.scanned, len(m.items)))
	} else if m.scanStop {
		title += "  " + sErr.Render(fmt.Sprintf("■ Stopped • %d scanned • %d shown", m.scanned, len(m.items)))
	} else if m.scanDone {
		title += "  " + sOk.Render(fmt.Sprintf("✓ Complete • %d scanned • %d shown", m.scanned, len(m.items)))
	}
	b.WriteString(title + "\n\n")

	if len(m.items) == 0 {
		b.WriteString(sDim.Render("  Waiting for items...") + "\n")
	} else {
		// Table header
		hdr := fmt.Sprintf("  %-4s %-10s %-6s %-12s %10s %10s %12s",
			"#", "Type", "Level", "Rarity", "Cost", "Value", "Profit")
		b.WriteString(sHeader.Render(hdr) + "\n")
		b.WriteString(sDim.Render("  "+strings.Repeat("─", 72)) + "\n")

		// Visible rows
		vis := m.visibleRows()
		end := m.offset + vis
		if end > len(m.items) {
			end = len(m.items)
		}
		for i := m.offset; i < end; i++ {
			b.WriteString(m.renderRow(i) + "\n")
		}

		// Scroll indicator
		if len(m.items) > vis {
			b.WriteString(sDim.Render(fmt.Sprintf("\n  %d-%d of %d", m.offset+1, end, len(m.items))) + "\n")
		}
	}

	b.WriteString("\n" + sHelp.Render("  ↑↓ navigate • b buy • s stop scan • r resume scan • q quit"))
	return b.String()
}

func (m *Model) startScan(reset bool) {
	m.state = stateScanning
	m.scanning = true
	m.scanDone = false
	m.scanStop = false

	if m.scanCancel != nil {
		m.scanCancel()
		m.scanCancel = nil
	}

	if reset {
		m.items = nil
		m.scanned = 0
		m.cursor = 0
		m.offset = 0
		m.bought = make(map[int]string)
	}

	m.itemCh = make(chan models.MarketItem, 50)
	ctx, cancel := context.WithCancel(context.Background())
	m.scanCancel = cancel
	go utils.ScanMarket(ctx, m.httpClient, m.store, m.baseURL, m.opts, m.itemCh)
}

func (m Model) viewDB() string {
	var left strings.Builder
	left.WriteString(sTitle.Render("🗄 Local DB") + "\n")

	if m.dbAction != "" {
		left.WriteString(m.dbAction + "\n")
	}
	if m.dbMessage != "" {
		left.WriteString(m.dbMessage + "\n")
	}
	if len(m.dbEntries) == 0 && strings.TrimSpace(m.dbQuery) != "" {
		left.WriteString(sErr.Render("Filter is hiding all rows. Press c to clear filter.") + "\n")
	}
	left.WriteString("\n")

	if m.dbInput != dbInputNone {
		if m.dbInput == dbInputSearch {
			left.WriteString(sInputLbl.Render("SEARCH MODE ACTIVE") + "\n")
			left.WriteString(sInputBox.Render("ID contains: "+m.dbInputVal+"|") + "\n\n")
		} else {
			left.WriteString(sInputLbl.Render("RANGE UPDATE MODE ACTIVE") + "\n")
			left.WriteString(sInputBox.Render("Range (start-end): "+m.dbInputVal+"|") + "\n\n")
		}
	}

	hdr := fmt.Sprintf("  %-4s %-20s %14s", "#", "ID", "Stored Value")
	left.WriteString(sHeader.Render(hdr) + "\n")
	left.WriteString(sDim.Render("  "+strings.Repeat("─", 43)) + "\n")

	if len(m.dbEntries) == 0 {
		left.WriteString(sDim.Render("  No local entries") + "\n")
	} else {
		vis := m.visibleRows()
		end := m.dbOffset + vis
		if end > len(m.dbEntries) {
			end = len(m.dbEntries)
		}
		for i := m.dbOffset; i < end; i++ {
			ptr := "  "
			if i == m.dbCursor {
				ptr = "> "
			}
			row := fmt.Sprintf("%s%-4d %-20s %14s", ptr, i+1, m.dbEntries[i].Key, m.dbEntries[i].Value)
			if i == m.dbCursor {
				left.WriteString(sSelected.Render(row) + "\n")
			} else {
				left.WriteString(sRow.Render(row) + "\n")
			}
		}
	}

	var right strings.Builder
	right.WriteString(sHeader.Render("Selected Entry") + "\n")
	right.WriteString(sDim.Render(strings.Repeat("─", 28)) + "\n")
	if len(m.dbEntries) > 0 && m.dbCursor < len(m.dbEntries) {
		entry := m.dbEntries[m.dbCursor]
		right.WriteString(sDim.Render("ID") + "\n")
		right.WriteString(sRow.Render(entry.Key) + "\n\n")
		right.WriteString(sDim.Render("Stored value") + "\n")
		right.WriteString(sProfit.Render(entry.Value) + "\n\n")
		right.WriteString(sDim.Render("Row in page") + "\n")
		right.WriteString(sRow.Render(fmt.Sprintf("%d/%d", m.dbCursor+1, len(m.dbEntries))) + "\n")
	} else {
		right.WriteString(sDim.Render("No selected row") + "\n")
	}

	right.WriteString("\n" + sHeader.Render("Filter") + "\n")
	right.WriteString(sDim.Render(strings.Repeat("─", 28)) + "\n")
	if strings.TrimSpace(m.dbQuery) == "" {
		right.WriteString(sDim.Render("none") + "\n")
	} else {
		right.WriteString(sRow.Render(fmt.Sprintf("contains: %q", m.dbQuery)) + "\n")
	}

	right.WriteString("\n" + sHeader.Render("Shortcuts") + "\n")
	right.WriteString(sDim.Render(strings.Repeat("─", 28)) + "\n")
	right.WriteString(sHelp.Render("u  update selected") + "\n")
	right.WriteString(sHelp.Render("x  update range") + "\n")
	right.WriteString(sHelp.Render("/  search") + "\n")
	right.WriteString(sHelp.Render("c  clear filter") + "\n")
	right.WriteString(sHelp.Render("r  reload") + "\n")

	leftWidth := m.width - 34
	if leftWidth < 70 {
		leftWidth = 70
	}
	rightPanel := lipgloss.NewStyle().
		Width(30).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(cDim).
		Padding(0, 1).
		Render(right.String())

	leftPanel := lipgloss.NewStyle().Width(leftWidth).Render(left.String())
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	return body
}

func parseIDRange(raw string) (int, int, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	startID, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start id: %w", err)
	}
	endID, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end id: %w", err)
	}

	if startID > endID {
		startID, endID = endID, startID
	}

	return startID, endID, nil
}

func isSearchTrigger(msg tea.KeyMsg) bool {
	key := msg.String()
	if key == "/" || key == "shift+7" || key == "?" || key == "f" || key == "F" {
		return true
	}
	if strings.Contains(key, "/") {
		return true
	}
	for _, r := range msg.Runes {
		if r == '/' || r == '?' || r == 'f' || r == 'F' {
			return true
		}
	}
	return false
}

func keyHasRune(msg tea.KeyMsg, expected rune) bool {
	for _, r := range msg.Runes {
		if unicode.ToLower(r) == unicode.ToLower(expected) {
			return true
		}
	}
	return false
}

func (m Model) renderRow(i int) string {
	item := m.items[i]
	diff := item.Diff()
	sel := i == m.cursor

	ptr := "  "
	if sel {
		ptr = "> "
	}

	// Plain text columns
	line := fmt.Sprintf("%s%-4d %-10s %-6s %-12s %10s %10s",
		ptr, i+1, item.Type, item.Level, item.Rarity,
		fmtGold(item.Gold), fmtGold(item.Value))

	// Profit with sign
	pStr := fmtGold(diff)
	if diff > 0 {
		pStr = "+" + pStr
	}
	profit := fmt.Sprintf(" %12s", pStr)

	// Buy status
	status := ""
	if s, ok := m.bought[i]; ok {
		status = "  " + s
	}
	if item.IsGoodWeaponDeal() {
		status += "  " + sWeapon.Render("WEAPON: ratio good")
	}
	if item.IsGoodCelestialDeal() {
		status += "  " + sCelBadge.Render("CELESTIAL: ratio good")
	}

	// Selected row gets full highlight
	if sel {
		return sSelected.Render(line+profit) + status
	}

	// Color profit column
	var ps lipgloss.Style
	switch {
	case diff > float64(utils.ProfitThresholdBuy):
		ps = sProfit
	case diff > 0:
		ps = sNeutral
	default:
		ps = sLoss
	}

	// Color row by rarity
	var rs lipgloss.Style
	if item.Rarity == "Celestial" {
		rs = sCelest
	} else {
		rs = sRow
	}

	return rs.Render(line) + ps.Render(profit) + status
}

func fmtGold(n float64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%.0f", n)
	var out strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(c)
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}

// Run starts the interactive TUI application.
func Run(httpClient *utils.HTTPClient, store keystore.KeyValueStore, baseURL string, opts utils.MarketOptions) error {
	p := tea.NewProgram(newModel(httpClient, store, baseURL, opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
