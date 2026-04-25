package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Location is a city-level position with the centroid coordinates of the matched place.
type Location struct {
	Lat  float64
	Lon  float64
	Name string // e.g. "St. George, Utah"
}

type nominatimResponse struct {
	Lat     string `json:"lat"`
	Lon     string `json:"lon"`
	Address struct {
		City    string `json:"city"`
		Town    string `json:"town"`
		Village string `json:"village"`
		County  string `json:"county"`
		State   string `json:"state"`
	} `json:"address"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// CityCenter reverse-geocodes lat/lon to the centroid of the matched city.
// zoom=10 asks Nominatim for city-level granularity; the returned coordinates
// are the centroid of that administrative boundary, not the input coordinates.
func CityCenter(ctx context.Context, lat, lon float64) (Location, error) {
	url := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?lat=%f&lon=%f&format=json&zoom=10",
		lat, lon,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Location{}, err
	}
	req.Header.Set("User-Agent", "roam-location (https://adriano.fyi/whereami)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()

	var result nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Location{}, err
	}

	resLat, err := strconv.ParseFloat(result.Lat, 64)
	if err != nil {
		return Location{}, fmt.Errorf("parsing lat %q: %w", result.Lat, err)
	}
	resLon, err := strconv.ParseFloat(result.Lon, 64)
	if err != nil {
		return Location{}, fmt.Errorf("parsing lon %q: %w", result.Lon, err)
	}

	a := result.Address
	place := a.City
	if place == "" {
		place = a.Town
	}
	if place == "" {
		place = a.Village
	}
	if place == "" {
		place = a.County
	}
	name := place
	if a.State != "" {
		if name != "" {
			name += ", "
		}
		name += a.State
	}

	return Location{Lat: resLat, Lon: resLon, Name: name}, nil
}
