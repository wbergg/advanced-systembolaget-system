package printer

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Enabled bool
	URL     string
}

type Client struct {
	enabled bool
	url     string
	http    *http.Client
	jobs    chan job
}

type job struct {
	kind string // "print", "cut", or "print_and_cut"
	text string
}

// cutDelay is how long the worker waits between the previous print POST
// returning and firing the cutter. Gives the printer hardware time to
// finish feeding paper before cut_command preempts the buffer.
const cutDelay = 10 * time.Second

func New(cfg Config) *Client {
	c := &Client{
		enabled: cfg.Enabled && cfg.URL != "",
		url:     cfg.URL,
		http:    &http.Client{Timeout: 3 * time.Second},
		jobs:    make(chan job, 32),
	}
	if c.enabled {
		go c.worker()
	}
	return c
}

// Send queues a print job. Non-blocking.
func (c *Client) Send(text string) {
	if c == nil || !c.enabled {
		return
	}
	c.enqueue(job{kind: "print", text: text})
}

// Cut queues a paper-cut job. Non-blocking. The worker inserts a short
// delay before the cut POST so any preceding print has time to finish.
func (c *Client) Cut() {
	if c == nil || !c.enabled {
		return
	}
	c.enqueue(job{kind: "cut"})
}

// SendAndCut prints text and cuts paper in a single POST (one HTTP request
// carrying both print_text and cut_command form fields). This sidesteps the
// firmware preemption issue where cut_command in a separate POST bypasses the
// print buffer. Non-blocking.
func (c *Client) SendAndCut(text string) {
	if c == nil || !c.enabled {
		return
	}
	c.enqueue(job{kind: "print_and_cut", text: text})
}

func (c *Client) enqueue(j job) {
	select {
	case c.jobs <- j:
	default:
		log.Printf("printer: job queue full, dropping %s", j.kind)
	}
}

// worker processes jobs one at a time, in FIFO order, for the lifetime of
// the Client. Single-worker serialization removes the POST-ordering race
// that existed when every Send/Cut spawned its own goroutine.
func (c *Client) worker() {
	for j := range c.jobs {
		switch j.kind {
		case "print":
			c.printText(j.text)
		case "cut":
			time.Sleep(cutDelay)
			c.cut()
		case "print_and_cut":
			c.printAndCut(j.text)
		}
	}
}

func (c *Client) printText(text string) {
	form := url.Values{}
	form.Set("stext", text)
	form.Set("print_text", "Print Text")
	c.post(form.Encode())
}

func (c *Client) cut() {
	form := url.Values{}
	form.Set("cut_command", "Cut")
	c.post(form.Encode())
}

func (c *Client) printAndCut(text string) {
	form := url.Values{}
	form.Set("stext", text)
	form.Set("print_text", "Print Text")
	form.Set("cut_command", "Cut")
	c.post(form.Encode())
}

func (c *Client) post(rawBody string) {
	resp, err := c.http.Post(c.url,
		"application/x-www-form-urlencoded",
		strings.NewReader(rawBody))
	if err != nil {
		log.Printf("printer: POST failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("printer: POST returned %d", resp.StatusCode)
	}
}
