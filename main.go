package main

import (
	"fmt"
	"strings"
	"time"
    "os"
    "math/rand"
    "encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	targetText string
	typedText  string
	started    bool
	startTime  time.Time
	endTime    time.Time
	wpm        int
	accuracy   float64
}

type WordList struct {
    Words []string `json:"commonWords"`
}

// Color Guide
// 15   Green
// 12   Blue
// 9    Red
// 8    Gray
// 3    Yellow

var (
	correctStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("15")).
            Bold(true)

	wrongStyle   = lipgloss.NewStyle().
            Foreground(lipgloss.Color("9")).
            Bold(true)

	cursorStyle  = lipgloss.NewStyle().
            Foreground(lipgloss.Color("12")).
            Bold(true)

	normalStyle  = lipgloss.NewStyle().
            Foreground(lipgloss.Color("8"))

    wpmStyle     = lipgloss.NewStyle().
            Foreground(lipgloss.Color("15")).
            Bold(true)

	layoutStyle  = lipgloss.NewStyle().
            Align(lipgloss.Center).
            Margin(2, 2)

    titleStyle = lipgloss.NewStyle().
                BorderStyle(lipgloss.RoundedBorder()).
                BorderForeground(lipgloss.Color("3")).
                Padding(0, 1).
                Foreground(lipgloss.Color("3")).
                SetString("TUIpe Race")
)

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func initialModel() model {
	return model{
		targetText: randomSentence(),
	}
}

// Better way to do loading JSON and randomizing string?
// kinda slow to do every race
func randomSentence() string {
    words, err := loadJSON("resources/wordlist.json")

    if err != nil {
        fmt.Println("Error loading JSON:", err)
        os.Exit(1)
    }

    rand.Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })
    selected := words[:30]

	return strings.Join(selected, " ")
}

func loadJSON(fileName string) ([]string, error){
    data, err := os.ReadFile(fileName)
    if err != nil {
        return nil, err
    }

    var wordList WordList
    err = json.Unmarshal(data, &wordList)
    if err != nil {
        return nil, err
    }

    return wordList.Words, nil
}

func (m model) Init() tea.Cmd {
	return nil
}


func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle command keypresses first
        // need ctrl+space to reset race
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		// Start timer on first keypress
		if !m.started && len(m.typedText) == 0 {
			m.started = true
			m.startTime = time.Now()
		}

		// Handle regular typing and backspace
		switch msg.String() {
		case "backspace":
			if len(m.typedText) > 0 {
				m.typedText = m.typedText[:len(m.typedText)-1]
			}
		case "enter":
			// End the test
			m.endTime = time.Now()
			m.calculateWPMAndAccuracy()
		default:
            // only register single character
			if len(msg.String()) == 1 {
				m.typedText += msg.String()
				m.calculateWPMAndAccuracy() // Update metrics in real-time
			}

            if m.finished() {
                m.endTime = time.Now()
                m.calculateWPMAndAccuracy()
                return m, func() tea.Msg { return tea.Quit() }
            }
		}
	}

	return m, nil
}

func (m model) View() string {
    if m.finished() {
        return layoutStyle.Render(fmt.Sprintf("Race finished!\n\nWPM: %d   Accuracy: %.2f%%", m.wpm, m.accuracy))
    }

	header := m.renderHeader()
	typingArea := m.renderTypingArea()
	return layoutStyle.Render(fmt.Sprintf("%s\n\n%s", header, typingArea))
}

func (m model) renderHeader() string {
    title := titleStyle.Render()

	// If test hasn't started yet, WPM and accuracy are 0
	wpm := 0
	accuracy := 0.0

	if m.started {
		wpm = m.wpm
		accuracy = m.accuracy
	}

	return fmt.Sprintf(
 	    "%s\n%s WPM: %d   Accuracy: %.2f%%\nPress Ctrl+C to quit.",
		title, wpmStyle.Render("Typing Test"), wpm, accuracy,
	)
}

func (m model) renderTypingArea() string {
    var renderedText strings.Builder
    targetRunes := []rune(m.targetText)
    typedRunes := []rune(m.typedText)
    incorrectString := false

    for i := 0; i < len(targetRunes); i++ {
        if i < len(typedRunes) {
            if typedRunes[i] == targetRunes[i] && !incorrectString {
                // Correct characters
                renderedText.WriteString(correctStyle.Render(string(typedRunes[i])))
            } else {
                // Once an incorrect character is found, set the flag
                incorrectString = true
                // Incorrect characters
                renderedText.WriteString(wrongStyle.Render(string(typedRunes[i])))
            }
        } else {
            // Cursor (next character to be typed)
            if i == len(typedRunes) {
                renderedText.WriteString(cursorStyle.Render(string(targetRunes[i])))
            } else {
                // Untyped characters
                renderedText.WriteString(normalStyle.Render(string(targetRunes[i])))
            }
        }
    }

    // adding nextLine escape characters every 10 strings
    textString := renderedText.String()
    stringArr := strings.Split(textString, " ")
    for i := 9; i < len(stringArr); i+=10 {
        stringArr[i] = stringArr[i] + "\n"
    }

    return strings.Join(stringArr, " ")
}


func (m *model) calculateWPMAndAccuracy() {
    elapsedMinutes := time.Since(m.startTime).Minutes()
    wordCount := len(strings.Fields(m.typedText))
    m.wpm = int(float64(wordCount) / elapsedMinutes)

    // cant have negative wpm
    // idk why this was a problem
    if m.wpm < 0 {
        m.wpm = 0
    }

    correctChars := 0
    for i := 0; i < len(m.typedText); i++ {
        if i < len(m.targetText) && m.typedText[i] == m.targetText[i] {
            correctChars++
        }
    }

    if len(m.typedText) > 0 {
        m.accuracy = (float64(correctChars) / float64(len(m.typedText))) * 100
    } else {
        m.accuracy = 0
    }
}


func (m model) finished() bool {
	return m.started && m.targetText == m.typedText
}

