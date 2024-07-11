package go_defender

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewClient(t *testing.T) {

	key := "key"
	limiter := rate.NewLimiter(
		rate.Every(time.Minute),
		10,
	)
	expiresAt := time.Now()

	client := &Client[string]{
		key:       key,
		limiter:   limiter,
		expiresAt: expiresAt,
		banned:    false,
	}

	newClient := NewClient(key, limiter, expiresAt)

	assert.Equal(
		t,
		client,
		newClient,
		"Should be equal",
	)

}

func TestClientGetters(t *testing.T) {

	t.Run("key getter", func(t *testing.T) {
		client := &Client[string]{
			key: "some key",
		}
		assert.Equal(t, client.key, client.Key(), "should return the same key")
	})

	t.Run("banned getter", func(t *testing.T) {
		client := &Client[string]{
			banned: true,
		}
		assert.Equal(t, client.banned, client.Banned(), "should return the same bool")
	})

	t.Run("expiresAt getter", func(t *testing.T) {
		client := &Client[string]{
			expiresAt: time.Now(),
		}
		assert.Equal(t, client.expiresAt, client.ExpiresAt(), "should return the same time")
	})

	t.Run("expired getter true", func(t *testing.T) {
		client := &Client[string]{
			expiresAt: time.Now(),
		}
		assert.Equal(t, client.expiresAt, client.ExpiresAt(), "should return the same time")
	})

}

func TestClientBan(t *testing.T) {
	t.Run("when client not banned", func(t *testing.T) {
		client := &Client[string]{
			banned: false,
		}
		client.Ban()
		assert.Equal(t, true, client.banned, "should return true")
	})

	t.Run("when client banned", func(t *testing.T) {
		client := &Client[string]{
			banned: true,
		}
		client.Ban()
		assert.Equal(t, true, client.banned, "should return true")
	})

}

func TestClientUnBan(t *testing.T) {
	t.Run("when client not banned", func(t *testing.T) {
		client := &Client[string]{
			banned: false,
		}
		client.Unban()
		assert.Equal(t, false, client.banned, "should return false")
	})

	t.Run("when client banned", func(t *testing.T) {
		client := &Client[string]{
			banned: true,
		}
		client.Unban()
		assert.Equal(t, false, client.banned, "should return false")
	})

}

func TestClientSetExpiration(t *testing.T) {
	newExpiration := time.Now().Add(time.Hour)
	t.Run("no initial expiration", func(t *testing.T) {
		client := &Client[string]{}
		client.SetExpiration(newExpiration)
		assert.Equal(t, newExpiration, client.expiresAt, "should return newExpiration")
	})

	t.Run("with initial expiration", func(t *testing.T) {
		client := &Client[string]{
			expiresAt: time.Now(),
		}
		client.SetExpiration(newExpiration)
		assert.Equal(t, newExpiration, client.expiresAt, "should return newExpiration")
	})
}

func TestClientExpired(t *testing.T) {

	t.Run("expired", func(t *testing.T) {
		client := &Client[string]{
			expiresAt: time.Now().Add(-time.Hour),
		}
		client.Expired()
		assert.Equal(t, true, client.Expired(), "should return true")
	})

	t.Run("not expired", func(t *testing.T) {
		client := &Client[string]{
			expiresAt: time.Now().Add(time.Hour),
		}
		client.Expired()
		assert.Equal(t, false, client.Expired(), "should return false")
	})

	t.Run("ban expired", func(t *testing.T) {
		client := &Client[string]{
			banned:    true,
			expiresAt: time.Now().Add(-time.Hour),
		}
		client.BanExpired()
		assert.Equal(t, true, client.BanExpired(), "should return true")
	})

	t.Run("ban not expired", func(t *testing.T) {
		client := &Client[string]{
			banned:    false,
			expiresAt: time.Now().Add(time.Hour),
		}
		client.BanExpired()
		assert.Equal(t, false, client.BanExpired(), "should return false")
	})

}

func TestClientReachedLimit(t *testing.T) {
	t.Run("not reached", func(t *testing.T) {
		client := &Client[string]{
			limiter: rate.NewLimiter(
				rate.Every(time.Second),
				1,
			),
		}
		assert.Equal(t, false, client.ReachedLimit(), "should return false")
	})

	t.Run("reached", func(t *testing.T) {
		client := &Client[string]{
			limiter: rate.NewLimiter(
				rate.Every(time.Second),
				1,
			),
		}
		assert.Equal(t, false, client.ReachedLimit(), "first reachedLimit should return false")
		assert.Equal(t, true, client.ReachedLimit(), "second reachedLimit should return true")
	})
}
