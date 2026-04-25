package webdav

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const flushInterval = 4 * time.Hour

// Client appends JSON entries to an array stored at a WebDAV URL.
// Entries are buffered in memory and flushed every 4 hours. On a day rollover,
// buffered entries are flushed to the outgoing day before loading the new day.
type Client struct {
	url      string
	username string
	password string
	http     *http.Client

	mu        sync.Mutex
	day       string
	dayURL    string
	entries   []json.RawMessage
	lastFlush time.Time
}

// NewClientFromEnv creates a Client from WEBDAV_URL, WEBDAV_USER, and
// WEBDAV_PASSWORD environment variables. Returns nil if WEBDAV_URL is unset.
func NewClientFromEnv() *Client {
	url := os.Getenv("WEBDAV_URL")
	if url == "" {
		return nil
	}
	return &Client{
		url:      url,
		username: os.Getenv("WEBDAV_USER"),
		password: os.Getenv("WEBDAV_PASSWORD"),
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

// Append buffers entry in memory and flushes to WebDAV every 4 hours.
// On a day boundary, any buffered entries are flushed to the outgoing day's
// file before the new day's file is loaded.
func (c *Client) Append(entry any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")

	if c.day != today {
		if len(c.entries) > 0 {
			if err := c.flush(); err != nil {
				return fmt.Errorf("webdav: flushing before day rollover: %w", err)
			}
		}
		if err := c.loadDay(today); err != nil {
			return err
		}
	}

	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("webdav: marshaling entry: %w", err)
	}
	c.entries = append(c.entries, json.RawMessage(entryBytes))

	if time.Since(c.lastFlush) >= flushInterval {
		return c.flush()
	}
	return nil
}

// loadDay fetches existing entries for the given day and resets the flush timer.
// Must be called with c.mu held.
func (c *Client) loadDay(day string) error {
	url := fmt.Sprintf("%s/%s.json", c.url, day)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("webdav GET: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("webdav GET: %w", err)
	}
	defer resp.Body.Close()

	c.entries = nil
	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("webdav GET read body: %w", err)
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if err := json.Unmarshal(body, &c.entries); err != nil {
				return fmt.Errorf("webdav: parsing existing array: %w", err)
			}
		}
		log.Printf("webdav: loaded %d existing entries for %s", len(c.entries), day)
	case http.StatusNotFound:
		log.Printf("webdav: no existing file for %s, starting fresh", day)
	default:
		return fmt.Errorf("webdav GET returned %s", resp.Status)
	}

	c.day = day
	c.dayURL = url
	c.lastFlush = time.Now()
	return nil
}

// flush writes all buffered entries to the current day's file.
// Must be called with c.mu held.
func (c *Client) flush() error {
	data, err := json.Marshal(c.entries)
	if err != nil {
		return fmt.Errorf("webdav: marshaling array: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, c.dayURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("webdav PUT: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("webdav PUT: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webdav PUT returned %s", resp.Status)
	}

	log.Printf("webdav: flushed %d entries to %s (%s)", len(c.entries), c.dayURL, resp.Status)
	c.lastFlush = time.Now()
	return nil
}
