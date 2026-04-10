package weather

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Data holds sensor readings from an Ambient Weather WS-5000 push.
// All numeric fields use *float64 so missing/optional sensors are omitted from JSON.
type Data struct {
	// Outdoor
	TempF         *float64 `json:"tempf,omitempty"`
	Humidity      *float64 `json:"humidity,omitempty"`
	FeelsLike     *float64 `json:"feelsLike,omitempty"`
	DewPoint      *float64 `json:"dewPoint,omitempty"`
	WindDir       *float64 `json:"winddir,omitempty"`
	WindSpeedMph  *float64 `json:"windspeedmph,omitempty"`
	WindGustMph   *float64 `json:"windgustmph,omitempty"`
	MaxDailyGust  *float64 `json:"maxdailygust,omitempty"`
	SolarRadiation *float64 `json:"solarradiation,omitempty"`
	UV            *float64 `json:"uv,omitempty"`

	// Rain
	HourlyRainIn   *float64 `json:"hourlyrainin,omitempty"`
	EventRainIn    *float64 `json:"eventrainin,omitempty"`
	DailyRainIn    *float64 `json:"dailyrainin,omitempty"`
	WeeklyRainIn   *float64 `json:"weeklyrainin,omitempty"`
	MonthlyRainIn  *float64 `json:"monthlyrainin,omitempty"`
	TotalRainIn    *float64 `json:"totalrainin,omitempty"`
	LastRain       string   `json:"lastRain,omitempty"`

	// Pressure
	BaromRelIn *float64 `json:"baromrelin,omitempty"`
	BaromAbsIn *float64 `json:"baromabsin,omitempty"`

	// Indoor
	TempInF      *float64 `json:"tempinf,omitempty"`
	HumidityIn   *float64 `json:"humidityin,omitempty"`
	FeelsLikeIn  *float64 `json:"feelsLikein,omitempty"`
	DewPointIn   *float64 `json:"dewPointin,omitempty"`

	// Optional sensors
	PM25      *float64 `json:"pm25,omitempty"`
	PM25_24h  *float64 `json:"pm25_24h,omitempty"`
	PM25In    *float64 `json:"pm25in,omitempty"`
	PM25In24h *float64 `json:"pm25in_24h,omitempty"`
	CO2       *float64 `json:"co2,omitempty"`

	// Meta
	StationType string    `json:"stationtype,omitempty"`
	DateUTC     string    `json:"dateutc,omitempty"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// Cache holds the latest weather reading from the station push.
type Cache struct {
	mu    sync.RWMutex
	data  Data
	ready bool
}

func NewCache() *Cache {
	return &Cache{}
}

// Update parses an Ambient Weather push request and stores the reading.
func (c *Cache) Update(r *http.Request) {
	q := r.URL.Query()
	d := Data{
		TempF:          parseFloat(q.Get("tempf")),
		Humidity:       parseFloat(q.Get("humidity")),
		FeelsLike:      parseFloat(q.Get("feelsLike")),
		DewPoint:       parseFloat(q.Get("dewPoint")),
		WindDir:        parseFloat(q.Get("winddir")),
		WindSpeedMph:   parseFloat(q.Get("windspeedmph")),
		WindGustMph:    parseFloat(q.Get("windgustmph")),
		MaxDailyGust:   parseFloat(q.Get("maxdailygust")),
		SolarRadiation: parseFloat(q.Get("solarradiation")),
		UV:             parseFloat(q.Get("uv")),
		HourlyRainIn:   parseFloat(q.Get("hourlyrainin")),
		EventRainIn:    parseFloat(q.Get("eventrainin")),
		DailyRainIn:    parseFloat(q.Get("dailyrainin")),
		WeeklyRainIn:   parseFloat(q.Get("weeklyrainin")),
		MonthlyRainIn:  parseFloat(q.Get("monthlyrainin")),
		TotalRainIn:    parseFloat(q.Get("totalrainin")),
		LastRain:       q.Get("lastRain"),
		BaromRelIn:     parseFloat(q.Get("baromrelin")),
		BaromAbsIn:     parseFloat(q.Get("baromabsin")),
		TempInF:        parseFloat(q.Get("tempinf")),
		HumidityIn:     parseFloat(q.Get("humidityin")),
		FeelsLikeIn:    parseFloat(q.Get("feelsLikein")),
		DewPointIn:     parseFloat(q.Get("dewPointin")),
		PM25:           parseFloat(q.Get("pm25")),
		PM25_24h:       parseFloat(q.Get("pm25_24h")),
		PM25In:         parseFloat(q.Get("pm25in")),
		PM25In24h:      parseFloat(q.Get("pm25in_24h")),
		CO2:            parseFloat(q.Get("co2")),
		StationType:    q.Get("stationtype"),
		DateUTC:        q.Get("dateutc"),
		LastUpdated:    time.Now().UTC(),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = d
	c.ready = true
}

// Get returns the latest cached reading and whether one has been received yet.
func (c *Cache) Get() (Data, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data, c.ready
}

func parseFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
