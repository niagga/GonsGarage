package cloudware_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainAndHandlerDoNotImportCloudware(t *testing.T) {
	t.Parallel()

	roots := []string{
		filepath.Join("..", "..", "..", "domain"),
		filepath.Join("..", "..", "..", "handler"),
	}
	cloudwareImport := "github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/cloudware"
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			fset := token.NewFileSet()
			file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if parseErr != nil {
				return parseErr
			}
			for _, imp := range file.Imports {
				pathLit := strings.Trim(imp.Path.Value, `"`)
				require.NotEqual(t, cloudwareImport, pathLit, "forbidden cloudware import in %s", path)
			}
			return nil
		})
		require.NoError(t, err)
	}
}
