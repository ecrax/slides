package main

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"log"
	"os"
	"os/user"
	"strings"
	"time"
)

type model struct {
	slides       []string
	currentSlide int
	viewport     viewport.Model
	fileName     string
	date         string
	author       string
}

type fileWatchMsg struct{}

var fileInfo os.FileInfo

func (m model) Init() tea.Cmd {
	if m.fileName == "" {
		return nil
	}
	fileInfo, _ = os.Stat(m.fileName)
	return fileWatchCmd()
}

func fileWatchCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return fileWatchMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left":
			if m.currentSlide > 0 {
				m.currentSlide--
			}
		case "right", " ":
			if m.currentSlide < len(m.slides)-1 {
				m.currentSlide++
			}
		}
	case fileWatchMsg:
		newFileInfo, err := os.Stat(m.fileName)
		if err == nil && newFileInfo.ModTime() != fileInfo.ModTime() {
			fileInfo = newFileInfo
			loadErr := m.Load()
			if loadErr != nil {
				log.Fatal(loadErr)
			}
			if m.currentSlide >= len(m.slides) {
				m.currentSlide = len(m.slides) - 1
			}
		}
		return m, fileWatchCmd()
	}

	return m, nil
}

var statusStyle = lipgloss.NewStyle().Padding(1)
var authorStyle = lipgloss.NewStyle().Align(lipgloss.Left).MarginLeft(2)
var dateStyle = lipgloss.NewStyle().Faint(true).Align(lipgloss.Left).Margin(0, 1)
var pageStyle = lipgloss.NewStyle().Align(lipgloss.Right).MarginRight(3)
var slideStyle = lipgloss.NewStyle().Padding(1)

func (m model) View() string {
	//log.Println(m.slides)
	r, _ := glamour.NewTermRenderer(glamour.WithStylesFromJSONFile("./theme.json"), glamour.WithWordWrap(m.viewport.Width))

	s := m.slides[m.currentSlide]
	s, err := r.Render(s)
	if err != nil {
		log.Fatalf("error when rendering markdown: %s", err)
	}
	s = slideStyle.Render(s)

	left := authorStyle.Render(m.author) + dateStyle.Render(m.date)
	right := pageStyle.Render(m.paging())

	status := statusStyle.Render(JoinHorizontal(left, right, m.viewport.Width))
	return JoinVertical(s, status, m.viewport.Height)
}

func (m model) paging() string {
	return fmt.Sprintf("%v / %v", m.currentSlide+1, len(m.slides))
}

const delimiter string = "---"

func (m *model) Load() error {
	if m.fileName == "" {
		return errors.New("no file specified")
	}

	s, err := os.Stat(m.fileName)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return errors.New("can not read directories")
	}

	b, err := os.ReadFile(m.fileName)
	if err != nil {
		return err
	}

	content := string(b)
	slides := strings.Split(content, delimiter)
	// TODO: some leading newlines still appear
	for i := 0; i < len(slides); i++ {
		slides[i] = strings.TrimRight(slides[i], "\r\n")
		slides[i] = strings.TrimLeft(slides[i], "\r\n")
	}
	m.slides = slides

	return nil
}

// JoinHorizontal joins two strings horizontally and fills the space in-between.
func JoinHorizontal(left, right string, width int) string {
	w := width - lipgloss.Width(right)
	return lipgloss.PlaceHorizontal(w, lipgloss.Left, left) + right
}

// JoinVertical joins two strings vertically and fills the space in-between.
func JoinVertical(top, bottom string, height int) string {
	h := height - lipgloss.Height(bottom)
	return lipgloss.PlaceVertical(h, lipgloss.Top, top) + bottom
}

func newModel() model {
	if len(os.Args) < 2 {
		log.Fatal("please specify a file to open")
	}
	currentUser, err := user.Current()
	var author string
	if err != nil {
		author = "me"
	} else {
		author = currentUser.Name
	}
	m := model{
		slides:       nil,
		currentSlide: 0,
		fileName:     os.Args[1],
		date:         time.Now().Format("02-01-2006"),
		author:       author,
	}

	err = m.Load()
	if err != nil {
		log.Fatalf("something went wrong: %s", err)
	}
	return m
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("There has been an error: %s", err)
	}
}
