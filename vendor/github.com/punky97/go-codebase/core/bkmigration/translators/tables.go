package translators

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gobuffalo/plush"
	"github.com/pkg/errors"
	"sort"
	"strings"
)

// Table is the table definition.
type Table struct {
	Name         string `db:"name"`
	Columns      []Column
	Indexes      []Index
	ForeignKeys  []ForeignKey
	primaryKeys  []string
	Options      map[string]interface{}
	columnsCache map[string]struct{}
}

// NewTable creates a new Table.
func NewTable(name string, opts map[string]interface{}) Table {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	// auto-timestamp as default
	if enabled, exists := opts["timestamps"]; !exists || enabled == true {
		opts["timestamps"] = true
	}
	return Table{
		Name:         name,
		Columns:      []Column{},
		Indexes:      []Index{},
		Options:      opts,
		columnsCache: map[string]struct{}{},
	}
}

func (t Table) String() string {
	return t.Ezm()
}

// Emz returns the fizz DDL to create the table.
func (t Table) Ezm() string {
	var buff bytes.Buffer
	var timestampsOpt = false
	if t.Options["timestamps"] != nil {
		timestampsOpt = t.Options["timestamps"].(bool)
	}
	// Write table options
	o := make([]string, 0, len(t.Options))
	for k, v := range t.Options {
		// Special handling for timestamps option
		if k == "timestamps" {
			continue
		}
		vv, _ := json.Marshal(v)
		o = append(o, fmt.Sprintf("%s: %s", k, string(vv)))
	}

	if len(o) > 0 {
		sort.SliceStable(o, func(i, j int) bool { return o[i] < o[j] })
		buff.WriteString(fmt.Sprintf("create_table(\"%s\", {%s}) {\n", t.Name, strings.Join(o, ", ")))
	} else {
		buff.WriteString(fmt.Sprintf("create_table(\"%s\") {\n", t.Name))
	}

	// Write columns
	if timestampsOpt {
		for _, c := range t.Columns {
			if c.Name == "created_at" || c.Name == "updated_at" {
				continue
			}
			buff.WriteString(fmt.Sprintf("\t%s\n", c.String()))
		}
	} else {
		for _, c := range t.Columns {
			buff.WriteString(fmt.Sprintf("\t%s\n", c.String()))
		}
	}

	if timestampsOpt {
		buff.WriteString("\tt.Timestamps()\n")
	}

	//Write primary key (single column pk will be written in inline form as the column opt)
	if len(t.primaryKeys) > 1 {
		pks := make([]string, len(t.primaryKeys))
		for i, pk := range t.primaryKeys {
			pks[i] = fmt.Sprintf("\"%s\"", pk)
		}
		buff.WriteString(fmt.Sprintf("\tt.PrimaryKey(%s)\n", strings.Join(pks, ", ")))
	}

	// Write indexes
	for _, i := range t.Indexes {
		buff.WriteString(fmt.Sprintf("\t%s\n", i.String()))
	}
	// Write foreign keys
	for _, fk := range t.ForeignKeys {
		buff.WriteString(fmt.Sprintf("\t%s\n", fk.String()))
	}
	buff.WriteString("}")
	return buff.String()
}

func (t Table) UnEzm() string {
	return fmt.Sprintf("drop_table(\"%s\")", t.Name)
}

func (f ezmer) CreateTable(name string, opts map[string]interface{}, help plush.HelperContext) error {
	t := NewTable(name, opts)
	if help.HasBlock() {
		ctx := help.Context.New()
		ctx.Set("t", &t)
		if _, err := help.BlockWith(ctx); err != nil {
			return errors.WithStack(err)
		}
	}
	if t.Options["timestamps"].(bool) {
		if !t.HasColumns("created_at", "updated_at") {
			t.Timestamps()
		}
	}
	f.add(f.Bubbler.CreateTable(t))
	return nil
}

func (f ezmer) DropTable(name string) {
	f.add(f.Bubbler.DropTable(Table{Name: name}))
}
func (f ezmer) RenameTable(old, new string) {
	f.add(f.Bubbler.RenameTable([]Table{
		{Name: old},
		{Name: new},
	}))
}

// Timestamps adds created_at and updated_at columns to the Table definition.
func (t *Table) Timestamps() error {
	if err := t.Timestamp("created_at"); err != nil {
		return err
	}
	return t.Timestamp("updated_at")
}

// Timestamp is a shortcut to add a timestamp column with default options.
func (t *Table) Timestamp(name string) error {
	return t.Column(name, "timestamp", Options{})
}

// Column adds a column to the table definition.
func (t *Table) Column(name string, colType string, options Options) error {
	if _, found := t.columnsCache[name]; found {
		return fmt.Errorf("duplicated column %s", name)
	}
	var primary bool
	if _, ok := options["primary"]; ok {
		if t.primaryKeys != nil {
			return errors.New("could not define multiple primary keys")
		}
		primary = true
		t.primaryKeys = []string{name}
	}
	c := Column{
		Name:    name,
		ColType: colType,
		Options: options,
		Primary: primary,
	}
	if t.columnsCache == nil {
		t.columnsCache = make(map[string]struct{})
	}
	t.columnsCache[name] = struct{}{}
	// Ensure id is first
	if name == "id" {
		t.Columns = append([]Column{c}, t.Columns...)
	} else {
		t.Columns = append(t.Columns, c)
	}
	return nil
}

func (t *Table) DisableTimestamps() {
	t.Options["timestamps"] = false
}

// PrimaryKey adds a primary key to the table. It's useful to define a composite
// primary key.
func (t *Table) PrimaryKey(pk ...string) error {
	if len(pk) == 0 {
		return errors.New("missing columns for primary key")
	}
	if t.primaryKeys != nil {
		return errors.New("duplicate primary key")
	}
	if !t.HasColumns(pk...) {
		return errors.New("columns must be declared before the primary key")
	}
	if len(pk) == 1 {
		for i, c := range t.Columns {
			if c.Name == pk[0] {
				t.Columns[i].Primary = true
				break
			}
		}
	}
	t.primaryKeys = make([]string, 0)
	t.primaryKeys = append(t.primaryKeys, pk...)
	return nil
}

// PrimaryKeys gets the list of registered primary key fields.
func (t *Table) PrimaryKeys() []string {
	return t.primaryKeys
}

// ColumnNames returns the names of the Table's columns.
func (t *Table) ColumnNames() []string {
	cols := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		cols[i] = c.Name
	}
	return cols
}

// HasColumns checks if the Table has all the given columns.
func (t *Table) HasColumns(args ...string) bool {
	for _, a := range args {
		if _, ok := t.columnsCache[a]; !ok {
			return false
		}
	}
	return true
}

// Index adds a new index to the table definition.
func (t *Table) Index(columns interface{}, options Options) error {
	i := Index{}
	switch tp := columns.(type) {
	default:
		return errors.Errorf("unexpected type %T for %s index columns", tp, t.Name) // %T prints whatever type t has
	case string:
		i.Columns = []string{tp}
	case []string:
		if len(tp) == 0 {
			return errors.Errorf("expected at least one column to apply %s index", t.Name)
		}
		i.Columns = tp
	case []interface{}:
		if len(tp) == 0 {
			return errors.Errorf("expected at least one column to apply %s index", t.Name)
		}
		cl := make([]string, len(tp))
		for i, c := range tp {
			cl[i] = c.(string)
		}
		i.Columns = cl
	}
	if options["name"] != nil {
		i.Name = options["name"].(string)
	} else {
		i.Name = fmt.Sprintf("%s_%s_idx", t.Name, strings.Join(i.Columns, "_"))
	}
	i.Unique = options["unique"] != nil && options["unique"].(bool)
	t.Indexes = append(t.Indexes, i)
	return nil
}

// ForeignKey adds a new foreign key to the table definition.
func (t *Table) ForeignKey(column string, refs interface{}, options Options) error {
	fkr, err := parseForeignKeyRef(refs)
	if err != nil {
		return errors.Wrap(err, "could not parse foreign key")
	}
	fk := ForeignKey{
		Column:     column,
		References: fkr,
		Options:    options,
	}

	if options["name"] != nil {
		fk.Name = options["name"].(string)
	} else {
		fk.Name = fmt.Sprintf("%s_%s_%s_fk", t.Name, fk.References.Table, strings.Join(fk.References.Columns, "_"))
	}

	t.ForeignKeys = append(t.ForeignKeys, fk)
	return nil
}
