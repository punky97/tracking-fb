package translators

import (
	"fmt"
	"github.com/blang/semver"
	"strings"
)

type SchemaQuery interface {
	ReplaceSchema(map[string]*Table)
	Build() error
	Version() (*semver.Version, error)
	TableInfo(string) (*Table, error)
	ReplaceColumn(table string, oldColumn string, newColumn Column) error
	ColumnInfo(table string, column string) (*Column, error)
	IndexInfo(table string, idx string) (*Index, error)
	Delete(string)
	SetTable(*Table)
	DeleteColumn(string, string)
}

type Schema struct {
	schema  map[string]*Table
	Builder SchemaQuery
	Name    string
	URL     string
}

func CreateSchema(name string, url string, schema map[string]*Table) Schema {
	return Schema{
		Name:   name,
		URL:    url,
		schema: schema,
	}
}

func (s *Schema) ReplaceSchema(newSchema map[string]*Table) {
	s.schema = newSchema
}

func (s *Schema) Build() error {
	return fmt.Errorf("build not implemented for this translator")
}

func (s *Schema) TableInfo(table string) (*Table, error) {
	if ti, ok := s.schema[table]; ok {
		return ti, nil
	}

	if s.Builder != nil {
		err := s.Builder.Build()
		if err != nil {
			return nil, err
		}
	} else {
		err := s.Build()
		if err != nil {
			return nil, err
		}
	}
	if ti, ok := s.schema[table]; ok {
		return ti, nil
	}
	return nil, fmt.Errorf("could not find table data for %s", table)
}

func (s *Schema) ReplaceColumn(table string, oldColumn string, newColumn Column) error {
	tableInfo, err := s.TableInfo(table)
	if err != nil {
		return err
	}
	for i, col := range tableInfo.Columns {
		if strings.ToLower(strings.TrimSpace(col.Name)) == strings.ToLower(strings.TrimSpace(oldColumn)) {
			tableInfo.Columns[i] = newColumn
			return nil
		}
	}
	return fmt.Errorf("could not find column(%s) in table(%s)", oldColumn, table)
}

func (s *Schema) ColumnInfo(table string, column string) (*Column, error) {
	ti, err := s.TableInfo(table)
	if err != nil {
		return nil, err
	}

	if ci, ok := s.findColumnInfo(ti, column); ok {
		return ci, nil
	}
	return nil, fmt.Errorf("could not find column data for %s in table %s", column, table)
}

func (s *Schema) IndexInfo(table string, idx string) (*Index, error) {
	ti, err := s.TableInfo(table)
	if err != nil {
		return nil, err
	}

	if i, ok := s.findIndexInfo(ti, idx); ok {
		return i, nil
	}
	return nil, fmt.Errorf("could not find index data for %s in table %s", idx, table)
}

func (s *Schema) Delete(table string) {
	delete(s.schema, table)
}

func (s *Schema) SetTable(table *Table) {
	s.schema[table.Name] = table
}

func (s *Schema) DeleteColumn(table string, column string) {
	tableInfo, err := s.TableInfo(table)
	if err != nil {
		return
	}
	for i, col := range tableInfo.Columns {
		if strings.ToLower(strings.TrimSpace(col.Name)) == strings.ToLower(strings.TrimSpace(column)) {
			tableInfo.Columns = append(tableInfo.Columns[:i], tableInfo.Columns[i+1:]...)
			return
		}
	}
}

func (s *Schema) findColumnInfo(tableInfo *Table, column string) (*Column, bool) {
	for _, col := range tableInfo.Columns {
		if strings.ToLower(strings.TrimSpace(col.Name)) == strings.ToLower(strings.TrimSpace(column)) {
			return &col, true
		}
	}
	return nil, false
}

func (s *Schema) findIndexInfo(tableInfo *Table, index string) (*Index, bool) {
	for _, ind := range tableInfo.Indexes {
		if strings.ToLower(strings.TrimSpace(ind.Name)) == strings.ToLower(strings.TrimSpace(index)) {
			return &ind, true
		}
	}
	return nil, false
}
