package version

const (
	EnvDev     = "dev"
	EnvDebug   = "debug"
	EnvRelease = "release"

	defaultVersion   = "0.0.0"
	defaultAuthor    = "dan"
	defaultBuildDate = "2025-01-25"
)

var (
	ENV        = EnvDev
	VERSION    = defaultVersion
	AUTHOR     = defaultAuthor
	BUILD_INFO = ""
	BUILD_DATE = ""
)
