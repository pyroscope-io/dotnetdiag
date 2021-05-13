package sampler

type thread struct {
	threadState
	lastBlockTime int64
	lastCPUTime   int64
	*callStack
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
		t.putSample(stack, t.lastCPUTime, rt)
	}
}

func (t *thread) putBlockSample(stack []uint64, rt int64) {
	if t.lastBlockTime > 0 {
		t.putSample(stack, t.lastBlockTime, rt)
	}
}

type callStack struct {
	frames []*frame
}

type frame struct {
	addr        uint64
	sampledTime int64
	callStack
}

func (t *callStack) putSample(stack []uint64, baseTime, relativeTime int64) {
	if len(stack) == 0 {
		return
	}
	i := len(stack) - 1
	x := stack[i]
	for _, f := range t.frames {
		if f.addr == x {
			f.sampledTime += relativeTime - baseTime
			if len(stack) > 1 {
				f.callStack.putSample(stack[:i], baseTime, relativeTime)
			}
			return
		}
	}
	n := &frame{addr: x, sampledTime: relativeTime - baseTime}
	if len(stack) > 1 {
		n.callStack.putSample(stack[:i], baseTime, relativeTime)
	}
	t.frames = append(t.frames, n)
}

func (t *callStack) walk(f func(int, *frame)) {
	for i, n := range t.frames {
		if i > 0 {
			f(i-1, n)
		} else {
			f(i, n)
		}
		n.callStack.walk(func(i int, n *frame) {
			f(i+1, n)
		})
	}
}
