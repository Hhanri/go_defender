package go_defender

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewDefender(t *testing.T) {
	duration := time.Hour
	banDuration := time.Hour * 4
	max := 160

	defender := &Defender[string]{
		clients:     map[string]*Client[string]{},
		Duration:    duration,
		BanDuration: banDuration,
		Max:         max,
		Mutex:       sync.Mutex{},
	}

	newDefender := NewDefender[string](max, duration, banDuration)

	assert.Equal(
		t,
		defender,
		newDefender,
		"should be equal",
	)
}

func TestDefenderBanList(t *testing.T) {
	bannedClient := &Client[string]{banned: true}
	unbannedClient := &Client[string]{banned: false}

	clients := map[string]*Client[string]{
		"key1": bannedClient,
		"key2": unbannedClient,
		"key3": bannedClient,
	}
	defender := Defender[string]{
		clients: clients,
	}

	assert.Equal(
		t,
		[]*Client[string]{bannedClient, bannedClient},
		defender.BanList(),
		"should return every banned clients",
	)
}

func TestDefenderClient(t *testing.T) {
	client := &Client[string]{banned: true, key: "key1"}

	clients := map[string]*Client[string]{
		client.key: client,
	}
	defender := Defender[string]{
		clients: clients,
	}

	t.Run("existing client", func(t *testing.T) {
		c, ok := defender.Client(client.key)
		assert.Equal(
			t,
			client,
			c,
			"should return client",
		)
		assert.Equal(
			t,
			true,
			ok,
			"should return true",
		)
	})
	t.Run("non existing client", func(t *testing.T) {
		c, ok := defender.Client("key2")
		assert.Equal(
			t,
			(*Client[string])(nil),
			c,
			"should return empty client",
		)
		assert.Equal(
			t,
			false,
			ok,
			"should return false",
		)
	})
}

func TestDefenderNewLimiter(t *testing.T) {

	max := 50
	duration := time.Minute
	defender := NewDefender[string](max, duration, time.Hour)
	rl := defender.newLimiter()
	assert.Equal(
		t,
		rate.NewLimiter(rate.Every(duration), max),
		rl,
		"should be equal",
	)

}

func TestDefenderNewClientExpiration(t *testing.T) {
	max := 50
	duration := time.Minute
	banDuration := time.Hour
	defender := NewDefender[string](max, duration, banDuration)
	now := time.Now()

	t.Run("valid client expiration", func(t *testing.T) {
		assert.Equal(
			t,
			now.Add(duration*Factor),
			defender.newValidClientExpiration(now),
		)
	})

	t.Run("banned client expiration", func(t *testing.T) {
		assert.Equal(
			t,
			now.Add(banDuration),
			defender.newBannedClientExpiration(now),
		)
	})
}

func TestDefenderCleanup(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	unbannedClient := &Client[string]{banned: false, key: "key1", expiresAt: future}
	bannedClient := &Client[string]{banned: false, key: "key2", expiresAt: future}
	expiredUnbannedClient := &Client[string]{banned: false, key: "key3", expiresAt: past}
	expiredBannedClient := &Client[string]{banned: false, key: "key4", expiresAt: past}

	clients := map[string]*Client[string]{
		unbannedClient.key:        unbannedClient,
		bannedClient.key:          bannedClient,
		expiredUnbannedClient.key: expiredUnbannedClient,
		expiredBannedClient.key:   expiredBannedClient,
	}
	defender := Defender[string]{
		clients: clients,
	}

	defender.Cleanup()

	assert.Equal(
		t,
		map[string]*Client[string]{
			unbannedClient.key: unbannedClient,
			bannedClient.key:   bannedClient,
		},
		defender.clients,
		"should only keep non expired clients",
	)
}

func TestDefenderIncrement(t *testing.T) {
	duration := time.Minute
	banDuration := time.Hour
	max := 50
	now := time.Now()

	type testCaseStruct struct {
		title           string
		initialClients  map[string]*Client[string]
		expectedClients map[string]*Client[string]
		key             string
		expectedResult  bool
	}

	unregisteredClientTestCase := testCaseStruct{
		title:          "unregistered client",
		initialClients: map[string]*Client[string]{},
		expectedClients: map[string]*Client[string]{
			"key1": NewClient(
				"key1",
				rate.NewLimiter(rate.Every(duration), max),
				now.Add(duration*Factor),
			),
		},
		key:            "key1",
		expectedResult: false,
	}

	banExpiredClientTestCase := func() testCaseStruct {
		limiter := rate.NewLimiter(rate.Every(duration), max)
		return testCaseStruct{
			title: "expired banned client",
			initialClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(-time.Hour),
					banned:    true,
				},
			},
			expectedClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(duration * Factor),
					banned:    false,
				},
			},
			key:            "key1",
			expectedResult: false,
		}
	}()

	banExpiredWithLimitReachedClientTestCase := func() testCaseStruct {
		limiter := rate.NewLimiter(rate.Every(duration), 1)
		limiter.AllowN(now, 1)
		return testCaseStruct{
			title: "expired banned client with limit reached",
			initialClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(-time.Hour),
					banned:    true,
				},
			},
			expectedClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(banDuration),
					banned:    true,
				},
			},
			key:            "key1",
			expectedResult: true,
		}
	}()

	bannedClientTestCase := func() testCaseStruct {
		limiter := rate.NewLimiter(rate.Every(duration), max)
		return testCaseStruct{
			title: "banned client, not expired",
			initialClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(time.Hour),
					banned:    true,
				},
			},
			expectedClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(time.Hour),
					banned:    true,
				},
			},
			key:            "key1",
			expectedResult: true,
		}
	}()

	compareClientsMap := func(t *testing.T, expected, actual map[string]*Client[string]) {
		for key, client := range expected {

			actualClient, ok := actual[key]
			if !ok {
				t.Errorf("Missing client %s\n", key)
			}

			assert.Equal(
				t,
				client,
				actualClient,
			)

		}
	}

	notBannedClientTestCase := func() testCaseStruct {
		limiter := rate.NewLimiter(rate.Every(duration), max)
		return testCaseStruct{
			title: "not banned client",
			initialClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(time.Hour),
					banned:    false,
				},
			},
			expectedClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(duration * Factor),
					banned:    false,
				},
			},
			key:            "key1",
			expectedResult: false,
		}
	}()

	notBannedWithLimitReachedClientTestCase := func() testCaseStruct {
		limiter := rate.NewLimiter(rate.Every(duration), 1)
		limiter.AllowN(now, 1)
		return testCaseStruct{
			title: "not banned client",
			initialClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(time.Hour),
					banned:    false,
				},
			},
			expectedClients: map[string]*Client[string]{
				"key1": {
					key:       "key1",
					limiter:   limiter,
					expiresAt: now.Add(banDuration),
					banned:    true,
				},
			},
			key:            "key1",
			expectedResult: true,
		}
	}()

	testCases := []testCaseStruct{
		unregisteredClientTestCase,
		banExpiredClientTestCase,
		banExpiredWithLimitReachedClientTestCase,
		bannedClientTestCase,
		notBannedClientTestCase,
		notBannedWithLimitReachedClientTestCase,
	}

	for _, testCase := range testCases {

		t.Run(testCase.title, func(t *testing.T) {

			defender := &Defender[string]{
				clients:     testCase.initialClients,
				Duration:    duration,
				BanDuration: banDuration,
				Max:         max,
				Mutex:       sync.Mutex{},
			}
			result := defender.Increment(testCase.key, now)
			assert.Equalf(
				t,
				testCase.expectedResult,
				result,
				"increment result should be %s",
				testCase.expectedResult,
			)
			compareClientsMap(t, testCase.expectedClients, defender.clients)

		})

	}

}
