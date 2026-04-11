package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/acaloiaro/roam-location/geocode"
	"github.com/acaloiaro/roam-location/service"
	"github.com/acaloiaro/roam-location/starlink"
	"github.com/acaloiaro/roam-location/weather"
	"github.com/acaloiaro/roam-location/webdav"
)

type locationEntry struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Name string  `json:"name,omitempty"`
}

type snapshot struct {
	RecordedAt time.Time          `json:"recorded_at"`
	Location   *locationEntry     `json:"location,omitempty"`
	Weather    *weather.Data      `json:"weather,omitempty"`
	Conditions *weather.Condition `json:"conditions,omitempty"`
}

type cache struct {
	mu    sync.RWMutex
	lat   float64
	lon   float64
	name  string
	ready bool
}

func (c *cache) set(lat, lon float64, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lat, c.lon, c.name, c.ready = lat, lon, name, true
}

func (c *cache) get() (lat, lon float64, name string, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lat, c.lon, c.name, c.ready
}

func pollInterval() time.Duration {
	if s := os.Getenv("POLL_INTERVAL_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 30 * time.Second
}

func main() {
	loc := &cache{}
	wCache := weather.NewCache()
	interval := pollInterval()
	wdClient := webdav.NewClientFromEnv()

	go func() {
		for {
			lat, lon, err := starlink.GetLocation(context.Background())
			if err != nil {
				log.Printf("location poll error: %v", err)
			} else {
				city, err := geocode.CityCenter(context.Background(), lat, lon)
				if err != nil {
					log.Printf("geocode error: %v", err)
				} else {
					loc.set(city.Lat, city.Lon, city.Name)
					log.Printf("location updated: %.6f, %.6f (%s)", city.Lat, city.Lon, city.Name)

					if wdClient != nil {
						snap := snapshot{
							RecordedAt: time.Now().UTC(),
							Location:   &locationEntry{Lat: city.Lat, Lon: city.Lon, Name: city.Name},
						}
						if data, ok := wCache.Get(); ok {
							snap.Weather = &data
							snap.Conditions = weather.Infer(data, city.Lat, city.Lon)
						}
						if err := wdClient.Append(snap); err != nil {
							log.Printf("webdav append error: %v", err)
						}
					}
				}
			}
			time.Sleep(interval)
		}
	}()

	service.Listen(loc.get, wCache)
}
