package main

import (
	"fmt"
	"strings"
    "strconv"
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
    wordList    WordList
	targetText  string
    targetTextLength int
    targetWords []string
	typedWords  []string
    currentWordIndex int

	started     bool
	startTime   time.Time
	endTime     time.Time
	wpm         int
	accuracy    float64
    incorrectCharCount int

    db *sql.DB
    leaderboard []LeaderboardEntry
}

type WordList struct {
    Words []string `json:"words"`
}

// Color Guide
// 15   White
// 12   Blue
// 9    Red
// 8    Gray
// 3    Yellow

var (
	correctCharStyle = lipgloss.NewStyle().
            Foreground(lipgloss.Color("15")).
            Bold(true)

	incorrectCharStyle   = lipgloss.NewStyle().
            Foreground(lipgloss.Color("9")).
            Bold(true)

    extraCharStyle  = lipgloss.NewStyle().
            Foreground(lipgloss.Color("1")).
            Bold(true)

	cursorStyle  = lipgloss.NewStyle().
            Foreground(lipgloss.Color("12")).
            Bold(true)

	normalCharStyle  = lipgloss.NewStyle().
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
    // instead of doing in main, write function to process args
    // 1. Word Count (-w [int num])
    // 2. Wipe leaderboard (-x) no args
    args := os.Args

    wordCount := DEFAULT_COUNT
    if len(args) == 2 {
        wordCount, _ = strconv.Atoi(args[1])
    }

    db, err := sql.Open("sqlite3", "resources/leaderboard.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    initializeDatabase(db)

	p := tea.NewProgram(initializeModel(db, wordCount))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func initializeModel(db *sql.DB, wordCount int) model {
    words, err := loadJSON("resources/wordlist.json")
    if err != nil {
        fmt.Println("Error loading JSON:", err)
        os.Exit(1)
    }

    wordList := WordList { Words: words }
    targetText := randomWords(wordList, wordCount)
    targetTextLength := len(targetText)
    targetWords := strings.Fields(targetText)
    leaderboard := fetchLeaderboard(db)

	return model{
        wordList:   wordList,
		targetText: targetText,
        targetWords: targetWords,
        typedWords: make([]string, len(targetWords)),
        currentWordIndex: 0,
        incorrectCharCount: 0,
        targetTextLength: targetTextLength,
        db: db,
        leaderboard: leaderboard,
	}
}

// load words from json file into []string
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

// select random words from wordList
func randomWords(words WordList, wordCount int) string {
    selectedWords := make([]string, 0)
    existing := make(map[int]struct{}, 0)
    for i := 0; i < wordCount; i++ {
        randomIndex := randomIndex(len(words.Words), existing)
        selectedWords = append(selectedWords, words.Words[randomIndex])
    }

    return strings.Join(selectedWords, " ")
}

// pick a random index that has not been selected
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
    m.targetText = randomWords(m.wordList, DEFAULT_COUNT)
    m.targetWords = strings.Fields(m.targetText)
    m.typedWords = make([]string, len(m.targetWords))
    m.incorrectCharCount = 0
    m.started = false
    m.currentWordIndex = 0
    m.wpm = 0
    m.accuracy = 0
    m.startTime = time.Time{}
    m.endTime = time.Time{}
}

func (m model) Init() tea.Cmd {
	return nil
}

// update function to process keypresses in bubbletea model
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

        // if race is finished, wont allow other keys to be processed
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
        case " ":
            // if not last word, increment index of word we are on
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
            // if backspace first character of a word, decrement to previous word
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

            if len(m.typedWords[m.currentWordIndex]) > len(m.targetWords[m.currentWordIndex]) {
                m.incorrectCharCount++
            } else if msg.String() != string(m.targetWords[m.currentWordIndex][len(m.typedWords[m.currentWordIndex])-1]) {
                m.incorrectCharCount++
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

	return fmt.Sprintf(
 	    "%s\n%s WPM: %d   Accuracy: %.2f%%\nPress Ctrl+C to quit, Ctrl+R to restart.",
		    title, wpmStyle.Render("Typing Test"), m.wpm, m.accuracy,
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
            renderedText.WriteString(m.styleText(targetWord, typedWord, i))
        } else {
            // upcoming words are all normal style
            renderedText.WriteString(normalCharStyle.Render(targetWord))
        }

        // spaces between words
        if i < len(m.targetWords) - 1 {
            renderedText.WriteString(" ")
        }
    }

    return renderedText.String()
}

// style written text into correct colors
// correct chars - white
// incorrect chars - red
// extra chars in word - dark red
// cursor - blue
// untyped chars - gray
func (m model) styleText(targetWord string, typedWord string, wordIndex int) string {
    var renderedText strings.Builder

    for i := 0; i < len(targetWord); i++ {
        // go up to length of typed text coloring each character
        if i < len(typedWord) {
            if targetWord[i] == typedWord[i] {
                // correct chars
                renderedText.WriteString(correctCharStyle.Render(string(targetWord[i])))
            } else {
                // incorrect chars
                renderedText.WriteString(incorrectCharStyle.Render(string(typedWord[i])))
            }
            continue

        } else if i == len(typedWord) {
            // cursor
            if wordIndex == m.currentWordIndex {
                renderedText.WriteString(cursorStyle.Render(string(targetWord[i])))
                continue
            }
        }

        //untyped text
        renderedText.WriteString(normalCharStyle.Render(string(targetWord[i])))
    }

    // extra chars in word are different color
    if len(typedWord) > len(targetWord) {
        renderedText.WriteString(extraCharStyle.Render(typedWord[len(targetWord):]))
    }

    return renderedText.String()
}

func (m *model) calculateWPMAndAccuracy() {
    elapsedMinutes := time.Since(m.startTime).Minutes()
    if elapsedMinutes == 0 {
        elapsedMinutes = 1.0 / 60.0
    }

    correctWords := 0
    correctChars := 0

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
        m.accuracy = 100 - ((float64(m.incorrectCharCount) / float64(correctChars)) * 100)
    } else {
        m.accuracy = 0
    }

    m.wpm = int(float64(correctWords) / elapsedMinutes)

    if m.wpm < 0 {
        m.wpm = 0
    }
}

// calculates the length of how many characters in the prefix of two words match
func matchingPrefixLength(a string, b string) int {
    // if either word is empty
    if len(a) == 0 || len(b) == 0 {
        return 0
    } else if a == b {
        return len(a)
    }

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
    // race must have been started to be finished
    if m.started {
        // space pressed on last word
        if m.currentWordIndex > len(m.targetWords) {
            return true
        }

        // last word typed correctly
        if m.currentWordIndex == len(m.targetWords) {
            return m.typedWords[len(m.targetWords)-1] == m.targetWords[len(m.targetWords)-1]
        }
    }
    return false
}
