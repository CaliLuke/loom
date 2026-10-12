package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestRecursiveAuthorizationValidationGenerated(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		node := dsl.Type("Node", func() {
			dsl.Attribute("value", dsl.String, func() {
				dsl.Pattern("^[a-z]+$")
			})
			dsl.Attribute("next", "Node")
			dsl.Required("value")
		})
		permission := dsl.Authorization("trees.read", node)
		dsl.Service("trees", func() {
			dsl.StrictAuthorization()
			dsl.Method("check", func() {
				dsl.Payload(node)
				dsl.Authorize(permission, func() {
					dsl.Bind("value", "value")
					dsl.Bind("next", "next")
				})
			})
		})
	})
	data := NewServicesData(root)
	dir := t.TempDir()
	const module = "example.com/recursiveauthorization"
	for _, svc := range root.Services {
		files := Files(module+"/gen", svc, data, make(map[string][]string))
		files = append(files, EndpointFile(module+"/gen", svc, data))
		for _, file := range files {
			_, err := file.Render(dir)
			require.NoError(t, err)
		}
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27.0\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "recursive_test.go"), []byte(recursiveAuthorizationValidationRuntime), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

const recursiveAuthorizationValidationRuntime = `package recursive_test
import (
 "context"
 "strings"
 "testing"
 svc "example.com/recursiveauthorization/gen/trees"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/stretchr/testify/require"
)
type implementation struct {
 calls,authorizations int
}
func (s *implementation) Check(context.Context,*svc.Node) error {
 s.calls++
 return nil
}
func (s *implementation) AuthorizeTreesRead(context.Context,*svc.Node) error {
 s.authorizations++
 return nil
}
func TestRecursiveAuthorization(t *testing.T) {
 for _,invalid:=range []bool{false,true} {
  impl:=&implementation{}
  endpoints:=svc.NewEndpoints(impl,impl)
  leaf:="good"
  if invalid {
   leaf="BAD"
  }
  node:=&svc.Node{Value:leaf}
  for i:=0;i<5;i++ {
   node=&svc.Node{Value:"good",Next:node}
  }
  _,err:=endpoints.Check(context.Background(),node)
  if invalid {
   var detail *loom.ServiceError
   require.ErrorAs(t,err,&detail)
   require.NotNil(t,detail.Field)
   require.Equal(t,"p"+strings.Repeat(".next",5)+".value",*detail.Field)
   require.Zero(t,impl.calls)
   require.Zero(t,impl.authorizations)
  } else {
   require.NoError(t,err)
   require.Equal(t,1,impl.calls)
   require.Equal(t,1,impl.authorizations)
  }
 }
}
`
