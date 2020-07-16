package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	location := currentLocation()
	byteArray, err := json.MarshalIndent(location, "", "  ")
	if err != nil {
		log.Fatalf("error providing current location: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(byteArray))
}

func currentLocation() (location Location) {
	dbPath := os.Getenv("ROAM_LOCATION_DB_PATH")

	fileHandle, err := os.Open(dbPath)

	if err != nil {
		panic("Cannot open file")
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
	http.HandleFunc("/current_location", handler)
	log.Fatal(http.ListenAndServe(":22495", nil))
}
