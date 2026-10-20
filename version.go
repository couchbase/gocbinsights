package cbinsights

const (
	versionStr = "v1.0.1"
)

// Version returns a string representation of the current SDK version.
func Version() string {
	return versionStr
}

// Identifier returns a string representation of the current SDK identifier.
func Identifier() string {
	return "gocb-insights/" + versionStr
}
