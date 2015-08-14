// Provides an executable for running cucumber tests on an OpenShift instance
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vbehar/openshift-cucumber/steps"

	"github.com/lsegal/gucumber"
)

func main() {
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
