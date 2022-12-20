package translators

import (
	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"
	"io"
	"os"
	"os/exec"
)

// Options is a generic map of options.
type Options map[string]interface{}

type ezmer struct {
	Bubbler *Bubbler
}

func (f ezmer) add(s string, err error) error {
	if err != nil {
		return errors.WithStack(err)
	}
	f.Bubbler.data = append(f.Bubbler.data, s)
	return nil
}

func (f ezmer) Exec(out io.Writer) func(string) error {
	return func(s string) error {
		args, err := shellquote.Split(s)
		if err != nil {
			return errors.Wrapf(err, "error parsing command: %s", s)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = out
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return errors.Wrapf(err, "error executing command: %s", s)
		}
		return nil
	}
}

// AString reads a fizz string, and translates its contents to SQL.
func AString(s string, t Translator) (string, error) {
	b := NewBubbler(t)
	return b.Bubble(s)
}
