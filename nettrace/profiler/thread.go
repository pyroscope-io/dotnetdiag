package profiler

type thread struct {
	lastBlockTime int64
	lastCPUTime   int64
	callTree
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

func (t *thread) addSample(sampleType clrThreadSampleType, relativeTime int64, stack []uint64) {
	switch sampleType {
	case sampleTypeError:
		return

	case sampleTypeManaged:
		switch t.state() {
		case uninitialized:
			t.putCPUSample(stack, relativeTime)
			t.lastBlockTime = -1
		case running:
			t.putCPUSample(stack, relativeTime)
		case blocked:
			t.putBlockSample(stack, relativeTime)
			t.lastBlockTime = -relativeTime
		}
		t.lastCPUTime = relativeTime

	case sampleTypeExternal:
		switch t.state() {
		case blocked, uninitialized:
			t.putBlockSample(stack, relativeTime)
		case running:
			t.putCPUSample(stack, relativeTime)
		}
		t.lastBlockTime = relativeTime
	}
}

func (t *thread) putCPUSample(stack []uint64, rt int64) {
	if t.lastCPUTime > 0 {
		t.put(stack, t.lastCPUTime, rt)
	}
}

func (t *thread) putBlockSample(stack []uint64, rt int64) {
	if t.lastBlockTime > 0 {
		t.put(stack, t.lastBlockTime, rt)
	}
}

type callTree []*frame

type frame struct {
	addr        uint64
	sampledTime int64
	callTree
}

func (t *callTree) put(stack []uint64, baseTime, relativeTime int64) {
	if len(stack) == 0 {
		return
	}
	i := len(stack) - 1
	x := stack[i]
	for _, f := range *t {
		if f.addr != x {
			continue
		}
		f.sampledTime += relativeTime - baseTime
		if len(stack) > 1 {
			f.put(stack[:i], baseTime, relativeTime)
		}
		return
	}
	f := &frame{addr: x, sampledTime: relativeTime - baseTime}
	if len(stack) > 1 {
		f.put(stack[:i], baseTime, relativeTime)
	}
	*t = append(*t, f)
}

func (t *callTree) walk(fn func(int, *frame)) {
	for i, f := range *t {
		if i > 0 {
			fn(i-1, f)
		} else {
			fn(i, f)
		}
		f.walk(func(i int, n *frame) {
			fn(i+1, n)
		})
	}
}
