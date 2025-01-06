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

	incorrectStyle   = lipgloss.NewStyle().
            Foreground(lipgloss.Color("9")).
            Bold(true)

    extraTextStyle  = lipgloss.NewStyle().
            Foreground(lipgloss.Color("1")).
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle command keypresses first
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
        case "ctrl+r":
            ResetModel(&m)
            return m, nil
		}

        if m.isRaceFinished() {
            return m, nil
        }

		// Start timer on first keypress
		if !m.started {
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

            if m.isRaceFinished() {
                m.endTime = time.Now()
                m.calculateWPMAndAccuracy()
                saveToLeaderboard(m.db, "Player One", m.wpm, m.accuracy)
                return m, tea.Quit

            }
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

                saveToLeaderboard(m.db, "Player One", m.wpm, m.accuracy)
                return m, tea.Quit
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
    m.leaderboard = fetchLeaderboard(m.db)
    render.WriteString("Leaderboard\n\n")
    for i, entry := range m.leaderboard {
        render.WriteString(fmt.Sprintf("%d. %s - %d WPM (%.2f%%)\n", i+1, entry.Name, entry.WPM, entry.Accuracy))
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

        if i <= m.currentWordIndex {
            renderedText.WriteString(styleText(targetWord, typedWord))
        } else {
            // upcoming words are all normal style
            renderedText.WriteString(normalStyle.Render(targetWord))
        }

        // spaces between words
        if i < len(m.targetWords) - 1 {
            renderedText.WriteString(" ")
        }
    }

    return renderedText.String()
}

// is it possible to underline a diff color from foreground?
func styleText(targetWord string, typedWord string) string {
    var renderedText strings.Builder

    for i := 0; i < len(targetWord); i++ {
        // go up to length of typed text coloring each character
        if i < len(typedWord) {
            if targetWord[i] == typedWord[i] {
                // correct chars
                renderedText.WriteString(correctStyle.Render(string(targetWord[i])))
            } else {
                // incorrect chars
                renderedText.WriteString(incorrectStyle.Render(string(typedWord[i])))
            }
        // this char is always cursor or " "
        } else if i == len(typedWord) {
            // cursor
            renderedText.WriteString(cursorStyle.Render(string(targetWord[i])))
        // anything past typed text in target is rendered as normal
        } else {
            renderedText.WriteString(normalStyle.Render(string(targetWord[i])))
        }
    }

    // extra chars in word are different color
    if len(typedWord) > len(targetWord) {
        renderedText.WriteString(extraTextStyle.Render(typedWord[len(targetWord):]))
    }

    return renderedText.String()
}

func (m *model) calculateWPMAndAccuracy() {
    elapsedMinutes := time.Since(m.startTime).Minutes()
    if elapsedMinutes == 0 {
        elapsedMinutes = 1.0 / 60.0
    }

    correctChars := 0
    correctWords := 0

    typedChars := 0

    for _, word := range m.typedWords {
        typedChars += len(word)
    }

    for i, typedWord := range m.typedWords {
        if i < len(m.targetWords) && typedWord == m.targetWords[i] {
            correctWords++
            correctChars += len(typedWord)
        } else if i < len(m.targetWords) {
            correctChars += matchingPrefixLength(typedWord, m.targetWords[i])
        }
    }

    if typedChars > 0 {
        m.accuracy = (float64(correctChars) / float64(typedChars)) * 100
    } else {
        m.accuracy = 0
    }

    m.wpm = int(float64(correctWords) / elapsedMinutes)

    if m.wpm < 0 {
        m.wpm = 0
    }
}

func matchingPrefixLength(a string, b string) int {
    length := 0

    for i := 0; i < len(a) && i < len(b); i++ {
        if a[i] == b[i] {
            length++
        } else {
            break
        }
    }

    return length
}

func (m model) isRaceFinished() bool {
    if m.started {
        if m.currentWordIndex >= len(m.targetWords) {
            return true
        }

        if len(m.typedWords) == len(m.targetWords) {
            if m.typedWords[len(m.targetWords)-1] == m.targetWords[len(m.targetWords)-1] {
                return true
            }
        }


    }

    return false

}


