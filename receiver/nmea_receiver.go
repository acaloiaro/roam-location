package receiver

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/adrianmo/go-nmea"
)

var dbFile *os.File
var err error

// Receive receives NMEA sentences from the Sierra Wireless RV55 router and writes parsed sentences to
// ROAM_LOCATION_DB_PATH
func Receive(dbPath string) {
	addr, _ := net.ResolveUDPAddr("udp", ":22335")
	sock, _ := net.ListenUDP("udp", addr)
	dbFile, err = os.OpenFile(dbPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("unable to open DB file: '%v'", err)
	}
	defer dbFile.Close()

	for {
		buf := make([]byte, 2048)
		rlen, _, err := sock.ReadFromUDP(buf)
		if err != nil {
			fmt.Println(err)
		}

		go handlePacket(buf, rlen)
	}
}

func handlePacket(buf []byte, rlen int) {
	sentences := fmt.Sprintln(string(buf[0:rlen]))
	sentence := strings.Split(sentences, "\n")[0] // We're not concerned with the second sentence, yet
	s, err := nmea.Parse(sentence)
	if err != nil {
		// ignoring bad nmea sentence
		log.Println("Ignoring bad NMEA sentence:", sentence)
		return
	}

	if s.DataType() == nmea.TypeGGA {
		m := s.(nmea.GGA)
		fmt.Printf("Raw sentence: %v\n", m)
		fmt.Printf("Time: %s\n", m.Time)
		fmt.Printf("Latitude GPS: %f\n", m.Latitude)
		fmt.Printf("Longitude GPS: %f\n", m.Longitude)

		appendLog(m.Latitude, m.Longitude, sentence)
	}
}

func appendLog(lat, lon float64, nmeaSentence string) {
	row := fmt.Sprintf("%f,%f,%s\n", lat, lon, nmeaSentence)

	// don't log unknown locations
	if lat == 0 || lon == 0 {
		return
	}

	if _, err := dbFile.WriteString(row); err != nil {
		log.Println("unable to write to database:", err)
	}
}
