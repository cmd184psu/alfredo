// Copyright 2025 Chris Delezenski <chris.delezenski@gmail.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alfredo

import (
	"context"
	"log"
	"runtime"
	"time"
)

// MemoryStats contains memory statistics
type MemoryStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NumGC      uint32
	HeapAlloc  uint64
	HeapSys    uint64
	HeapIdle   uint64
	HeapInUse  uint64
}

// Options for memory manager
type MemoryManagementOptions struct {
	// Interval between memory checks
	Interval time.Duration
	// ForceGC determines if garbage collection should be forced
	ForceGC bool
	// MemoryThreshold in bytes to trigger alerts (0 for disabled)
	MemoryThreshold uint64
	// Custom logger (optional)
	Logger *log.Logger
}

// DefaultOptions returns sensible default options
func DefaultMMOptions() *MemoryManagementOptions {
	return &MemoryManagementOptions{
		Interval:        30 * time.Second,
		ForceGC:         false,
		MemoryThreshold: 1024 * 1024 * 1024, // 1GB
		Logger:          log.Default(),
	}
}

func (options *MemoryManagementOptions) WithInterval(interval time.Duration) *MemoryManagementOptions {
	options.Interval = interval
	return options
}
func (options *MemoryManagementOptions) WithForceGC(forceGC bool) *MemoryManagementOptions {
	options.ForceGC = forceGC
	return options
}
func (options *MemoryManagementOptions) WithMemoryThreshold(threshold uint64) *MemoryManagementOptions {
	options.MemoryThreshold = threshold
	return options
}
func (options *MemoryManagementOptions) WithLogger(logger *log.Logger) *MemoryManagementOptions {
	options.Logger = logger
	return options
}
func (options *MemoryManagementOptions) WithDefaultLogger() *MemoryManagementOptions {
	options.Logger = log.Default()
	return options
}

// MemoryManager manages memory and outputs to logs
type MemoryManager struct {
	options MemoryManagementOptions
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a new memory manager with the given options
func NewMemoryManager(options MemoryManagementOptions) *MemoryManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &MemoryManager{
		options: options,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins memory monitoring in a goroutine
func (mm *MemoryManager) Start() {
	go mm.monitor()
}

// Stop cancels the memory monitoring goroutine
func (mm *MemoryManager) Stop() {
	mm.cancel()
}

// monitor continuously checks memory usage until canceled
func (mm *MemoryManager) monitor() {
	ticker := time.NewTicker(mm.options.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			mm.options.Logger.Println("Memory manager stopped")
			return
		case <-ticker.C:
			stats := mm.collectStats()
			mm.logStats(stats)

			// Optionally force GC
			if mm.options.ForceGC {
				runtime.GC()
				mm.options.Logger.Println("Forced garbage collection")
			}

			// Check if memory usage exceeds threshold
			if mm.options.MemoryThreshold > 0 && stats.HeapAlloc > mm.options.MemoryThreshold {
				mm.options.Logger.Printf("WARNING: Memory usage exceeds threshold: %d MB\n", stats.HeapAlloc/1024/1024)
			}
		}
	}
}

// collectStats gathers memory statistics
func (mm *MemoryManager) collectStats() MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MemoryStats{
		Alloc:      memStats.Alloc,
		TotalAlloc: memStats.TotalAlloc,
		Sys:        memStats.Sys,
		NumGC:      memStats.NumGC,
		HeapAlloc:  memStats.HeapAlloc,
		HeapSys:    memStats.HeapSys,
		HeapIdle:   memStats.HeapIdle,
		HeapInUse:  memStats.HeapInuse,
	}
}

// logStats logs the current memory statistics
func (mm *MemoryManager) logStats(stats MemoryStats) {
	mm.options.Logger.Printf("Memory stats: Alloc=%d MB, TotalAlloc=%d MB, Sys=%d MB, NumGC=%d\n",
		stats.Alloc/1024/1024,
		stats.TotalAlloc/1024/1024,
		stats.Sys/1024/1024,
		stats.NumGC)
	mm.options.Logger.Printf("Heap stats: HeapAlloc=%d MB, HeapSys=%d MB, HeapIdle=%d MB, HeapInUse=%d MB\n",
		stats.HeapAlloc/1024/1024,
		stats.HeapSys/1024/1024,
		stats.HeapIdle/1024/1024,
		stats.HeapInUse/1024/1024)
}
