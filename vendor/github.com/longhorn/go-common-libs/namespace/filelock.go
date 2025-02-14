package namespace

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/longhorn/go-common-libs/types"
	"github.com/longhorn/go-common-libs/utils"
)

// LockFile switches to the process namespace and locks a file at the specified path.
// It returns the file handle.
func LockFile(procName, path string) (result *os.File, err error) {
	defer func() {
		err = errors.Wrapf(err, "failed to lock file %s", path)
	}()

	fn := func() (interface{}, error) {
		return utils.LockFile(path)
	}

	rawResult, err := RunFunc(fn, procName, types.HostProcDirectory, 0)
	if err != nil {
		return nil, err
	}

	var ableToCast bool
	result, ableToCast = rawResult.(*os.File)
	if !ableToCast {
		return nil, errors.Errorf(types.ErrNamespaceCastResultFmt, result, rawResult)
	}
	return result, nil
}

// FileLock is a struct responsible for locking a file.
type FileLock struct {
	FilePath string        // The path of the file to lock.
	File     *os.File      // The file handle aquired after successful lock.
	Timeout  time.Duration // The maximum time to wait for lock acquisition.

	done  chan struct{} // A channel for signaling lock release.
	mutex *sync.Mutex   // Mutex to prevent concurrent access to the file handle.
}

// NewLock creates a new FileLock instance.
func NewLock(filepath string, timeout time.Duration) *FileLock {
	if timeout == 0 {
		timeout = types.FileLockDefaultTimeout
	}

	return &FileLock{
		FilePath: filepath,
		Timeout:  timeout,
		done:     make(chan struct{}),
		mutex:    &sync.Mutex{},
	}
}

// Lock locks a file. It starts a goroutine to lock the file and returns the file
// handle. If the lock acquisition exceeds the specified timeout, the function
// unlocks the file and returns an error.
// It also starts another goroutine to wait for lock to release and unlock the file.
func (lock *FileLock) Lock() error {
	var err error
	defer func() {
		err = errors.Wrapf(err, "failed to lock file %s", lock.FilePath)
	}()

	log := logrus.WithField("file", lock.FilePath)

	// Use a buffered channel for error handling to prevent goroutine leak.
	errCh := make(chan error, 1)

	// Use a buffered channel for signaling successful lock acquisition.
	resultCh := make(chan struct{}, 1)

	// Use a context with timeout for handling the lock timeout.
	ctx, cancel := context.WithTimeout(context.Background(), lock.Timeout)
	defer cancel()

	go func() {
		lock.mutex.Lock()
		defer lock.mutex.Unlock()

		result, err := LockFile(types.ProcessNone, lock.FilePath)
		if err != nil {
			errCh <- err
			return
		}

		lock.File = result
		resultCh <- struct{}{}
	}()

	select {
	case <-resultCh:
		log.Trace("Locked file")
	case <-ctx.Done():
		log.Trace("Timeout waiting for file to lock")

		lock.mutex.Lock()
		defer lock.mutex.Unlock()

		if lock.File != nil {
			err := utils.UnlockFile(lock.File)
			if err != nil {
				return errors.Wrapf(err, "failed to unlock timed out lock file %v", lock.FilePath)
			}
			lock.File = nil
		}

		return fmt.Errorf("timed out waiting for file to lock %v", lock.FilePath)
	}

	// Wait for unlock
	go func() {
		<-lock.done
		log.Trace("Received done signal to unlock file")

		lock.mutex.Lock()
		defer lock.mutex.Unlock()

		if lock.File != nil {
			err := utils.UnlockFile(lock.File)
			if err != nil {
				logrus.WithError(err).Error("Failed to gracefully unlock file")
			}
			lock.File = nil

		}
	}()
	return nil
}

// Unlock closes the done channel to signal the lock to release.
func (lock *FileLock) Unlock() {
	close(lock.done)
}
