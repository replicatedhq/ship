package version

import "time"

var (
	build Build
)

type Build struct {
	Version      string
	GitSHA       string
	BuildTime    time.Time
	TimeFallback string `json:"time_fallback,omitempty"`
}

func Init() {
	build.Version = version
	if len(gitSHA) >= 7 {
		build.GitSHA = gitSHA[:7]
	}
	var err error
	build.BuildTime, err = time.Parse(time.RFC3339, buildTime)
	if err != nil {
		build.TimeFallback = buildTime
	}
}

func GetBuild() Build {
	return build
}

func Version() string {
	return build.Version
}

func GitSHA() string {
	return build.GitSHA
}

func BuildTime() time.Time {
	return build.BuildTime
}
