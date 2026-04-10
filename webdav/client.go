package webdav

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Client appends JSON entries to an array stored at a WebDAV URL.
type Client struct {
	url      string
	username string
	password string
	http     *http.Client
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

// dailyURL returns the URL for today's date file under the base directory.
func (c *Client) dailyURL() string {
	return fmt.Sprintf("%s/%s.json", c.url, time.Now().UTC().Format("2006-01-02"))
}

// Append fetches the existing JSON array for today's date file, appends entry,
// and writes the result back. A missing file (404) is treated as an empty array.
func (c *Client) Append(entry any) error {
	url := c.dailyURL()
	log.Printf("webdav: fetching %s", url)
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

	var arr []json.RawMessage
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("webdav GET read body: %w", err)
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if err := json.Unmarshal(body, &arr); err != nil {
				return fmt.Errorf("webdav: parsing existing array: %w", err)
			}
		}
		log.Printf("webdav: got existing array with %d entries", len(arr))
	} else if resp.StatusCode == http.StatusNotFound {
		log.Printf("webdav: file not found, starting new array")
	} else {
		return fmt.Errorf("webdav GET returned %s", resp.Status)
	}

	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("webdav: marshaling entry: %w", err)
	}
	arr = append(arr, json.RawMessage(entryBytes))

	updated, err := json.Marshal(arr)
	if err != nil {
		return fmt.Errorf("webdav: marshaling array: %w", err)
	}

	log.Printf("webdav: writing %d entries (%d bytes) to %s", len(arr), len(updated), url)
	putReq, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(updated))
	if err != nil {
		return fmt.Errorf("webdav PUT: %w", err)
	}
	putReq.SetBasicAuth(c.username, c.password)
	putReq.Header.Set("Content-Type", "application/json")

	putResp, err := c.http.Do(putReq)
	if err != nil {
		return fmt.Errorf("webdav PUT: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return fmt.Errorf("webdav PUT returned %s", putResp.Status)
	}

	log.Printf("webdav: write succeeded (%s)", putResp.Status)
	return nil
}
