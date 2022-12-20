package mysql

import (
	"database/sql"
	"strings"

	"github.com/spf13/viper"
)

const (
	driverName = "mysql"
)

// BeginTransaction -- Create a transaction
func (st *SQLTool) BeginTransaction() error {
	tx, err := st.db.Begin()
	if err != nil {
		return err
	}
	st.tx = tx
	return nil
}

// Rollback -- Rollback transaction
func (st *SQLTool) Rollback() error {
	return st.tx.Rollback()
}

// Commit -- Commit transaction
func (st *SQLTool) Commit() error {
	return st.tx.Commit()
}

// UsingTransaction --
func (st *SQLTool) UsingTransaction(callback func() []*Command) (results []sql.Result, err error) {
	var t *sql.Tx
	results = make([]sql.Result, 0)
	commands := callback()
	t, err = st.db.Begin()
	if err != nil {
		return
	}
	// If commit successful, let's publish to queue
	defer func() {
		if err == nil && len(results) == len(commands) && viper.GetBool("mysql_migrate.enable") {
			st.publishQueues(commands)
		}
	}()
	// Transaction must be end its by commit or rollback
	defer func() {
		// If panic show up, let's recover and rollback and re-throw panic
		if r := recover(); r != nil {
			if errs := t.Rollback(); errs != nil {
				err = errs
			}
			panic(r)
			// If any statement fail in execution, this transaction will be rollback
		} else if err != nil {
			if errs := t.Rollback(); errs != nil {
				err = errs
			}
			// Else let's commit it
		} else {
			if errs := t.Commit(); errs != nil {
				err = errs
			}
		}
	}()
	var stmt *sql.Stmt
	var result sql.Result
	// Run pipeline of commands to execute each statement
	if st.ctx != nil {
		for _, c := range commands {
			stmt, err = t.PrepareContext(st.ctx, c.Statement)
			if err != nil {
				return
			}
			defer stmt.Close()
			result, err = stmt.ExecContext(st.ctx, c.Arguments...)
			if err != nil {
				return
			}
			// Try to get Last Insert ID to make sure new record has written
			if _, err = result.LastInsertId(); err != nil {
				return
			}
			results = append(results, result)
		}
	} else {
		for _, c := range commands {
			stmt, err = t.Prepare(c.Statement)
			if err != nil {
				return
			}
			defer stmt.Close()
			result, err = stmt.Exec(c.Arguments...)
			if err != nil {
				return
			}
			// Try to get Last Insert ID to make sure new record has written
			if _, err = result.LastInsertId(); err != nil {
				return
			}
			results = append(results, result)
		}
	}
	return
}

func (st *SQLTool) QueryToCommand(query string, args ...interface{}) []*Command {
	commands := make([]*Command, 0)
	commands = append(commands, NewCommand(query, args...))
	return commands
}

func (st *SQLTool) QueryToCommands(query string, argsList [][]interface{}) []*Command {
	commands := make([]*Command, 0)
	for _, args := range argsList {
		commands = append(commands, QueryToCommand(query, args...))
	}
	return commands
}

func (st *SQLTool) publishQueues(commands []*Command) {
	exchangeName := viper.GetString("mysql_migrate.exchange_name")
	for _, c := range commands {
		st.PublishToRmq(exchangeName, c.Method, driverName, c.Statement, c.Arguments)
	}
}

// Command -- Command for INSERT/UPDATE/DELETE
type Command struct {
	Statement string
	Method    string
	Arguments []interface{}
}

// NewCommand --
func NewCommand(statement string, arguments ...interface{}) *Command {
	var command = &Command{
		Statement: statement,
		Arguments: arguments,
	}

	command.Method = command.identifyCommand()
	return command
}

// IdentifyCommand - Identify which type of command (INSERT/DELETE/UPDATE)
func (c *Command) identifyCommand() string {
	return strings.ToTitle(strings.Split(strings.TrimSpace(c.Statement), " ")[0])
}

func QueryToCommand(query string, args ...interface{}) *Command {
	return NewCommand(query, args)
}
