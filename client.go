package go_defender

import (
	"time"

	"golang.org/x/time/rate"
)

type Client[T comparable] struct {
	limiter   *rate.Limiter
	expiresAt time.Time
	banned    bool
	key       T
}

func (c *Client[T]) Key() interface{} { return c.key }

func (c *Client[T]) Banned() bool { return c.banned }

func (c *Client[T]) ExpiresAt() time.Time { return c.expiresAt }

func (c *Client[T]) Expired() bool {
	return time.Now().After(c.expiresAt)
}

func (c *Client[T]) BanExpired() bool {
	return c.banned && c.Expired()
}

func (c *Client[T]) Ban() {
	c.banned = true
}

func (c *Client[T]) Unban() {
	c.banned = false
}

func (c *Client[T]) SetExpiration(expireAt time.Time) {
	c.expiresAt = expireAt
}

func (c *Client[T]) ReachedLimit() bool {
	return !c.limiter.AllowN(time.Now(), 1)
}

func NewClient[T comparable](key T, limiter *rate.Limiter, expiresAt time.Time) *Client[T] {
	return &Client[T]{
		key:       key,
		limiter:   limiter,
		expiresAt: expiresAt,
	}
}
