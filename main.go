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
	targetText  string
	typedText   string
	started     bool
	startTime   time.Time
	endTime     time.Time
	wpm         int
	accuracy    float64
    wordList    WordList
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
    words, err := loadJSON("resources/wordlist.json")
    if err != nil {
        fmt.Println("Error loading JSON:", err)
        os.Exit(1)
    }

    wordList := WordList { Words: words }

	return model{
        wordList:   wordList,
		targetText: randomSentence(wordList),
	}
}

// Better way to do loading JSON and randomizing string?
// kinda slow to do every race
func randomSentence(words WordList) string {
    rand.Shuffle(len(words.Words), func(i, j int) { words.Words[i], words.Words[j] = words.Words[j], words.Words[i] })
    selected := words.Words[:30]

	return strings.Join(selected, " ")
}

func loadJSON(fileName string) ([]string, error){
    file, err := os.Open(fileName)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var wordList WordList
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&wordList); err != nil {
        return nil, err
    }

    return wordList.Words, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func ResetModel(m *model) {
    m.targetText = randomSentence(m.wordList)
    m.typedText = ""
    m.started = false
    m.wpm = 0
    m.accuracy = 0
    m.startTime = time.Time{}
    m.endTime = time.Time{}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle command keypresses first
        // need ctrl+r to reset race
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
        case "ctrl+r":
            ResetModel(&m)
            return m, nil
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
		default:
            // only register single character
			if len(msg.String()) == 1 {
				m.typedText += msg.String()
				m.calculateWPMAndAccuracy() // Update metrics in real-time
			}

            if m.finished() {
                m.endTime = time.Now()
                m.calculateWPMAndAccuracy()
                // TODO
                // dont allow users to type after finish screen is showing
                // or make it so it doesnt bring up race screen again
                return m, func() tea.Msg { return tea.Quit() }
            }
		}
	}

	return m, nil
}

func (m model) View() string {
    if m.finished() {
        return layoutStyle.Render(
            fmt.Sprintf("Race finished!\n\nWPM: %d   Accuracy: %.2f%%",
                        m.wpm, m.accuracy))
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
 	    "%s\n%s WPM: %d   Accuracy: %.2f%%\nPress Ctrl+C to quit, Ctrl+R to restart.",
		    title, wpmStyle.Render("Typing Test"), wpm, accuracy,
	)
}

func (m model) renderTypingArea() string {
    var renderedText strings.Builder
    targetRunes := []rune(m.targetText)
    typedRunes := []rune(m.typedText)

    targetLength := len(targetRunes)
    typedLength := len(typedRunes)

    incorrectString := false

    for i := 0; i < targetLength; i++ {
        if i < typedLength {
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
            if i == typedLength {
                renderedText.WriteString(cursorStyle.Render(string(targetRunes[i])))
            } else {
                // Untyped characters
                renderedText.WriteString(normalStyle.Render(string(targetRunes[i])))
            }
        }
    }

    // adding nextLine escape characters every 10 strings
    stringArr := strings.Fields(renderedText.String())
    for i := 9; i < len(stringArr); i+=10 {
        stringArr[i] += "\n"
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
    typedLength := len(m.typedText)
    targetLength := len(m.targetText)

    for i := 0; i < typedLength && i < targetLength; i++ {
        if m.typedText[i] == m.targetText[i] {
            correctChars++
        }
    }

    if typedLength > 0 {
        m.accuracy = (float64(correctChars) / float64(typedLength)) * 100
    } else {
        m.accuracy = 0
    }
}


func (m model) finished() bool {
	return m.started && m.targetText == m.typedText
}
