package main

import (
	"fmt"
	"io"
	"net/http"
)

type ReadSeekerAt interface {
	io.ReaderAt
	io.Seeker
}

func WordCountParallel(file ReadSeekerAt) (int64, string) {
	length, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Printf("Error getting length of file")
		return -1, err.Error()
	}

	partLength := length / 10

	ch := make(chan string)
	done := make(chan struct{})

	for i := int64(0); i < 10; i++ {
		start := partLength * i
		end := partLength * (i + 1)

		if i == 9 {
			end = length
		}

		go WordCountPartial(file, start, end, ch, done)
	}

	go func() {
		for i := 0; i < 10; i++ {
			<-done
		}

		close(ch)
	}()

	return reduce(ch)
}

func isWhitespace(r byte) bool {
	switch r {
	case ' ':
		return true
	case '\n':
		return true
	case '\r':
		return true
	default:
		return false
	}
}

func WordCountPartial(input io.ReaderAt, start int64, end int64, output chan<- string, done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
	}()

	buffer := make([]byte, 100)
	bufferFill := 0
	word := ""

	isEOF := false

	inWord := false
	snoozing := start != 0

	for i, j := start, 0; ; {
		if snoozing && i == end {
			break
		}

		if !inWord && i > end {
			break
		}

		if j == bufferFill {
			if isEOF {
				break
			}

			j = 0
			nread, err := input.ReadAt(buffer, i)
			if nread == 0 {
				break
			}

			if err == io.EOF {
				isEOF = true
			} else if err != nil {
				fmt.Println(err)
				return
			}

			bufferFill = nread
		}

		if isWhitespace(buffer[j]) {
			inWord = false
			if word != "" {
				output <- word
				word = ""
			}

			if snoozing {
				snoozing = false
			}
		} else if !snoozing && !inWord {
			inWord = true
		}

		if inWord {
			word = word + string(buffer[j])
		}

		i++
		j++
	}

	if word != "" {
		output <- word
	}
}

func reduce(ch <-chan string) (int64, string) {
	frequency := map[string]int64{}
	max := int64(0)
	maxString := ""

	for word := range ch {
		fmt.Printf("word %s\n", word)
		frequency[word] = frequency[word] + 1
		if frequency[word] > max {
			max = frequency[word]
			maxString = word
		}
	}

	return max, maxString
}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/wordcount", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Header", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		defer file.Close()

		wordCount, word := WordCountParallel(file)

		fmt.Fprintf(w, "The number of words is %d (%s)", wordCount, word)
	})

	http.ListenAndServe(":5000", mux)
}
