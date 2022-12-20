package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/punky97/go-codebase/core/bkgrpc"
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"encoding/gob"
	"encoding/json"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const DefaultLimit = 20

var (
	defaultReplacer = []string{
		`&`, "u0026",
		`<`, "u003c",
		`>`, "u003e",
	}

	mysqlReplacer = []string{}
	pgsqlReplacer = []string{
		"\u0000", "",
	}
)

// Table -- define table information: table name, columns
type Table struct {
	Name                      string
	IgnoreColumns             []string
	DateTimeColumns           []string
	AutoUpdateDateTimeColumns []string
	AutoCreateDateTimeColumns []string
	ForeignKeys               []string
	NotNullColumns            []string
	SelectedColumns           []string
	DefaultColumns            []string
	ConvertJSONKey2Column     map[string]string
	AIColumns                 string
	AIColumnsLimitValue       int64
	ForceAIRemoveColumns      bool

	// xss
	XssIgnoreColumns []string

	ColumnCustomAction map[string]ColumnFunction
}

// SQLTool --
type SQLTool struct {
	db                  *sql.DB
	tx                  *sql.Tx
	table               Table
	columns             []string
	column2FieldName    map[string]string
	column2Kind         map[string]reflect.Kind
	column2Type         map[string]reflect.Type
	includeColumn       map[string]bool
	ignoreColumn        map[string]bool
	values              []interface{}
	proto               interface{}
	rabbitMQProducer    *queue.Producer
	ctx                 context.Context
	defaultLimit        int
	defaultOffset       int
	selectAllField      bool
	onlyUseDefaultField bool
	useTransaction      bool
	ignoreAIColumn      bool
	actionType          string
	aiColumnReachLimit  bool
	MaxArgsQuery        int

	// xss
	xssEnabled       bool
	xssReplacer      *strings.Replacer
	xssIgnoreColumns map[string]bool

	// sql replacer
	mysqlReplacer *strings.Replacer
	pgsqlReplacer *strings.Replacer
}

type ColumnFunction func(interface{}) interface{}

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionSelect = "select"
	ActionDelete = "delete"
)

func (st *SQLTool) initOption(isUpdate bool) {
	st.ignoreAIColumn = false
	st.includeColumn = make(map[string]bool)
	st.ignoreColumn = make(map[string]bool)

	// xss init
	st.xssEnabled = viper.GetBool("security.xss.enabled")
	st.xssIgnoreColumns = make(map[string]bool)

	if st.xssEnabled {
		replacer := viper.GetStringSlice("security.xss.replacer")
		if len(replacer) < 1 {
			replacer = defaultReplacer
		}

		st.xssReplacer = strings.NewReplacer(replacer...)

		if len(st.table.XssIgnoreColumns) > 0 {
			for _, col := range st.table.XssIgnoreColumns {
				st.xssIgnoreColumns[col] = true
			}
		}
	}

	if viper.GetBool("security.mysql.enabled") {
		mysqlRepl := viper.GetStringSlice("security.mysql.replacer")
		if len(mysqlRepl) < 1 {
			mysqlRepl = mysqlReplacer
		}
		st.mysqlReplacer = strings.NewReplacer(mysqlRepl...)
	}

	if viper.GetBool("security.pgsql.enabled") {
		pgsqlRepl := viper.GetStringSlice("security.pgsql.replacer")
		if len(pgsqlRepl) < 1 {
			pgsqlRepl = pgsqlReplacer
		}
		st.pgsqlReplacer = strings.NewReplacer(pgsqlRepl...)
	}
	st.MaxArgsQuery = viper.GetInt("dms.max_args_query")

	var isChecked bool
	if len(st.table.IgnoreColumns) > 0 {
		for i := range st.table.IgnoreColumns {
			st.ignoreColumn[st.table.IgnoreColumns[i]] = true
		}
		isChecked = true
	}
	if len(st.table.SelectedColumns) > 0 {
		for i := range st.table.SelectedColumns {
			st.includeColumn[st.table.SelectedColumns[i]] = true
		}
		isChecked = true
	}
	if st.table.AIColumnsLimitValue <= 0 {
		st.table.AIColumnsLimitValue = 1000000000000000
	}
	if st.ctx == nil {
		return
	}
	md, ok := metadata.FromIncomingContext(st.ctx)
	if !ok {
		return
	}
	if !st.onlyUseDefaultField {
		if !st.selectAllField {
			if fIgnores := md.Get(utils.UseAllWithoutFieldHeader); len(fIgnores) > 0 {
				isChecked = true
				if st.ignoreColumn == nil {
					st.ignoreColumn = make(map[string]bool)
				}
				for _, cl := range fIgnores {
					st.ignoreColumn[cl] = true
				}
			} else if fSelects := md.Get(utils.FieldSelectHeader); len(fSelects) > 0 {
				isChecked = true
				st.includeColumn = make(map[string]bool)
				for _, cl := range fSelects {
					st.includeColumn[cl] = true
				}
			}
		}
	}

	if !isUpdate {
		if limitRecord := md.Get(utils.LimitHeader); len(limitRecord) > 0 && cast.ToInt(limitRecord[0]) > 0 {
			st.defaultLimit = cast.ToInt(limitRecord[0])
		} else if defaultLimit := md.Get(utils.DefaultLimitHeader); len(defaultLimit) > 0 && cast.ToBool(defaultLimit[0]) {
			if limit := viper.GetInt("limit_record.number"); limit > 0 {
				st.defaultLimit = limit
			} else {
				st.defaultLimit = DefaultLimit
			}
		}
		if offsetStr := md.Get(utils.OffsetHeader); len(offsetStr) > 0 && cast.ToInt(offsetStr[0]) > 0 {
			st.defaultOffset = cast.ToInt(offsetStr[0])
		}
	}

	// If field and ignore field was defined by params or metadata of request. Do not use default
	if isChecked {
		return
	}
	if useAllFieldStr := md.Get(utils.UseAllFieldHeader); st.selectAllField || (len(useAllFieldStr) > 0 && cast.ToBool(useAllFieldStr[0])) {
		return
	}
	if limitFieldStr := md.Get(utils.DefaultSelectHeader); st.onlyUseDefaultField || (len(limitFieldStr) > 0 && cast.ToBool(limitFieldStr[0])) {
		for _, cl := range st.table.DefaultColumns {
			st.includeColumn[cl] = true
		}
	}
}
func (st *SQLTool) initTool() {
	st.initOption(false)
}

// StreamExecute - execute func of stream data
// If this func return error != nil
// -> stream will be stop
type StreamExecute func(data interface{}, err error) error

// GetSQLTool --
func GetSQLTool(db *sql.DB, table Table) (st SQLTool) {
	st.db = db
	st.table = table
	return
}

// UseTransaction -- Enable use transaction for CUD (Create/Update/Delete)
func (st *SQLTool) UseTransaction() {
	st.useTransaction = true
}

// NewSelectQuery -- create new select query
func NewSelectQuery(table Table, protoObj interface{}) (st SQLTool) {
	st.table = table
	st.parseColumns(protoObj)
	return
}

// NewSelectQuery -- create new select query
func NewTransactions(ctx context.Context, db *sql.DB) (st SQLTool) {
	st.ctx = ctx
	st.db = db

	tx, err := st.db.Begin()
	if err != nil {
		return
	}

	st.tx = tx
	return
}

// NewSelect -- prepare select query
func NewSelect(ctx context.Context, db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.table = table
	st.actionType = ActionSelect
	st.initTool()
	st.parseColumns(protoObj)
	return
}

// NewSelectAll -- prepare select query
func NewSelectAll(ctx context.Context, db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.table = table
	st.selectAllField = true
	st.actionType = ActionSelect
	st.initTool()
	st.parseColumns(protoObj)
	return
}

// NewSelectDefault -- prepare select query
func NewSelectDefault(ctx context.Context, db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.table = table
	st.onlyUseDefaultField = true
	st.actionType = ActionSelect
	st.initTool()
	st.parseColumns(protoObj)
	return
}

// NewInsert -- prepare insert query
func NewInsert(ctx context.Context, db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.table = table
	st.actionType = ActionCreate
	st.initOption(true)
	st.parseColumns(protoObj)
	st.fillValues(protoObj, false)
	st.proto = protoObj
	return
}

// PrepareUpdate -- prepare update query
func NewUpdate(ctx context.Context, db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.table = table
	st.actionType = ActionUpdate
	st.initOption(true)
	st.parseUpdateColumns(protoObj, nil)
	st.fillValues(protoObj, true)
	return
}

// NewDelete -- prepare delete query
func NewDelete(ctx context.Context, db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.table = table
	st.actionType = ActionDelete
	st.initTool()
	st.parseColumns(protoObj)
	return
}

// PrepareSelect -- prepare select query
// Deprecated: Please use NewSelect func to prepare select instead
func PrepareSelect(db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.db = db
	st.table = table
	st.actionType = ActionSelect
	st.initTool()
	st.parseColumns(protoObj)
	return
}

// PrepareSelectCustomColumn -- prepare select query
// Deprecated:: Please use NewSelect func to prepare select instead
func PrepareSelectCustomColumn(db *sql.DB, table Table, protoObj interface{}, column []string) (st SQLTool, col string) {
	st.db = db
	st.table = table
	st.actionType = ActionSelect
	st.initTool()
	col = st.parseCustomColumns(protoObj, column)
	return
}

// PrepareInsert -- prepare insert query
// Deprecated: : Please use NewInsert func to prepare insert instead
func PrepareInsert(db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.db = db
	st.table = table
	st.actionType = ActionCreate
	st.initTool()
	st.parseColumns(protoObj)
	st.fillValues(protoObj, false)
	st.proto = protoObj
	return
}

// PrepareUpdate -- prepare update query
// Deprecated: : Please use NewUpdate func to prepare select instead
func PrepareUpdate(db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.db = db
	st.table = table
	st.actionType = ActionUpdate
	st.initTool()
	st.parseUpdateColumns(protoObj, nil)
	st.fillValues(protoObj, true)
	return
}

// PrepareDelete -- prepare delete query
// Deprecated: Please use NewDelete func instead
func PrepareDelete(db *sql.DB, table Table, protoObj interface{}) (st SQLTool) {
	st.db = db
	st.table = table
	st.actionType = ActionDelete
	st.initTool()
	st.parseColumns(protoObj)
	return
}

// NewUpdateWithColumns -- prepare update query with update columns
func NewUpdateWithColumns(ctx context.Context, db *sql.DB, table Table, protoObj interface{}, columns []string) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	st.actionType = ActionUpdate
	st.PrepareUpdateWithColumns(table, protoObj, columns)
	return
}

// NewTool -- generic mysql tool for raw query/exec
func NewTool(ctx context.Context, db *sql.DB) (st SQLTool) {
	st.ctx = ctx
	st.db = db
	return
}

func (st *SQLTool) Context() context.Context {
	if st.ctx != nil {
		return st.ctx
	}

	return context.Background()
}

// SetDB -- set sql db
func (st *SQLTool) SetDB(db *sql.DB) {
	st.db = db
}

// InitRmq --
func (st *SQLTool) InitRmq() *queue.Producer {
	p := queue.NewRMQProducerFromDefConf()
	if p == nil {
		//logger.BkLog.Info("nil producer")
	} else {
		if err := p.Connect(); err != nil {
			logger.BkLog.Error("Cannot connect to rabbitmq. Please check configuration file for more information", err)
		}
		p.Start()
		utils.HandleSigterm(func() {
			if p != nil {
				logger.BkLog.Debug("Stop producer")
				p.Close()
			}
		})
	}
	return p
}

// InitRedis --
func (st *SQLTool) InitRedis(ctx context.Context) (*bkredis.RedisClient, error) {
	redisConn, err := bkredis.NewConnection(ctx, bkredis.DefaultRedisConnectionFromConfig())
	return redisConn, err
}

// PublishToRmq --
func (st *SQLTool) PublishToRmq(exchangeName, method, db, query string, args []interface{}) {
	p := st.InitRmq()
	if p == nil {
		logger.BkLog.Info("nil producer")
		return
	}
	//p.Start()
	//defer p.Close()
	rand.Seed(time.Now().Unix())
	payload := MigrateDataInput{}
	payload.ID = fmt.Sprintf("%v_%v", rand.Int63(), time.Now().Unix())
	payload.Method = method
	payload.Table = st.table
	payload.Query = query
	payload.Values = args
	payload.Columns = st.columns
	gob.Register(time.Time{})
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(payload)
	if err != nil {
		logger.BkLog.Warnf("Encoding failed %v, err %v", string(b.Bytes()), err)
	}
	err = p.PublishRouting(exchangeName, db, b.Bytes())
	if err != nil {
		logger.BkLog.Warnf("fail to publish routing, table %v, query %v, err %v", st.GetTableName(), query, err)
	}
}

// MigrateDataInput --
type MigrateDataInput struct {
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Db      string        `json:"db"`
	Table   Table         `json:"table"`
	Query   string        `json:"query"`
	Values  []interface{} `json:"values"`
	Columns []string      `json:"columns"`
	Retry   int           `json:"retry"`
}

// Insert -- do insert
func (st *SQLTool) Insert(query string, args ...interface{}) (result sql.Result, err error) {
	// sqli protect
	query, args = st.ignoreValue(query, args...)
	err = ScanSQLInjection(st.ctx, query, len(args))
	if err != nil {
		return
	}

	// limit auto incr
	if st.aiColumnReachLimit {
		logger.BkLog.Infow("[auto_incr] Query blocked",
			"query", query,
			"source_pod_name", meta.GetDMSSourcePodName(st.ctx),
			"source_ip", meta.GetDMSSourceIP(st.ctx),
		)
		err = bkgrpc.ErrAIColReachLimit
		return
	}

	if st.ctx != nil {
		result, err = st.execContext(st.ctx, query, args...)
	} else {
		result, err = st.exec(query, args...)
	}
	if err != nil {
		return
	}

	if viper.GetBool("mysql_migrate.enable") {
		exchName := viper.GetString("mysql_migrate.exchange_name")
		st.PublishToRmq(exchName, "Insert", "mysql", query, args)
	}
	return
}

// Update -- do update
func (st *SQLTool) Update(query string, args ...interface{}) (result sql.Result, err error) {
	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "UPDATE", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end
	// sqli protect
	query, args = st.ignoreValue(query, args...)
	err = ScanSQLInjection(st.ctx, query, len(args))
	if err != nil {
		return
	}

	// limit auto incr
	if st.aiColumnReachLimit {
		logger.BkLog.Infow("[auto_incr] Query blocked",
			"query", query,
			"source_pod_name", meta.GetDMSSourcePodName(st.ctx),
			"source_ip", meta.GetDMSSourceIP(st.ctx),
		)
		err = bkgrpc.ErrAIColReachLimit
		return
	}

	if st.ctx != nil {
		result, err = st.execContext(st.ctx, query, args...)
	} else {
		result, err = st.exec(query, args...)
	}
	if err != nil {
		return
	}

	if viper.GetBool("mysql_migrate.enable") {
		exchName := viper.GetString("mysql_migrate.exchange_name")
		st.PublishToRmq(exchName, "Update", "mysql", query, args)
	}
	return
}

// Exec -- exec raw query
func (st *SQLTool) Exec(query string, args ...interface{}) (result sql.Result, err error) {
	// sqli protect
	query, args = st.ignoreValue(query, args...)
	err = ScanSQLInjection(st.ctx, query, len(args))
	if err != nil {
		return
	}

	// limit auto incr
	if st.aiColumnReachLimit {
		logger.BkLog.Infow("[auto_incr] Query blocked",
			"query", query,
			"source_pod_name", meta.GetDMSSourcePodName(st.ctx),
			"source_ip", meta.GetDMSSourceIP(st.ctx),
		)
		err = bkgrpc.ErrAIColReachLimit
		return
	}

	if st.ctx != nil {
		result, err = st.execContext(st.ctx, query, args...)
	} else {
		result, err = st.exec(query, args...)
	}
	if err != nil {
		return
	}

	if viper.GetBool("mysql_migrate.enable") {
		exchName := viper.GetString("mysql_migrate.exchange_name")
		st.PublishToRmq(exchName, "Exec", "mysql", query, args)
	}
	return
}

// ExecStoredProcedure -- exec stored procedure
func (st *SQLTool) ExecStoredProcedure(procedureName string, args ...interface{}) (result sql.Result, err error) {
	paramQuestions := make([]string, len(args))
	for idx := range args {
		paramQuestions[idx] = "?"
	}
	query := fmt.Sprintf("CALL %v(%v)", procedureName, strings.Join(paramQuestions, ", "))
	return st.Exec(query, args...)
}

// UpdateMany --
func (st *SQLTool) UpdateMany(query string, listArgs [][]interface{}) (results []sql.Result, errs []error) {
	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "UPDATE", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end
	if st.ctx != nil {
		results, errs = st.execManyContext(st.ctx, query, listArgs)
	} else {
		results, errs = st.execMany(query, listArgs)
	}
	if errs != nil {
		return
	}

	if viper.GetBool("mysql_migrate.enable") {
		exchName := viper.GetString("mysql_migrate.exchange_name")
		for _, args := range listArgs {
			st.PublishToRmq(exchName, "Update", "mysql", query, args)
		}
	}
	return
}

// Delete -- do delete
func (st *SQLTool) Delete(query string, args ...interface{}) (result sql.Result, err error) {
	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "DELETE", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end
	// sqli protect
	query, args = st.ignoreValue(query, args...)
	err = ScanSQLInjection(st.ctx, query, len(args))
	if err != nil {
		return
	}

	if st.ctx != nil {
		result, err = st.execContext(st.ctx, query, args...)
	} else {
		result, err = st.exec(query, args...)
	}
	if err != nil {
		return
	}

	if viper.GetBool("mysql_migrate.enable") {
		exchName := viper.GetString("mysql_migrate.exchange_name")
		st.PublishToRmq(exchName, "Delete", "mysql", query, args)
	}
	return
}

// PrepareSelect -- set table and parse columns for query
func (st *SQLTool) PrepareSelect(table Table, protoObj interface{}) {
	st.table = table
	st.initTool()
	st.parseColumns(protoObj)
}

// PrepareInsert -- set table and parse columns for insert query
func (st *SQLTool) PrepareInsert(table Table, protoObj interface{}) {
	st.table = table
	st.initOption(true)
	st.parseColumns(protoObj)
	st.fillValues(protoObj, false)
	st.proto = protoObj
}

// PrepareInsert -- set table and parse columns for insert query
func (st *SQLTool) PrepareUpdate(table Table, protoObj interface{}) {
	st.table = table
	st.initOption(true)
	st.parseUpdateColumns(protoObj, []string{})
	st.fillValues(protoObj, true)
	st.proto = protoObj
}

func (st *SQLTool) PrepareUpdateWithColumns(table Table, protoObj interface{}, columns []string) {
	st.table = table
	st.initOption(true)
	st.parseUpdateColumns(protoObj, columns)
	st.fillValues(protoObj, true)
	st.proto = protoObj
}

// PrepareInsert -- set table and parse columns for insert query
func (st *SQLTool) PrepareDelete(table Table, protoObj interface{}) {
	st.table = table
	st.initTool()
	st.parseColumns(protoObj)
}

func (st *SQLTool) ExecTransQuery(query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := st.tx.Prepare(query)
	if err != nil {
		return
	}

	defer stmt.Close()
	result, err = stmt.Exec(args...)
	return
}

func (st *SQLTool) SelectTransQuery(dest interface{}, query string, args ...interface{}) error {
	stmt, err := st.tx.Prepare(query)
	if err != nil {
		return err
	}

	query = st.addDefaultLimit(query)

	var rows *sql.Rows
	rows, err = st.selectQueryWithStmtTx(stmt, query, args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	var vp reflect.Value

	value := reflect.ValueOf(dest)

	// json.Unmarshal returns errors for these
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(value)

	slice, err := baseType(value.Type(), reflect.Slice)
	if err != nil {
		return err
	}

	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := Deref(slice.Elem())

	//empty := true

	for rows.Next() {
		vp = reflect.New(base)
		err = st.Scan(rows, vp.Interface())
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error while scan, details: %v", err))
			continue
		}
		//empty = false
		// append
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
	}

	return err
}

func (st *SQLTool) SelectQueryTransQuery(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := st.tx.Prepare(query)
	if err != nil {
		return nil, err
	}

	return st.selectQueryWithStmtTx(stmt, query, args...)
}

func (st *SQLTool) selectQueryWithStmtTx(stmt *sql.Stmt, query string, args ...interface{}) (rows *sql.Rows, err error) {
	if st.ctx != nil {
		rows, err = stmt.QueryContext(st.ctx, args...)
	} else {
		rows, err = stmt.Query(args...)
	}

	//defer stmt.Close() // Prepared statements take up server resources and should be closed after use.

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (st *SQLTool) GetTransQuery(dest interface{}, query string, args ...interface{}) error {
	stmt, err := st.tx.Prepare(query)
	if err != nil {
		return err
	}

	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "SELECT", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end

	var rows *sql.Rows
	rows, err = st.selectQueryWithStmtTx(stmt, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	var vp reflect.Value

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if v.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(v)

	base := Deref(v.Type())

	if rows.Next() {
		vp = reflect.New(base)
		err = st.Scan(rows, vp.Interface())
		if err == nil {
			direct.Set(vp.Elem())
		}
	} else {
		err = sql.ErrNoRows
	}

	return err
}

func (st *SQLTool) RollbackTransactions() error {
	return st.tx.Rollback()
}

func (st *SQLTool) CommitTransactions() error {
	return st.tx.Commit()
}

// SelectQuery --
// Deprecated: Please use function Select or Get or customize function with selectQuery
func (st *SQLTool) SelectQuery(query string, args ...interface{}) (rows *sql.Rows, err error) {
	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "SELECT", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end

	if st.ctx != nil {
		rows, err = st.queryContext(st.ctx, query, args...)
	} else {
		rows, err = st.query(query, args...)
	}
	return
}

// selectQuery --
func (st *SQLTool) selectQuery(query string, args ...interface{}) (rows *sql.Rows, err error) {
	// sqli protect
	err = ScanSQLInjection(st.ctx, query, len(args))
	if err != nil {
		return
	}

	if st.ctx != nil {
		rows, err = st.queryContext(st.ctx, query, args...)
	} else {
		rows, err = st.query(query, args...)
	}
	return
}

func (st *SQLTool) selectStreamOption(destType reflect.Type, query string,
	size int64, ctx context.Context, breakChan chan bool, args ...interface{}) (
	chDestO chan interface{}, chErrorO chan error, err error) {
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "SELECT", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end
	rows, err := st.selectQuery(query, args...)
	if err != nil {
		logger.BkLog.Infof("Error to select query: %v", err)
		if rows != nil {
			rows.Close()
		}
		return
	}
	chDestO = make(chan interface{}, 5)
	chErrorO = make(chan error, 5)
	go func(chDest chan interface{}, errorChan chan error) {
		var err error
		defer func() {
			rows.Close()
			chDest <- nil
			logger.BkLog.Debugf("defer run func")
		}()
		var isSlice = destType.Kind() == reflect.Slice
		if isSlice {
			destType = destType.Elem()
		}
		var slice = reflect.MakeSlice(reflect.SliceOf(destType), 0, 0)
		isPtr := destType.Kind() == reflect.Ptr
		base := Deref(destType)
		var index = 1
		var run = 1
		for rows.Next() {
			logger.BkLog.Debugf("index: %v, run: %v", index, run)
			index++
			select {
			case <-ctx.Done():
				logger.BkLog.Debugf("context done")
				return
			case <-breakChan:
				logger.BkLog.Debugf("force close stream")
				return
			default:
				run++
				var vp = reflect.New(base)
				err = st.Scan(rows, vp.Interface())
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Error while scan, details: %v", err))
					errorChan <- err
				}
				if !isSlice {
					if isPtr {
						chDest <- vp.Interface()
					} else {
						chDest <- reflect.Indirect(vp).Interface()
					}
				} else {
					if isPtr {
						slice = reflect.Append(slice, vp)
					} else {
						slice = reflect.Append(slice, reflect.Indirect(vp))
					}
				}
				if slice.Len() >= int(size) {
					chDest <- slice.Interface()
					slice = reflect.MakeSlice(reflect.SliceOf(destType), 0, 0)
				}
			}
		}
		logger.BkLog.Debugf("End for rows.Next()")
		if slice.Len() > 0 {
			chDest <- slice.Interface()
		}
	}(chDestO, chErrorO)
	return
}

// StreamOne - select data from mysql with stream process
// args:
//   - dest: dest type you want to response
//   - query: sql query
//   - ctx: context of stream. when ctx done then close connection and channel
//   - execute: func execute data return of stream. If this func return error != nil, stream will be close
//   - args: .....
func (st *SQLTool) StreamOne(dest interface{}, query string,
	ctx context.Context, execute StreamExecute, args ...interface{}) error {
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Struct && Deref(destType).Kind() != reflect.Struct {
		return errors.New("dest must be struct or ptr of struct")
	}
	breakChan := make(chan bool, 1)
	dataChan, errChan, err := st.selectStreamOption(destType, query, 1, ctx, breakChan, args...)
	if err != nil {
		return err
	}
	defer func() {
		close(dataChan)
		close(errChan)
	}()
loop:
	for {
		select {
		case revChan := <-dataChan:
			if revChan == nil {
				break loop
			}
			err := execute(revChan, nil)
			if err != nil {
				breakChan <- true
			}
		case err := <-errChan:
			errProcess := execute(nil, err)
			if errProcess != nil {
				breakChan <- true
			}
		}
	}
	return nil
}

// StreamSlice - select data from mysql with stream process
// args:
//   - dest: dest type you want to response
//   - query: sql query
//   - size: size of slice you want to rev
//   - ctx: context of stream. when ctx done then close connection and channel
//   - execute: func execute data return of stream. If this func return error != nil, stream will be close
//   - args: .....
func (st *SQLTool) StreamSlice(dest interface{}, query string, size int64,
	ctx context.Context, execute StreamExecute, args ...interface{}) error {
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Slice && Deref(destType.Elem()).Kind() != reflect.Struct {
		return errors.New("dest must be slice of struct or slice ptr of struct")
	}
	breakChan := make(chan bool, 1)
	dataChan, errChan, err := st.selectStreamOption(destType, query, size, ctx, breakChan, args...)
	if err != nil {
		logger.BkLog.Infof("Error to select stream %v", err)
		return err
	}
	defer func() {
		close(dataChan)
		close(errChan)
	}()
loop:
	for {
		select {
		case revChan := <-dataChan:
			logger.BkLog.Debugf("rev chan")
			if revChan == nil {
				break loop
			}
			err := execute(revChan, nil)
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("error run: %v", err))
				breakChan <- true
			}
		case err := <-errChan:
			errProcess := execute(nil, err)
			if errProcess != nil {
				logger.BkLog.Errorw(fmt.Sprintf("error run: %v", err))
				breakChan <- true
			}
		}
	}
	return nil
}

func (st *SQLTool) addDefaultLimit(query string) string {
	if st.defaultLimit > 0 && !strings.Contains(strings.ToLower(query), " limit ") {
		query = fmt.Sprintf("%s limit %d", query, st.defaultLimit)
	}
	if st.defaultOffset > 0 && !strings.Contains(strings.ToLower(query), " offset ") {
		query = fmt.Sprintf("%s offset %d", query, st.defaultOffset)
	}
	return query
}

// Select -- do select, same as get but return empty list in case of not found instead of error
func (st *SQLTool) Select(dest interface{}, query string, args ...interface{}) error {
	if st.MaxArgsQuery > 0 && len(args) > st.MaxArgsQuery {
		return fmt.Errorf("too many args input. len_args: %d, max: %d", len(args), st.MaxArgsQuery)
	}
	query = st.addDefaultLimit(query)
	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "SELECT", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end

	rows, err := st.selectQuery(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var vp reflect.Value

	value := reflect.ValueOf(dest)

	// json.Unmarshal returns errors for these
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(value)

	slice, err := baseType(value.Type(), reflect.Slice)
	if err != nil {
		return err
	}

	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := Deref(slice.Elem())

	//empty := true

	for rows.Next() {
		vp = reflect.New(base)
		err = st.Scan(rows, vp.Interface())
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error while scan, details: %v", err))
			continue
		}
		//empty = false
		// append
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
	}

	return err
}

// reflect helpers
func baseType(t reflect.Type, expected reflect.Kind) (reflect.Type, error) {
	t = Deref(t)
	if t.Kind() != expected {
		return nil, fmt.Errorf("expected %s but got %s", expected, t.Kind())
	}
	return t, nil
}

// Deref is Indirect for reflect.Types
func Deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

var _scannerInterface = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

// isScannable takes the reflect.Type and the actual dest value and returns
// whether or not it's Scannable.  Something is scannable if:
//   - it is not a struct
//   - it implements sql.Scanner
//   - it has no exported fields
func isScannable(t reflect.Type) bool {
	if reflect.PtrTo(t).Implements(_scannerInterface) {
		return true
	}
	if t.Kind() != reflect.Struct {
		return true
	}

	return false
}

// Get -- do select on
func (st *SQLTool) Get(dest interface{}, query string, args ...interface{}) (err error) {

	//start newrelic segment
	/*	nrm := tracing.NewNonWebTransactionMonitor(query)
		nrm.StartDataStoreSegment(query, "SELECT", newrelic.DatastoreMySQL)
		defer nrm.EndDataStoreSegment()
		defer nrm.End()*/
	// segment end

	rows, err := st.selectQuery(query, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	var vp reflect.Value

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if v.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	direct := reflect.Indirect(v)

	base := Deref(v.Type())
	// scannable := isScannable(base)
	// log.Print("base: ", base, " scannable: ", scannable, " direct canSet: ", direct.CanSet())

	if rows.Next() {
		vp = reflect.New(base)
		err = st.Scan(rows, vp.Interface())
		if err == nil {
			direct.Set(vp.Elem())
		}
	} else {
		err = sql.ErrNoRows
	}

	return
}

// GetTableName -- return table name
func (st *SQLTool) GetTableName() string {
	return st.table.Name
}

// GetQueryColumnList -- use for SELECT, INSERT query
func (st *SQLTool) GetQueryColumnList() string {
	if st.ignoreAIColumn {
		var newCol = make([]string, 0)
		for _, col := range st.columns {
			if col != st.table.AIColumns {
				newCol = append(newCol, col)
			}
		}
		return strings.Join(newCol, ",")
	}
	return strings.Join(st.columns, ",")
}

// GetQueryColumnListWithPrefix -- use for SELECT, INSERT query
func (st *SQLTool) GetQueryColumnListWithPrefix(prefix string) string {
	columnsWithPrefix := []string{}
	for _, c := range st.columns {
		columnsWithPrefix = append(columnsWithPrefix, prefix+"."+c)
	}
	return strings.Join(columnsWithPrefix, ",")
}

// GetCreatePostGresQueryColumnList -- use for SELECT, INSERT query
func (st *SQLTool) GetCreatePostGresQueryColumnList() string {
	return st.GetQueryColumnList()
}

// GetQueryValueList -- get column should ignore AIColumns
func (st *SQLTool) getColumnValueList() []string {
	return st.columns
}

// GetQueryValueList -- use for INSERT query
func (st *SQLTool) GetQueryValueList() string {
	questionMarks := make([]string, 0)
	newCol := st.getColumnValueList()
	for index := 0; index < len(newCol); index++ {
		questionMarks = append(questionMarks, "?")
	}
	return strings.Join(questionMarks, ",")
}

// GetPostGresQueryValueList -- use for INSERT query
func (st *SQLTool) GetPostGresQueryValueList() string {
	questionMarks := make([]string, 0)
	for index := 0; index < len(st.columns); index++ {
		questionMarks = append(questionMarks, fmt.Sprintf("$%v", index+1))
	}
	return strings.Join(questionMarks, ",")
}

// GetCreatePostGresQueryValueList -- use for INSERT query
func (st *SQLTool) GetCreatePostGresQueryValueList() string {
	questionMarks := make([]string, 0)
	newCol := st.getColumnValueList()
	for index := 0; index < len(newCol); index++ {
		questionMarks = append(questionMarks, fmt.Sprintf("$%v", index+1))
	}
	return strings.Join(questionMarks, ",")
}

// GetQueryAssignmentList -- use for UPDATE query
func (st *SQLTool) GetQueryAssignmentList() string {
	p := make([]string, 0)

	for _, column := range st.columns {
		temp := column + " = ?"
		p = append(p, temp)
	}

	return strings.Join(p, ",")
}

// GetQueryAssignmentList -- use for UPDATE query
func (st *SQLTool) GetPostGresQueryAssignmentList() string {
	p := make([]string, 0)
	var decrese = 0
	for index, column := range st.columns {
		temp := column + fmt.Sprintf(" = $%d", index+1-decrese)
		p = append(p, temp)
	}

	return strings.Join(p, ",")
}

func (st *SQLTool) GetNumColumn() int {
	return len(st.columns)
}

// GetQueryCustomAssignmentList -- use for UPDATE query
func (st *SQLTool) GetQueryCustomAssignmentList(column []string) string {
	p := make([]string, 0)

	for _, column := range column {
		temp := column + " = ?"
		p = append(p, temp)
	}

	return strings.Join(p, ",")
}

// parse columns
func (st *SQLTool) parseColumns(i interface{}) {
	st.columns = make([]string, 0)
	st.column2FieldName = make(map[string]string, 0)
	st.column2Kind = make(map[string]reflect.Kind, 0)
	st.column2Type = make(map[string]reflect.Type, 0)

	t := reflect.TypeOf(i).Elem()
	for index := 0; index < t.NumField(); index++ {
		f := t.Field(index)
		name := f.Name
		typeV := f.Type
		kind := f.Type.Kind()
		jsonKey := getKeyFromJSONTag(f.Tag.Get("json"))
		if jsonKey == "-" || utils.IsIgnoreThisField(name) {
			continue
		}
		st.addColumnFieldNameAndKind(jsonKey, name, kind, typeV)
	}
	return
}

// parse columns for update query
func (st *SQLTool) parseUpdateColumns(i interface{}, allowFields []string) {
	// merge with updated datetime columns
	if len(allowFields) > 0 {
		for _, v := range st.table.AutoUpdateDateTimeColumns {
			allowFields = append(allowFields, v)
		}
	}
	st.columns = make([]string, 0)
	st.column2FieldName = make(map[string]string, 0)
	st.column2Kind = make(map[string]reflect.Kind, 0)
	st.column2Type = make(map[string]reflect.Type, 0)
	updatedFieldsBitMask, isDefault := utils.GetUpdatedFieldsBitMask(reflect.ValueOf(i).Elem())
	if isDefault {
		logger.BkLog.Debug("Use default updated field bitmask with all value is true => update all field")
	}
	t := reflect.TypeOf(i).Elem()
	for index := 0; index < t.NumField(); index++ {
		f := t.Field(index)
		name := f.Name
		typeV := f.Type
		kind := f.Type.Kind()
		jsonKey := getKeyFromJSONTag(f.Tag.Get("json"))
		if !updatedFieldsBitMask[index] || jsonKey == "-" || utils.IsIgnoreThisField(name) ||
			(len(allowFields) > 0 && !utils.SliceContain(allowFields, jsonKey)) {
			continue
		}
		st.addColumnFieldNameAndKind(jsonKey, name, kind, typeV)
	}
	return
}

// Use to add column, FieldName, Kind to SQLTool
func (st *SQLTool) addColumnFieldNameAndKind(jsonKey string, name string, kind reflect.Kind, typeV reflect.Type) {
	column, ok := st.table.ConvertJSONKey2Column[jsonKey]
	if !ok {
		column = jsonKey
	}
	if st.isIgnoreColumn(column) {
		return
	}
	st.columns = append(st.columns, column)
	st.column2FieldName[column] = name
	st.column2Kind[column] = kind
	st.column2Type[column] = typeV
}

func (st *SQLTool) isIgnoreColumn(column string) bool {
	if st.actionType == ActionUpdate && utils.IsStringSliceContains(st.table.AutoUpdateDateTimeColumns, column) {
		return false
	}
	if st.actionType == ActionCreate &&
		(utils.IsStringSliceContains(st.table.AutoUpdateDateTimeColumns, column) ||
			utils.IsStringSliceContains(st.table.AutoCreateDateTimeColumns, column)) {
		return false
	}
	if _, ok := st.ignoreColumn[column]; ok {
		return true
	}
	if len(st.includeColumn) < 1 {
		return false
	}
	if _, ok := st.includeColumn[column]; !ok {
		return true
	}
	return false
}

// parse custom columns
func (st *SQLTool) parseCustomColumns(i interface{}, col []string) string {
	columnStr := ""
	st.columns = make([]string, 0)
	st.column2FieldName = make(map[string]string, 0)
	st.column2Kind = make(map[string]reflect.Kind, 0)
	st.column2Type = make(map[string]reflect.Type, 0)
	mapCol := map[string]bool{}

	for _, value := range col {
		mapCol[value] = true
	}

	t := reflect.TypeOf(i).Elem()
	for index := 0; index < t.NumField(); index++ {
		f := t.Field(index)
		name := f.Name
		kind := f.Type.Kind()
		jsonKey := getKeyFromJSONTag(f.Tag.Get("json"))
		if jsonKey == "-" || utils.IsIgnoreThisField(name) {
			continue
		}

		// convert to column name
		column, ok := st.table.ConvertJSONKey2Column[jsonKey]
		if !ok {
			column = jsonKey
		}
		if st.isIgnoreColumn(column) {
			continue
		}
		if _, ok := mapCol[column]; ok {
			st.columns = append(st.columns, column)
			st.column2FieldName[column] = name
			st.column2Kind[column] = kind
			if columnStr == "" {
				columnStr = column
			} else {
				columnStr = fmt.Sprintf("%s,%s", columnStr, column)
			}
		}
	}
	return columnStr
}

func (st *SQLTool) isReachLimitValue(x interface{}) bool {
	value := cast.ToInt64(x)

	return value >= st.table.AIColumnsLimitValue
}

func (st *SQLTool) fillValues(i interface{}, isUpdate bool) {
	st.values = make([]interface{}, 0)
	v := reflect.ValueOf(i).Elem()
	columnCustom := st.table.ColumnCustomAction
	if len(columnCustom) < 1 {
		columnCustom = make(map[string]ColumnFunction, 0)
	}
	for _, column := range st.columns {
		// get field name
		fieldName := st.column2FieldName[column]
		// kind, _ := smth.Column2Kind[column]
		fieldValue := v.FieldByName(fieldName)
		fieldValueInterface := fieldValue.Interface()

		if customFunc := columnCustom[column]; customFunc != nil {
			fieldValueInterface = customFunc(fieldValueInterface)
		}

		convertedValue := fieldValueInterface
		if column == st.table.AIColumns {
			if IsZeroOfUnderlyingType(fieldValueInterface) || st.table.ForceAIRemoveColumns {
				st.ignoreAIColumn = true
			} else if st.isReachLimitValue(fieldValueInterface) {
				st.aiColumnReachLimit = true
			}
		}

		// if nullable column, check if zero
		if !utils.IsStringSliceContains(st.table.NotNullColumns, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is foreign key
		if utils.IsStringSliceContains(st.table.ForeignKeys, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is datetime
		if utils.IsStringSliceContains(st.table.DateTimeColumns, column) {
			if utils.IsStringSliceContains(st.table.AutoUpdateDateTimeColumns, column) {
				convertedValue = time.Now()
			} else if !isUpdate && utils.IsStringSliceContains(st.table.AutoCreateDateTimeColumns, column) {
				convertedValue = time.Now()
			} else {
				if IsZeroOfUnderlyingType(fieldValueInterface) {
					convertedValue = nil
				} else {
					convertedValue = time.Unix(fieldValue.Int(), 0)
				}
			}
		}

		if st.column2Kind[column] == reflect.Slice && st.column2Type[column].Elem().Kind() == reflect.Uint8 {
			st.values = append(st.values, convertedValue)
			continue
		}

		var errMarshal error
		switch st.column2Kind[column] {
		case reflect.Slice, reflect.Struct, reflect.Ptr, reflect.Map:
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			} else {
				convertedValue, errMarshal = json.Marshal(convertedValue)
			}
		}
		if errMarshal != nil {
			panic(errMarshal)
		}

		convertedValue = st.preventXSS(fieldValue, column, convertedValue)
		st.values = append(st.values, convertedValue)
	}
	st.removeAIField()
}

func (st *SQLTool) removeAIField() {
	if !st.ignoreAIColumn || st.table.AIColumns == "" {
		return
	}
	tmpColumns := make([]string, 0)
	tmpValues := make([]interface{}, 0)
	for index, column := range st.columns {
		if column == st.table.AIColumns {
			continue
		}
		tmpColumns = append(tmpColumns, column)
		tmpValues = append(tmpValues, st.values[index])
	}
	st.columns = tmpColumns
	st.values = tmpValues

}

func (st *SQLTool) GetFillValues(i interface{}) []interface{} {
	values := make([]interface{}, 0)
	v := reflect.ValueOf(i).Elem()
	for _, column := range st.columns {
		// get field name
		fieldName, _ := st.column2FieldName[column]
		// kind, _ := smth.Column2Kind[column]
		fieldValue := v.FieldByName(fieldName)
		fieldValueInterface := fieldValue.Interface()
		convertedValue := fieldValueInterface

		// if nullable column, check if zero
		if !utils.IsStringSliceContains(st.table.NotNullColumns, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is foreign key
		if utils.IsStringSliceContains(st.table.ForeignKeys, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is datetime
		if utils.IsStringSliceContains(st.table.DateTimeColumns, column) {
			if utils.IsStringSliceContains(st.table.AutoUpdateDateTimeColumns, column) {
				convertedValue = time.Now()
			} else if utils.IsStringSliceContains(st.table.AutoCreateDateTimeColumns, column) && IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = time.Now()
			} else {
				if IsZeroOfUnderlyingType(fieldValueInterface) {
					convertedValue = nil
				} else {
					convertedValue = time.Unix(fieldValue.Int(), 0)
				}
			}
		}

		if st.column2Kind[column] == reflect.Slice && st.column2Type[column].Elem().Kind() == reflect.Uint8 {
			values = append(values, convertedValue)
			continue
		}

		var errMarshal error
		switch st.column2Kind[column] {
		case reflect.Slice, reflect.Struct, reflect.Ptr, reflect.Map:
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			} else {
				convertedValue, errMarshal = json.Marshal(convertedValue)
			}
		}
		if errMarshal != nil {
			panic(errMarshal)
		}

		convertedValue = st.preventXSS(fieldValue, column, convertedValue)
		values = append(values, convertedValue)
	}

	return values
}

// using custom data
func (st *SQLTool) fillCustomValues(i interface{}, columns []string) {
	st.values = make([]interface{}, 0)
	v := reflect.ValueOf(i).Elem()
	for _, column := range columns {
		// get field name
		fieldName, _ := st.column2FieldName[column]
		// kind, _ := smth.Column2Kind[column]
		fieldValue := v.FieldByName(fieldName)
		fieldValueInterface := fieldValue.Interface()
		convertedValue := fieldValueInterface

		// if nullable column, check if zero
		if !utils.IsStringSliceContains(st.table.NotNullColumns, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is foreign key
		if utils.IsStringSliceContains(st.table.ForeignKeys, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is datetime
		if utils.IsStringSliceContains(st.table.DateTimeColumns, column) {
			if utils.IsStringSliceContains(st.table.AutoUpdateDateTimeColumns, column) {
				convertedValue = time.Now()
			} else {
				if IsZeroOfUnderlyingType(fieldValueInterface) {
					convertedValue = nil
				} else {
					convertedValue = time.Unix(fieldValue.Int(), 0)
				}
			}
		}

		convertedValue = st.preventXSS(fieldValue, column, convertedValue)
		st.values = append(st.values, convertedValue)
	}
}

// IsZeroOfUnderlyingType --
func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

// GetFilledValues -- get filled values
func (st *SQLTool) GetFilledValues() []interface{} {
	return st.values
}

// GetCustomFilledValues -- get filled values
func (st *SQLTool) GetCustomFilledValues(i interface{}, column []string) []interface{} {
	st.fillCustomValues(i, column)
	return st.values
}

// Scan -- scan row to object
func (st *SQLTool) Scan(rows *sql.Rows, dest interface{}) (err error) {
	st.values = make([]interface{}, 0)
	for _, column := range st.columns {
		// get field name
		// fieldName, _ := smth.Column2FieldName[column]
		kind, _ := st.column2Kind[column]
		switch kind {
		case reflect.String:
			var nstr = &sql.NullString{}
			st.values = append(st.values, nstr)
		case reflect.Bool:
			var nbool = &sql.NullBool{}
			st.values = append(st.values, nbool)
		case reflect.Float32:
			var nfloat32 = &sql.NullFloat64{}
			st.values = append(st.values, nfloat32)
		case reflect.Float64:
			var nfloat64 = &sql.NullFloat64{}
			st.values = append(st.values, nfloat64)
		case reflect.Int64, reflect.Int32, reflect.Int:
			if utils.IsStringSliceContains(st.table.DateTimeColumns, column) {
				var ntime = &mysql.NullTime{}
				st.values = append(st.values, ntime)
			} else {
				var nint64 = &sql.NullInt64{}
				st.values = append(st.values, nint64)
			}
		case reflect.Slice, reflect.Struct, reflect.Map, reflect.Ptr:
			var nbytes = &sql.RawBytes{}
			st.values = append(st.values, nbytes)
		}
	}
	err = rows.Scan(st.values...)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Sql tool: Scan error: %v", err), "fields", st.columns)
		return
	}

	v := reflect.ValueOf(dest)
	// log.Print(v.Kind())
	if v.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if v.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}

	ve := v.Elem()
	// log.Print(ve.Kind())
	for index, column := range st.columns {
		// get field name
		fieldName, _ := st.column2FieldName[column]
		kind, _ := st.column2Kind[column]
		value := st.values[index]

		switch kind {
		case reflect.String:
			ve.FieldByName(fieldName).SetString(value.(*sql.NullString).String)
		case reflect.Bool:
			ve.FieldByName(fieldName).SetBool(value.(*sql.NullBool).Bool)
		case reflect.Float64:
			ve.FieldByName(fieldName).SetFloat(value.(*sql.NullFloat64).Float64)
		case reflect.Int64:
			if utils.IsStringSliceContains(st.table.DateTimeColumns, column) {
				var vtime = value.(*mysql.NullTime)
				if vtime.Valid && vtime.Time.Unix() >= 0 {
					ve.FieldByName(fieldName).SetInt(vtime.Time.Unix())
				}
			} else {
				ve.FieldByName(fieldName).SetInt(value.(*sql.NullInt64).Int64)
			}
		case reflect.Int32, reflect.Int:
			ve.FieldByName(fieldName).SetInt(value.(*sql.NullInt64).Int64)
		case reflect.Slice:
			val := value.(*sql.RawBytes)

			// If the slice is not slice of bytes
			if ve.FieldByName(fieldName).Type().Elem().Kind() != reflect.Uint8 {
				if len(*val) < 1 {
					break
				}
				v := make([]interface{}, 0)
				s := reflect.MakeSlice(st.column2Type[column], 0, 0)
				if err = json.Unmarshal(*val, &v); err == nil {
					s, _ = scanWithRecursive(v, s, st.column2Type[column].Elem())
					ve.FieldByName(fieldName).Set(s)
				} else {
					logger.BkLog.Error(err)
				}
			} else {
				var tmp = make([]byte, len(*val))
				copy(tmp, *val)
				ve.FieldByName(fieldName).SetBytes(tmp)
			}
		case reflect.Map, reflect.Ptr:
			val := value.(*sql.RawBytes)

			// If the slice is not slice of bytes
			if ve.FieldByName(fieldName).Type().Elem().Kind() != reflect.Uint8 {
				if len(*val) < 1 {
					break
				}

				v := make(map[string]interface{}, 0)
				if err = json.Unmarshal(*val, &v); err == nil {
					dataValue := reflect.New(Deref(st.column2Type[column]))
					scanWithRecursive(v, dataValue.Elem(), st.column2Type[column].Elem())
					ve.FieldByName(fieldName).Set(dataValue)
				} else {
					logger.BkLog.Error(err)
				}
			}
		}
	}

	return
}

func (st *SQLTool) GetUpdateMap() map[string]interface{} {
	m := make(map[string]interface{})

	for k, f := range strings.Split(st.GetQueryColumnList(), ",") {
		if f == "id" {
			continue
		}

		m[f] = st.GetFilledValues()[k]
	}

	return m
}

// getKeyFromJSONTag -- get json key from struct json tag
func getKeyFromJSONTag(tag string) string {
	pieces := utils.StringSlice(tag, ",")
	return pieces[0]
}

// GetColumn2FieldName --
func (st *SQLTool) GetColumn2FieldName(fieldName string) string {
	if val, ok := st.column2FieldName[fieldName]; ok {
		return val
	}
	return ""
}

// Execute query with args in non-context
func (st *SQLTool) exec(query string, args ...interface{}) (result sql.Result, err error) {
	if st.useTransaction {
		results, errs := st.UsingTransaction(func() []*Command {
			return st.QueryToCommand(query, args...)
		})
		if errs != nil {
			return nil, errs
		}
		return results[0], nil
	}
	var stmt *sql.Stmt
	stmt, err = st.db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()
	return stmt.Exec(args...)
}

// Execute query with args in context
func (st *SQLTool) execContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	if st.useTransaction {
		results, errs := st.UsingTransaction(func() []*Command {
			return st.QueryToCommand(query, args...)
		})
		if errs != nil {
			return nil, errs
		}
		return results[0], nil
	}

	var stmt *sql.Stmt
	stmt, err = st.db.PrepareContext(ctx, query)
	if err != nil {
		return
	}
	defer stmt.Close()

	utils.SetCtxData(ctx, meta.CtxSqlDB, "master")

	return stmt.ExecContext(ctx, args...)
}

// Execute query with many args in non-context
func (st *SQLTool) execMany(query string, listArgs [][]interface{}) (results []sql.Result, errs []error) {
	if st.useTransaction {
		var err error
		results, err = st.UsingTransaction(func() []*Command {
			return st.QueryToCommands(query, listArgs)
		})
		if err != nil {
			errs = append(errs, err)
		}
		return
	}

	stmt, err := st.db.Prepare(query)
	if err != nil {
		errs = append(errs, err)
		return
	}
	defer stmt.Close()
	for _, args := range listArgs {
		result, err := stmt.Exec(args...)
		if err != nil {
			errs = append(errs, err)
			return
		}
		results = append(results, result)
	}
	return
}

// Execute query with many args in context
func (st *SQLTool) execManyContext(ctx context.Context, query string, listArgs [][]interface{}) (results []sql.Result, errs []error) {
	if st.useTransaction {
		var err error
		results, err = st.UsingTransaction(func() []*Command {
			return st.QueryToCommands(query, listArgs)
		})
		if err != nil {
			errs = append(errs, err)
		}
		return
	}

	stmt, err := st.db.PrepareContext(ctx, query)
	if err != nil {
		errs = append(errs, err)
		return
	}
	defer stmt.Close()
	for _, args := range listArgs {
		result, err := stmt.ExecContext(ctx, args...)
		if err != nil {
			errs = append(errs, err)
			return
		}
		results = append(results, result)
	}
	return
}

// Query with args in non-context
func (st *SQLTool) query(query string, args ...interface{}) (rows *sql.Rows, err error) {
	var stmt *sql.Stmt
	stmt, err = st.db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()
	return stmt.Query(args...)
}

var mActionForceMaster = map[string]bool{
	ActionUpdate: true,
	ActionCreate: true,
	ActionDelete: true,
}

func (st *SQLTool) queryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	var stmt *sql.Stmt
	stmt, err = st.db.PrepareContext(ctx, query)
	if err != nil {
		return
	}
	defer stmt.Close()
	return stmt.QueryContext(ctx, args...)
	//if val := utils.GetCtxData(ctx, meta.CtxSqlDB); val == "master" || mActionForceMaster[st.actionType] {
	//	if stCluster, ok := stmt.(StCluster); ok {
	//		logger.BkLog.Debugf("Force using master conn for query")
	//		stCluster.ForceMaster(true)
	//	}
	//}
}

func correctDataType(v interface{}, dataType reflect.Kind) interface{} {
	var err error
	vString := cast.ToString(v)

	switch dataType {
	case reflect.String:
		return vString
	case reflect.Int64:
		if v, err = strconv.ParseInt(vString, 10, 64); err == nil {
			return v
		}
	case reflect.Int32:
		if v, err = strconv.ParseInt(vString, 10, 32); err == nil {
			return v
		}
	case reflect.Float64:
		if v, err = strconv.ParseFloat(vString, 64); err == nil {
			return v
		}
	case reflect.Float32:
		if v, err = strconv.ParseFloat(vString, 32); err == nil {
			return v
		}
	case reflect.Uint64:
		if v, err = strconv.ParseUint(vString, 10, 64); err == nil {
			return v
		}
	case reflect.Uint32:
		if v, err = strconv.ParseUint(vString, 10, 32); err == nil {
			return v
		}
	case reflect.Bool:
		if v, err = strconv.ParseBool(vString); err == nil {
			return v
		}
	}

	return nil
}

// scanWithRecursive -- scan with check func is recursive call
func scanWithRecursive(data interface{}, ve reflect.Value, sliceType reflect.Type) (res reflect.Value, err error) {
	switch document := data.(type) {
	case map[string]interface{}:
		numField := ve.NumField()
		for i := 0; i < numField; i++ {
			sliceTag := utils.StringSlice(ve.Type().Field(i).Tag.Get("json"), ",")
			if (len(sliceTag) > 0 && sliceTag[0] == "-") || len(sliceTag) == 0 {
				continue
			}

			destinationFName := ve.Type().Field(i).Name
			documentFieldName := sliceTag[0]
			var documentFieldValue interface{}
			var documentHasField bool
			if documentFieldValue, documentHasField = document[documentFieldName]; !documentHasField {
				continue
			}
			destinationFVReflect := ve.FieldByName(destinationFName)
			if !destinationFVReflect.CanSet() {
				continue
			}

			fieldValueString := cast.ToString(documentFieldValue)
			switch destinationFVReflect.Kind() {
			case reflect.Int64:
				if v, err := strconv.ParseInt(fieldValueString, 10, 64); err == nil {
					ve.FieldByName(destinationFName).SetInt(v)
				}
			case reflect.Float64:
				if v, err := strconv.ParseFloat(fieldValueString, 64); err == nil {
					ve.FieldByName(destinationFName).SetFloat(v)
				}
			case reflect.Slice:
				if v, ok := documentFieldValue.([]interface{}); v != nil && ok {
					embeddedDocuments := reflect.MakeSlice(destinationFVReflect.Type(), 0, 0)
					kind := destinationFVReflect.Type().Elem().Kind()
					if reflect.TypeOf(documentFieldValue).Kind() == reflect.Slice {
						for i := 0; i <= len(v)-1; i++ {
							var dataElement reflect.Value
							switch kind {
							case reflect.Slice:
								dataElement := reflect.New(Deref(destinationFVReflect.Type().Elem()))
								scanWithRecursive(v[i], dataElement.Elem(), dataElement.Type().Elem())
							case reflect.Map, reflect.Ptr:
								dataElement = reflect.New(Deref(destinationFVReflect.Type().Elem()))
								scanWithRecursive(v[i], dataElement.Elem(), nil)
							default:
								dataElement = reflect.ValueOf(utils.CorrectDataType(v[i], destinationFVReflect.Type().Elem().Kind()))
								//dataElement = reflect.ValueOf(v[i])
							}

							embeddedDocuments = reflect.Append(embeddedDocuments, dataElement)
						}
						ve.FieldByName(destinationFName).Set(embeddedDocuments)
					}
				}
			case reflect.Map:
				if reflect.TypeOf(documentFieldValue).Kind() == reflect.Map {
					mapField := ve.FieldByName(destinationFName)
					mapField.Set(reflect.MakeMap(mapField.Type()))
					mapValue, ok := documentFieldValue.(map[string]interface{})
					if ok {
						for mapK, mapV := range mapValue {
							dataElement := reflect.New(destinationFVReflect.Elem().Type())
							scanWithRecursive(mapV, dataElement, nil)
							mapField.SetMapIndex(reflect.ValueOf(mapK), dataElement)
						}
					} else {
						logger.BkLog.Warnf("documentFieldValue is not map")
					}
				} else {
					logger.BkLog.Warnf("Field `%v` value `%v` is not maps", destinationFName, documentFieldValue)
				}
			case reflect.Ptr:
				if reflect.TypeOf(documentFieldValue).Kind() == reflect.Map {
					if v, ok := documentFieldValue.(map[string]interface{}); ok {
						dataValue := reflect.New(destinationFVReflect.Type().Elem())
						_, err = scanWithRecursive(v, dataValue.Elem(), nil)
						ve.FieldByName(destinationFName).Set(dataValue)
					}

				}
			default:
				if vCorrected := correctDataType(documentFieldValue, destinationFVReflect.Kind()); vCorrected != nil {
					ve.FieldByName(destinationFName).Set(reflect.ValueOf(vCorrected))
				} else {
					logger.BkLog.Warnf(
						"Unknown field type `%v` with value %v name `%v`",
						destinationFVReflect.Kind(),
						fieldValueString,
						documentFieldName,
					)
				}
			}
		}
	case []interface{}:
		if ve.Kind() == reflect.Slice {
			for i := 0; i <= len(document)-1; i++ {
				switch sliceType.Kind() {
				case reflect.String, reflect.Int64, reflect.Int32, reflect.Int8, reflect.Float64, reflect.Float32:
					ve = reflect.Append(ve, reflect.ValueOf(utils.CorrectDataType(document[i], ve.Type().Elem().Kind())))
				case reflect.Slice, reflect.Struct, reflect.Ptr, reflect.Map:
					dataElement := reflect.New(sliceType.Elem())
					scanWithRecursive(document[i], dataElement.Elem(), nil)
					ve = reflect.Append(ve, dataElement)
				}
			}
			res = ve
		}
	}
	return
}

// html escape for prevent xss
func (st *SQLTool) preventXSS(field reflect.Value, col string, val interface{}) interface{} {
	if !st.xssEnabled || st.xssIgnoreColumns[col] {
		return val
	}

	if val == nil || st.column2Kind[col] != reflect.String {
		return val
	}

	newVal := st.xssReplacer.Replace(val.(string))
	field.SetString(newVal)
	return newVal
}

// ignore value for database
func (st *SQLTool) ignoreValue(query string, args ...interface{}) (string, []interface{}) {
	replacer := st.mysqlReplacer
	//if st.db.IsPgSQL {
	//	replacer = st.pgsqlReplacer
	//}

	if replacer == nil {
		return query, args
	}

	for i, arg := range args {
		if val, ok := arg.(string); ok {
			args[i] = replacer.Replace(val)
		}
	}

	query = replacer.Replace(query)
	return query, args
}
