// Provides an executable for running cucumber tests on an OpenShift instance
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vbehar/openshift-cucumber/reporter"
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
	reporterName := flags.StringP("reporter", "r", "", "reporter (junit)")
	outputFile := flags.StringP("output", "o", "", "output file")
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

	features := []string{}
	for _, dir := range os.Args[1:] {
		firstLevelFiles, _ := filepath.Glob(filepath.Join(dir, "*.feature"))
		subLevelsFiles, _ := filepath.Glob(filepath.Join(dir, "**", "*.feature"))

		features = append(features, firstLevelFiles...)
		features = append(features, subLevelsFiles...)
	}

	c := steps.NewContext(&gucumber.GlobalContext)
	runner, err := c.RunFiles(features)
	if err != nil {
		log.Fatalf("Got error %v\n", err)
	}

	if *reporterName == "junit" {
		junit := &reporter.JunitReporter{}
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file %s: %v", *outputFile, err)
		} else {
			defer f.Close()

			w := bufio.NewWriter(f)
			if err = junit.GenerateReport(runner.Results, w); err != nil {
				log.Fatalf("Failed to generate JUnit Report to %s: %v", *outputFile, err)
			}
		}

	}

}
