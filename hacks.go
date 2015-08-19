package main

/*

This file is a hack to fix the cross-arch non-awareness of godep.
See https://github.com/tools/godep/issues/174 for more details.

TL;DR
To run "godep save ./..." on darwin, uncomment the following block.
It will force godep to see the linux/windows-specific dependencies.

*/

/*

import (
	// required for linux build
	"github.com/docker/libcontainer/cgroups/fs"

	// required for windows build
	"github.com/Sirupsen/logrus"
)

var (
	// required for linux build
	_ = &fs.Manager{}

	// required for windows build
	_ = &logrus.Entry{}
)

*/
