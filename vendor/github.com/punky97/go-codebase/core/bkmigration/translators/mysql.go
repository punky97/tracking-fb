package translators

import (
	"github.com/punky97/go-codebase/core/bkmigration/utils"
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strings"
)

const strDefaultSize = 255

type MySQL struct {
	Schema         SchemaQuery
	strDefaultSize int
}

func NewMySQL(url, name string) *MySQL {
	sch := &mysqlSchema{Schema{URL: url, Name: name, schema: map[string]*Table{}}}
	sch.Builder = sch
	return &MySQL{
		strDefaultSize: strDefaultSize,
		Schema:         sch,
	}
}

func (p *MySQL) CreateTable(t Table) (string, error) {
	sql := make([]string, 0)
	cols := make([]string, 0)

	for _, c := range t.Columns {
		cols = append(cols, p.buildColumn(c))
		if c.Primary {
			cols = append(cols, fmt.Sprintf("PRIMARY KEY(`%s`)", c.Name))
		}
	}

	for _, fk := range t.ForeignKeys {
		cols = append(cols, p.buildForeignKey(t, fk, true))
	}

	primaryKeys := t.PrimaryKeys()
	if len(primaryKeys) > 1 {
		pks := make([]string, len(primaryKeys))
		for i, pk := range primaryKeys {
			pks[i] = fmt.Sprintf("`%s`", pk)
		}
		cols = append(cols, fmt.Sprintf("PRIMARY KEY(%s)", strings.Join(pks, ", ")))
	}

	s := fmt.Sprintf("CREATE TABLE %s (\n%s\n) ENGINE=InnoDB;", utils.EscapeIdentifier(t.Name), strings.Join(cols, ",\n"))
	sql = append(sql, s)

	for _, i := range t.Indexes {
		s, err := p.AddIndex(Table{
			Name:    t.Name,
			Indexes: []Index{i},
		})
		if err != nil {
			return "", err
		}
		sql = append(sql, s)
	}

	return strings.Join(sql, "\n"), nil
}

func (p *MySQL) DropTable(t Table) (string, error) {
	return fmt.Sprintf("DROP TABLE %s;", utils.EscapeIdentifier(t.Name)), nil
}

func (p *MySQL) RenameTable(t []Table) (string, error) {
	if len(t) < 2 {
		return "", errors.New("not enough table names supplied")
	}
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s;", utils.EscapeIdentifier(t[0].Name), utils.EscapeIdentifier(t[1].Name)), nil
}

func (p *MySQL) ChangeColumn(t Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", errors.New("not enough columns supplied")
	}
	c := t.Columns[0]
	s := fmt.Sprintf("ALTER TABLE %s MODIFY %s;", utils.EscapeIdentifier(t.Name), p.buildColumn(c))
	return s, nil
}

func (p *MySQL) AddColumn(t Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", errors.New("not enough columns supplied")
	}

	if _, ok := t.Columns[0].Options["first"]; ok {
		c := t.Columns[0]
		s := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s FIRST;", utils.EscapeIdentifier(t.Name), p.buildColumn(c))
		return s, nil
	}

	if val, ok := t.Columns[0].Options["after"]; ok {
		c := t.Columns[0]
		s := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s AFTER `%s`;", utils.EscapeIdentifier(t.Name), p.buildColumn(c), val)
		return s, nil
	}

	c := t.Columns[0]
	s := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", utils.EscapeIdentifier(t.Name), p.buildColumn(c))
	return s, nil
}

func (p *MySQL) DropColumn(t Table) (string, error) {
	if len(t.Columns) == 0 {
		return "", errors.New("not enough columns supplied")
	}
	c := t.Columns[0]
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN `%s`;", utils.EscapeIdentifier(t.Name), c.Name), nil
}

func (p *MySQL) RenameColumn(t Table) (string, error) {
	if len(t.Columns) < 2 {
		return "", errors.New("not enough columns supplied")
	}
	oc := t.Columns[0]
	nc := t.Columns[1]

	ti, err := p.Schema.TableInfo(t.Name)

	if err != nil {
		return "", err
	}
	var c Column
	for _, c = range ti.Columns {
		if c.Name == oc.Name {
			break
		}
	}
	col := p.buildColumn(c)
	col = strings.Replace(col, oc.Name, fmt.Sprintf("%s` `%s", oc.Name, nc.Name), -1)
	s := fmt.Sprintf("ALTER TABLE %s CHANGE %s;", utils.EscapeIdentifier(t.Name), col)
	return s, nil
}

func (f *MySQL) AddIndex(t Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", errors.New(" not enough indexes supplied")
	}
	i := t.Indexes[0]
	cols := make([]string, 0)

	for _, c := range i.Columns {
		cols = append(cols, fmt.Sprintf("`%s`", c))
	}
	s := fmt.Sprintf("CREATE INDEX `%s` ON %s (%s);", i.Name, utils.EscapeIdentifier(t.Name), strings.Join(cols, ", "))
	if i.Unique {
		s = strings.Replace(s, "CREATE", "CREATE UNIQUE", 1)
	}
	return s, nil
}

func (p *MySQL) DropIndex(t Table) (string, error) {
	if len(t.Indexes) == 0 {
		return "", errors.New("not enough indexes supplied")
	}
	i := t.Indexes[0]
	return fmt.Sprintf("DROP INDEX `%s` ON %s;", i.Name, utils.EscapeIdentifier(t.Name)), nil
}

func (p *MySQL) RenameIndex(t Table) (string, error) {
	sch := p.Schema.(*mysqlSchema)
	version, err := sch.Version()
	if err != nil {
		return "", errors.WithStack(err)
	}
	if version.LT(mysql57Version) {
		return "", errors.New("renaming indexes on MySQL versions less than 5.7 is not supported; use raw SQL instead")
	}
	ix := t.Indexes
	if len(ix) < 2 {
		return "", errors.New("not enough indexes supplied")
	}
	oi := ix[0]
	ni := ix[1]
	return fmt.Sprintf("ALTER TABLE %s RENAME INDEX `%s` TO `%s`;", utils.EscapeIdentifier(t.Name), oi.Name, ni.Name), nil
}

func (p *MySQL) AddForeignKey(t Table) (string, error) {
	if len(t.ForeignKeys) == 0 {
		return "", errors.New("not enough foreign keys supplied")
	}

	return p.buildForeignKey(t, t.ForeignKeys[0], false), nil
}

func (p *MySQL) DropForeignKey(t Table) (string, error) {
	if len(t.ForeignKeys) == 0 {
		return "", errors.New("not enough foreign keys supplied")
	}

	fk := t.ForeignKeys[0]

	var ifExists string
	if v, ok := fk.Options["if_exists"]; ok && v.(bool) {
		ifExists = "IF EXISTS"
	}

	s := fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s `%s`;", utils.EscapeIdentifier(t.Name), ifExists, fk.Name)
	return s, nil
}

func (p *MySQL) buildForeignKey(t Table, fk ForeignKey, onCreate bool) string {
	rcols := []string{}
	for _, c := range fk.References.Columns {
		rcols = append(rcols, fmt.Sprintf("`%s`", c))
	}
	refs := fmt.Sprintf("%s (%s)", utils.EscapeIdentifier(fk.References.Table), strings.Join(rcols, ", "))
	s := fmt.Sprintf("FOREIGN KEY (`%s`) REFERENCES %s", fk.Column, refs)

	if onUpdate, ok := fk.Options["on_update"]; ok {
		s += fmt.Sprintf(" ON UPDATE %s", onUpdate)
	}

	if onDelete, ok := fk.Options["on_delete"]; ok {
		s += fmt.Sprintf(" ON DELETE %s", onDelete)
	}

	if !onCreate {
		s = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT `%s` %s;", utils.EscapeIdentifier(t.Name), fk.Name, s)
	}

	return s
}

func (p *MySQL) buildColumn(c Column) string {
	s := fmt.Sprintf("`%s` %s", c.Name, p.colType(c))
	if c.Options["unsigned"] != nil {
		s = fmt.Sprintf("%s unsigned", s)
	}

	if c.Options["null"] == nil || c.Primary {
		s = fmt.Sprintf("%s NOT NULL", s)
	}

	if c.Options["default"] != nil {
		d := fmt.Sprintf("%#v", c.Options["default"])
		re := regexp.MustCompile("^(\")(.+)(\")$")
		d = re.ReplaceAllString(d, "'$2'")
		s = fmt.Sprintf("%s DEFAULT %s", s, d)
	}

	if c.Options["default_raw"] != nil {
		d := fmt.Sprintf("%s", c.Options["default_raw"])
		s = fmt.Sprintf("%s DEFAULT %s", s, d)
	}

	if c.Options["collate"] != nil {
		collate := fmt.Sprintf("%#v", c.Options["collate"])
		s = fmt.Sprintf("%s COLLATE %s", s, collate)
	}

	if c.Options["comment"] != nil {
		cmt := fmt.Sprintf("%#v", c.Options["comment"])
		s = fmt.Sprintf("%s COMMENT %s", s, cmt)
	}

	if c.Primary && (c.ColType == "integer" || strings.ToLower(c.ColType) == "int" || strings.ToLower(c.ColType) == "bigint") {
		s = fmt.Sprintf("%s AUTO_INCREMENT", s)
	}
	return s
}

func (p *MySQL) colType(c Column) string {
	switch strings.ToLower(c.ColType) {
	case "string", "varchar":
		s := fmt.Sprintf("%d", p.strDefaultSize)
		if c.Options["size"] != nil {
			s = fmt.Sprintf("%d", c.Options["size"])
		}
		return fmt.Sprintf("VARCHAR (%s)", s)
	case "int", "integer":
		return "INTEGER"
	case "uuid": // for cassandra database
		return "char(36)"
	case "timestamp", "time", "datetime":
		return "DATETIME"
	case "blob", "[]byte":
		return "BLOB"
	case "tinyint", "TINYINT":
		size := "4" //default size tinyint type
		if c.Options["size"] != nil {
			size = fmt.Sprintf("%d", c.Options["size"])
		}
		return fmt.Sprintf("tinyint(%s)", size)
	case "float", "FLOAT":
		if c.Options["precision"] != nil {
			precision := c.Options["precision"]
			if c.Options["scale"] != nil {
				scale := c.Options["scale"]
				return fmt.Sprintf("FLOAT(%d,%d)", precision, scale)
			}
			return fmt.Sprintf("FLOAT(%d)", precision)
		}
		return "FLOAT"
	case "numeric", "decimal":
		if c.Options["precision"] != nil {
			precision := c.Options["precision"]
			if c.Options["scale"] != nil {
				scale := c.Options["scale"]
				return fmt.Sprintf("decimal(%d,%d)", precision, scale)
			}
			return fmt.Sprintf("decimal(%d)", precision)
		}
		return "DECIMAL"
	case "double", "DOUBLE":
		if c.Options["precision"] != nil {
			precision := c.Options["precision"]
			if c.Options["scale"] != nil {
				scale := c.Options["scale"]
				return fmt.Sprintf("double(%d,%d)", precision, scale)
			}
			return fmt.Sprintf("double(%d)", precision)
		}
		return "double"
	case "json":
		return "JSON"
	default:
		return c.ColType
	}
}
