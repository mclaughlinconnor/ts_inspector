package utils

type LocationError interface {
	error
	GetEndColumn() uint
	GetEndLine() uint
	GetFilename() string
	GetMessage() string
	GetRange() Range
	GetStartColumn() uint
	GetStartLine() uint
}
