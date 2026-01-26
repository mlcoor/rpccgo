package rpcruntime

import (
	"sync"
	"sync/atomic"
	"time"
)

type errorRecord struct {
	msg       []byte
	expiresAt time.Time
}

var (
	registryMu sync.Mutex
	registry   = make(map[uint64]errorRecord)

	nextErrorID atomic.Uint64
)

var errorTTL = 3 * time.Second

// StoreError stores an error message in the global registry and returns its id.
//
// A returned id of 0 indicates "no error" (i.e. err is nil).
func StoreError(err error) uint64 {
	if err == nil {
		return 0
	} else {
		return StoreErrorMsg([]byte(err.Error()))
	}
}

// StoreErrorMsg stores msg in the global registry and returns its id.
//
// The stored bytes are copied.
func StoreErrorMsg(msg []byte) uint64 {
	startCleanerOnce.Do(startCleaner)

	id := nextErrorID.Add(1)
	copied := make([]byte, len(msg))
	copy(copied, msg)

	record := errorRecord{
		msg:       copied,
		expiresAt: time.Now().Add(errorTTL),
	}

	registryMu.Lock()
	registry[id] = record
	registryMu.Unlock()

	return id
}

// GetErrorMsgBytes returns a copy of the stored message bytes.
//
// If the record is expired, it is removed and ok is false.
func GetErrorMsgBytes(errorID uint64) (msg []byte, ok bool) {
	if errorID == 0 {
		return nil, false
	} else {
		now := time.Now()

		registryMu.Lock()
		defer registryMu.Unlock()
		record, exists := registry[errorID]
		if !exists {
			return nil, false
		}

		if now.After(record.expiresAt) {
			delete(registry, errorID)
			return nil, false
		} else {
			copied := make([]byte, len(record.msg))
			copy(copied, record.msg)
			return copied, true
		}
	}
}

// cleanupExpired removes expired entries from the registry.
//
// It returns the number of removed records.
func cleanupExpired(now time.Time) int {
	registryMu.Lock()
	defer registryMu.Unlock()

	removed := 0
	for id, record := range registry {
		if now.After(record.expiresAt) {
			delete(registry, id)
			removed++
		}
	}

	return removed
}
