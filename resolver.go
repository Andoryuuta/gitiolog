package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const b62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func encodeID(id uint64) string {
	var out []byte

	for id > 0 {
		tmp := id % 62
		id /= 62
		out = append(out, b62Alphabet[tmp])
	}

	return string(out)
}

// ResolvedShortlink represents a single resolved shortlink
type ResolvedShortlink struct {
	isError     bool
	url         string
	resolvedURL string
}

// Resolver resolves git.io shortlinks
type Resolver struct {
	requestCounter uint64
	startTime      time.Time
	workerCount    int
}

func newResolver() *Resolver {
	return &Resolver{
		workerCount: 100,
		startTime:   time.Now(),
	}
}

// GetRPS returns req/sec
func (r *Resolver) GetRPS() float64 {
	return float64(r.requestCounter) / time.Since(r.startTime).Seconds()
}

func (r *Resolver) startWorker(queue chan string, output chan ResolvedShortlink) {
	go func() {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 1024,
				TLSHandshakeTimeout: 0 * time.Second,
			},
		}

		for {
			url := <-queue
			atomic.AddUint64(&r.requestCounter, 1)

			resp, err := client.Head(url)
			if err != nil {
				output <- ResolvedShortlink{
					isError: true,
				}
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == 302 {
				resolvedURL, err := resp.Location()
				if err != nil {
					output <- ResolvedShortlink{
						isError: true,
					}
					continue
				}

				output <- ResolvedShortlink{
					isError:     false,
					url:         url,
					resolvedURL: resolvedURL.String(),
				}
			} else if resp.StatusCode == 404 {
				output <- ResolvedShortlink{
					isError:     false,
					url:         url,
					resolvedURL: "",
				}
			} else {
				output <- ResolvedShortlink{
					isError: true,
				}
			}
		}
	}()
}

// ResolveRange resolves (bruteforce) a range of git.io shortlinks.
func (r *Resolver) ResolveRange(start uint64, end uint64, outputChannel chan ResolvedShortlink) {
	workChannel := make(chan string)

	for i := 0; i < r.workerCount; i++ {
		r.startWorker(workChannel, outputChannel)
	}

	for i := start; i < end; i++ {
		url := fmt.Sprintf("https://git.io/%s", encodeID(i))
		workChannel <- url
	}
}
