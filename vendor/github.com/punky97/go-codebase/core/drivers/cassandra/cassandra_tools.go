package cassandra

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

// Table -- define table information: table name, columns
type Table struct {
	Name                      string
	IgnoreColumns             []string
	DateTimeColumns           []string
	AutoUpdateDateTimeColumns []string
	PartitionKeys             []string
	ClusteringKeys            []string
	ForeignKeys               []string
	PrimaryKeys               []string
	NotNullColumns            []string
	ConvertJSONKey2Column     map[string]string
}

// CQLTool --
type CQLTool struct {
	session          *gocql.Session
	table            Table
	columns          []string
	column2FieldName map[string]string
	column2Kind      map[string]reflect.Kind
	column2Type      map[string]reflect.Type
	values           []interface{}
	rabbitMQProducer *queue.Producer
}

// GetCassandraTool --
func GetCassandraTool(session *gocql.Session, table Table) (ct CQLTool) {
	ct.session = session
	ct.table = table
	return
}

// NewSelectQuery -- create new select query
func NewSelectQuery(table Table, protoObj interface{}) (ct CQLTool) {
	ct.table = table
	ct.parseColumns(protoObj, false)
	return
}

// PrepareSelect -- prepare select query
func PrepareSelect(session *gocql.Session, table Table, protoObj interface{}) (ct CQLTool) {
	ct.session = session
	ct.table = table
	ct.parseColumns(protoObj, false)
	return
}

// NewInsertQuery -- create new insert query
func NewInsertQuery(table Table, protoObj interface{}) (ct CQLTool) {
	ct.table = table
	ct.parseColumns(protoObj, false)
	ct.fillValues(protoObj)
	return
}

// PrepareInsert -- prepare insert query
func PrepareInsert(session *gocql.Session, table Table, protoObj interface{}) (ct CQLTool) {
	ct.session = session
	ct.table = table
	ct.parseColumns(protoObj, false)
	ct.fillValues(protoObj)
	return
}

// NewUpdateQuery -- create new update query
func NewUpdateQuery(table Table, protoObj interface{}) (ct CQLTool) {
	ct.table = table
	ct.parseColumns(protoObj, false)
	ct.fillValues(protoObj)
	return
}

// PrepareUpdate -- prepare update query
func Prepare(session *gocql.Session) (ct CQLTool) {
	ct.session = session
	return
}

// PrepareUpdate -- prepare update query
func PrepareUpdate(session *gocql.Session, table Table, protoObj interface{}) (ct CQLTool) {
	ct.session = session
	ct.table = table
	ct.parseColumns(protoObj, true)
	ct.fillValues(protoObj)
	return
}

// PrepareDelete -- prepare delete query
func PrepareDelete(session *gocql.Session, table Table, protoObj interface{}) (ct CQLTool) {
	ct.session = session
	ct.table = table
	ct.parseColumns(protoObj, false)
	return
}

// InitRmq --
func (ct *CQLTool) InitRmq() *queue.Producer {
	p := queue.NewRMQProducerFromDefConf()
	if p == nil {
		//logger.BkLog.Info("nil producer")
	} else {
		if err := p.Connect(); err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Cannot connect to rabbitmq. Please check configuration file for more information: %v", err))
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
func (ct *CQLTool) InitRedis(ctx context.Context) (*bkredis.RedisClient, error) {
	redisConn, err := bkredis.NewConnection(ctx, bkredis.DefaultRedisConnectionFromConfig())
	return redisConn, err
}

// PublishToRmq --
func (ct *CQLTool) PublishToRmq(exchangeName, method, db, query string, args []interface{}) {
	p := ct.InitRmq()
	if p == nil {
		logger.BkLog.Info("nil producer")
		return
	}
	// p.Start()
	// defer p.Close()
	rand.Seed(time.Now().Unix())
	payload := MigrateDataInput{}
	payload.ID = fmt.Sprintf("%v_%v", rand.Int63(), time.Now().Unix())
	payload.Method = method
	payload.Table = ct.table
	payload.Query = query
	payload.Values = args
	payload.Columns = ct.columns
	gob.Register(time.Time{})
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(payload)
	if err != nil {
		logger.BkLog.Warnf("Encoding failed %v, err %v", string(b.Bytes()), err)
	}
	err = p.PublishRouting(exchangeName, db, b.Bytes())
	if err != nil {
		logger.BkLog.Warnf("fail to publish routing, table %v, query %v, err %v", ct.GetTableName(), query, err)
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

// SetDB -- set cassandara db
func (ct *CQLTool) SetDB(session *gocql.Session) {
	ct.session = session
}

// SetPageSize -- wrap gocql set page size - using in case want to query using IN and ORDER BY at same time
func (ct *CQLTool) SetPageSize(n int) {
	ct.session.SetPageSize(n)
}

// Insert -- do insert
func (ct *CQLTool) Insert(query string, args ...interface{}) (err error) {
	if viper.GetBool("cassandra_migrate.enable") {
		exchName := viper.GetString("cassandra_migrate.exchange_name")
		ct.PublishToRmq(exchName, "Insert", "cassandra", query, args)
	}
	return ct.session.Query(query, args...).Exec()
}

// Update -- do update
func (ct *CQLTool) Update(query string, args ...interface{}) (err error) {
	if viper.GetBool("cassandra_migrate.enable") {
		exchName := viper.GetString("cassandra_migrate.exchange_name")
		ct.PublishToRmq(exchName, "Update", "cassandra", query, args)
	}
	return ct.session.Query(query, args...).Exec()
}

// Delete -- do delete
func (ct *CQLTool) Delete(query string, args ...interface{}) (err error) {
	if viper.GetBool("cassandra_migrate.enable") {
		exchName := viper.GetString("cassandra_migrate.exchange_name")
		ct.PublishToRmq(exchName, "Delete", "cassandra", query, args)
	}
	return ct.session.Query(query, args...).Exec()
}

// Select -- do select
func (ct *CQLTool) Select(dest interface{}, query string, args ...interface{}) (err error) {
	iter := ct.session.Query(query, args...).Iter()
	defer func() {
		if queryError := iter.Close(); queryError != nil {
			err = queryError
		}
	}()
	value := reflect.ValueOf(dest)
	// json.Unmarshal returns errors for these
	if value.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if value.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	slice, err := baseType(value.Type(), reflect.Slice)
	if err != nil {
		return err
	}
	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := Deref(slice.Elem())
	empty := true
	ct.getMapValuesScan()
	direct := reflect.Indirect(value)
	for iter.Scan(ct.values...) {
		vp := reflect.New(base)
		if err = ct.fillData(vp.Interface()); err != nil {
			logger.BkLog.Warnf("Error while scan, details: %v", err)
			continue
		}
		empty = false
		// append
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
	}
	if empty {
		err = gocql.ErrNotFound
	}
	return
}

func (ct *CQLTool) SelectWithPageState(dest interface{}, query string, pageSize int, pageState []byte, args ...interface{}) (newPageState []byte, err error) {
	cqlQuery := ct.session.Query(query, args...)

	if pageSize > 0 {
		cqlQuery = cqlQuery.PageSize(pageSize)
	}

	cqlQuery = cqlQuery.PageState(pageState)

	iter := cqlQuery.Iter()

	newPageState = iter.PageState()

	defer func() {
		if queryError := iter.Close(); queryError != nil {
			err = queryError
		}
	}()

	value := reflect.ValueOf(dest)

	// json.Unmarshal returns errors for these
	if value.Kind() != reflect.Ptr {
		return nil, errors.New("must pass a pointer, not a value, to StructScan destination")
	}

	if value.IsNil() {
		return nil, errors.New("nil pointer passed to StructScan destination")
	}

	slice, err := baseType(value.Type(), reflect.Slice)
	if err != nil {
		return nil, err
	}

	isPtr := slice.Elem().Kind() == reflect.Ptr
	base := Deref(slice.Elem())

	empty := true

	ct.getMapValuesScan()

	direct := reflect.Indirect(value)

	for iter.Scan(ct.values...) {
		vp := reflect.New(base)

		if err = ct.fillData(vp.Interface()); err != nil {
			logger.BkLog.Warnf("Error while scan, details: %v", err)
			continue
		}

		empty = false

		// append
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
	}

	if empty {
		err = gocql.ErrNotFound
	}

	return
}

func (ct *CQLTool) SelectCount(dest interface{}, query string, args ...interface{}) (err error) {
	iter := ct.session.Query(query, args...).Iter()

	defer func() {
		if queryError := iter.Close(); queryError != nil {
			err = queryError
		}
	}()

	iter.Scan(dest)

	return
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

// Get -- do select on
func (ct *CQLTool) Get(dest interface{}, query string, args ...interface{}) (err error) {
	row := ct.session.Query(query, args...).Consistency(gocql.One)
	// fmt.Println("rows:", rows)
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
	ct.getMapValuesScan()
	if err = row.Scan(ct.values...); err == nil {
		vp = reflect.New(base)
		err = ct.fillData(vp.Interface())
		if err == nil {
			direct.Set(vp.Elem())
		}
	}

	return
}

// GetTableName -- return table name
func (ct *CQLTool) GetTableName() string {
	return ct.table.Name
}

// GetQueryColumnList -- use for SELECT, INSERT query
func (ct *CQLTool) GetQueryColumnList() string {
	return strings.Join(ct.columns, ",")
}

func (ct *CQLTool) GetPartitionKeys() []string {
	return ct.table.PartitionKeys
}

func (ct *CQLTool) GetClusteringKeys() []string {
	return ct.table.ClusteringKeys
}

// GetQueryValueList -- use for INSERT query
func (ct *CQLTool) GetQueryValueList() string {
	questionMarks := make([]string, 0)
	for index := 0; index < len(ct.columns); index++ {
		questionMarks = append(questionMarks, "?")
	}
	return strings.Join(questionMarks, ",")
}

// GetQueryAssignmentList -- use for UPDATE query
func (ct *CQLTool) GetQueryAssignmentList() string {
	p := make([]string, 0)

	for _, column := range ct.columns {
		temp := column + " = ?"
		p = append(p, temp)
	}

	return strings.Join(p, ",")
}

// parse columns
func (ct *CQLTool) parseColumns(i interface{}, ignorePrimary bool) {
	ct.columns = make([]string, 0)
	ct.column2FieldName = make(map[string]string, 0)
	ct.column2Kind = make(map[string]reflect.Kind, 0)
	ct.column2Type = make(map[string]reflect.Type, 0)

	t := reflect.TypeOf(i).Elem()
	for index := 0; index < t.NumField(); index++ {
		f := t.Field(index)
		name := f.Name
		typeV := f.Type
		kind := f.Type.Kind()
		jsonKey := getKeyFromJSONTag(f.Tag.Get("json"))
		if jsonKey == "-" || utils.IsIgnoreThisField(name) || (ignorePrimary && utils.IsStringSliceContains(ct.table.PrimaryKeys, jsonKey)) {
			continue
		}

		// convert to column name
		column, ok := ct.table.ConvertJSONKey2Column[jsonKey]
		if !ok {
			column = jsonKey
		}
		if len(ct.table.IgnoreColumns) > 0 {
			if utils.IsStringSliceContains(ct.table.IgnoreColumns, column) {
				continue
			}
		}
		ct.columns = append(ct.columns, column)
		ct.column2FieldName[column] = name
		ct.column2Kind[column] = kind
		ct.column2Type[column] = typeV
	}
	return
}

func (ct *CQLTool) fillValues(i interface{}) {
	ct.values = make([]interface{}, 0)
	v := reflect.ValueOf(i).Elem()
	for _, column := range ct.columns {
		// get field name
		fieldName, _ := ct.column2FieldName[column]
		// kind, _ := smth.Column2Kind[column]
		fieldValue := v.FieldByName(fieldName)
		fieldValueInterface := fieldValue.Interface()
		convertedValue := fieldValueInterface

		// if nullable column, check if zero
		if !utils.IsStringSliceContains(ct.table.NotNullColumns, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is foreign key
		if utils.IsStringSliceContains(ct.table.ForeignKeys, column) {
			if IsZeroOfUnderlyingType(fieldValueInterface) {
				convertedValue = nil
			}
		}

		// check if column is datetime
		if utils.IsStringSliceContains(ct.table.DateTimeColumns, column) {
			if utils.IsStringSliceContains(ct.table.AutoUpdateDateTimeColumns, column) {
				convertedValue = time.Now()
			} else {
				if IsZeroOfUnderlyingType(fieldValueInterface) {
					convertedValue = nil
				} else {
					convertedValue = time.Unix(fieldValue.Int(), 0)
				}
			}
		}

		ct.values = append(ct.values, convertedValue)
	}
}

// IsZeroOfUnderlyingType --
func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

// GetFilledValues -- get filled values
func (ct *CQLTool) GetFilledValues() []interface{} {
	return ct.values
}

func (ct *CQLTool) getMapValuesScan() []interface{} {
	ct.values = make([]interface{}, 0)

	for _, column := range ct.columns {
		if columnKind, ok := ct.column2Kind[column]; ok {
			switch {
			case columnKind == reflect.Int64 && utils.IsStringSliceContains(ct.table.DateTimeColumns, column):
				ct.values = append(ct.values, new(time.Time))
			default:
				if columnType, ok := ct.column2Type[column]; ok {
					ct.values = append(ct.values, reflect.New(columnType).Interface())
				}
			}
		}
	}

	return ct.values
}

func (ct *CQLTool) fillData(dest interface{}) (err error) {
	destVOF := reflect.ValueOf(dest)
	if destVOF.Kind() != reflect.Ptr {
		return errors.New("must pass a pointer, not a value, to StructScan destination")
	}
	if destVOF.IsNil() {
		return errors.New("nil pointer passed to StructScan destination")
	}
	destVOFE := destVOF.Elem()
	for index, column := range ct.columns {
		// Get field name
		fieldName, _ := ct.column2FieldName[column]
		kind, _ := ct.column2Kind[column]
		value := ct.values[index]
		switch kind {
		case reflect.Int64:
			if utils.IsStringSliceContains(ct.table.DateTimeColumns, column) {
				if v := *value.(*time.Time); v.Unix() >= 0 {
					destVOFE.FieldByName(fieldName).SetInt(v.Unix())
				}
			} else {
				destVOFE.FieldByName(fieldName).SetInt(cast.ToInt64(value))
			}
		case reflect.Map:
			mapField := destVOFE.FieldByName(fieldName)
			mapField.Set(reflect.MakeMap(reflect.TypeOf(value).Elem()))
			a := reflect.ValueOf(value).Elem()
			keys := a.MapKeys()
			for _, key := range keys {
				mapField.SetMapIndex(key, a.MapIndex(key))
			}
		default:
			valueKind := reflect.TypeOf(value).Kind()
			if valueKind == reflect.Ptr {
				if v := reflect.ValueOf(value).Elem(); v.Kind() == destVOFE.FieldByName(fieldName).Kind() {
					destVOFE.FieldByName(fieldName).Set(v)
				}
			} else if valueKind == destVOFE.FieldByName(fieldName).Kind() {
				destVOFE.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
	return
}

// getKeyFromJSONTag -- get json key from struct json tag
func getKeyFromJSONTag(tag string) string {
	pieces := utils.StringSlice(tag, ",")
	return pieces[0]
}
