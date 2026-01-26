package rpcruntime

import (
	"sync"
	"testing"
	"time"
)

func resetForTest() {
	registryMu.Lock()
	registry = make(map[uint64]errorRecord)
	registryMu.Unlock()
	nextErrorID.Store(0)
	startCleanerOnce = sync.Once{}
}

func TestStoreAndLookup(t *testing.T) {
	resetForTest()

	id := StoreErrorMsg([]byte("hello"))
	if id != 0 {
		msg, ok := GetErrorMsgBytes(id)
		if ok {
			if string(msg) == "hello" {
				// ok
			} else {
				t.Fatalf("unexpected msg: %q", string(msg))
			}
		} else {
			t.Fatalf("expected ok")
		}
	} else {
		t.Fatalf("expected non-zero id")
	}
}

func TestExpiresAfterTTL(t *testing.T) {
	resetForTest()

	oldTTL := errorTTL
	oldInterval := cleanupInterval
	errorTTL = 30 * time.Millisecond
	cleanupInterval = 10 * time.Millisecond
	t.Cleanup(func() {
		errorTTL = oldTTL
		cleanupInterval = oldInterval
	})

	id := StoreErrorMsg([]byte("bye"))
	time.Sleep(3 * errorTTL)

	cleanupExpired(time.Now())

	_, ok := GetErrorMsgBytes(id)
	if ok {
		t.Fatalf("expected expired")
	}
}

func TestConcurrencySafety(t *testing.T) {
	resetForTest()

	oldTTL := errorTTL
	errorTTL = 200 * time.Millisecond
	t.Cleanup(func() {
		errorTTL = oldTTL
	})

	var wg sync.WaitGroup
	storeCount := 200
	wg.Add(storeCount)

	ids := make([]uint64, storeCount)
	for i := 0; i < storeCount; i++ {
		i := i
		go func() {
			defer wg.Done()
			ids[i] = StoreErrorMsg([]byte("msg"))
		}()
	}
	wg.Wait()

	wg.Add(storeCount)
	for i := 0; i < storeCount; i++ {
		id := ids[i]
		go func() {
			defer wg.Done()
			_, _ = GetErrorMsgBytes(id)
		}()
	}
	wg.Wait()
}
