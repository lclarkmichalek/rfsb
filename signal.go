package rfsb

// Signal is a wrapper for a string. Define your own if you feel the need
type Signal byte

const (
	// Finished is emitted by ResourceGraph when a Resource finishes being evaluated. Regardless of what the evaluation
	// of the resource entailed, the Finished signal is involved. This includes:
	// 1. The resource's dependencies finished, but did not emit the signals the resource depended on.
	// 2. The resource's dependencies were met, and the resource was skipped
	// 3. The resource's dependencies were met, and the resource was materialized
	Finished Signal = iota
	// Unevaluated is emitted by ResourceGraph when a Resource's dependencies emit the Finished signal, and the resource
	// was dependent on a different signal.
	Unevaluated
	// Evaluated is emitted by ResourceGraph when a resource is Skipped or Materialized
	Evaluated
	// Skipped is emitted by ResourceGraph when a Resource's ShouldSkip function is called, and returns true, causing
	// the Materialization to be skipped. It is not called when a resource is not run due to missing dependencies (see
	// Evaluated)
	Skipped
	// Materialized is emitted by ResourceGraph when a Resource's Materialize function is called.
	Materialized
)

func (s Signal) String() string {
	switch s {
	case Finished:
		return "Finished"
	case Unevaluated:
		return "Unevaluated"
	case Evaluated:
		return "Evaluated"
	case Skipped:
		return "Skipped"
	case Materialized:
		return "Materialized"
	default:
		return "UNKNOWN_SIGNAL"
	}
}
