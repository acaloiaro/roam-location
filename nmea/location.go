package nmea

import (
	"strconv"
	"strings"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// ParseLocation parses nmea sentences into Location structures
func ParseLocation(nmeaSentence string) (l Location) {
	fields := strings.Split(nmeaSentence, ",")

	if f, err := strconv.ParseFloat(fields[0], 64); err == nil {
		l.Lat = f
	}

	if f, err := strconv.ParseFloat(fields[1], 64); err == nil {
		l.Lon = f
	}

	return
}
