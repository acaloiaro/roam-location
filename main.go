package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/acaloiaro/roam-location/receiver"
	"github.com/acaloiaro/roam-location/service"
)

var dbPath string

func init() {
	var ok bool
	if dbPath, ok = os.LookupEnv("ROAM_LOCATION_DB_PATH"); !ok {
		log.Fatalf("ROAM_LOCATION_DB_PATH environment variable not set. Please set it before running")
	}
}

func main() {
	modePtr := flag.String("mode", "listener", "set application mode to either 'listener' (listening for NMEA sentences) or 'service' (providing current coordinates)")

	flag.Parse()

	switch *modePtr {
	case "listener":
		fmt.Println("Listener mode")
		receiver.Receive(dbPath)

	case "service":
		fmt.Println("Service mode")
		service.Listen(dbPath)
	}

}
