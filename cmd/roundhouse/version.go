package main

import (
	"regexp"
	"runtime/debug"
	"strings"
)

var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// buildVersion is the version this binary reports. The release build sets version; a binary from
// `go install ...@vX.Y.Z` knows its tag from the module information instead. Anything else (a local
// build, a pseudo-version of an untagged commit) stays "dev".
func buildVersion() string {
	if version != devVersion {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return moduleVersion(info.Main.Version)
	}
	return devVersion
}

// moduleVersion turns the module version of a build into a roundhouse version, without the leading v
// that release builds leave out; only a plain release tag counts.
func moduleVersion(v string) string {
	if !releaseTag.MatchString(v) {
		return devVersion
	}
	return strings.TrimPrefix(v, "v")
}
