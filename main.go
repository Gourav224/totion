package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	valutDir string

	blue      = lipgloss.Color("#3B82F6") // Tailwind Blue-500
	blueLight = lipgloss.Color("#60A5FA")
	textWhite = lipgloss.Color("#FFFFFF")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textWhite).
			Background(blue).
			Padding(0, 2).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(blueLight).
			MarginTop(1)

	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(blue).
			Padding(1, 2)

	cursorStyle = lipgloss.NewStyle().
			Foreground(blueLight)
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error getting home directory", err)
	}
	valutDir = fmt.Sprintf("%s/.totion", homeDir)
}

// ---------------- Items ----------------

type item struct {
	title, desc string
}

func (i item) Title() string       { return "📘 " + i.title }
func (i item) FilterValue() string { return i.title }
func (i item) Description() string { return i.desc }

// ---------------- Model ----------------

type model struct {
	width, height int

	newFileInput           textinput.Model
	createFileInputVisible bool
	currentFile            *os.File
	noteTextArea           textarea.Model
	list                   list.Model
	showingList            bool
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// ---------------- Update ----------------

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	// Responsive window resize
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.list.SetSize(msg.Width-8, msg.Height-12)
		m.noteTextArea.SetWidth(msg.Width - 8)
		m.noteTextArea.SetHeight(msg.Height - 12)
		m.newFileInput.Width = msg.Width - 10

		return m, nil

	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "ctrl+n":
			m.createFileInputVisible = true
			m.showingList = false
			return m, nil

		case "ctrl+l":
			m.showingList = true
			m.createFileInputVisible = false
			return m, nil

		case "esc":
			if m.createFileInputVisible {
				m.createFileInputVisible = false
				m.newFileInput.SetValue("")
				return m, nil
			}

			if m.showingList {
				m.showingList = false
				return m, nil
			}

			if m.currentFile != nil {
				m.currentFile.Close()
				m.currentFile = nil
				m.noteTextArea.SetValue("")
				return m, nil
			}

			return m, nil

		case "ctrl+s":
			if m.currentFile == nil {
				break
			}

			m.currentFile.Truncate(0)
			m.currentFile.Seek(0, 0)
			m.currentFile.WriteString(m.noteTextArea.Value())
			m.currentFile.Close()

			m.currentFile = nil
			m.noteTextArea.SetValue("")
			return m, nil

		case "enter":
			if m.showingList {
				item, ok := m.list.SelectedItem().(item)
				if ok {
					filepath := fmt.Sprintf("%s/%s", valutDir, item.title[2:])
					content, _ := os.ReadFile(filepath)

					m.noteTextArea.SetValue(string(content))
					f, _ := os.OpenFile(filepath, os.O_RDWR, 0644)
					m.currentFile = f
					m.showingList = false
				}
				return m, nil
			}

			if m.createFileInputVisible {
				filename := m.newFileInput.Value()

				if filename != "" {
					filepath := fmt.Sprintf("%s/%s.md", valutDir, filename)
					f, err := os.Create(filepath)
					if err != nil {
						log.Fatalf("%v", err)
					}
					m.currentFile = f
					m.createFileInputVisible = false
					m.newFileInput.SetValue("")
				}
				return m, nil
			}
		}
	}

	// Route events
	if m.createFileInputVisible {
		m.newFileInput, cmd = m.newFileInput.Update(msg)
	}
	if m.currentFile != nil {
		m.noteTextArea, cmd = m.noteTextArea.Update(msg)
	}
	if m.showingList {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

// ---------------- View ----------------

func (m model) View() string {

	header := titleStyle.Render("🔷 Totion — Minimal Note Vault 🧠")

	help := helpStyle.Render("🆕 New: Ctrl+N · 📂 List: Ctrl+L · 💾 Save: Ctrl+S · ❌ Quit: Q · ESC: Back")

	mainContainer := containerStyle.Width(m.width - 4)

	var view string

	switch {
	case m.createFileInputVisible:
		view = mainContainer.Render(m.newFileInput.View())

	case m.currentFile != nil:
		view = mainContainer.Render(m.noteTextArea.View())

	case m.showingList:
		view = mainContainer.Render(m.list.View())

	default:
		view = mainContainer.Render("Press Ctrl+N to create a note or Ctrl+L to view notes.")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, view, help)
}

// ---------------- Init Model ----------------

func initialModel() model {
	os.MkdirAll(valutDir, 0750)

	ti := textinput.New()
	ti.Placeholder = "Name your note..."
	ti.CharLimit = 120
	ti.Width = 50
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = cursorStyle
	ti.TextStyle = cursorStyle

	ta := textarea.New()
	ta.Placeholder = "Start writing..."
	ta.Focus()

	items := listFiles()
	l := list.New(items, list.NewDefaultDelegate(), 40, 20)
	l.Title = "Your Notes"
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(blue)

	return model{
		newFileInput: ti,
		noteTextArea: ta,
		list:         l,
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// ---------------- List Files ----------------

func listFiles() []list.Item {
	items := make([]list.Item, 0)
	entries, _ := os.ReadDir(valutDir)

	for _, ent := range entries {
		if !ent.IsDir() {
			info, _ := ent.Info()
			mod := info.ModTime().Format("02 Jan 06 15:04")

			items = append(items, item{
				title: ent.Name(),
				desc:  fmt.Sprintf("Updated: %s", mod),
			})
		}
	}
	return items
}
