package fcache

import (
	"errors"
	"github.com/opensvc/testhelper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

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
