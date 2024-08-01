package version

import (
	"fmt"
	"runtime/debug"
)

var (
	Tag      string
	Revision string
	BuildAt  string
	Dirty    bool
)

func init() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	for _, setting := range buildInfo.Settings {
		// https://pkg.go.dev/runtime/debug#BuildSetting
		switch setting.Key {
		case "vcs.revision":
			Revision = setting.Value
		case "vcs.time":
			BuildAt = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				Dirty = true
			}
		}
	}
}

func String() string {
	if Revision == "" {
		return "dev"
	}

	s := fmt.Sprintf("%s %s at %s", Tag, Revision, BuildAt)
	if Dirty {
		s += " dirty"
	}
	return s
}
