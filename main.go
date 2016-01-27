// Provides an executable for running cucumber tests on an OpenShift instance
package main

import (
	"bufio"
	"flag"
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

func init() {
	// disable glog logging to stderr by default
	// because we don't want port-forwarding info messages in the console output
	flag.Set("logtostderr", "false")
}

func main() {
	flags := pflag.NewFlagSet("openshift-cucumber", pflag.ExitOnError)
	printVersion := flags.BoolP("version", "v", false, "print version")
	featuresFilesOrDirs := flags.StringSliceP("features", "f", []string{}, "paths to .feature files or directories")
	reporterName := flags.StringP("reporter", "r", "", "reporter (junit)")
	outputFile := flags.StringP("output", "o", "", "output file")
	//util.AddFlagSetToPFlagSet(flag.CommandLine, flags) // import glog flags
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

	if len(*featuresFilesOrDirs) == 0 {
		flags.PrintDefaults()
		os.Exit(1)
	}

	features := []string{}
	for _, dir := range *featuresFilesOrDirs {
		info, err := os.Stat(dir)
		if err != nil {
			log.Printf("Failed to open path %s: %v", dir, err)
			continue
		}

		if info.IsDir() {
			if files, _ := filepath.Glob(filepath.Join(dir, "*.feature")); files != nil {
				features = append(features, files...)
			}
			if files, _ := filepath.Glob(filepath.Join(dir, "**", "*.feature")); files != nil {
				features = append(features, files...)
			}
		} else {
			features = append(features, dir)
		}
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
