// Package fcache provides named cache for functions results
//
// supported functions are:
//   - Output for OutPutters: Output() ([]byte, error)
//
// cache backend is filesystem directory, and functions are protected by a Lock/Unlock lock := func(name string) Locker
package fcache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type (
	Locker interface {
		Lock(time.Duration, string) (err error)
		UnLock() (err error)
	}

	Outputter interface {
		Output() ([]byte, error)
	}
)

var (
	mkdirAll           = os.MkdirAll
	mkdirAllMaxRetries = 3
)

// Purge cache backend directory 'cacheDir'
func Purge(cacheDir string) (err error) {
	return os.RemoveAll(cacheDir)
}

// Output function returns previous o.Output() results named 'sig' from 'cacheDir', if not found in cache a new value
// is computed, and cache is updated.
//
// A lock object associated with 'sig' (created by function 'lockP') is used,
//
// When lock error:
//   Returns o.Output() (no update cache).
//
// When locked:
//   returns found cache entry associated with 'sig' from 'cacheDir'.
//   If entry is not present, it runs 'o.Output()'.
//     In case of error during o.Output(), the error is returned and cache is not updated,
//     else cache associated with 'sig' is updated and new cached value is returned.
//
// dir is created if absent, lockDuration is the maximum duration to wait for lock step.
func Output(o Outputter, sig string, dir string, lockDuration time.Duration, lockP func(name string) Locker) (out []byte, err error) {
	if err = mkdirAllRetry(dir); err != nil {
		return nil, err
	}
	sig = normalize(sig)
	lock := lockP(sig)
	if err = lock.Lock(lockDuration, "cache"); err != nil {
		return o.Output()
	}
	defer func(lock Locker) {
		_ = lock.UnLock()
	}(lock)
	outfile := cacheFile(dir, sig)
	if fileExist(outfile) {
		return ioutil.ReadFile(outfile)
	}
	out, err = o.Output()
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(outfile, out, 0600)
	return out, err
}

func Clear(sig string, dir string, lockDuration time.Duration, lockP func(name string) Locker) error {
	sig = normalize(sig)
	lock := lockP(sig)
	if err := lock.Lock(lockDuration, "cache"); err != nil {
		return err
	}
	defer func(lock Locker) {
		_ = lock.UnLock()
	}(lock)
	outfile := cacheFile(dir, sig)
	if !fileExist(outfile) {
		return nil
	}
	return os.Remove(outfile)
}

func mkdirAllRetry(path string) (err error) {
	for i := 0; i < mkdirAllMaxRetries; i++ {
		if err = mkdirAll(path, 0700); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	return
}

func cacheFile(cacheDir, name string) string {
	return filepath.Join(cacheDir, name) + ".out"
}

func fileExist(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func normalize(name string) string {
	return strings.ReplaceAll(name, "/", "(slash)")
}
