# roam-location
The purpose of this repo is to make my current location available on adriano.fyi. The coordinates from the two services here are used populate the map at https://adriano.fyi/whereami


# About

I can't imagine this is useful to many others. This repo consits of a receiver of GPS data and a web service that makes the data avaialble.

## Receiver
The receiver receives UDP packets containing NMEA sentences from my Sierra Wireless RV55 router. The sentences are parsed and appended to a log file.

## Location Service

The location service simply finds the last location appended to the log file and makes it available as a JSON object over HTTP.

# Running

ROAM_LOCATION_DB_PATH=/path/to/nmea_logs.log go run main.go
