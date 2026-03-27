package main

import (
"fmt"
"os"
    "time"

"github.com/adzin/port-forward-cli/internal/db"
"github.com/adzin/port-forward-cli/internal/ui"
tea "github.com/charmbracelet/bubbletea"
)

func main() {
    f, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    tea.LogToFile("debug.log", "debug")
	os.Setenv("HOME", "/home/adzin") // ensure db works
	if err := db.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "db init: %v\n", err)
		os.Exit(1)
	}

	app := ui.NewApp()
	p := tea.NewProgram(app)
    go func() {
        time.Sleep(500 * time.Millisecond)
        p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
        time.Sleep(500 * time.Millisecond)
        p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    }()
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
    f.Close()
}
