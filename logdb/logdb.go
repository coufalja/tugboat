package logdb

// LogDBInfo is the info provided when LogDBCallback is invoked.
type LogDBInfo struct {
	Shard uint64
	Busy  bool
}

// LogDBCallback is called by the LogDB layer whenever NodeHost is required to
// be notified for the status change of the LogDB.
type LogDBCallback func(LogDBInfo)
