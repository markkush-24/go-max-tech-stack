package runtimeinfo

import (
	"encoding/json"
	"net/http"
	"runtime"
	runtimemetrics "runtime/metrics"
	"time"
)

var selectedMetricNames = []string{
	"/cpu/classes/gc/pause:cpu-seconds",
	"/cpu/classes/gc/total:cpu-seconds",
	"/cpu/classes/user:cpu-seconds",
	"/gc/cycles/automatic:gc-cycles",
	"/gc/cycles/forced:gc-cycles",
	"/gc/cycles/total:gc-cycles",
	"/gc/gogc:percent",
	"/gc/gomemlimit:bytes",
	"/gc/heap/allocs:bytes",
	"/gc/heap/frees:bytes",
	"/gc/heap/goal:bytes",
	"/gc/heap/live:bytes",
	"/gc/heap/objects:objects",
	"/gc/limiter/last-enabled:gc-cycle",
	"/memory/classes/heap/objects:bytes",
	"/memory/classes/heap/released:bytes",
	"/memory/classes/metadata/other:bytes",
	"/memory/classes/profiling/buckets:bytes",
	"/memory/classes/total:bytes",
	"/sched/gomaxprocs:threads",
	"/sched/goroutines:goroutines",
}

type Snapshot struct {
	Timestamp string                  `json:"timestamp"`
	Go        GoSnapshot              `json:"go"`
	MemStats  MemStatsSnapshot        `json:"memstats"`
	Metrics   map[string]MetricSample `json:"metrics"`
}

type GoSnapshot struct {
	Version      string `json:"version"`
	NumCPU       int    `json:"num_cpu"`
	GOMAXPROCS   int    `json:"gomaxprocs"`
	NumGoroutine int    `json:"num_goroutine"`
}

type MemStatsSnapshot struct {
	AllocBytes             uint64  `json:"alloc_bytes"`
	TotalAllocBytes        uint64  `json:"total_alloc_bytes"`
	SysBytes               uint64  `json:"sys_bytes"`
	RuntimeMemoryUsedBytes uint64  `json:"runtime_memory_used_bytes"`
	Mallocs                uint64  `json:"mallocs"`
	Frees                  uint64  `json:"frees"`
	HeapAllocBytes         uint64  `json:"heap_alloc_bytes"`
	HeapSysBytes           uint64  `json:"heap_sys_bytes"`
	HeapIdleBytes          uint64  `json:"heap_idle_bytes"`
	HeapInuseBytes         uint64  `json:"heap_inuse_bytes"`
	HeapReleasedBytes      uint64  `json:"heap_released_bytes"`
	HeapObjects            uint64  `json:"heap_objects"`
	StackInuseBytes        uint64  `json:"stack_inuse_bytes"`
	MSpanInuseBytes        uint64  `json:"mspan_inuse_bytes"`
	MCacheInuseBytes       uint64  `json:"mcache_inuse_bytes"`
	GCSysBytes             uint64  `json:"gc_sys_bytes"`
	OtherSysBytes          uint64  `json:"other_sys_bytes"`
	NextGCBytes            uint64  `json:"next_gc_bytes"`
	LastGCUnixNano         uint64  `json:"last_gc_unix_nano"`
	PauseTotalNs           uint64  `json:"pause_total_ns"`
	NumGC                  uint32  `json:"num_gc"`
	NumForcedGC            uint32  `json:"num_forced_gc"`
	GCCPUFraction          float64 `json:"gc_cpu_fraction"`
}

type MetricSample struct {
	Kind    string            `json:"kind"`
	Uint64  *uint64           `json:"uint64,omitempty"`
	Float64 *float64          `json:"float64,omitempty"`
	Hist    *HistogramSummary `json:"histogram,omitempty"`
}

type HistogramSummary struct {
	Count uint64  `json:"count"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ReadSnapshot(time.Now()))
}

func ReadSnapshot(now time.Time) Snapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return Snapshot{
		Timestamp: now.UTC().Format(time.RFC3339Nano),
		Go: GoSnapshot{
			Version:      runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			GOMAXPROCS:   runtime.GOMAXPROCS(0),
			NumGoroutine: runtime.NumGoroutine(),
		},
		MemStats: snapshotMemStats(ms),
		Metrics:  readRuntimeMetrics(),
	}
}

func snapshotMemStats(ms runtime.MemStats) MemStatsSnapshot {
	runtimeMemoryUsed := ms.Sys
	if ms.HeapReleased <= ms.Sys {
		runtimeMemoryUsed = ms.Sys - ms.HeapReleased
	}

	return MemStatsSnapshot{
		AllocBytes:             ms.Alloc,
		TotalAllocBytes:        ms.TotalAlloc,
		SysBytes:               ms.Sys,
		RuntimeMemoryUsedBytes: runtimeMemoryUsed,
		Mallocs:                ms.Mallocs,
		Frees:                  ms.Frees,
		HeapAllocBytes:         ms.HeapAlloc,
		HeapSysBytes:           ms.HeapSys,
		HeapIdleBytes:          ms.HeapIdle,
		HeapInuseBytes:         ms.HeapInuse,
		HeapReleasedBytes:      ms.HeapReleased,
		HeapObjects:            ms.HeapObjects,
		StackInuseBytes:        ms.StackInuse,
		MSpanInuseBytes:        ms.MSpanInuse,
		MCacheInuseBytes:       ms.MCacheInuse,
		GCSysBytes:             ms.GCSys,
		OtherSysBytes:          ms.OtherSys,
		NextGCBytes:            ms.NextGC,
		LastGCUnixNano:         ms.LastGC,
		PauseTotalNs:           ms.PauseTotalNs,
		NumGC:                  ms.NumGC,
		NumForcedGC:            ms.NumForcedGC,
		GCCPUFraction:          ms.GCCPUFraction,
	}
}

func readRuntimeMetrics() map[string]MetricSample {
	supported := supportedRuntimeMetrics()

	samples := make([]runtimemetrics.Sample, 0, len(selectedMetricNames))
	for _, name := range selectedMetricNames {
		if _, ok := supported[name]; ok {
			samples = append(samples, runtimemetrics.Sample{Name: name})
		}
	}

	runtimemetrics.Read(samples)

	out := make(map[string]MetricSample, len(samples))
	for _, sample := range samples {
		if value, ok := metricValue(sample.Value); ok {
			out[sample.Name] = value
		}
	}
	return out
}

func supportedRuntimeMetrics() map[string]struct{} {
	descriptions := runtimemetrics.All()
	out := make(map[string]struct{}, len(descriptions))
	for _, d := range descriptions {
		out[d.Name] = struct{}{}
	}
	return out
}

func metricValue(v runtimemetrics.Value) (MetricSample, bool) {
	switch v.Kind() {
	case runtimemetrics.KindUint64:
		n := v.Uint64()
		return MetricSample{Kind: "uint64", Uint64: &n}, true
	case runtimemetrics.KindFloat64:
		n := v.Float64()
		return MetricSample{Kind: "float64", Float64: &n}, true
	case runtimemetrics.KindFloat64Histogram:
		return MetricSample{Kind: "float64_histogram", Hist: summarizeHistogram(v.Float64Histogram())}, true
	default:
		return MetricSample{}, false
	}
}

func summarizeHistogram(h *runtimemetrics.Float64Histogram) *HistogramSummary {
	var count uint64
	min := 0.0
	max := 0.0
	found := false

	for i, bucketCount := range h.Counts {
		if bucketCount == 0 {
			continue
		}
		count += bucketCount
		if !found {
			min = h.Buckets[i]
			found = true
		}
		max = h.Buckets[i+1]
	}

	return &HistogramSummary{
		Count: count,
		Min:   min,
		Max:   max,
	}
}
