package dictionary

import (
    "os"
    "strings"


    "encoding/json"
    "math/rand/v2"
)

type Dictionary struct {
    Words []string `json:"words"`
}

// load words from json file into []string
func LoadJSON(fileName string) ([]string, error){
    file, err := os.Open(fileName)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var wordList Dictionary
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&wordList); err != nil {
        return nil, err
    }

    return wordList.Words, nil
}

// select random words from wordList
func SelectRandomWords(dict Dictionary, wordCount int) string {
    selectedWords := make([]string, 0)
    existing := make(map[int]struct{}, 0)
    for i := 0; i < wordCount; i++ {
        randomIndex := randomIndex(len(dict.Words), existing)
        selectedWords = append(selectedWords, dict.Words[randomIndex])
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
