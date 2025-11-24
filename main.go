package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	valutDir string

	blue      = lipgloss.Color("#3B82F6")
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
	currentFilePath        string
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
			if m.currentFile != nil {
				m.currentFile.Close()
			}
			return m, tea.Quit

		case "ctrl+n":
			m.createFileInputVisible = true
			m.showingList = false
			m.newFileInput.Focus()
			return m, m.newFileInput.Focus()

		case "ctrl+l":
			// Refresh file list
			items := listFiles()
			m.list.SetItems(items)
			m.showingList = true
			m.createFileInputVisible = false
			return m, nil

		case "esc":
			if m.createFileInputVisible {
				m.createFileInputVisible = false
				m.newFileInput.SetValue("")
				m.newFileInput.Blur()
				return m, nil
			}

			if m.showingList {
				m.showingList = false
				return m, nil
			}

			if m.currentFile != nil {
				m.currentFile.Close()
				m.currentFile = nil
				m.currentFilePath = ""
				m.noteTextArea.SetValue("")
				m.noteTextArea.Blur()
				return m, nil
			}

			return m, nil

		case "ctrl+s":
			if m.currentFile == nil {
				break
			}

			// Write to file
			if err := m.currentFile.Truncate(0); err != nil {
				log.Printf("Error truncating file: %v", err)
				break
			}
			if _, err := m.currentFile.Seek(0, 0); err != nil {
				log.Printf("Error seeking file: %v", err)
				break
			}
			if _, err := m.currentFile.WriteString(m.noteTextArea.Value()); err != nil {
				log.Printf("Error writing file: %v", err)
				break
			}
			if err := m.currentFile.Sync(); err != nil {
				log.Printf("Error syncing file: %v", err)
			}

			return m, nil

		case "enter":
			if m.showingList {
				selectedItem, ok := m.list.SelectedItem().(item)
				if ok {
					// Remove emoji prefix to get actual filename
					filename := strings.TrimPrefix(selectedItem.title, "📘 ")
					filepath := fmt.Sprintf("%s/%s", valutDir, filename)

					content, err := os.ReadFile(filepath)
					if err != nil {
						log.Printf("Error reading file: %v", err)
						return m, nil
					}

					f, err := os.OpenFile(filepath, os.O_RDWR, 0644)
					if err != nil {
						log.Printf("Error opening file: %v", err)
						return m, nil
					}

					m.noteTextArea.SetValue(string(content))
					m.noteTextArea.Focus()
					m.currentFile = f
					m.currentFilePath = filepath
					m.showingList = false
				}
				return m, m.noteTextArea.Focus()
			}

			if m.createFileInputVisible {
				filename := strings.TrimSpace(m.newFileInput.Value())

				if filename != "" {
					// Add .md extension if not present
					if !strings.HasSuffix(filename, ".md") {
						filename = filename + ".md"
					}

					filepath := fmt.Sprintf("%s/%s", valutDir, filename)
					f, err := os.Create(filepath)
					if err != nil {
						log.Printf("Error creating file: %v", err)
						return m, nil
					}

					m.currentFile = f
					m.currentFilePath = filepath
					m.createFileInputVisible = false
					m.newFileInput.SetValue("")
					m.newFileInput.Blur()
					m.noteTextArea.Focus()

					return m, m.noteTextArea.Focus()
				}
				return m, nil
			}
		}
	}

	// Route events to appropriate components
	if m.createFileInputVisible {
		m.newFileInput, cmd = m.newFileInput.Update(msg)
		return m, cmd
	}
	if m.currentFile != nil && !m.showingList {
		m.noteTextArea, cmd = m.noteTextArea.Update(msg)
		return m, cmd
	}
	if m.showingList {
		m.list, cmd = m.list.Update(msg)
		return m, cmd
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
		prompt := "Enter note name:\n\n"
		view = mainContainer.Render(prompt + m.newFileInput.View())

	case m.currentFile != nil:
		filename := ""
		if m.currentFilePath != "" {
			parts := strings.Split(m.currentFilePath, "/")
			filename = parts[len(parts)-1]
		}
		noteView := fmt.Sprintf("Editing: %s\n\n%s", filename, m.noteTextArea.View())
		view = mainContainer.Render(noteView)

	case m.showingList:
		view = mainContainer.Render(m.list.View())

	default:
		welcome := "Welcome to Totion! 📝\n\n" +
			"Press Ctrl+N to create a new note\n" +
			"Press Ctrl+L to view your notes\n" +
			"Press Q to quit"
		view = mainContainer.Render(welcome)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, view, help)
}

// ---------------- Init Model ----------------

func initialModel() model {
	if err := os.MkdirAll(valutDir, 0750); err != nil {
		log.Fatalf("Error creating vault directory: %v", err)
	}

	ti := textinput.New()
	ti.Placeholder = "Name your note..."
	ti.CharLimit = 120
	ti.Width = 50
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = cursorStyle
	ti.TextStyle = cursorStyle

	ta := textarea.New()
	ta.Placeholder = "Start writing your note..."
	ta.ShowLineNumbers = false

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
	entries, err := os.ReadDir(valutDir)
	if err != nil {
		log.Printf("Error reading vault directory: %v", err)
		return items
	}

	for _, ent := range entries {
		if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".md") {
			info, err := ent.Info()
			if err != nil {
				continue
			}
			mod := info.ModTime().Format("02 Jan 06 15:04")

			items = append(items, item{
				title: ent.Name(),
				desc:  fmt.Sprintf("Updated: %s", mod),
			})
		}
	}
	return items
}
