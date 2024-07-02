package sqlwrapper

import "errors"

// errors and error format texts

var (
	ErrNilPointer    = errors.New("destination pointer is nil")
	ErrNotPointer    = errors.New("target is not a pointer")
	ErrElemNotStruct = errors.New("elem of target is not a struct")
	ErrElemNotSlice  = errors.New("elem of target is not a slice")
	ErrInvalidPKType = errors.New("invalid primary key type (should be one of int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64 and string)")
	ErrNotEnoughArgs = errors.New("not enough arguments")
	ErrTooManyArgs   = errors.New("too many arguments")

	ErrInvalidJoinCondType      = errors.New(`invalid join condition type (should be either "on" or "using")`)
	ErrDialectAlreadyRegistered = errors.New("dialect has already been registered")
)

const (
	f1 = "unsupported conversion from type %T into type %s"
	f2 = "converting value type %T (%v) to type %s: %s"
	f3 = "unsupported source type: %T"
	f4 = "primary key field '%s' should not be empty or zero value"
	f5 = "cannot find any fields related to column '%s'"

	fx1 = "fail to create transaction: %s"
)
