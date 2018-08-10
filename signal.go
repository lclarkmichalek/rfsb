package rfsb

// Signal is a wrapper for a string. Define your own if you feel the need
type Signal string

const (
	// SignalSkipped is a signal emitted by the ResourceGraph materializer when a resource is skipped
	SignalSkipped Signal = "skipped"
	// SignalMaterialized is a signal emitted by the ResourceGraph materializer when a resource is materialized
	SignalMaterialized Signal = "materialized"
	// SignalFinished is a signal emitted by the ResourceGraph materializer when a resource has finished being
	// materialized (or skipped, or is otherwise no longer relevant)
	SignalFinished Signal = "finished"
)
