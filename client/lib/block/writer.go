package block

// Writer defines Writer that work on block level
type Writer interface {
	// WriteBlock write the value to the underlying writer
	WriteBlock(key, value []byte) (written int, err error)
}
