//go:build aix || (solaris && !illumos)

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"fmt"
	"os"
	"sync"
	"syscall"
)

type fcntlProcessLock struct {
	key   string
	state *fcntlProcessLockState
}

type fcntlProcessLockState struct {
	mu   sync.Mutex
	refs int
}

var fcntlProcessLocks = struct {
	sync.Mutex
	locks map[string]*fcntlProcessLockState
	owned map[*os.File]*fcntlProcessLock
}{
	locks: make(map[string]*fcntlProcessLockState),
	owned: make(map[*os.File]*fcntlProcessLock),
}

func fcntlInfoKey(info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("file has unsupported identity metadata")
	}
	return fmt.Sprintf("%T:%d:%d", stat, stat.Dev, stat.Ino), nil
}

func acquireFcntlProcessLock(file *os.File) (*fcntlProcessLock, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	key, err := fcntlInfoKey(info)
	if err != nil {
		return nil, err
	}

	fcntlProcessLocks.Lock()
	state := fcntlProcessLocks.locks[key]
	if state == nil {
		state = &fcntlProcessLockState{}
		fcntlProcessLocks.locks[key] = state
	}
	state.refs++
	fcntlProcessLocks.Unlock()

	state.mu.Lock()
	return &fcntlProcessLock{key: key, state: state}, nil
}

func releaseFcntlProcessLock(file *os.File, held *fcntlProcessLock) {
	held.state.mu.Unlock()

	fcntlProcessLocks.Lock()
	delete(fcntlProcessLocks.owned, file)
	held.state.refs--
	if held.state.refs == 0 {
		delete(fcntlProcessLocks.locks, held.key)
	}
	fcntlProcessLocks.Unlock()
}

func lockFile(file *os.File) error {
	held, err := acquireFcntlProcessLock(file)
	if err != nil {
		return err
	}

	if err := syscall.FcntlFlock(file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Type: syscall.F_WRLCK,
	}); err != nil {
		held.state.mu.Unlock()
		fcntlProcessLocks.Lock()
		held.state.refs--
		if held.state.refs == 0 {
			delete(fcntlProcessLocks.locks, held.key)
		}
		fcntlProcessLocks.Unlock()
		return err
	}

	fcntlProcessLocks.Lock()
	fcntlProcessLocks.owned[file] = held
	fcntlProcessLocks.Unlock()
	return nil
}

func unlockFile(file *os.File) error {
	err := syscall.FcntlFlock(file.Fd(), syscall.F_SETLK, &syscall.Flock_t{
		Type: syscall.F_UNLCK,
	})

	fcntlProcessLocks.Lock()
	held := fcntlProcessLocks.owned[file]
	fcntlProcessLocks.Unlock()
	if held != nil {
		releaseFcntlProcessLock(file, held)
	}
	return err
}
