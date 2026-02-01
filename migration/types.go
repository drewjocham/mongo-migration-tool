package migration

type Direction int

const (
	DirectionUp Direction = iota
	DirectionDown
)

func (d Direction) String() string {
	switch d {
	case DirectionUp:
		return "up"
	case DirectionDown:
		return "down"
	default:
		return "unknown"
	}
}

type ErrNotSupported struct {
	Operation string
}

func (e ErrNotSupported) Error() string {
	return "operation not supported: " + e.Operation
}
