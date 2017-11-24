package jwt

// Method represents the operation method
type Method uint8

// String implements Stringer.String
func (m Method) String() string {
	if str, ok := _MethodMap[m]; ok {
		return str
	}
	return ""
}

// Method enum
const (
	MethodWrite Method = 1 << iota
	MethodRead
	MethodDelete
	MethodAdmin
)

// contains all method strings
const _MethodStrings = "writereaddeleteadmin"

// slice mapping to the individual method strings
var _MethodMap = map[Method]string{
	MethodWrite:  _MethodStrings[0:5],
	MethodRead:   _MethodStrings[5:9],
	MethodDelete: _MethodStrings[9:15],
	MethodAdmin:  _MethodStrings[15:20],
}
