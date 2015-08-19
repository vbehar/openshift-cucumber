package describe

import (
	"strings"
	"testing"

	"github.com/gonum/graph"
	"github.com/gonum/graph/concrete"
	kapi "k8s.io/kubernetes/pkg/api"
	ktestclient "k8s.io/kubernetes/pkg/client/testclient"
	kutil "k8s.io/kubernetes/pkg/util"

	"github.com/openshift/origin/pkg/client/testclient"
	imagegraph "github.com/openshift/origin/pkg/image/graph/nodes"
)

func TestChainDescriber(t *testing.T) {
	tests := []struct {
		testName         string
		namespaces       kutil.StringSet
		output           string
		defaultNamespace string
		name             string
		tag              string
		path             string
		humanReadable    map[string]int
		dot              []string
		expectedErr      error
		includeInputImg  bool
	}{
		{
			testName:         "human readable test - single namespace",
			namespaces:       kutil.NewStringSet("test"),
			output:           "",
			defaultNamespace: "test",
			name:             "ruby-20-centos7",
			tag:              "latest",
			path:             "../../../../pkg/cmd/experimental/buildchain/test/single-namespace-bcs.yaml",
			humanReadable: map[string]int{
				"imagestreamtag/ruby-20-centos7:latest":        1,
				"\tbc/ruby-hello-world":                        1,
				"\t\timagestreamtag/ruby-hello-world:latest":   1,
				"\tbc/ruby-sample-build":                       1,
				"\t\timagestreamtag/origin-ruby-sample:latest": 1,
			},
			expectedErr: nil,
		},
		{
			testName:         "dot test - single namespace",
			namespaces:       kutil.NewStringSet("test"),
			output:           "dot",
			defaultNamespace: "test",
			name:             "ruby-20-centos7",
			tag:              "latest",
			path:             "../../../../pkg/cmd/experimental/buildchain/test/single-namespace-bcs.yaml",
			dot: []string{
				"digraph \"ruby-20-centos7:latest\" {",
				"// Node definitions.",
				"[label=\"BuildConfig|test/ruby-hello-world\"];",
				"[label=\"BuildConfig|test/ruby-sample-build\"];",
				"[label=\"ImageStreamTag|test/ruby-hello-world:latest\"];",
				"[label=\"ImageStreamTag|test/ruby-20-centos7:latest\"];",
				"[label=\"ImageStreamTag|test/origin-ruby-sample:latest\"];",
				"",
				"// Edge definitions.",
				"[label=\"BuildOutput\"];",
				"[label=\"BuildOutput\"];",
				"[label=\"BuildInputImage,BuildTriggerImage\"];",
				"[label=\"BuildInputImage,BuildTriggerImage\"];",
				"}",
			},
			expectedErr: nil,
		},
		{
			testName:         "human readable test - multiple namespaces",
			namespaces:       kutil.NewStringSet("test", "master", "default"),
			output:           "",
			defaultNamespace: "master",
			name:             "ruby-20-centos7",
			tag:              "latest",
			path:             "../../../../pkg/cmd/experimental/buildchain/test/multiple-namespaces-bcs.yaml",
			humanReadable: map[string]int{
				"<master imagestreamtag/ruby-20-centos7:latest>":         1,
				"\t<default bc/ruby-hello-world>":                        1,
				"\t\t<test imagestreamtag/ruby-hello-world:latest>":      1,
				"\t<test bc/ruby-sample-build>":                          1,
				"\t\t<another imagestreamtag/origin-ruby-sample:latest>": 1,
			},
			expectedErr: nil,
		},
		{
			testName:         "dot test - multiple namespaces",
			namespaces:       kutil.NewStringSet("test", "master", "default"),
			output:           "dot",
			defaultNamespace: "master",
			name:             "ruby-20-centos7",
			tag:              "latest",
			path:             "../../../../pkg/cmd/experimental/buildchain/test/multiple-namespaces-bcs.yaml",
			dot: []string{
				"digraph \"ruby-20-centos7:latest\" {",
				"// Node definitions.",
				"[label=\"BuildConfig|default/ruby-hello-world\"];",
				"[label=\"BuildConfig|test/ruby-sample-build\"];",
				"[label=\"ImageStreamTag|test/ruby-hello-world:latest\"];",
				"[label=\"ImageStreamTag|master/ruby-20-centos7:latest\"];",
				"[label=\"ImageStreamTag|another/origin-ruby-sample:latest\"];",
				"",
				"// Edge definitions.",
				"[label=\"BuildOutput\"];",
				"[label=\"BuildOutput\"];",
				"[label=\"BuildInputImage,BuildTriggerImage\"];",
				"[label=\"BuildInputImage,BuildTriggerImage\"];",
				"}",
			},
			expectedErr: nil,
		},
		{
			testName:         "human readable - multiple triggers - triggeronly",
			name:             "ruby-20-centos7",
			defaultNamespace: "test",
			tag:              "latest",
			path:             "../../../../pkg/cmd/experimental/buildchain/test/multiple-trigger-bcs.yaml",
			namespaces:       kutil.NewStringSet("test"),
			humanReadable: map[string]int{
				"imagestreamtag/ruby-20-centos7:latest":   1,
				"\tbc/parent1":                            1,
				"\t\timagestreamtag/parent1img:latest":    1,
				"\t\t\tbc/child2":                         2,
				"\t\t\t\timagestreamtag/child2img:latest": 2,
				"\tbc/parent2":                            1,
				"\t\timagestreamtag/parent2img:latest":    1,
				"\t\t\tbc/child3":                         2,
				"\t\t\t\timagestreamtag/child3img:latest": 2,
				"\t\t\tbc/child1":                         1,
				"\t\t\t\timagestreamtag/child1img:latest": 1,
				"\tbc/parent3":                            1,
				"\t\timagestreamtag/parent3img:latest":    1,
			},
		},
		{
			testName:         "human readable - multiple triggers - trigger+input",
			name:             "ruby-20-centos7",
			defaultNamespace: "test",
			tag:              "latest",
			path:             "../../../../pkg/cmd/experimental/buildchain/test/multiple-trigger-bcs.yaml",
			namespaces:       kutil.NewStringSet("test"),
			includeInputImg:  true,
			humanReadable: map[string]int{
				"imagestreamtag/ruby-20-centos7:latest":   1,
				"\tbc/parent1":                            1,
				"\t\timagestreamtag/parent1img:latest":    1,
				"\t\t\tbc/child1":                         2,
				"\t\t\t\timagestreamtag/child1img:latest": 2,
				"\t\t\tbc/child2":                         2,
				"\t\t\t\timagestreamtag/child2img:latest": 2,
				"\t\t\tbc/child3":                         3,
				"\t\t\t\timagestreamtag/child3img:latest": 3,
				"\tbc/parent2":                            1,
				"\t\timagestreamtag/parent2img:latest":    1,
				"\tbc/parent3":                            1,
				"\t\timagestreamtag/parent3img:latest":    1,
			},
		},
	}

	for _, test := range tests {
		o := ktestclient.NewObjects(kapi.Scheme, kapi.Scheme)
		if len(test.path) > 0 {
			if err := ktestclient.AddObjectsFromPath(test.path, o, kapi.Scheme); err != nil {
				t.Fatal(err)
			}
		}

		oc, _ := testclient.NewFixtureClients(o)
		ist := imagegraph.MakeImageStreamTagObjectMeta(test.defaultNamespace, test.name, test.tag)

		desc, err := NewChainDescriber(oc, test.namespaces, test.output).Describe(ist, test.includeInputImg)
		t.Logf("%s: output:\n%s\n\n", test.testName, desc)
		if err != test.expectedErr {
			t.Fatalf("%s: error mismatch: expected %v, got %v", test.testName, test.expectedErr, err)
		}

		got := strings.Split(desc, "\n")

		switch test.output {
		case "dot":
			if len(test.dot) != len(got) {
				t.Fatalf("%s: expected %d lines, got %d:\n%s", test.testName, len(test.dot), len(got), desc)
			}
			for _, expected := range test.dot {
				if !strings.Contains(desc, expected) {
					t.Errorf("%s: unexpected description:\n%s\nexpected line in it:\n%s", test.testName, desc, expected)
				}
			}
		case "":
			if lenReadable(test.humanReadable) != len(got) {
				t.Fatalf("%s: expected %d lines, got %d:\n%s", test.testName, lenReadable(test.humanReadable), len(got), desc)
			}
			for _, line := range got {
				if _, ok := test.humanReadable[line]; !ok {
					t.Errorf("%s: unexpected line: %s", test.testName, line)
				}
				test.humanReadable[line]--
			}
			for line, cnt := range test.humanReadable {
				if cnt != 0 {
					t.Errorf("%s: unexpected number of lines for [%s]: %d", test.testName, line, cnt)
				}
			}
		}
	}
}

func lenReadable(value map[string]int) int {
	length := 0
	for _, cnt := range value {
		length += cnt
	}
	return length
}

func TestDepthFirst(t *testing.T) {
	g := concrete.NewDirectedGraph()

	a := concrete.Node(g.NewNodeID())
	b := concrete.Node(g.NewNodeID())

	g.AddNode(a)
	g.AddNode(b)
	g.SetEdge(concrete.Edge{F: a, T: b}, 1)
	g.SetEdge(concrete.Edge{F: b, T: a}, 1)

	count := 0

	df := &DepthFirst{
		EdgeFilter: func(graph.Edge) bool {
			return true
		},
		Visit: func(u, v graph.Node) {
			count++
			t.Logf("%d -> %d\n", u.ID(), v.ID())
		},
	}

	df.Walk(g, a, func(n graph.Node) bool {
		if count > 100 {
			t.Fatalf("looped")
			return true
		}
		return false
	})
}
