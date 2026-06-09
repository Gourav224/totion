package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const noteExtension = ".md"

var (
	vaultDir string

	ink         = lipgloss.Color("#111827")
	muted       = lipgloss.Color("#6B7280")
	blue        = lipgloss.Color("#2563EB")
	green       = lipgloss.Color("#059669")
	red         = lipgloss.Color("#DC2626")
	border      = lipgloss.Color("#D1D5DB")
	white       = lipgloss.Color("#FFFFFF")
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(white).Background(blue).Padding(0, 2).MarginBottom(1)
	helpStyle   = lipgloss.NewStyle().Foreground(muted).MarginTop(1)
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(border).Padding(1, 2)
	labelStyle  = lipgloss.NewStyle().Bold(true).Foreground(ink)
	statusStyle = lipgloss.NewStyle().Foreground(green).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(red).MarginTop(1)
	cursorStyle = lipgloss.NewStyle().Foreground(blue)
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("get home directory:", err)
	}
	vaultDir = filepath.Join(homeDir, ".totion")
}

type noteItem struct {
	filename string
	modified string
	modTime  int64
}

func (n noteItem) Title() string       { return n.filename }
func (n noteItem) FilterValue() string { return n.filename }
func (n noteItem) Description() string { return "Updated " + n.modified }

type appModel struct {
	width  int
	height int

	newNoteInput       textinput.Model
	newNoteInputActive bool
	noteTextArea       textarea.Model
	noteList           list.Model
	noteListActive     bool
	activeNotePath     string
	statusMessage      string
	errorMessage       string
}

func (m appModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentWidth := max(20, msg.Width-8)
		contentHeight := max(8, msg.Height-12)

		m.noteList.SetSize(contentWidth, contentHeight)
		m.noteTextArea.SetWidth(contentWidth)
		m.noteTextArea.SetHeight(contentHeight)
		m.newNoteInput.Width = max(20, msg.Width-12)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "ctrl+n":
			m.clearMessages()
			m.newNoteInputActive = true
			m.noteListActive = false
			m.newNoteInput.Focus()
			return m, m.newNoteInput.Focus()

		case "ctrl+l":
			m.clearMessages()
			m.refreshNoteList()
			m.noteListActive = true
			m.newNoteInputActive = false
			m.noteTextArea.Blur()
			return m, nil

		case "esc":
			m.clearMessages()
			if m.newNoteInputActive {
				m.newNoteInputActive = false
				m.newNoteInput.SetValue("")
				m.newNoteInput.Blur()
				return m, nil
			}

			if m.noteListActive {
				m.noteListActive = false
				if m.activeNotePath != "" {
					m.noteTextArea.Focus()
					return m, m.noteTextArea.Focus()
				}
				return m, nil
			}

			if m.activeNotePath != "" {
				m.activeNotePath = ""
				m.noteTextArea.SetValue("")
				m.noteTextArea.Blur()
			}

			return m, nil

		case "ctrl+s":
			if m.activeNotePath == "" {
				m.setError("Open or create a note before saving.")
				return m, nil
			}

			if err := os.WriteFile(m.activeNotePath, []byte(m.noteTextArea.Value()), 0600); err != nil {
				m.setError(fmt.Sprintf("Could not save note: %v", err))
				return m, nil
			}

			m.statusMessage = fmt.Sprintf("Saved %s", filepath.Base(m.activeNotePath))
			m.errorMessage = ""
			m.refreshNoteList()
			return m, nil

		case "enter":
			if m.noteListActive {
				selectedNote, ok := m.noteList.SelectedItem().(noteItem)
				if !ok {
					return m, nil
				}

				return m.openNote(selectedNote.filename)
			}

			if m.newNoteInputActive {
				return m.createNote()
			}
		}
	}

	if m.newNoteInputActive {
		m.newNoteInput, cmd = m.newNoteInput.Update(msg)
		return m, cmd
	}

	if m.activeNotePath != "" && !m.noteListActive {
		m.noteTextArea, cmd = m.noteTextArea.Update(msg)
		return m, cmd
	}

	if m.noteListActive {
		m.noteList, cmd = m.noteList.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m appModel) View() string {
	header := titleStyle.Render("Totion - Markdown Note Vault")
	help := helpStyle.Render("Ctrl+N New  Ctrl+L Notes  Ctrl+S Save  Esc Back  Q Quit")

	panelWidth := max(24, m.width-4)
	panel := panelStyle.Width(panelWidth)

	var body string
	switch {
	case m.newNoteInputActive:
		body = panel.Render(labelStyle.Render("Create note") + "\n\n" + m.newNoteInput.View())

	case m.activeNotePath != "":
		filename := filepath.Base(m.activeNotePath)
		body = panel.Render(labelStyle.Render("Editing "+filename) + "\n\n" + m.noteTextArea.View())

	case m.noteListActive:
		body = panel.Render(m.noteList.View())

	default:
		body = panel.Render(
			labelStyle.Render("No note open") + "\n\n" +
				"Create a note with Ctrl+N or open your vault with Ctrl+L.\n" +
				"Notes are saved as Markdown files in " + vaultDir + ".",
		)
	}

	parts := []string{header, body}
	if m.errorMessage != "" {
		parts = append(parts, errorStyle.Render(m.errorMessage))
	} else if m.statusMessage != "" {
		parts = append(parts, statusStyle.Render(m.statusMessage))
	}
	parts = append(parts, help)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func initialModel() appModel {
	if err := os.MkdirAll(vaultDir, 0750); err != nil {
		log.Fatalf("create vault directory: %v", err)
	}

	noteNameInput := textinput.New()
	noteNameInput.Placeholder = "daily-plan"
	noteNameInput.CharLimit = 120
	noteNameInput.Width = 50
	noteNameInput.Cursor.Style = cursorStyle
	noteNameInput.PromptStyle = cursorStyle
	noteNameInput.TextStyle = cursorStyle

	editor := textarea.New()
	editor.Placeholder = "Write your note..."
	editor.ShowLineNumbers = false

	noteList := list.New(listNotes(vaultDir), list.NewDefaultDelegate(), 40, 20)
	noteList.Title = "Notes"
	noteList.SetShowStatusBar(false)
	noteList.SetFilteringEnabled(true)
	noteList.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(blue)
	noteList.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(muted)
	noteList.Styles.HelpStyle = lipgloss.NewStyle().Foreground(muted)

	return appModel{
		newNoteInput:  noteNameInput,
		noteTextArea:  editor,
		noteList:      noteList,
		statusMessage: "Vault ready at " + vaultDir,
	}
}

func main() {
	program := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func (m *appModel) openNote(filename string) (tea.Model, tea.Cmd) {
	notePath := filepath.Join(vaultDir, filepath.Base(filename))
	content, err := os.ReadFile(notePath)
	if err != nil {
		m.setError(fmt.Sprintf("Could not open note: %v", err))
		return *m, nil
	}

	m.noteTextArea.SetValue(string(content))
	m.noteTextArea.Focus()
	m.activeNotePath = notePath
	m.noteListActive = false
	m.newNoteInputActive = false
	m.statusMessage = "Opened " + filepath.Base(notePath)
	m.errorMessage = ""

	return *m, m.noteTextArea.Focus()
}

func (m *appModel) createNote() (tea.Model, tea.Cmd) {
	filename := normalizeNoteFilename(m.newNoteInput.Value())
	if filename == "" {
		m.setError("Enter a note name.")
		return *m, nil
	}

	notePath := filepath.Join(vaultDir, filename)
	file, err := os.OpenFile(notePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		m.setError(fmt.Sprintf("Could not create note: %v", err))
		return *m, nil
	}
	if err := file.Close(); err != nil {
		m.setError(fmt.Sprintf("Could not close note: %v", err))
		return *m, nil
	}

	m.activeNotePath = notePath
	m.noteTextArea.SetValue("")
	m.noteTextArea.Focus()
	m.newNoteInputActive = false
	m.newNoteInput.SetValue("")
	m.newNoteInput.Blur()
	m.statusMessage = "Created " + filename
	m.errorMessage = ""
	m.refreshNoteList()

	return *m, m.noteTextArea.Focus()
}

func (m *appModel) refreshNoteList() {
	m.noteList.SetItems(listNotes(vaultDir))
}

func (m *appModel) clearMessages() {
	m.statusMessage = ""
	m.errorMessage = ""
}

func (m *appModel) setError(message string) {
	m.errorMessage = message
	m.statusMessage = ""
}

func normalizeNoteFilename(value string) string {
	filename := strings.TrimSpace(value)
	if filename == "" {
		return ""
	}

	filename = filepath.Base(filename)
	if filename == "." || filename == string(filepath.Separator) {
		return ""
	}

	if !strings.HasSuffix(strings.ToLower(filename), noteExtension) {
		filename += noteExtension
	}

	return filename
}

func listNotes(directory string) []list.Item {
	entries, err := os.ReadDir(directory)
	if err != nil {
		log.Printf("read vault directory: %v", err)
		return nil
	}

	notes := make([]noteItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), noteExtension) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			log.Printf("read note metadata for %s: %v", entry.Name(), err)
			continue
		}

		notes = append(notes, noteItem{
			filename: entry.Name(),
			modified: info.ModTime().
				Format("02 Jan 2006, 15:04"),
			modTime: info.ModTime().UnixNano(),
		})
	}

	sort.Slice(notes, func(i, j int) bool {
		if notes[i].modTime == notes[j].modTime {
			return notes[i].filename < notes[j].filename
		}
		return notes[i].modTime > notes[j].modTime
	})

	items := make([]list.Item, len(notes))
	for i, note := range notes {
		items[i] = note
	}

	return items
}
