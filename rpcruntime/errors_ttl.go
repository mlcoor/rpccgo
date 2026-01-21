package rpcruntime

import (
	"sync"
	"time"
)

var startCleanerOnce sync.Once

var cleanupInterval = 1 * time.Second

func startCleaner() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for now := range ticker.C {
			_ = cleanupExpired(now)
		}
	}()
}
