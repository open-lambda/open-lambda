package common

type RuntimeType int

const (
	RT_PYTHON RuntimeType = iota
	RT_NATIVE             = iota
)

// LambdaFileExtension is the file extension used for lambda packages.
// TODO: This should be configurable in the future to support different archive formats.
const LambdaFileExtension = ".tar.gz"
