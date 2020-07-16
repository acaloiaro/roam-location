package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var port = 22495

// Listen listens for location requests from clients and return my current coordinates from the db located at dbPath
// my last known location is always the tail end of the file
func Listen(dbPath string) {
	http.HandleFunc("/current_location", listenHandler(dbPath))
	log.Printf("Listening: 0.0.0.0:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func listenHandler(dbPath string) (handler func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		location := currentLocation(dbPath)
		byteArray, err := json.MarshalIndent(location, "", "  ")
		if err != nil {
			log.Fatalf("error providing current location: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, string(byteArray))
	}
}

func currentLocation(dbPath string) (location Location) {
	var err error
	fileHandle, err := os.Open(dbPath)
	if err != nil {
		log.Fatalf("unable to open db file: %v", err)
	}
	defer fileHandle.Close()

	line := ""
	stat, _ := fileHandle.Stat()
	fs := stat.Size()
	var cursor int64 = fs - 1 // start before the newline
	for {
		cursor--

		char := make([]byte, 1)
		fileHandle.ReadAt(char, cursor)

		if cursor != -1 && (char[0] == '\n') { // stop if we find a line
			break
		}

		line = fmt.Sprintf("%s%s", string(char), line) // there is more efficient way

		if cursor == -fs { // stop if we are at the begining
			break
		}
	}

	fields := strings.Split(line, ",")

	if f, err := strconv.ParseFloat(fields[0], 64); err == nil {
		location.Lat = f
	}

	if f, err := strconv.ParseFloat(fields[1], 64); err == nil {
		location.Lon = f
	}

	return
}

func main() {
}
