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
// directly overhead at sea level. This is the physical upper bound; actual
// noon peaks are lower everywhere outside the tropics and scale with sin(elevation).
const atmosphericCeiling = 950.0

// solarNoonPeak returns the approximate clear-sky solar radiation (W/m²) at
// solar noon for the given latitude and date. It accounts for both seasonal
// declination and latitude so the normalization in Infer stays accurate year-round.
func solarNoonPeak(lat float64, t time.Time) float64 {
	declinationDeg := 23.45 * math.Sin(2*math.Pi/365*float64(t.YearDay()-81))
	elevationDeg := 90.0 - math.Abs(lat-declinationDeg)
	if elevationDeg <= 0 {
		return 0
	}
	return atmosphericCeiling * math.Sin(elevationDeg * math.Pi / 180)
}

// twilightSolar is the solar radiation threshold (W/m²) below which the sun is considered
// low in the sky, producing dawn or dusk conditions.
const twilightSolar = 30.0

// sunFraction returns the fraction of peak solar radiation expected at the given
// local time of day, using a simple sinusoidal model (sunrise ~6, sunset ~18).
// Returns 0 outside of daylight hours.
func sunFraction(localMinutes int) float64 {
	localMinutes = ((localMinutes % 1440) + 1440) % 1440
	localHourF := float64(localMinutes) / 60.0
	if localHourF < 6 || localHourF >= 18 {
		return 0
	}
	return math.Sin(math.Pi * (localHourF - 6) / 12.0)
}

// Infer derives a weather condition from sensor readings and approximate location.
// Solar radiation is normalized against the expected clear-sky maximum for the
// current time of day, so conditions are accurate at any hour.
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

	// Near-rain: very high humidity and overcast.
	if humidity >= 95 && solar < 10 {
		return &Condition{"Foggy", "🌫️"}
	}

	// Estimate local time from longitude (rough UTC offset).
	// Use the current wall-clock time rather than d.LastUpdated so that a stale
	// cache entry (e.g. after a connectivity gap) doesn't cause the sun-angle
	// calculation to reflect an hour from earlier in the day.
	now := time.Now().UTC()
	utcMinutes := now.Hour()*60 + now.Minute()
	localMinutes := utcMinutes + int(math.Round(lon/15))*60
	localMinutes = ((localMinutes % 1440) + 1440) % 1440
	fraction := sunFraction(localMinutes)
	isDaytime := fraction > 0
	isMorning := localMinutes < 720

	// Dawn/dusk: sensor detects low but positive solar radiation while the sun is
	// geometrically near the horizon (fraction < 0.2 or before/after model daylight).
	// isMorning distinguishes the two without hard-coding sunrise/sunset hours.
	if solar > 0 && solar < twilightSolar && (!isDaytime || fraction < 0.2) {
		if isMorning {
			return &Condition{"Dawn", "🌅"}
		}
		return &Condition{"Dusk", "🌇"}
	}

	if !isDaytime || solar < 10 {
		if !isDaytime {
			if humidity > 80 {
				return &Condition{"Cloudy night", "☁️"}
			}
			return &Condition{"Clear night", "🌙"}
		}
		// Daytime with no solar — heavy overcast or dense cloud.
		return &Condition{"Overcast", "☁️"}
	}

	// Normalize solar radiation against the expected clear-sky max for this hour.
	// Clamp to [0, 1] to absorb sensor noise or unusually bright readings (e.g.
	// cloud-edge reflection).
	peak := solarNoonPeak(lat, now)
	if peak == 0 {
		return &Condition{"Clear night", "🌙"}
	}
	normalized := math.Min(solar/(peak*fraction), 1.0)

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

