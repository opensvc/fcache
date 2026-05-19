package fcache

import (
	"errors"
	"github.com/opensvc/testhelper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

// mockLocker is a simple in-memory lock for testing
type mockLocker struct {
	locked bool
}

func (m *mockLocker) Lock(time.Duration, string) error {
	if m.locked {
		return errors.New("already locked")
	}
	m.locked = true
	return nil
}

func (m *mockLocker) UnLock() error {
	m.locked = false
	return nil
}

func newMockLock() func(name string) Locker {
	return func(name string) Locker {
		return &mockLocker{locked: false}
	}
}

func TestCreateCacheDir(t *testing.T) {
	mkdirCalls := 0
	successAfter := 0

	fakeMkdirAll := func(_ string, _ os.FileMode) error {
		if mkdirCalls >= successAfter {
			return nil
		}
		mkdirCalls = mkdirCalls + 1
		return errors.New("fail after retries")
	}

	t.Run("return error if max retry exceeded", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()
		orig := mkdirAll
		defer func() { mkdirAll = orig }()

		mkdirCalls = 0
		successAfter = 9
		mkdirAll = fakeMkdirAll

		err := mkdirAllRetry(td)
		assert.NotNil(t, err)
		assert.Equal(t, "fail after retries", err.Error())
	})

	t.Run("return nil if succeed before max retry exceeded", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()
		orig := mkdirAll
		defer func() { mkdirAll = orig }()

		mkdirCalls = 0
		successAfter = 2
		mkdirAll = fakeMkdirAll

		err := mkdirAllRetry(td)
		assert.Nil(t, err)
	})
}

func TestAge(t *testing.T) {
	t.Run("returns error when cache file does not exist", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()

		age, err := Age("test", td, time.Second, newMockLock())
		assert.NotNil(t, err)
		assert.True(t, os.IsNotExist(err))
		assert.Equal(t, time.Duration(0), age)
	})

	t.Run("returns age of cache file", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()

		// Create cache file 1 second ago
		sig := "test"
		cachePath := cacheFile(td, sig)
		content := []byte("test content")
		err := os.WriteFile(cachePath, content, 0600)
		assert.Nil(t, err)

		// Set mod time to 1 second ago
		now := time.Now()
		oneSecAgo := now.Add(-1 * time.Second)
		err = os.Chtimes(cachePath, oneSecAgo, oneSecAgo)
		assert.Nil(t, err)

		// Age should be approximately 1 second
		age, err := Age(sig, td, time.Second, newMockLock())
		assert.Nil(t, err)
		assert.True(t, age >= time.Second && age < 2*time.Second, "age should be approximately 1 second, got %v", age)
	})

	t.Run("returns error when lock fails", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()

		// Create a lock that always fails
		badLock := func(name string) Locker {
			return &mockLocker{locked: true}
		}

		age, err := Age("test", td, time.Second, badLock)
		assert.NotNil(t, err)
		assert.Equal(t, time.Duration(0), age)
	})
}
