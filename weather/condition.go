package weather

import (
	"math"
	"time"
)

// Condition is an inferred weather condition with a human label and emoji.
type Condition struct {
	Label string `json:"label"`
	Emoji string `json:"emoji"`
}

// atmosphericCeiling is the clear-sky solar radiation (W/m²) with the sun
// directly overhead at sea level. Actual clear-sky radiation at any other
// elevation angle scales by sin(elevation).
const atmosphericCeiling = 950.0

// solarPosition returns the solar elevation angle (radians) and hour angle
// (degrees) for the given latitude, longitude, and UTC time.
//
// Elevation > 0 means the sun is above the horizon.
// Hour angle < 0 means the sun is east of the meridian (morning);
// > 0 means west of the meridian (afternoon).
func solarPosition(lat, lon float64, t time.Time) (elevationRad, hourAngleDeg float64) {
	decRad := 23.45 * math.Pi / 180 * math.Sin(2*math.Pi/365*float64(t.YearDay()-81))
	utcHours := float64(t.Hour()) + float64(t.Minute())/60.0
	// math.Remainder normalises to [-180, 180], handling midnight crossings cleanly.
	hourAngleDeg = math.Remainder((utcHours-(12.0-lon/15.0))*15, 360)
	hourAngleRad := hourAngleDeg * math.Pi / 180
	latRad := lat * math.Pi / 180
	sinElev := math.Sin(latRad)*math.Sin(decRad) + math.Cos(latRad)*math.Cos(decRad)*math.Cos(hourAngleRad)
	elevationRad = math.Asin(math.Max(-1, math.Min(1, sinElev)))
	return
}

// Infer derives a weather condition from sensor readings and position.
// Solar radiation is normalised against the clear-sky expectation for the
// actual solar elevation angle, so conditions are accurate at any time of day
// without any hardcoded sunrise or sunset times.
func Infer(d Data, lat, lon float64) *Condition {
	solar := 0.0
	if d.SolarRadiation != nil {
		solar = *d.SolarRadiation
	}
	humidity := 0.0
	if d.Humidity != nil {
		humidity = *d.Humidity
	}

	// Rain takes priority.
	if d.HourlyRainIn != nil && *d.HourlyRainIn > 0.02 {
		if *d.HourlyRainIn > 0.1 {
			return &Condition{"Heavy rain", "⛈️"}
		}
		return &Condition{"Rainy", "🌧️"}
	}

	// Near-rain: very high humidity with no solar.
	if humidity >= 95 && solar < 10 {
		return &Condition{"Foggy", "🌫️"}
	}

	elevationRad, hourAngleDeg := solarPosition(lat, lon, time.Now().UTC())
	isMorning := hourAngleDeg < 0 // sun east of meridian

	if elevationRad <= 0 {
		// Sun is below the horizon.
		if solar > 0 {
			// Sensor still catching scattered twilight.
			if isMorning {
				return &Condition{"Dawn", "🌅"}
			}
			return &Condition{"Dusk", "🌇"}
		}
		if humidity > 80 {
			return &Condition{"Cloudy night", "☁️"}
		}
		return &Condition{"Clear night", "🌙"}
	}

	// Sun is above the horizon. Normalise against the clear-sky radiation
	// expected at this elevation angle.
	expectedClearSky := atmosphericCeiling * math.Sin(elevationRad)
	if solar < 10 {
		return &Condition{"Overcast", "☁️"}
	}
	normalized := math.Min(solar/expectedClearSky, 1.0)

	if normalized > 0.85 {
		return &Condition{"Sunny", "🌞"}
	}
	if normalized > 0.55 {
		return &Condition{"Partly cloudy", "⛅"}
	}
	if normalized > 0.15 {
		return &Condition{"Mostly cloudy", "🌥️"}
	}
	return &Condition{"Overcast", "☁️"}
}
