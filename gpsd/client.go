package gpsd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

const defaultHost = "localhost:2947"

func host() string {
	if h := os.Getenv("GPSD_HOST"); h != "" {
		return h
	}
	return defaultHost
}

// GetLocation fetches the current GPS coordinates from the local gpsd daemon.
func GetLocation(ctx context.Context) (lat, lon float64, err error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", host())
	if err != nil {
		return 0, 0, fmt.Errorf("connecting to gpsd: %w", err)
	}
	defer conn.Close()

	// Unblock the read when the context is cancelled.
	go func() {
		<-ctx.Done()
		conn.SetDeadline(time.Now())
	}()

	if _, err := fmt.Fprintf(conn, "?WATCH={\"enable\":true,\"json\":true}\n"); err != nil {
		return 0, 0, fmt.Errorf("sending watch command: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var msg struct {
			Class string  `json:"class"`
			Mode  int     `json:"mode"`
			Lat   float64 `json:"lat"`
			Lon   float64 `json:"lon"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Class != "TPV" || msg.Mode < 2 {
			continue
		}
		return msg.Lat, msg.Lon, nil
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return 0, 0, ctx.Err()
		}
		return 0, 0, fmt.Errorf("reading from gpsd: %w", err)
	}
	return 0, 0, fmt.Errorf("gpsd closed connection without a valid fix")
}
