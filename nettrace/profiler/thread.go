package profiler

type thread struct {
	lastBlockTime int64
	lastCPUTime   int64
	// StackID -> sampled time
	samples map[int32]int64
}

type threadState int

const (
	_ threadState = iota - 1

	uninitialized
	running
	blocked
)

func (t *thread) state() threadState {
	switch {
	case t.lastBlockTime < 0:
		return running
	case t.lastBlockTime > 0:
		return blocked
	default:
		return uninitialized
	}
}

func (t *thread) addSample(sampleType clrThreadSampleType, relativeTime int64, stackID int32) {
	switch sampleType {
	case sampleTypeError:
		return

	case sampleTypeManaged:
		switch t.state() {
		case uninitialized:
			t.putCPUSample(stackID, relativeTime)
			t.lastBlockTime = -1
		case running:
			t.putCPUSample(stackID, relativeTime)
		case blocked:
			t.putBlockSample(stackID, relativeTime)
			t.lastBlockTime = -relativeTime
		}
		t.lastCPUTime = relativeTime

	case sampleTypeExternal:
		switch t.state() {
		case blocked, uninitialized:
			t.putBlockSample(stackID, relativeTime)
		case running:
			t.putCPUSample(stackID, relativeTime)
		}
		t.lastBlockTime = relativeTime
	}
}

func (t *thread) putCPUSample(stackID int32, rt int64) {
	if t.lastCPUTime > 0 {
		t.samples[stackID] += t.lastCPUTime - rt
	}
}

func (t *thread) putBlockSample(stackID int32, rt int64) {
	if t.lastBlockTime > 0 {
		t.samples[stackID] += t.lastBlockTime - rt
	}
}
