package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	gitVersion = "v0.0.0-master+$Format:%H$" // taged version $(git describe --tags --dirty)
	gitCommit  = "$Format:%H$"               // sha1 from git, output of $(git rev-parse HEAD)
	buildDate  = "1970-01-01T00:00:00Z"      // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

type Version struct {
	GitVersion string
	GitCommit  string
	BuildDate  string
	GoVersion  string
	Compiler   string
	Platform   string
}

func Get() Version {
	return Version{
		GitVersion: gitVersion,
		GitCommit:  gitCommit,
		BuildDate:  buildDate,
		GoVersion:  runtime.Version(),
		Compiler:   runtime.Compiler,
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func (v Version) String() string {
	bts, _ := json.MarshalIndent(v, "", "  ")
	return string(bts)
}
