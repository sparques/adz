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
