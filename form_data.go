package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
)

type InputFile interface {
	io.ReadSeeker
	io.ReaderAt
}

func WordCountParallel(input InputFile) {
	length, _ := input.Seek(0, io.SeekEnd)

	partLength := length / 10

	sectionReaders := make([]*io.SectionReader, 10)
	for i := int64(0); i < 10; i++ {
		sectionReaders[i] = io.NewSectionReader(input, partLength*i, partLength)
	}

	for i := 0; i < 10; i++ {
		go func(i int) {
			count := WordCount(sectionReaders[i])
			fmt.Println("count = ", count)
		}(i)
	}
}

func WordCount(input io.Reader) int {
	scanner := bufio.NewScanner(input)
	scanner.Split(bufio.ScanWords)

	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}

	return count
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

		wordCount := WordCount(file)

		fmt.Fprintf(w, "The number of words is %d", wordCount)
	})

	http.ListenAndServe(":5000", mux)
}
