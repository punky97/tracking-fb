package bkmigration

import (
	"github.com/punky97/go-codebase/core/bkmigration/translators"
	"fmt"
)

func newSchemaMigrations(name string) translators.Table {
	return translators.Table{
		Name: name,
		Columns: []translators.Column{
			{
				Name:    "id",
				ColType: "int",
				Primary: true,
			},
			{
				Name:    "table_name",
				ColType: "varchar",
				Options: map[string]interface{}{
					"size": 255,
				},
			},
			{
				Name:    "migration_name",
				ColType: "varchar",
				Options: map[string]interface{}{
					"size": 255,
				},
			},
			{
				Name:    "version",
				ColType: "string",
				Options: map[string]interface{}{
					"size": 14,
				},
			},
		},
		Indexes: []translators.Index{
			{Name: fmt.Sprintf("%s_version_idx", name), Columns: []string{"version"}, Unique: true},
		},
	}
}