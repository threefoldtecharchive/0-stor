package stats

import (
	"time"

	"sync"

	"github.com/paulbellamy/ratecounter"
)

type stat struct {
	readReq  *ratecounter.RateCounter
	writeReq *ratecounter.RateCounter
}

func newStat() stat {
	return stat{
		readReq:  ratecounter.NewRateCounter(time.Hour),
		writeReq: ratecounter.NewRateCounter(time.Hour),
	}
}

// global (private) variables used to store state for this module
var (
	globalNamespaceStat     = make(map[string]stat)
	globalNamespaceStatLock sync.RWMutex
)

// AddNamespace create a rate counter this a namespace
func AddNamespace(label string) {
	_, ok := globalNamespaceStat[label]
	if ok {
		return
	}

	globalNamespaceStat[label] = newStat()
	return
}

// IncrRead increments the read request counter for a namespace
func IncrRead(label string) {
	globalNamespaceStatLock.Lock()
	defer globalNamespaceStatLock.Unlock()

	stat, ok := globalNamespaceStat[label]
	if !ok {
		AddNamespace(label)
		stat = globalNamespaceStat[label]
	}

	stat.readReq.Incr(1)
}

// IncrWrite increments the write request counter for a namespace
func IncrWrite(label string) {
	globalNamespaceStatLock.Lock()
	defer globalNamespaceStatLock.Unlock()

	stat, ok := globalNamespaceStat[label]
	if !ok {
		AddNamespace(label)
		stat = globalNamespaceStat[label]
	}

	stat.writeReq.Incr(1)
}

// Rate return the numer of request per hour for a namespace
func Rate(label string) (read, write int64) {
	globalNamespaceStatLock.RLock()
	defer globalNamespaceStatLock.RUnlock()

	stat, ok := globalNamespaceStat[label]
	if !ok {
		AddNamespace(label)
		stat = globalNamespaceStat[label]
	}

	return stat.readReq.Rate(), stat.writeReq.Rate()
}
