// Package fcache_test provides blackbox tests on fcache
//
package fcache_test

import (
	"errors"
	"github.com/opensvc/fcache"
	"github.com/opensvc/testhelper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

type (
	mockOutputter struct {
		output []byte
		err    error
	}

	mockLocker struct {
		lockReturn   error
		unLockReturn error
	}
)

var (
	lockerP         = func(name string) fcache.Locker { return mockLocker{} }
	lockerPFailLock = func(name string) fcache.Locker { return mockLocker{lockReturn: errors.New("lock error")} }
	maxLockDuration = time.Millisecond
)

func (t mockOutputter) Output() ([]byte, error) {
	return t.output, t.err
}

func (m mockLocker) Lock(time.Duration, string) error {
	return m.lockReturn
}

func (m mockLocker) UnLock() error {
	return m.unLockReturn
}

func randomS() string {
	return time.Now().Format("2-15-04-05.000000000")
}

func randomB() []byte {
	return []byte(randomS())
}

func TestNew(t *testing.T) {
	cacheDir, cleanup := testhelper.Tempdir(t)
	defer cleanup()

	t.Run("return value from outputter", func(t *testing.T) {
		for range []int{1, 2, 3} {
			expected := randomB()
			result, err := fcache.Output(mockOutputter{expected, nil}, randomS(), cacheDir, maxLockDuration, lockerP)
			assert.Nil(t, err)
			assert.Equalf(t, expected, result, "%q vs %q\n", expected, result)
		}
	})

	t.Run("return value from cache", func(t *testing.T) {
		// warmup cache for sig
		sig := randomS()
		expected := randomB()
		_, _ = fcache.Output(mockOutputter{expected, nil}, sig, cacheDir, maxLockDuration, lockerP)

		for range []int{1, 2, 3} {
			result, err := fcache.Output(mockOutputter{randomB(), nil}, sig, cacheDir, maxLockDuration, lockerP)
			assert.Nil(t, err)
			assert.Equalf(t, expected, result, "%q vs %q\n", expected, result)
		}
	})

	t.Run("when lock fail, return value from Outputter", func(t *testing.T) {
		sig := randomS()

		for range []int{1, 2, 3} {
			expected := randomB()
			result, err := fcache.Output(mockOutputter{expected, nil}, sig, cacheDir, maxLockDuration, lockerPFailLock)
			assert.Nil(t, err)
			assert.Equalf(t, expected, result, "%q vs %q\n", expected, result)
		}
	})

	t.Run("return error from Outputter", func(t *testing.T) {
		sig := randomS()

		for range []int{1, 2, 3} {
			_, err := fcache.Output(mockOutputter{[]byte{}, errors.New("outputter error")}, sig, cacheDir, maxLockDuration, lockerP)
			assert.NotNil(t, err)
			assert.Equal(t, "outputter error", err.Error())
		}
	})

	t.Run("return error when error from outputter", func(t *testing.T) {
		sig := randomS()

		for range []int{1, 2, 3} {
			_, err := fcache.Output(mockOutputter{[]byte{}, errors.New("outputter error")}, sig, cacheDir, maxLockDuration, lockerP)
			assert.NotNil(t, err)
			assert.Equal(t, "outputter error", err.Error())
		}
	})
}

func TestPurge(t *testing.T) {
	t.Run("allowed on empty cache", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()
		err := fcache.Purge(td)
		assert.Nil(t, err)
	})

	t.Run("purge populated cache", func(t *testing.T) {
		td, cleanup := testhelper.Tempdir(t)
		defer cleanup()
		var err error
		for range []int{1, 2, 3} {
			_, _ = fcache.Output(mockOutputter{}, randomS(), td, time.Millisecond, lockerP)
		}
		err = fcache.Purge(td)
		assert.Nil(t, err)
		_, err = os.Stat(td)
		assert.NotNil(t, err)
		assert.True(t, os.IsNotExist(err))
	})
}
