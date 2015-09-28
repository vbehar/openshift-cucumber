// Package reporter provides reporters for generating reports with the results of the tests
package reporter

import (
	"io"

	"github.com/lsegal/gucumber"
)

// Reporter allows to generate a report of the results
type Reporter interface {
	GenerateReport([]gucumber.RunnerResult, io.Writer) error
}
