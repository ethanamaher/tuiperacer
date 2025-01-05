package main

import (
	"fmt"
	"strings"
	"time"
    "os"
    "log"
    "math/rand/v2"
    "encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

const (
    DEFAULT_COUNT int = 15
)
type model struct {
	targetText  string
    targetWords []string
	typedWords  []string
    currentWordIndex int

	started     bool
	startTime   time.Time
	endTime     time.Time
	wpm         int
	accuracy    float64
    wordList    WordList

    db *sql.DB
    leaderboard []LeaderboardEntry
}

type WordList struct {
    Words []string `json:"words"`
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
            Margin(2, 2).
            Width(60)

    titleStyle = lipgloss.NewStyle().
            BorderStyle(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("3")).
            Padding(0, 1).
            Foreground(lipgloss.Color("3")).
            SetString("TUIpe Race")
)

func main() {
    db, err := sql.Open("sqlite3", "resources/leaderboard.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    initializeDatabase(db)

	p := tea.NewProgram(initializeModel(db))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func initializeModel(db *sql.DB) model {
    words, err := loadJSON("resources/wordlist.json")
    if err != nil {
        fmt.Println("Error loading JSON:", err)
        os.Exit(1)
    }

    wordList := WordList { Words: words }
    targetText := randomSentence(wordList, DEFAULT_COUNT)
    targetWords := strings.Fields(targetText)
    leaderboard := fetchLeaderboard(db)
	return model{
        wordList:   wordList,
		targetText: targetText,
        targetWords: targetWords,
        typedWords: make([]string, len(targetWords)),
        currentWordIndex: 0,
        db: db,
        leaderboard: leaderboard,
	}
}

// improve speed on json loading
// go-json?
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

func randomSentence(words WordList, wordCount int) string {
    selectedWords := make([]string, 0)
    existing := make(map[int]struct{}, 0)
    for i := 0; i < wordCount; i++ {
        randomIndex := randomIndex(len(words.Words), existing)
        selectedWords = append(selectedWords, words.Words[randomIndex])
    }

    return strings.Join(selectedWords, " ")
}

func randomIndex(size int, existingIndexes map[int]struct{}) int {
    for {
        randomIndex := rand.IntN(size)

        _, exists := existingIndexes[randomIndex]
        if !exists {
            existingIndexes[randomIndex] = struct{}{}
            return randomIndex
        }
    }
}

func ResetModel(m *model) {
    m.targetText = randomSentence(m.wordList, DEFAULT_COUNT)
    m.targetWords = strings.Fields(m.targetText)
    m.typedWords = make([]string, len(m.targetWords))
    m.started = false
    m.currentWordIndex = 0
    m.wpm = 0
    m.accuracy = 0
    m.startTime = time.Time{}
    m.endTime = time.Time{}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // prevent typing after race end
    if m.isRaceFinished() {
        return m, nil
    }

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
		if !m.started && len(m.typedWords) == 0 {
			m.started = true
			m.startTime = time.Now()
		}

		// Handle regular keystrokes
		switch msg.String() {
        // go to next word on " "
        case " ":
            if m.currentWordIndex < len(m.targetWords) {
                m.currentWordIndex++
            }
        return m, nil
		case "backspace":
            // go to previous word
            if m.currentWordIndex > 0 && len(m.typedWords[m.currentWordIndex]) == 0 {
                m.currentWordIndex--
            } else if len(m.typedWords[m.currentWordIndex]) > 0 {
                m.typedWords[m.currentWordIndex] = m.typedWords[m.currentWordIndex][:len(m.typedWords[m.currentWordIndex])-1]
            }
		default:
			if len(msg.String()) == 1 {
                if m.currentWordIndex < len(m.targetWords) {
                    m.typedWords[m.currentWordIndex] += msg.String()
                }
            }

            m.calculateWPMAndAccuracy()

            if m.isRaceFinished() {
                m.endTime = time.Now()

                saveToLeaderboard(m.db, "Player One", m.wpm)

                m.leaderboard = fetchLeaderboard(m.db)
                return m, func() tea.Msg { return tea.Quit() }
            }
        }
	}

	return m, nil
}

func (m model) View() string {
    if m.isRaceFinished() {
        leaderboardView := m.renderLeaderboard()
        return layoutStyle.Render(
            fmt.Sprintf("Race finished!\n\nWPM: %d   Accuracy: %.2f%%\n\n%s",
                        m.wpm, m.accuracy, leaderboardView),)
    }


	header := m.renderHeader()
	typingArea := m.renderTypingArea()
	return layoutStyle.Render(fmt.Sprintf("%s\n\n%s", header, typingArea))
}

func (m model) renderLeaderboard() string {
    var render strings.Builder
    render.WriteString("Leaderboard\n\n")
    for i, entry := range m.leaderboard {
        render.WriteString(fmt.Sprintf("%d. %s - %d WPM\n", i+1, entry.Name, entry.WPM))
    }
    return render.String()
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

    for i, targetWord := range m.targetWords {
        typedWord := ""
        if i < len(m.typedWords) {
            typedWord = m.typedWords[i]
        }

        if i < m.currentWordIndex {
            if typedWord == targetWord {
                renderedText.WriteString(correctStyle.Render(targetWord))
            } else {
                renderedText.WriteString(wrongStyle.Render(typedWord))
            }
        } else if i == m.currentWordIndex {
            if typedWord == targetWord {
                renderedText.WriteString(correctStyle.Render(targetWord))
            } else if strings.HasPrefix(targetWord, typedWord) {
                renderedText.WriteString(correctStyle.Render(typedWord))
                renderedText.WriteString(cursorStyle.Render(string(targetWord[len(typedWord)])))
                renderedText.WriteString(normalStyle.Render(targetWord[len(typedWord)+1:]))
            } else {
                renderedText.WriteString(wrongStyle.Render(typedWord))
            }
        } else {
            renderedText.WriteString(normalStyle.Render(targetWord))
        }

        if i < len(m.targetWords) - 1 {
            renderedText.WriteString(" ")
        }
    }

    return renderedText.String()
}

func (m *model) calculateWPMAndAccuracy() {
    elapsedMinutes := time.Since(m.startTime).Minutes()
    wordCount := len(m.targetWords)

    correctChars := 0
    typedLength := len(m.typedWords)
    targetLength := len(m.targetText)

    for i := 0; i < typedLength && i < targetLength; i++ {
        if m.typedWords[i] == m.targetWords[i] {
            correctChars++
        }
    }

    if typedLength > 0 {
        m.accuracy = (float64(correctChars) / float64(typedLength)) * 100
        m.wpm = int(float64(wordCount) / elapsedMinutes)

        if m.wpm < 0 {
            m.wpm = 0
        }

    } else {
        m.accuracy = 0
        m.wpm = 0
    }
}

func (m model) isRaceFinished() bool {
	return m.started && m.currentWordIndex >= len(m.targetWords) - 1
}
