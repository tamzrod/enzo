// internal/state/dict.go
package state

// Template is a minimal v1 template: [CONST][VAR][CONST]
type Template struct {
	ID     uint16
	ConstA []byte
	ConstB []byte
}

// Dict is per-connection agreement state.
type Dict struct {
	Templates map[uint16]*Template
}

func NewDict() *Dict {
	return &Dict{
		Templates: make(map[uint16]*Template),
	}
}
