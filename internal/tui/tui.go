package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(1, 0)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	priceStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00"))

	cheapestStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Background(lipgloss.Color("#2E2E2E")).
			Padding(0, 1)

	tableStyle = table.DefaultStyles()

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)
)

type Model struct {
	textInput  textinput.Model
	spinner    spinner.Model
	table      table.Model
	results    []PriceResult
	searching  bool
	error      string
	success    string
	mode       string // "input", "results", "register"
	width      int
	height     int
}

type PriceResult struct {
	Registrar string
	Price     float64
	Renewal   float64
	Available bool
	Features  string
}

type searchMsg struct {
	domain  string
	results []PriceResult
	err     error
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.Focus()
	ti.CharLimit = 253
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	columns := []table.Column{
		{Title: "Registrar", Width: 15},
		{Title: "Price", Width: 12},
		{Title: "Renewal", Width: 12},
		{Title: "Status", Width: 12},
		{Title: "Features", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	t.SetStyles(tableStyle)

	return Model{
		textInput: ti,
		spinner:   s,
		table:     t,
		mode:      "input",
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			if m.mode == "input" && !m.searching {
				domain := m.textInput.Value()
				if domain != "" {
					m.searching = true
					m.error = ""
					m.success = ""
					return m, tea.Batch(
						m.spinner.Tick,
						searchDomain(domain),
					)
				}
			} else if m.mode == "results" {
				// Register selected domain
				selected := m.table.Cursor()
				if selected < len(m.results) {
					m.success = fmt.Sprintf("Registering with %s...", m.results[selected].Registrar)
					// TODO: Actually register the domain
					time.Sleep(1 * time.Second)
					m.success = fmt.Sprintf("Domain registered successfully with %s!", m.results[selected].Registrar)
				}
			}
			return m, nil

		case "esc":
			if m.mode == "results" {
				m.mode = "input"
				m.results = nil
				m.textInput.SetValue("")
				return m, nil
			}
		}

	case searchMsg:
		m.searching = false
		if msg.err != nil {
			m.error = msg.err.Error()
			return m, nil
		}

		m.results = msg.results
		m.mode = "results"

		// Update table with results
		rows := []table.Row{}
		for _, r := range m.results {
			status := "❌ Unavailable"
			if r.Available {
				status = "✅ Available"
			}
			rows = append(rows, table.Row{
				r.Registrar,
				fmt.Sprintf("$%.2f", r.Price),
				fmt.Sprintf("$%.2f/yr", r.Renewal),
				status,
				r.Features,
			})
		}
		m.table.SetRows(rows)

		return m, nil

	case spinner.TickMsg:
		if m.searching {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update inputs
	if m.mode == "input" {
		m.textInput, cmd = m.textInput.Update(msg)
	} else if m.mode == "results" {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	var s strings.Builder

	// Header
	s.WriteString(titleStyle.Render("🌐 regctl - Domain Registration CLI"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render("Compare prices across 897 TLDs and register at the cheapest registrar"))
	s.WriteString("\n\n")

	// Error message
	if m.error != "" {
		s.WriteString(errorStyle.Render("❌ Error: " + m.error))
		s.WriteString("\n\n")
	}

	// Success message
	if m.success != "" {
		s.WriteString(successStyle.Render("✅ " + m.success))
		s.WriteString("\n\n")
	}

	switch m.mode {
	case "input":
		// Search input
		s.WriteString("Enter domain to search:\n\n")
		s.WriteString(m.textInput.View())
		s.WriteString("\n\n")

		if m.searching {
			s.WriteString(m.spinner.View() + " Checking domain availability and comparing prices...\n")
		} else {
			s.WriteString(subtitleStyle.Render("Press Enter to search, q to quit"))
		}

	case "results":
		// Results table
		s.WriteString("📊 Price Comparison Results:\n\n")
		s.WriteString(m.table.View())
		s.WriteString("\n\n")

		// Find cheapest
		cheapest := ""
		cheapestPrice := 999999.0
		for _, r := range m.results {
			if r.Available && r.Price < cheapestPrice {
				cheapestPrice = r.Price
				cheapest = r.Registrar
			}
		}

		if cheapest != "" {
			s.WriteString(cheapestStyle.Render(fmt.Sprintf("💰 Best Deal: %s at $%.2f", cheapest, cheapestPrice)))
			s.WriteString("\n\n")
			s.WriteString(subtitleStyle.Render("Press Enter to register, Esc to search again, q to quit"))
		} else {
			s.WriteString(errorStyle.Render("❌ Domain not available at any registrar"))
			s.WriteString("\n\n")
			s.WriteString(subtitleStyle.Render("Press Esc to search again, q to quit"))
		}
	}

	return s.String()
}

func searchDomain(domain string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Actually call the API
		// Simulating API call with mock data
		time.Sleep(2 * time.Second)

		results := []PriceResult{
			{
				Registrar: "Spaceship",
				Price:     8.88,
				Renewal:   10.99,
				Available: true,
				Features:  "Free WHOIS privacy, DNS",
			},
			{
				Registrar: "Porkbun",
				Price:     7.95,
				Renewal:   9.99,
				Available: true,
				Features:  "Free WHOIS, Email forwarding",
			},
			{
				Registrar: "Cloudflare",
				Price:     9.15,
				Renewal:   9.15,
				Available: true,
				Features:  "At-cost pricing, Fast DNS",
			},
			{
				Registrar: "Value Domain",
				Price:     12.00,
				Renewal:   12.00,
				Available: false,
				Features:  "JP support, Credit card",
			},
		}

		return searchMsg{
			domain:  domain,
			results: results,
		}
	}
}

func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
