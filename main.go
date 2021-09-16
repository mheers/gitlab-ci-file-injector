package main

import (
	"runtime"

	"github.com/mheers/gitlab-ci-file-injector/cmd"
	"github.com/sirupsen/logrus"
)

// build flags
var (
	VERSION    string
	BuildTime  string
	CommitHash string
	GoVersion  string
	GitTag     string
	GitBranch  string
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cmd.VERSION = VERSION
	cmd.BuildTime = BuildTime
	cmd.CommitHash = CommitHash
	cmd.GoVersion = GoVersion
	cmd.GitTag = GitTag
	cmd.GitBranch = GitBranch

	// execeute the command
	err := cmd.Execute()
	if err != nil {
		logrus.Fatalf("Execute failed: %+v", err)
	}
}
