package translators

import (
	"github.com/gobuffalo/plush"
	"os"

	"strings"
)

type BubbleType int

type Bubbler struct {
	Translator
	data []string
}

func NewBubbler(t Translator) *Bubbler {
	return &Bubbler{
		Translator: t,
		data:       []string{},
	}
}
func (b *Bubbler) String() string {
	return strings.Join(b.data, "\n")
}

func (b *Bubbler) Bubble(s string) (string, error) {
	f := ezmer{b}
	ctx := plush.NewContextWith(map[string]interface{}{
		"exec":             f.Exec(os.Stdout),
		"create_table":     f.CreateTable,
		"rename_table":     f.RenameTable,
		"drop_table":       f.DropTable,
		"add_column":       f.AddColumn,
		"change_column":    f.ChangeColumn,
		"rename_column":    f.RenameColumn,
		"drop_column":      f.DropColumn,
		"sql":              f.RawSQL,
		"add_index":        f.AddIndex,
		"drop_index":       f.DropIndex,
		"rename_index":     f.RenameIndex,
		"add_foreign_key":  f.AddForeignKey,
		"drop_foreign_key": f.DropForeignKey,
	})
	err := plush.RunScript(s, ctx)
	return b.String(), err
}
