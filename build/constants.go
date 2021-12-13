package build

// DO NOT EDIT THIS FILE DIRECTLY. These are build-time constants
// set through ‘make’.
var (
	GoVersion = ""

	BuildDate = ""

	// Go get development tag.
	goGetTag = "DEVELOPMENT.GOGET"

	// Version - version time.RFC3339.
	Version = ""
	// ReleaseTag - release tag in TAG.%Y-%m-%dT%H-%M-%SZ.
	ReleaseTag = ""
	// CommitID - latest commit id.
	CommitID = ""
	// ShortCommitID - first 12 characters from CommitID.
	ShortCommitID = ""
)