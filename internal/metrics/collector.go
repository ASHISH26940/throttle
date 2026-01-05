package metrics

import (
	"sync/atomic"
	"time"
)

type TrafficPattern int

const (
	PatternUnknown TrafficPattern = iota
	PatternBursty      // High variance, sudden spikes
	PatternSteady      // Low variance, constant rate
	PatternSpikey      // Periodic spikes
	PatternIdle        // Very low traffic
)

func (p TrafficPattern) String() string{
	switch p {
	case PatternBursty:
		return "Bursty"
	case PatternSteady:
		return "Steady"
	case PatternSpikey:
		return "Spikey"
	case PatternIdle:
		return "Idle"
	default:
		return "Unknown"
	}
}

type TrafficMetrics struct{
	windowStart atomic.Int64
	windowSize time.Duration

	requests atomic.Int64
	allowed atomic.Int64
	denied atomic.Int64

	avgRPS atomic.Value
	peakRPS atomic.Value
	variance atomic.Value

	pattern atomic.Value
}

func NewTrafficMetrics(windowSize time.Duration) *TrafficMetrics {
	tm := &TrafficMetrics{
		windowSize: windowSize,
	}
	tm.windowStart.Store(time.Now().UnixNano())
	tm.avgRPS.Store(float64(0))
	tm.peakRPS.Store(float64(0))
	tm.variance.Store(float64(0))
	tm.pattern.Store(PatternUnknown)
	return tm
}

func (tm *TrafficMetrics) Record(allowed bool){
	tm.requests.Add(1)
	if allowed{
		tm.allowed.Add(1)
	}else{
		tm.denied.Add(1)
	}
}

type MetricsSnapshot struct {
	WindowStart  time.Time
	WindowSize   time.Duration
	Requests     int64
	Allowed      int64
	Denied       int64
	AvgRPS       float64
	PeakRPS      float64
	Variance     float64
	BurstRatio   float64 // peak/avg
	DenialRate   float64 // denied/total
	Pattern      TrafficPattern
}

func (tm *TrafficMetrics) Snapshot() MetricsSnapshot{
	requests:=tm.requests.Load()
	allowed:=tm.allowed.Load()
	denied:=tm.denied.Load()
	avgRPS:=tm.avgRPS.Load().(float64)
	peakRPS:=tm.peakRPS.Load().(float64)

	var burstRatio float64
	if avgRPS >0{
		burstRatio=peakRPS/avgRPS
	}

	var denialRate float64
	if requests>0{
		denialRate=float64(denied)/float64(requests)
	}

	return MetricsSnapshot{
		WindowStart: time.Unix(0, tm.windowStart.Load()),
		WindowSize:  tm.windowSize,
		Requests:    requests,
		Allowed:     allowed,
		Denied:      denied,
		AvgRPS:      avgRPS,
		PeakRPS:     peakRPS,
		Variance:    tm.variance.Load().(float64),
		BurstRatio:  burstRatio,
		DenialRate:  denialRate,
		Pattern:     tm.pattern.Load().(TrafficPattern),
	}
}

func (tm *TrafficMetrics) DetectPattern() TrafficPattern{
	snap:=tm.Snapshot()

	if snap.AvgRPS <1.0{
		tm.pattern.Store(PatternIdle)
		return PatternIdle
	}

	if snap.BurstRatio >3.0 && snap.Variance >0.5{
		tm.pattern.Store(PatternBursty)
		return  PatternBursty
	}

	if snap.Variance <0.2{
		tm.pattern.Store(PatternSteady)
		return PatternSteady
	}

	if snap.BurstRatio > 1.5 && snap.BurstRatio < 3.0 {
		tm.pattern.Store(PatternSpikey)
		return PatternSpikey
	}

	return PatternUnknown
}

func (tm *TrafficMetrics) Reset() {
	tm.windowStart.Store(time.Now().UnixNano())
	tm.requests.Store(0)
	tm.allowed.Store(0)
	tm.denied.Store(0)
	tm.avgRPS.Store(float64(0))
	tm.peakRPS.Store(float64(0))
	tm.variance.Store(float64(0))
	tm.pattern.Store(PatternUnknown)
}