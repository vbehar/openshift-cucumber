/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl"

	"github.com/spf13/cobra"
)

// AddPrinterFlags adds printing related flags to a command (e.g. output format, no headers, template path)
func AddPrinterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml|template|templatefile|wide.")
	cmd.Flags().String("output-version", "", "Output the formatted object with the given version (default api-version).")
	cmd.Flags().Bool("no-headers", false, "When using the default output, don't print headers.")
	cmd.Flags().StringP("template", "t", "", "Template string or path to template file to use when -o=template or -o=templatefile.  The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview]")
}

// OutputVersion returns the preferred output version for generic content (JSON, YAML, or templates)
func OutputVersion(cmd *cobra.Command, defaultVersion string) string {
	outputVersion := GetFlagString(cmd, "output-version")
	if len(outputVersion) == 0 {
		outputVersion = defaultVersion
	}
	return outputVersion
}

// PrinterForCommand returns the default printer for this command.
// Requires that printer flags have been added to cmd (see AddPrinterFlags).
func PrinterForCommand(cmd *cobra.Command) (kubectl.ResourcePrinter, bool, error) {
	outputFormat := GetFlagString(cmd, "output")
	templateFile := GetFlagString(cmd, "template")
	if len(outputFormat) == 0 && len(templateFile) != 0 {
		outputFormat = "template"
	}

	return kubectl.GetPrinter(outputFormat, templateFile)
}
