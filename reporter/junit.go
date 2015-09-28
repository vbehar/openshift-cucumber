package reporter

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/jstemmer/go-junit-report/parser"

	"github.com/lsegal/gucumber"
)

// JunitReporter generates tests reports in the JUnit XML format
type JunitReporter struct {
	Packages []*parser.Package
}

// GenerateReport writes a report in the JUnit XML format using the given writer
func (jr *JunitReporter) GenerateReport(results []gucumber.RunnerResult, w io.Writer) error {
	for i, res := range results {

		var pkg *parser.Package
		if i > 0 && results[i-1].Feature == res.Feature {
			// same feature => re-use the previous package
			// (1 feature <=> 1 package)
			pkg = jr.Packages[len(jr.Packages)-1]

			if results[i-1].Scenario.Line == res.Scenario.Line {
				// same scenario => skip to the next result
				// (1 scenario <=> 1 test)
				continue
			}
		} else {
			// new feature => create a new package
			pkg = &parser.Package{
				Name: res.Feature.Title,
			}
			jr.Packages = append(jr.Packages, pkg)
		}

		// new test (for the scenario)
		test := &parser.Test{
			Name: res.Scenario.Title,
		}
		switch {
		case res.Failed():
			test.Result = parser.FAIL
			test.Output = []string{}
			for _, err := range res.Errors() {
				test.Output = append(test.Output, err.String())
			}
		case res.Skipped():
			test.Result = parser.SKIP
		default:
			test.Result = parser.PASS
		}

		pkg.Tests = append(pkg.Tests, test)
	}

	report := &parser.Report{}
	for _, pkg := range jr.Packages {
		report.Packages = append(report.Packages, *pkg)
	}

	return JUnitReportXML(report, false, w)
}

// the following code has been copy-pasted from
// https://github.com/jstemmer/go-junit-report/blob/master/junit-formatter.go

// JUnitTestSuites is a collection of JUnit test suites.
type JUnitTestSuites struct {
	XMLName xml.Name `xml:"testsuites"`
	Suites  []JUnitTestSuite
}

// JUnitTestSuite is a single JUnit test suite which may contain many
// testcases.
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Time       string          `xml:"time,attr"`
	Name       string          `xml:"name,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase
}

// JUnitTestCase is a single test case with its result.
type JUnitTestCase struct {
	XMLName     xml.Name          `xml:"testcase"`
	Classname   string            `xml:"classname,attr"`
	Name        string            `xml:"name,attr"`
	Time        string            `xml:"time,attr"`
	SkipMessage *JUnitSkipMessage `xml:"skipped,omitempty"`
	Failure     *JUnitFailure     `xml:"failure,omitempty"`
}

// JUnitSkipMessage contains the reason why a testcase was skipped.
type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

// JUnitProperty represents a key/value pair used to define properties.
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitFailure contains data related to a failed test.
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitReportXML writes a JUnit xml representation of the given report to w
// in the format described at http://windyroad.org/dl/Open%20Source/JUnit.xsd
func JUnitReportXML(report *parser.Report, noXMLHeader bool, w io.Writer) error {
	suites := JUnitTestSuites{}

	// convert Report to JUnit test suites
	for _, pkg := range report.Packages {
		ts := JUnitTestSuite{
			Tests:      len(pkg.Tests),
			Failures:   0,
			Time:       formatTime(pkg.Time),
			Name:       pkg.Name,
			Properties: []JUnitProperty{},
			TestCases:  []JUnitTestCase{},
		}

		classname := pkg.Name
		if idx := strings.LastIndex(classname, "/"); idx > -1 && idx < len(pkg.Name) {
			classname = pkg.Name[idx+1:]
		}

		// properties
		ts.Properties = append(ts.Properties, JUnitProperty{"go.version", runtime.Version()})
		if pkg.CoveragePct != "" {
			ts.Properties = append(ts.Properties, JUnitProperty{"coverage.statements.pct", pkg.CoveragePct})
		}

		// individual test cases
		for _, test := range pkg.Tests {
			testCase := JUnitTestCase{
				Classname: classname,
				Name:      test.Name,
				Time:      formatTime(test.Time),
				Failure:   nil,
			}

			if test.Result == parser.FAIL {
				ts.Failures++
				testCase.Failure = &JUnitFailure{
					Message:  "Failed",
					Type:     "",
					Contents: strings.Join(test.Output, "\n"),
				}
			}

			if test.Result == parser.SKIP {
				testCase.SkipMessage = &JUnitSkipMessage{strings.Join(test.Output, "\n")}
			}

			ts.TestCases = append(ts.TestCases, testCase)
		}

		suites.Suites = append(suites.Suites, ts)
	}

	// to xml
	bytes, err := xml.MarshalIndent(suites, "", "\t")
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(w)

	if !noXMLHeader {
		writer.WriteString(xml.Header)
	}

	writer.Write(bytes)
	writer.WriteByte('\n')
	writer.Flush()

	return nil
}

func countFailures(tests []parser.Test) (result int) {
	for _, test := range tests {
		if test.Result == parser.FAIL {
			result++
		}
	}
	return
}

func formatTime(time int) string {
	return fmt.Sprintf("%.3f", float64(time)/1000.0)
}
