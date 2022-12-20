package sqlhooks

import (
	"context"
	sqlDriver "database/sql/driver"
	"errors"
)

type Hook func(ctx context.Context, query string, args ...interface{}) (context.Context, error)
type ErrorHook func(ctx context.Context, err error, query string, args ...interface{}) error

type Hooks interface {
	Before(ctx context.Context, query string, args ...interface{}) (context.Context, error)
	After(ctx context.Context, query string, args ...interface{}) (context.Context, error)
}

type HookErrors interface {
	OnError(ctx context.Context, err error, query string, args ...interface{}) error
}

type Driver struct {
	sqlDriver.Driver
	Hooks
}

func (drv *Driver) Open(name string) (sqlDriver.Conn, error) {
	conn, err := drv.Driver.Open(name)
	if err != nil {
		return conn, err
	}

	wrapped := &Conn{conn, drv.Hooks}
	iE, iQ := isExecer(conn), isQueryer(conn)
	if iE && iQ {
		return &ExecerQueryerContext{wrapped, &ExecerContext{wrapped}, &QueryerContext{wrapped}}, nil
	}
	if iE {
		return &ExecerContext{wrapped}, nil
	}
	if iQ {
		return &QueryerContext{wrapped}, nil
	}
	return wrapped, nil
}

type Conn struct {
	sqlDriver.Conn
	Hooks
}

func (conn *Conn) Prepare(query string) (sqlDriver.Stmt, error) { return conn.Conn.Prepare(query) }
func (conn *Conn) Close() error                                 { return conn.Conn.Close() }
func (conn *Conn) Begin() (sqlDriver.Tx, error)                 { return conn.Conn.Begin() }
func (conn *Conn) BeginTx(ctx context.Context, opts sqlDriver.TxOptions) (sqlDriver.Tx, error) {
	if ciCtx, is := conn.Conn.(sqlDriver.ConnBeginTx); is {
		return ciCtx.BeginTx(ctx, opts)
	}
	return nil, errors.New("[sql] driver does not support non-default isolation level")
}

func (conn *Conn) PrepareContext(ctx context.Context, query string) (sqlDriver.Stmt, error) {
	var (
		stmt sqlDriver.Stmt
		err  error
	)

	if c, ok := conn.Conn.(sqlDriver.ConnPrepareContext); ok {
		stmt, err = c.PrepareContext(ctx, query)
	} else {
		stmt, err = conn.Prepare(query)
	}

	if err != nil {
		return stmt, err
	}

	return &Stmt{stmt, conn.Hooks, query}, nil
}

func (conn *Conn) ExecContext(ctx context.Context, query string, args []sqlDriver.NamedValue) (sqlDriver.Result, error) {
	var err error
	cArgs := namedValueToInterface(args)

	if ctx, err = conn.Hooks.Before(ctx, query, cArgs...); err != nil {
		return nil, err
	}

	result, err := conn.execContext(ctx, query, args)
	if err != nil {
		return result, handlerError(ctx, conn.Hooks, err, query, cArgs...)
	}

	if ctx, err = conn.Hooks.After(ctx, query, cArgs...); err != nil {
		return nil, err
	}

	return result, err
}

func (conn *Conn) execContext(ctx context.Context, query string, args []sqlDriver.NamedValue) (sqlDriver.Result, error) {
	switch c := conn.Conn.(type) {
	case sqlDriver.ExecerContext:
		return c.ExecContext(ctx, query, args)
	case sqlDriver.Execer:
		cArgs, err := namedValueToValue(args)
		if err != nil {
			return nil, err
		}
		return c.Exec(query, cArgs)
	default:
		return nil, errors.New("[sql-hook] ExecContext has created, but something went wrong")
	}
}

type ExecerContext struct{ *Conn }

func (ec *ExecerContext) execContext(ctx context.Context, query string, args []sqlDriver.NamedValue) (sqlDriver.Result, error) {
	switch c := ec.Conn.Conn.(type) {
	case sqlDriver.ExecerContext:
		return c.ExecContext(ctx, query, args)
	case sqlDriver.Execer:
		cArgs, err := namedValueToValue(args)
		if err != nil {
			return nil, err
		}
		return c.Exec(query, cArgs)
	default:
		return nil, errors.New("[sql-hook] ExecContext has created, but something went wrong")
	}
}

func (ec *ExecerContext) ExecContext(ctx context.Context, query string, args []sqlDriver.NamedValue) (sqlDriver.Result, error) {
	var err error
	cArgs := namedValueToInterface(args)

	if ctx, err = ec.Hooks.Before(ctx, query, cArgs...); err != nil {
		return nil, err
	}

	results, err := ec.execContext(ctx, query, args)
	if err != nil {
		return results, handlerError(ctx, ec.Hooks, err, query, cArgs...)
	}

	if ctx, err = ec.Hooks.After(ctx, query, cArgs...); err != nil {
		return nil, err
	}

	return results, err
}

type QueryerContext struct{ *Conn }

func (qc *QueryerContext) queryContext(ctx context.Context, query string, args []sqlDriver.NamedValue) (sqlDriver.Rows, error) {
	switch c := qc.Conn.Conn.(type) {
	case sqlDriver.QueryerContext:
		return c.QueryContext(ctx, query, args)
	case sqlDriver.Queryer:
		cArgs, err := namedValueToValue(args)
		if err != nil {
			return nil, err
		}
		return c.Query(query, cArgs)
	default:
		return nil, errors.New("[sql-hook] QueryContext has created, but something went to wrong")
	}
}

func (qc *QueryerContext) QueryContext(ctx context.Context, query string, args []sqlDriver.NamedValue) (sqlDriver.Rows, error) {
	var err error
	cArgs := namedValueToInterface(args)

	if ctx, err = qc.Hooks.Before(ctx, query, cArgs...); err != nil {
		return nil, err
	}

	results, err := qc.queryContext(ctx, query, args)
	if err != nil {
		return results, handlerError(ctx, qc.Hooks, err, query, cArgs...)
	}

	if ctx, err = qc.Hooks.After(ctx, query, cArgs...); err != nil {
		return nil, err
	}

	return results, err
}

type ExecerQueryerContext struct {
	*Conn
	*ExecerContext
	*QueryerContext
}
type Stmt struct {
	sqlDriver.Stmt
	Hooks
	query string
}

func (stmt *Stmt) Close() error                                          { return stmt.Stmt.Close() }
func (stmt *Stmt) NumInput() int                                         { return stmt.Stmt.NumInput() }
func (stmt *Stmt) Exec(args []sqlDriver.Value) (sqlDriver.Result, error) { return stmt.Stmt.Exec(args) }
func (stmt *Stmt) Query(args []sqlDriver.Value) (sqlDriver.Rows, error)  { return stmt.Stmt.Query(args) }

func (stmt *Stmt) execContext(ctx context.Context, args []sqlDriver.NamedValue) (sqlDriver.Result, error) {
	if s, ok := stmt.Stmt.(sqlDriver.StmtExecContext); ok {
		return s.ExecContext(ctx, args)
	}

	values := make([]sqlDriver.Value, len(args))
	for _, arg := range args {
		values[arg.Ordinal-1] = arg.Value
	}

	return stmt.Exec(values)
}

func (stmt *Stmt) ExecContext(ctx context.Context, args []sqlDriver.NamedValue) (sqlDriver.Result, error) {
	var err error
	cArgs := namedValueToInterface(args)

	if ctx, err = stmt.Hooks.Before(ctx, stmt.query, cArgs...); err != nil {
		return nil, err
	}

	results, err := stmt.execContext(ctx, args)

	if err != nil {
		return results, handlerError(ctx, stmt.Hooks, err, stmt.query, cArgs...)
	}

	if ctx, err = stmt.Hooks.After(ctx, stmt.query, cArgs...); err != nil {
		return nil, err
	}

	return results, err
}

func (stmt *Stmt) queryContext(ctx context.Context, args []sqlDriver.NamedValue) (sqlDriver.Rows, error) {
	if s, ok := stmt.Stmt.(sqlDriver.StmtQueryContext); ok {
		return s.QueryContext(ctx, args)
	}

	values := make([]sqlDriver.Value, len(args))
	for _, arg := range args {
		values[arg.Ordinal-1] = arg.Value
	}
	return stmt.Query(values)
}

func (stmt *Stmt) QueryContext(ctx context.Context, args []sqlDriver.NamedValue) (sqlDriver.Rows, error) {
	var err error
	cArgs := namedValueToInterface(args)

	if ctx, err = stmt.Hooks.Before(ctx, stmt.query, cArgs...); err != nil {
		return nil, err
	}

	rows, err := stmt.queryContext(ctx, args)
	if err != nil {
		return rows, handlerError(ctx, stmt.Hooks, err, stmt.query, cArgs...)
	}

	if ctx, err = stmt.Hooks.After(ctx, stmt.query, cArgs...); err != nil {
		return nil, err
	}

	return rows, err
}

func Wrap(drv sqlDriver.Driver, hooks Hooks) sqlDriver.Driver {
	return &Driver{drv, hooks}
}

func isExecer(conn sqlDriver.Conn) bool {
	switch conn.(type) {
	case sqlDriver.ExecerContext:
		return true
	case sqlDriver.Execer:
		return true
	default:
		return false
	}
}

func isQueryer(conn sqlDriver.Conn) bool {
	switch conn.(type) {
	case sqlDriver.QueryerContext:
		return true
	case sqlDriver.Queryer:
		return true
	default:
		return false
	}
}

func handlerError(ctx context.Context, hooks Hooks, err error, query string, args ...interface{}) error {
	h, ok := hooks.(HookErrors)
	if !ok {
		return err
	}

	if err := h.OnError(ctx, err, query, args...); err != nil {
		return err
	}

	return err
}

func namedValueToValue(named []sqlDriver.NamedValue) ([]sqlDriver.Value, error) {
	args := make([]sqlDriver.Value, len(named))
	for i, n := range named {
		if len(n.Name) > 0 {
			return nil, errors.New("[sql]: driver does not support the use of Named Parameter")
		}
		args[i] = n.Value
	}
	return args, nil
}

func namedValueToInterface(named []sqlDriver.NamedValue) []interface{} {
	itf := make([]interface{}, len(named))
	for i, n := range named {
		itf[i] = n
	}
	return itf
}
