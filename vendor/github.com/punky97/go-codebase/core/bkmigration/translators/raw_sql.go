package translators

import "strings"

func (e ezmer) RawSQL(sql string) {
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}
	e.add(sql, nil)
}

// Deprecated: use RawSQL instead.
func (f ezmer) RawSql(sql string) {
	f.RawSQL(sql)
}
