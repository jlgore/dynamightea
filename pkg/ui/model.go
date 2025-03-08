package ui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jg/dynamightea/pkg/db"
)

type viewMode string

const (
	tableListMode viewMode = "tables"
	tableViewMode viewMode = "table"
	indexViewMode viewMode = "index"
)

// Model represents the UI state
type Model struct {
	tables       []string
	selectedTable int
	viewMode     viewMode
	tableData    *db.TableInfo
	width        int
	height       int
	loading      bool
	error        error
	client       *db.DynamoClient
}

// NewModel creates a new UI model
func NewModel() Model {
	return Model{
		tables:       []string{},
		selectedTable: 0,
		viewMode:     tableListMode,
		loading:      true,
		client:       db.NewDynamoClient(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return loadTables
}

// Update handles messages and user input
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Cycle through view modes
			switch m.viewMode {
			case tableListMode:
				if len(m.tables) > 0 {
					m.viewMode = tableViewMode
					return m, loadTableInfo(m.tables[m.selectedTable])
				}
			case tableViewMode:
				m.viewMode = indexViewMode
			case indexViewMode:
				m.viewMode = tableListMode
			}
		case "up", "k":
			if m.selectedTable > 0 {
				m.selectedTable--
			}
		case "down", "j":
			if m.selectedTable < len(m.tables)-1 {
				m.selectedTable++
			}
		case "enter":
			if m.viewMode == tableListMode && len(m.tables) > 0 {
				m.viewMode = tableViewMode
				return m, loadTableInfo(m.tables[m.selectedTable])
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tablesLoadedMsg:
		m.tables = msg.tables
		m.loading = false
	case tableInfoLoadedMsg:
		m.tableData = msg.tableInfo
		m.loading = false
	case errorMsg:
		m.error = msg.err
		m.loading = false
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.loading {
		return "Loading..."
	}

	if m.error != nil {
		return "Error: " + m.error.Error()
	}

	var content string
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFF00")).Render

	switch m.viewMode {
	case tableListMode:
		content = titleStyle("DynamoDB Tables") + "\n\n"
		for i, table := range m.tables {
			if i == m.selectedTable {
				content += "> " + table + "\n"
			} else {
				content += "  " + table + "\n"
			}
		}
		content += "\n[↑/↓]: Navigate [Enter]: Select [Tab]: Switch View [q]: Quit"
	
	case tableViewMode:
		if m.tableData == nil {
			content = "Loading table data..."
		} else {
			content = titleStyle("Table: " + m.tableData.TableName) + "\n\n"
			content += "Primary Key:\n"
			for _, attr := range m.tableData.KeySchema {
				content += "  " + attr.AttributeName + " (" + attr.KeyType + ")\n"
			}
			content += "\nAttributes:\n"
			for name, attrType := range m.tableData.AttributeDefinitions {
				content += "  " + name + ": " + attrType + "\n"
			}
			content += "\n[Tab]: View Indexes [q]: Quit"
		}
	
	case indexViewMode:
		if m.tableData == nil {
			content = "Loading table data..."
		} else {
			content = titleStyle("Indexes: " + m.tableData.TableName) + "\n\n"
			
			// GSIs
			content += lipgloss.NewStyle().Bold(true).Render("Global Secondary Indexes:") + "\n"
			if len(m.tableData.GSIs) == 0 {
				content += "  None\n"
			} else {
				for _, gsi := range m.tableData.GSIs {
					content += "  " + gsi.IndexName + ":\n"
					for _, key := range gsi.KeySchema {
						content += "    " + key.AttributeName + " (" + key.KeyType + ")\n"
					}
					content += "\n"
				}
			}
			
			// LSIs
			content += lipgloss.NewStyle().Bold(true).Render("Local Secondary Indexes:") + "\n"
			if len(m.tableData.LSIs) == 0 {
				content += "  None\n"
			} else {
				for _, lsi := range m.tableData.LSIs {
					content += "  " + lsi.IndexName + ":\n"
					for _, key := range lsi.KeySchema {
						content += "    " + key.AttributeName + " (" + key.KeyType + ")\n"
					}
					content += "\n"
				}
			}
			content += "\n[Tab]: View Tables [q]: Quit"
		}
	}

	return content
}

// Messages
type tablesLoadedMsg struct {
	tables []string
}

type tableInfoLoadedMsg struct {
	tableInfo *db.TableInfo
}

type errorMsg struct {
	err error
}

// Commands
func loadTables() tea.Msg {
	client := db.NewDynamoClient()
	tables, err := client.ListTables()
	if err != nil {
		return errorMsg{err}
	}
	return tablesLoadedMsg{tables}
}

func loadTableInfo(tableName string) tea.Cmd {
	return func() tea.Msg {
		client := db.NewDynamoClient()
		tableInfo, err := client.DescribeTable(tableName)
		if err != nil {
			return errorMsg{err}
		}
		return tableInfoLoadedMsg{tableInfo}
	}
}