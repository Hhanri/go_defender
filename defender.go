package go_defender

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const Factor = 10

type Defender[T comparable] struct {
	clients map[T]*Client[T]

	Duration    time.Duration
	BanDuration time.Duration
	Max         int

	sync.Mutex
}

func NewDefender[T comparable](max int, duration, banDuration time.Duration) *Defender[T] {
	return &Defender[T]{
		clients:     map[T]*Client[T]{},
		Duration:    duration,
		BanDuration: banDuration,
		Max:         max,
		Mutex:       sync.Mutex{},
	}
}

func (d *Defender[T]) BanList() []*Client[T] {
	banList := make([]*Client[T], 0, len(d.clients))

	for _, client := range d.clients {
		if client.banned {
			banList = append(banList, client)
		}
	}

	return banList
}

func (d *Defender[T]) Client(key T) (*Client[T], bool) {
	d.Lock()
	defer d.Unlock()
	client, ok := d.clients[key]
	return client, ok
}

func (d *Defender[T]) newLimiter() *rate.Limiter {
	return rate.NewLimiter(
		rate.Every(d.Duration),
		d.Max,
	)
}

func (d *Defender[T]) newClientExpiration(now time.Time, duration time.Duration) time.Time {
	return now.Add(duration)
}

func (d *Defender[T]) newValidClientExpiration(now time.Time) time.Time {
	return d.newClientExpiration(now, d.Duration*Factor)
}

func (d *Defender[T]) newBannedClientExpiration(now time.Time) time.Time {
	return d.newClientExpiration(now, d.BanDuration)
}

func (d *Defender[T]) Increment(key T, now time.Time) bool {
	d.Lock()
	defer d.Unlock()

	client, found := d.clients[key]

	if !found {
		d.clients[key] = NewClient(
			key,
			d.newLimiter(),
			d.newValidClientExpiration(now),
		)
		return false
	}

	// Check if the client is not banned anymore and the cleanup hasn't been run yet
	if client.BanExpired() {
		client.Unban()
	}

	// Check if client is banned
	if client.Banned() {
		return true
	}

	// Update the client expiration
	client.SetExpiration(
		d.newValidClientExpiration(now),
	)

	// Check the rate limiter
	if client.ReachedLimit() {
		client.Ban()
		client.SetExpiration(
			d.newBannedClientExpiration(now),
		)
		return true
	}

	return false
}

func (d *Defender[T]) Cleanup() {
	d.Lock()
	defer d.Unlock()

	for key, client := range d.clients {
		if client.Expired() {
			delete(d.clients, key)
		}
	}
}

// use this inside a Goroutine
func (d *Defender[T]) CleanupTask(quitCh <-chan struct{}) {
	t := time.NewTicker(d.Duration * Factor)
	for {
		select {
		case <-quitCh:
			return
		case <-t.C:
			d.Cleanup()
		}
	}
}
