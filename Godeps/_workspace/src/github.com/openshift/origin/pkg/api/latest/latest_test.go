package latest

import (
	"testing"

	"k8s.io/kubernetes/pkg/api/meta"
)

func TestRESTRootScope(t *testing.T) {
	for _, v := range [][]string{{"v1beta3"}, {"v1"}} {
		mapping, err := RESTMapper.RESTMapping("Node", v...)
		if err != nil {
			t.Fatal(err)
		}
		if mapping.Scope.Name() != meta.RESTScopeNameRoot {
			t.Errorf("Node should have a root scope: %#v", mapping.Scope)
		}
	}
}
