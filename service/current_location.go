package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/acaloiaro/roam-location/weather"
)

const port = 22495

type locationData struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name,omitempty"`
}

type whereAmI struct {
	Location   *locationData      `json:"location,omitempty"`
	Weather    *weather.Data      `json:"weather,omitempty"`
	Conditions *weather.Condition `json:"conditions,omitempty"`
}

var allowedOrigins = map[string]bool{
	"https://adriano.fyi":   true,
	"http://localhost:1313": true,
}

func setCORS(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); allowedOrigins[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
}

// Listen registers all HTTP endpoints and starts the server.
func Listen(getLocation func() (lat, lon float64, name string, ok bool), wCache *weather.Cache) {
	http.HandleFunc("/whereami", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, r)

		resp := whereAmI{}

		if lat, lon, name, ok := getLocation(); ok {
			resp.Location = &locationData{Lat: lat, Lon: lon, Name: name}
		}

		if data, ok := wCache.Get(); ok {
			resp.Weather = &data
			if resp.Location != nil {
				resp.Conditions = weather.Infer(data, resp.Location.Lat, resp.Location.Lon)
			} else {
				resp.Conditions = weather.Infer(data, 0, 0)
			}
		}

		if resp.Location == nil && resp.Weather == nil {
			http.Error(w, "no data yet", http.StatusServiceUnavailable)
			return
		}

		b, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			log.Printf("error marshaling whereami: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, string(b))
	})

	// Receives pushes from the Ambient Weather WS-5000 via the awnet custom server config.
	// The station sends params as /data/report/&key=val&... (& instead of ?) so we fix that up.
	http.HandleFunc("/data/report/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery == "" {
			if i := strings.Index(r.RequestURI, "&"); i >= 0 {
				r.URL.RawQuery = r.RequestURI[i+1:]
			}
		}
		wCache.Update(r)
		if data, ok := wCache.Get(); ok {
			tempStr := "n/a"
			if data.TempF != nil {
				tempStr = fmt.Sprintf("%.1f°F", *data.TempF)
			}
			log.Printf("weather update received: temp=%s station=%s", tempStr, data.StationType)
		}
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("listening on 0.0.0.0:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
