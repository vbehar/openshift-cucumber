// Provides an executable for running cucumber tests on an OpenShift instance
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vbehar/openshift-cucumber/steps"

	"github.com/lsegal/gucumber"
	"github.com/spf13/pflag"
)

var (
	gitCommit   string
	buildNumber string
)

func main() {
	flags := pflag.NewFlagSet("openshift-cucumber", pflag.ExitOnError)
	printVersion := flags.BoolP("version", "v", false, "print version")
	flags.Parse(os.Args[1:])

	if *printVersion {
		if len(gitCommit) == 0 {
			fmt.Printf("openshift-cucumber - DEV\n")
		} else if len(buildNumber) == 0 {
			fmt.Printf("openshift-cucumber - Commit %s\n", gitCommit)
		} else {
			fmt.Printf("openshift-cucumber - Commit %s Build #%s\n", gitCommit, buildNumber)
		}
		os.Exit(0)
	}

	if len(os.Args) == 1 {
		usage := `%[1]s will run tests for all the cucumber '*.feature' files found in the provided paths.

Usage:
%[1]s /path/to/directory-with-cucumber-feature-files
`
		fmt.Printf(usage, os.Args[0])
		os.Exit(1)
	}

	c := steps.NewContext(&gucumber.GlobalContext)

	for _, dir := range os.Args[1:] {
		log.Printf("Running openshift-cucumber on directory %s ...\n", dir)
		c.RunDir(dir)
	}

}
