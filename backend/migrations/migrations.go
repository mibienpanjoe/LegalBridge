package migrations

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed *.sql
var files embed.FS

// All returns the contents of every embedded migration, ordered by filename.
// Each migration is idempotent, so the full list is applied on every run.
func All() ([]string, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	sqls := make([]string, 0, len(names))
	for _, name := range names {
		b, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		sqls = append(sqls, string(b))
	}
	return sqls, nil
}
