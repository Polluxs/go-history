package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type model struct {
	choices  []string // Items on the list
	cursor   int      // Which item our cursor is pointing at
	selected string   // Which item is selected
}

func initialModel(paths []string) model {
	return model{
		choices: paths,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Which path do you want to select?\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\nPress enter to select, q to quit.\n"
	return s
}

func main() {
	// Get the current user's home directory
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	homeDir := usr.HomeDir

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}

	log.Printf("Current working directory: %v", cwd)

	// Path to the history file
	// Find the appropriate history file
	var historyFile string
	bashHistoryFile := filepath.Join(homeDir, ".bash_history")
	zshHistoryFile := filepath.Join(homeDir, ".zsh_history")

	// Check if the Zsh history file exists, otherwise use Bash history
	if _, err := os.Stat(zshHistoryFile); err == nil {
		historyFile = zshHistoryFile
	} else if _, err := os.Stat(bashHistoryFile); err == nil {
		historyFile = bashHistoryFile
	} else {
		log.Fatal("No history file found.")
	}

	// Open and read the history file
	file, err := os.Open(historyFile)
	if err != nil {
		log.Fatalf("Error opening history file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Read the entire history file
	content, err := os.ReadFile(historyFile)
	if err != nil {
		log.Fatalf("Error reading history file: %v", err)
	}

	// Split the file content into lines
	lines := strings.Split(string(content), "\n")
	uniquePaths := make(map[string]struct{})
	orderedPaths := []string{}

	addPath := func(path string) {
		if _, exists := uniquePaths[path]; !exists {
			uniquePaths[path] = struct{}{}
			orderedPaths = append(orderedPaths, path)
		}
	}

	// Process lines in reverse order
	for i := len(lines) - 1; i >= 0; i-- {
		// just show 10 for now
		if len(uniquePaths) > 10 {
			continue
		}

		line := lines[i]
		if strings.HasPrefix(strings.ToLower(line), "cd ") {
			cdPath := strings.TrimSpace(line[3:])
			var newPath string

			if strings.HasPrefix(cdPath, "~") {
				newPath = filepath.Join(homeDir, cdPath[1:])
			} else if filepath.IsAbs(cdPath) {
				newPath = cdPath
			} else {
				newPath = filepath.Join(cwd, cdPath)
			}

			newPath = filepath.Clean(newPath)
			newPath, err = filepath.Abs(newPath)
			if err != nil {
				log.Printf("Error resolving path '%s': %v", cdPath, err)
				continue
			}

			addPath(newPath)
			cwd = newPath
		}
	}

	// Create a Bubble Tea program and start it
	p := tea.NewProgram(initialModel(orderedPaths))
	m, err := p.Run()
	if err != nil {
		log.Fatalf("Error running Bubble Tea program: %v", err)
	}

	// Cast the returned model and navigate to the selected path
	finalModel := m.(model)
	selectedPath := finalModel.selected

	// Copying to clipboard using pbcopy
	cmdText := fmt.Sprintf("cd %s", selectedPath)
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(cmdText)
	err = cmd.Run()
	if err != nil {
		log.Println("If you install pbcopy you can just paste")
		log.Println(cmdText)
	} else {
		fmt.Printf("The command 'cd %s' has been copied to the clipboard.\n", selectedPath)
	}

	os.Exit(0)
}
