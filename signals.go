package adz

// Signals are akin to unix signals (but not the same thing!) Interpreters will keep running until they run out of commands
// or if they get a signal. Signals are implemented via go channels.
type Signal int

const (
	SignalRun Signal = iota
	SignalBreak
	SignalStop
	SignalAbort
	SignalKill
)

var signalMap = map[Signal]string{
	SignalRun:   "run",
	SignalBreak: "break",
	SignalStop:  "stop",
	SignalAbort: "abort",
	SignalKill:  "kill",
}

func (sig Signal) String() string {
	return signalMap[sig]
}

func (sig Signal) Signal() {
}
