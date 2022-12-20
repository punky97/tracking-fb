package mongo

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/drivers/mongo/mgo_access_layout"
	"github.com/punky97/go-codebase/core/drivers/mongo/mgo_mock"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const DefaultLimit = 20

// CollectionInfo --
type CollectionInfo struct {
	Name                    string
	IDFieldName             string
	DateTimeFields          []string
	EmbeddedDocumentFields  []CollectionInfo
	ConvertToSnakeCaseField []string
	AutoIncrementId         bool
	StringDynamicField      []string
	DBRefFields             []DBRef
	UpdatedAtFields         []string
	DefaultFields           []string
	DefaultIgnoreFields     []string
	DefaultLimit            int
	FixDataMapFunc          FixDataMap
	Struct                  func() proto.Message
}

// MGTool --
type MGTool struct {
	Collection           *CollectionInfo
	Db                   mgo_access_layout.DatabaseAL
	IncrementIdGenerator func(collectionName string) (id int64, err error)
	rabbitMQProducer     *queue.Producer
	includeFields        map[string]int8
	ignoreFields         map[string]int8
	fieldValueNil        map[string]bool
	defaultLimit         int
	defaultOffset        int
	ctx                  context.Context
}

var defaultStructEmbedValue = struct{}{}

var reUpdateFieldNull = regexp.MustCompile(`cannot use the part \(.*\) to traverse the element \(\{(.*): null\}\)`)

// StreamExecute - execute func of stream data
// If this func return error != nil
// -> stream will be stop
type StreamExecute func(data interface{}, err error) error

type FixDataMap func(data interface{}) (map[string]interface{}, error)

// DBRef --
type DBRef struct {
	CollectionRef string
	Name          string
}

func (mgt *MGTool) initTool(isUpdate bool) {
	var isChecked bool

	if mgt.ctx == nil {
		return
	}
	md, _ := metadata.FromIncomingContext(mgt.ctx)
	if fIgnores := md.Get(utils.UseAllWithoutFieldHeader); len(fIgnores) > 0 {
		isChecked = true
		mgt.ignoreFields = make(map[string]int8)
		for _, cl := range fIgnores {
			mgt.ignoreFields[cl] = 0
			if strings.Contains(cl, "_") {
				mgt.ignoreFields[utils.ToCamelInitCaseKeepAll(cl, false)] = 0
			}
		}
	} else if fSelects := md.Get(utils.FieldSelectHeader); len(fSelects) > 0 {
		isChecked = true
		mgt.includeFields = make(map[string]int8)
		for _, cl := range fSelects {
			mgt.includeFields[cl] = 1
			if strings.Contains(cl, "_") {
				mgt.includeFields[utils.ToCamelInitCaseKeepAll(cl, false)] = 1
			}
		}
	}

	if useAllFieldStr := md.Get(utils.UseAllFieldHeader); len(useAllFieldStr) > 0 && cast.ToBool(useAllFieldStr[0]) {
		mgt.includeFields = nil
		mgt.ignoreFields = nil
		isChecked = true
	}

	if !isUpdate {
		if limitRecord := md.Get(utils.LimitHeader); len(limitRecord) > 0 && cast.ToInt(limitRecord[0]) > 0 {
			mgt.defaultLimit = cast.ToInt(limitRecord[0])
		} else if defaultLimit := md.Get(utils.DefaultLimitHeader); len(defaultLimit) > 0 && cast.ToBool(defaultLimit[0]) {
			if limit := viper.GetInt("limit_record.number"); limit > 0 {
				mgt.defaultLimit = limit
			} else {
				mgt.defaultLimit = DefaultLimit
			}
		}
		if offsetStr := md.Get(utils.OffsetHeader); len(offsetStr) > 0 && cast.ToInt(offsetStr[0]) > 0 {
			mgt.defaultOffset = cast.ToInt(offsetStr[0])
		}
	}

	// If field and ignore field was defined by params or metadata of request. Do not use default
	if isChecked {
		return
	}

	var useDefaultLimitSelectField bool
	if limitFieldStr := md.Get(utils.DefaultSelectHeader); viper.GetBool("limit_field_select_record.default_field") || (len(limitFieldStr) > 0 && cast.ToBool(limitFieldStr[0])) {
		useDefaultLimitSelectField = true
	}
	if useDefaultLimitSelectField {
		if mgt.includeFields == nil {
			mgt.includeFields = make(map[string]int8)
		}
		for _, cl := range mgt.Collection.DefaultFields {
			mgt.includeFields[cl] = 1
			if strings.Contains(cl, "_") {
				mgt.includeFields[utils.ToCamelInitCaseKeepAll(cl, false)] = 1
			}
		}
		if mgt.ignoreFields == nil {
			mgt.ignoreFields = make(map[string]int8)
		}
		for _, cl := range mgt.Collection.DefaultIgnoreFields {
			//mgt.ignoreFields[cl] = 0
			if strings.Contains(cl, "_") {
				mgt.ignoreFields[utils.ToCamelInitCaseKeepAll(cl, false)] = 0
			}
		}
	}
}

// GetRawMGTool --
func GetRawMGTool(c *BkMongo) *MGTool {
	c.Session.Refresh()
	return &MGTool{
		Db:                   c.GetDB(),
		IncrementIdGenerator: c.incrementIDGenerator,
	}
}

// InitRmq --
func (mgt MGTool) InitRmq() *queue.Producer {
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
func (mgt MGTool) InitRedis(ctx context.Context) (*bkredis.RedisClient, error) {
	redisConn, err := bkredis.NewConnection(ctx, bkredis.DefaultRedisConnectionFromConfig())
	return redisConn, err
}

// PublishToRmq --
func (mgt MGTool) PublishToRmq(exchangeName, method, db string, collection CollectionInfo, query interface{}, args bson.M) {
	p := mgt.InitRmq()
	if p == nil {
		logger.BkLog.Info("nil producer")
		return
	}
	// p.Start()
	// defer p.Close()
	rand.Seed(time.Now().Unix())
	payload := MigrateDataInput{}
	payload.Method = method
	payload.Db = db
	payload.Collection = collection
	payload.Query = query
	payload.Args = args
	payload.ID = fmt.Sprintf("%v_%v", rand.Int63(), time.Now().Unix())
	gob.Register(time.Time{})
	gob.Register(func() proto.Message {
		return nil
	})
	gob.Register(bson.M{})
	gob.Register(map[string]string{})
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(payload)
	if err != nil {
		logger.BkLog.Warnf("Encoding failed %v, err %v", string(b.Bytes()), err)
	}
	err = p.PublishRouting(exchangeName, db, b.Bytes())
	if err != nil {
		logger.BkLog.Warnf("fail to publish routing, table %v, query %v, err %v", collection.Name, query, err)
	}
}

// MigrateDataInput --
type MigrateDataInput struct {
	ID         string         `json:"id"`
	Method     string         `json:"method"`
	Db         string         `json:"db"`
	Collection CollectionInfo `json:"collection"`
	Query      interface{}    `json:"query"`
	Args       bson.M         `json:"args"`
	Retry      int            `json:"retry"`
}

// GetCollection --
func (mgt MGTool) GetCollection() (collection *mgo.Collection) {
	if db, ok := mgt.Db.(*mgo.Database); ok {
		return db.C(mgt.Collection.Name)
	}
	return
}

// GetCollection --
func (mgt MGTool) getCollection() (collection mgo_access_layout.CollectionAL) {
	if db, ok := mgt.Db.(*mgo.Database); ok {
		return db.C(mgt.Collection.Name)
	}

	if db, ok := mgt.Db.(*mgo_mock.Database); ok {
		return db.C(mgt.Collection.Name)
	}
	return
}

// FindWithIncludeFields --
func (mgt MGTool) FindWithIncludeFields(query interface{}, protoObj proto.Message, includes []string) *mgo.Query {
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		if len(includes) < 1 {
			return collection.Find(query).Select(mgt.GetSelectFields(protoObj))
		} else {
			return collection.Find(query).Select(mgt.makeSelectFields(includes))
		}
	}
	return nil
}

// FindId --
func (mgt MGTool) FindId(query interface{}) *mgo.Query {
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		fields := make(map[string]int8)
		fields["_id"] = 1
		return collection.Find(query).Select(fields)
	}
	return nil
}

// Pipe --
func (mgt MGTool) Pipe(pipeline interface{}) *mgo.Pipe {
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		return collection.Pipe(pipeline)
	}
	return nil
}

// Find --
func (mgt MGTool) Find(query interface{}, protoObj proto.Message) *mgo.Query {
	var mgoQuery *mgo.Query
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		if len(mgt.includeFields) > 0 {
			mgoQuery = collection.Find(query).Select(mgt.includeFields)
		} else {
			mgoQuery = collection.Find(query).Select(mgt.GetSelectFields(protoObj))
		}
		if mgt.defaultLimit > 0 {
			mgoQuery = mgoQuery.Limit(mgt.defaultLimit)
		}
		if mgt.defaultOffset > 0 {
			mgoQuery = mgoQuery.Skip(mgt.defaultOffset)
		}
	}
	return mgoQuery
}

func (mgt MGTool) getQuery(query interface{}, protoObj interface{}, limit, offset int, sort []string,
	fields ...string) mgo_access_layout.QueryAL {
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		var mgoQuery *mgo.Query
		mgoQuery = collection.Find(query)
		if len(fields) > 0 {
			mgoQuery = mgoQuery.Select(mgt.parseField2Map(fields))
		} else if len(mgt.includeFields) > 0 {
			mgoQuery = mgoQuery.Select(mgt.includeFields)
		} else if protoObj != nil {
			mgoQuery = mgoQuery.Select(mgt.GetSelectFields(protoObj))
		}

		if limit > 0 {
			mgoQuery = mgoQuery.Limit(limit)
		}
		if offset > 0 {
			mgoQuery = mgoQuery.Skip(offset)
		}
		for _, sortField := range sort {
			mgoQuery = mgoQuery.Sort(sortField)
		}
		return mgoQuery
	}

	if collection, ok := mgt.getCollection().(*mgo_mock.Collection); ok {
		var mgoQuery *mgo_mock.Query
		mgoQuery = collection.Find(query)
		if len(fields) > 0 {
			mgoQuery = mgoQuery.Select(mgt.parseField2Map(fields))
		} else if len(mgt.includeFields) > 0 {
			mgoQuery = mgoQuery.Select(mgt.includeFields)
		} else if protoObj != nil {
			mgoQuery = mgoQuery.Select(mgt.GetSelectFields(protoObj))
		}

		if limit > 0 {
			mgoQuery = mgoQuery.Limit(limit)
		}
		if offset > 0 {
			mgoQuery = mgoQuery.Skip(offset)
		}
		for _, sortField := range sort {
			mgoQuery = mgoQuery.Sort(sortField)
		}
		return mgoQuery
	}
	return nil
}

func (mgt MGTool) parseField2Map(fields []string) map[string]int8 {
	res := make(map[string]int8)
	for _, field := range fields {
		res[field] = 1
	}
	return res
}

// Update --
func (mgt MGTool) Update(query interface{}, protoObj proto.Message) error {
	return mgt.UpdateWithDefinedField(query, protoObj, nil)
}

// Increase --
func (mgt MGTool) Increase(query interface{}, fieldName string, conditionsInc interface{}) error {
	change := bson.M{"$inc": bson.M{fieldName: conditionsInc}}
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Update", "mongo", *mgt.Collection, query, change)
	}
	return mgt.getCollection().Update(query, change)
}

// IncreaseMulti --
func (mgt MGTool) IncreaseMulti(query interface{}, data bson.M) error {
	change := bson.M{"$inc": data}
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Update", "mongo", *mgt.Collection, query, change)
	}
	return mgt.getCollection().Update(query, change)
}

// PullUpdate --
func (mgt MGTool) PullUpdate(query interface{}, fieldName string, conditionsPull interface{}) error {
	change := bson.M{"$pull": bson.M{fieldName: conditionsPull}}
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Update", "mongo", *mgt.Collection, query, change)
	}
	return mgt.getCollection().Update(query, change)
}

// PushUpdate --
func (mgt MGTool) PushUpdate(query interface{}, fieldName string, protoObj proto.Message) error {
	change, err := mgt.getPushFields(protoObj, fieldName, mgt.Collection)
	if err != nil {
		return err
	}

	change = bson.M{"$push": change}
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Update", "mongo", *mgt.Collection, query, change)
	}
	return mgt.getCollection().Update(query, change)
}

// PushMultipleValue --
func (mgt MGTool) PushMultipleValue(query interface{}, fieldName string, conditionsPush interface{}) error {
	change := bson.M{"$push": bson.M{fieldName: bson.M{"$each": conditionsPush}}}
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Update", "mongo", *mgt.Collection, query, change)
	}
	return mgt.getCollection().Update(query, change)
}

// Set --
func (mgt MGTool) Set(query interface{}, changes interface{}) error {
	change := bson.M{
		"$set": changes,
	}
	_, err := mgt.getCollection().UpdateAll(query, change)
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "UpdateAll", "mongo", *mgt.Collection, query, change)
	}
	return err
}

// UpdateWithDefinedField --
func (mgt MGTool) UpdateWithDefinedField(query interface{}, protoObj proto.Message, enableFields []string) error {
	_, err := mgt.UpdateFields(query, protoObj, enableFields, true)
	return err
}

// UpdateFields --
func (mgt MGTool) UpdateFields(query interface{}, protoObj proto.Message, enableFields []string, updateAll bool) (int, error) {
	var moreIncludes = make(map[string]int8)
	if len(enableFields) > 0 {
		for _, field := range enableFields {
			moreIncludes[field] = 1
		}
	}
	change, err := mgt.getCreateUpdateFields(protoObj, true, mgt.Collection, "", false, moreIncludes)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when get field: %v", err), "field", enableFields)
		return 0, err
	}
	change = bson.M{
		"$set": change,
	}
	return mgt.update(query, change, updateAll, 0)
}

func (mgt MGTool) update(query interface{}, change map[string]interface{}, updateAll bool, recursive int) (int, error) {
	if recursive > 5 {
		logger.BkLog.Errorw("CHECK NOW - recursive too many time", "change", utils.ToJSONString(change))
		return 0, errors.New("CHECK NOW - recursive too many time")
	}
	var err error
	if updateAll {
		var info *mgo.ChangeInfo
		info, err = mgt.getCollection().UpdateAll(query, change)
		if info != nil {
			return info.Updated, err
		}
		if err != nil {
			logger.BkLog.Debugf("Error when query err: %v, update: % - change: %v", err, utils.ToJSONString(query), utils.ToJSONString(change))
			if fieldName := getFieldUpdateNull(err); fieldName != "" {
				change, errChange := mgt.fixChangeData(change, fieldName)
				if errChange != nil {
					return 0, err
				}
				return mgt.update(query, change, updateAll, recursive+1)
			}
		}
		if viper.GetBool("mongo_migrate.enable") && err == nil {
			exchName := viper.GetString("mongo_migrate.exchange_name")
			mgt.PublishToRmq(exchName, "UpdateAll", "mongo", *mgt.Collection, query, change)
		}
		return 0, err
	} else {
		err = mgt.getCollection().Update(query, change)
		if err != nil {
			logger.BkLog.Debugf("Error when query, update: % - change: %v", utils.ToJSONString(query), utils.ToJSONString(change))
			if fieldName := getFieldUpdateNull(err); fieldName != "" {
				change, errChange := mgt.fixChangeData(change, fieldName)
				if errChange != nil {
					return 0, err
				}
				return mgt.update(query, change, updateAll, recursive+1)
			}
		}
		if viper.GetBool("mongo_migrate.enable") && err == nil {
			exchName := viper.GetString("mongo_migrate.exchange_name")
			mgt.PublishToRmq(exchName, "Update", "mongo", *mgt.Collection, query, change)
		}
		return 1, err
	}
}

func getFieldUpdateNull(err error) string {
	if err == nil {
		return ""
	}
	res := reUpdateFieldNull.FindStringSubmatch(err.Error())
	if len(res) != 2 {
		return ""
	}
	return res[1]
}

// UpsertFields --
func (mgt MGTool) UpsertFields(query interface{}, protoObj proto.Message, enableFields []string) error {
	var moreIncludes map[string]int8
	if len(enableFields) > 0 {
		moreIncludes := make(map[string]int8)
		for _, field := range enableFields {
			moreIncludes[field] = 1
		}
	}
	change, err := mgt.getCreateUpdateFields(protoObj, true,
		mgt.Collection, "", false, moreIncludes)
	if err != nil {
		return err
	}
	change = bson.M{
		"$set": change,
	}

	if mgt.Collection.AutoIncrementId {
		idValue, err := mgt.IncrementIdGenerator(mgt.Collection.Name)
		if err != nil {
			logger.BkLog.Errorw("Error when increase _id", "err", err)
			return err
		}
		change["$setOnInsert"] = bson.M{
			"_id": idValue,
		}
	}

	_, err = mgt.getCollection().Upsert(query, change)
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Upsert", "mongo", *mgt.Collection, query, change)
	}
	return err
}

// Remove single document matching with query--
func (mgt MGTool) Remove(query interface{}) error {
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Remove", "mongo", *mgt.Collection, query, nil)
	}
	return mgt.getCollection().Remove(query)
}

// RemoveAll Remove all documents matching with query--
func (mgt MGTool) RemoveAll(query interface{}) error {
	_, err := mgt.getCollection().RemoveAll(query)
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "RemoveAll", "mongo", *mgt.Collection, query, nil)
	}
	return err
}

// Create --
func (mgt MGTool) Create(protoObj proto.Message) (int64, error) {
	document, err := mgt.getCreateFields(protoObj)
	if err != nil {
		return 0, err
	}
	var id int64
	if document["_id"] != nil {
		id, _ = document["_id"].(int64)
	}
	if viper.GetBool("mongo_migrate.enable") {
		exchName := viper.GetString("mongo_migrate.exchange_name")
		mgt.PublishToRmq(exchName, "Insert", "mongo", *mgt.Collection, bson.M{}, document)
	}

	err = mgt.getCollection().Insert(document)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Insert mongo error: %v", err), "id", id, "doc", utils.ToJSONString(document), "input", utils.ToJSONString(protoObj))
	}
	return id, err
}

// Count --
func (mgt MGTool) Count(query interface{}) (int, error) {
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		return collection.Find(query).Count()
	}
	return 0, nil
}

// TODU
// FindAndModify --
func (mgt MGTool) FindAndModify(query interface{}, change mgo.Change) (result bson.M, err error) {
	if collection, ok := mgt.getCollection().(*mgo.Collection); ok {
		_, err = collection.Find(query).Apply(change, &result)
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Could not update query: %v", err), "query", utils.ToJSONString(query), "change", utils.ToJSONString(change))
		}
	}

	if collection, ok := mgt.getCollection().(*mgo_mock.Collection); ok {
		_, err = collection.Find(query).Apply(change, &result)
	}
	return
}

// Deref is Indirect for reflect.Types
func Deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// Scan --
func (mgt MGTool) Scan(document map[string]interface{}, destination interface{}) (err error) {
	return mgt.ScanWithRecursive(document, destination, mgt.Collection)
}

// ScanWithRecursive -- scan with check func is recursive call
func (mgt MGTool) ScanWithRecursive(document map[string]interface{}, destination interface{}, collection *CollectionInfo) (err error) {
	if v := reflect.ValueOf(destination); v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("mongo tools scan: Destination must be a pointer and is not nil pointer")
	}

	ve := reflect.ValueOf(destination).Elem()

	for documentFieldName, documentFieldValue := range document {
		if documentFieldValue == nil {
			logger.BkLog.Debugf(
				"Field `%v` found in `%v._id=%v` has been ignored due to null value",
				documentFieldName,
				collection.Name,
				document["_id"],
			)
			continue
		}

		destinationFName := fmt.Sprintf("%v%v", strings.ToUpper(string(documentFieldName[0])), string(documentFieldName[1:]))
		// Convert snake_case to camelCase
		if parts := strings.Split(destinationFName, "_"); len(parts) > 1 {
			if len(strings.TrimSpace(parts[0])) > 0 && len(strings.TrimSpace(parts[len(parts)-1])) > 0 {
				destinationFName = ""
				for _, fName := range parts {
					destinationFName += fmt.Sprintf(
						"%v%v",
						strings.ToUpper(string(fName[0])),
						string(fName[1:]),
					)
				}
			}
		}

		// Check field is save
		if utils.IsStringSliceContains(collection.StringDynamicField, documentFieldName) {
			if destinationFVReflect := ve.FieldByName(destinationFName); destinationFVReflect.CanSet() &&
				destinationFVReflect.Kind() == reflect.String {
				var strDynamic string
				switch reflect.TypeOf(documentFieldValue).Kind() {
				case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Ptr:
					valueRaw, err := json.Marshal(documentFieldValue)
					if err != nil {
						logger.BkLog.Errorw(fmt.Sprintf("Mgo - cannot Marshal documentFieldValue, detail: %v", err), "documentFieldName", documentFieldName)
					} else {
						strDynamic = string(valueRaw)
					}

				default:
					strDynamic = cast.ToString(documentFieldValue)
				}
				ve.FieldByName(destinationFName).SetString(string(strDynamic))
			}
			continue
		}

		if destinationFName == "_id" && len(collection.IDFieldName) > 0 {
			destinationFName = collection.IDFieldName
		}

		if destinationFVReflect := ve.FieldByName(destinationFName); destinationFVReflect.CanSet() {
			fieldValueString := cast.ToString(documentFieldValue)
			switch destinationFVReflect.Kind() {
			case reflect.Int64:
				if utils.IsStringSliceContains(collection.DateTimeFields, documentFieldName) {
					if reflect.TypeOf(documentFieldValue).String() == "time.Time" && documentFieldValue.(time.Time).Unix() >= 0 {
						ve.FieldByName(destinationFName).SetInt(documentFieldValue.(time.Time).Unix())
					}
				} else if v, err := strconv.ParseInt(fieldValueString, 10, 64); err == nil {
					ve.FieldByName(destinationFName).SetInt(v)
				}
			case reflect.Float64:
				if v, err := strconv.ParseFloat(fieldValueString, 64); err == nil {
					ve.FieldByName(destinationFName).SetFloat(v)
				}
			case reflect.Slice:
				embeddedDocuments := reflect.MakeSlice(destinationFVReflect.Type(), 0, 0)
				embeddedDocument, _ := mgt.getEmbeddedDocumentStruct(documentFieldName, collection)

				// If found embedded document and same type
				if embeddedDocument != nil && reflect.TypeOf(embeddedDocument) == destinationFVReflect.Type().Elem() {
					if reflect.TypeOf(documentFieldValue).Kind() == reflect.Slice {
						if values, ok := documentFieldValue.([]interface{}); values != nil && ok {
							for _, value := range values {
								embeddedDocument, collectionChild := mgt.getEmbeddedDocumentStruct(documentFieldName, collection)
								var valueMapData map[string]interface{}
								var canAssertion bool
								valueMapData, canAssertion = value.(map[string]interface{})
								if !canAssertion && collectionChild.FixDataMapFunc != nil {
									valueMapData, err = collectionChild.FixDataMapFunc(value)
									if err != nil {
										return errors.New(fmt.Sprintf("type of struct is not match - FixDataMapFunc: %s", err.Error()))
									}
									canAssertion = true
								}
								if canAssertion {
									err = mgt.ScanWithRecursive(valueMapData, embeddedDocument, collectionChild)
									embeddedDocuments = reflect.Append(embeddedDocuments, reflect.ValueOf(embeddedDocument))
								} else {
									return errors.New(fmt.Sprintf("type of struct is not match - value: %v, documentFieldName: %s, collectionName: %s",
										value, documentFieldName, collection.Name))
								}
							}

							ve.FieldByName(destinationFName).Set(embeddedDocuments)
						}
					} else if reflect.TypeOf(documentFieldValue).Kind() == reflect.Map {
						var values map[string]interface{}
						var ok bool
						if values, ok = documentFieldValue.(map[string]interface{}); values != nil && ok {
							for _, value := range values {
								var valueMapData map[string]interface{}
								var canAssertion bool
								embeddedDocumentChild, collectionChild := mgt.getEmbeddedDocumentStruct(documentFieldName, collection)
								valueMapData, canAssertion = value.(map[string]interface{})
								if !canAssertion && collectionChild.FixDataMapFunc != nil {
									valueMapData, err = collectionChild.FixDataMapFunc(value)
									if err != nil {
										return errors.New(fmt.Sprintf("type of struct is not match - FixDataMapFunc: %s", err.Error()))
									}
									canAssertion = true
								}
								if canAssertion {
									err = mgt.ScanWithRecursive(valueMapData, embeddedDocumentChild, collectionChild)
									embeddedDocuments = reflect.Append(embeddedDocuments, reflect.ValueOf(embeddedDocumentChild))
								} else {
									return errors.New("type of struct is not match")
								}
							}
							ve.FieldByName(destinationFName).Set(embeddedDocuments)
						}
					} else {
						logger.BkLog.Warnf("Field `%v` must be a Slice or Maps", documentFieldName)
					}
				} else {
					if v, ok := documentFieldValue.([]interface{}); v != nil && ok {
						if reflect.TypeOf(documentFieldValue).Kind() == reflect.Slice {
							sliceValueType := reflect.TypeOf(destinationFVReflect.Interface()).Elem().Kind()
							for i := 0; i <= len(v)-1; i++ {
								if vCorrected := mgt.correctDataType(v[i], sliceValueType); vCorrected != nil {
									embeddedDocuments = reflect.Append(embeddedDocuments, reflect.ValueOf(vCorrected))
								}
							}
							ve.FieldByName(destinationFName).Set(embeddedDocuments)
						}
					} else if values, ok := documentFieldValue.(map[string]interface{}); values != nil && ok {
						sliceValueType := reflect.TypeOf(destinationFVReflect.Interface()).Elem().Kind()
						for i := range values {
							if vCorrected := mgt.correctDataType(values[i], sliceValueType); vCorrected != nil {
								embeddedDocuments = reflect.Append(embeddedDocuments, reflect.ValueOf(vCorrected))
							}
						}
						ve.FieldByName(destinationFName).Set(embeddedDocuments)
					}
				}
			case reflect.Map:
				if reflect.TypeOf(documentFieldValue).Kind() == reflect.Map {
					mapField := ve.FieldByName(destinationFName)
					mapField.Set(reflect.MakeMap(mapField.Type()))

					embeddedStruct, _ := mgt.getEmbeddedDocumentStruct(documentFieldName, collection)
					if embeddedStruct != nil && reflect.TypeOf(embeddedStruct) == destinationFVReflect.Type().Elem() {
						mapValue, ok := documentFieldValue.(map[string]interface{})
						if ok {
							for mapK := range mapValue {
								embeddedStruct, collectionChild := mgt.getEmbeddedDocumentStruct(documentFieldName, collection)
								tmp := mapValue[mapK].(map[string]interface{})
								mgt.ScanWithRecursive(tmp, embeddedStruct, collectionChild)
								mapField.SetMapIndex(reflect.ValueOf(mapK), reflect.ValueOf(embeddedStruct))
							}
						}

					} else {
						mapVType := reflect.TypeOf(mapField.Interface()).Elem().Kind()
						for mapK, mapV := range documentFieldValue.(map[string]interface{}) {
							if vCorrected := mgt.correctDataType(mapV, mapVType); vCorrected != nil {
								mapField.SetMapIndex(reflect.ValueOf(mapK), reflect.ValueOf(vCorrected))
							}
						}
					}
				} else {
					logger.BkLog.Warnf("Field `%v` value `%v` is not maps", destinationFName, documentFieldValue)
				}
			case reflect.Ptr:
				if reflect.TypeOf(documentFieldValue).Kind() == reflect.Map {
					if embeddedDocumentStruct, collectionChild := mgt.getEmbeddedDocumentStruct(documentFieldName, collection); embeddedDocumentStruct != nil {
						if v, ok := documentFieldValue.(map[string]interface{}); ok {
							err = mgt.ScanWithRecursive(v, embeddedDocumentStruct, collectionChild)
							ve.FieldByName(destinationFName).Set(reflect.ValueOf(embeddedDocumentStruct))
						}
					}
				}
			default:
				if vCorrected := mgt.correctDataType(documentFieldValue, destinationFVReflect.Kind()); vCorrected != nil {
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
	}

	return
}

// GetSelectFields --
func (mgt MGTool) GetSelectFields(protoObj interface{}) (fields map[string]int8) {
	fields = make(map[string]int8)
	for _, field := range mgt.GetFields(protoObj) {
		if field == "Id" {
			fields["_id"] = 1
		} else {
			field := fmt.Sprintf("%v%v", strings.ToLower(string(field[0])), string(field[1:]))
			fields[field] = 1
		}
	}
	for fieldIgnore := range mgt.ignoreFields {
		delete(fields, fieldIgnore)
	}
	return
}

// GetSelectFields --
func (mgt MGTool) makeSelectFields(includeFields []string) (fields map[string]int8) {
	fields = make(map[string]int8)
	for _, field := range includeFields {
		if field == "id" || field == "Id" {
			fields["_id"] = 1
		} else {
			fields[field] = 1
		}
	}
	return
}

// GetFields --
func (mgt MGTool) GetFields(protoObj interface{}) (fields []string) {
	val := reflect.ValueOf(protoObj).Elem()
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		if utils.IsIgnoreThisField(typeField.Name) {
			continue
		}
		fields = append(fields, typeField.Name)
	}
	return
}

func (mgt MGTool) getEmbeddedDocumentStruct(fieldName string, collection *CollectionInfo) (proto.Message, *CollectionInfo) {
	if collection == nil {
		return nil, nil
	}
	for i := 0; i <= len(collection.EmbeddedDocumentFields)-1; i++ {
		if collection.EmbeddedDocumentFields[i].Name == fieldName {
			return collection.EmbeddedDocumentFields[i].Struct(), &collection.EmbeddedDocumentFields[i]
		}
	}

	return nil, nil
}

func (mgt MGTool) correctDataType(v interface{}, dataType reflect.Kind) interface{} {
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

func (mgt MGTool) getCreateFields(protoObj interface{}) (fields bson.M, err error) {
	return mgt.getCreateUpdateFields(protoObj, false, mgt.Collection, "", false, nil)
}

func (mgt MGTool) isIncludeField(parentFieldName, fieldName string, moreIncludes map[string]int8) bool {
	if parentFieldName != "" {
		fieldName = fmt.Sprintf("%s.%s", parentFieldName, fieldName)
	}
	if len(moreIncludes) > 0 {
		if _, ok := moreIncludes[fieldName]; ok {
			return true
		} else {
			return false
		}
	}
	if len(mgt.includeFields) > 0 {
		if _, ok := mgt.includeFields[fieldName]; ok {
			return true
		} else {
			return false
		}
	}

	return true
}

func (mgt MGTool) isIgnore(parentFieldName, fieldName string) bool {
	if len(mgt.ignoreFields) > 0 {
		if _, ok := mgt.ignoreFields[fieldName]; ok {
			return true
		}
	}
	return false
}

func (mgt MGTool) getCreateUpdateFields(protoObj interface{}, isUpdateFunc bool, collection *CollectionInfo,
	parentFieldName string, updateAll bool, moreIncludes map[string]int8) (fields bson.M, err error) {
	fields = make(bson.M)
	reflectValue := reflect.ValueOf(protoObj).Elem()
	numField := reflectValue.NumField()
	updatedFieldsBitMask := make([]bool, numField)
	isBitMaskUseless := false
	if isUpdateFunc {
		updatedFieldsBitMask, isBitMaskUseless = utils.GetUpdatedFieldsBitMask(reflectValue)
	}
	if isBitMaskUseless {
		//logger.BkLog.Debug("Use default updated field bit mask with all value is true=> update all field")
	}
	for i := 0; i < numField; i++ {
		var updateFieldName = parentFieldName
		sliceTag := utils.StringSlice(reflectValue.Type().Field(i).Tag.Get("json"), ",")
		if (len(sliceTag) > 0 && sliceTag[0] == "-") || len(sliceTag) == 0 {
			continue
		}

		reflectCurrentFieldValue := reflectValue.Field(i)

		if reflectCurrentFieldValue.Interface() == nil || reflectCurrentFieldValue.Interface() == 0 {
			continue
		}

		typeField := reflectValue.Type().Field(i)
		if utils.IsIgnoreThisField(typeField.Name) {
			continue
		}
		currentFieldName := strings.ToLower(string(typeField.Name[0])) + string(typeField.Name[1:])
		var jsonTag string
		if len(sliceTag) > 0 {
			jsonTag = sliceTag[0]
		} else {
			jsonTag = utils.ToSnake(currentFieldName)
		}
		if mgt.isIgnore(parentFieldName, jsonTag) || mgt.isIgnore(parentFieldName, currentFieldName) {
			continue
		}
		isUpdateAll := !isUpdateFunc || updateAll || mgt.isIncludeField(parentFieldName, jsonTag, moreIncludes) ||
			mgt.isIncludeField(parentFieldName, currentFieldName, moreIncludes)

		if isUpdateFunc && (!updatedFieldsBitMask[i]) {
			continue
		}
		var (
			embeddedStruct proto.Message
		)
		convertToSnakeCaseField := collection.ConvertToSnakeCaseField
		if utils.IsStringSliceContains(convertToSnakeCaseField, currentFieldName) {
			currentFieldName = jsonTag
		}
		// getCreateUpdateFields is not call recursive when parentFieldName is blank
		if updateFieldName != "" {
			updateFieldName = updateFieldName + "." + currentFieldName
		} else {
			updateFieldName = currentFieldName
		}

		embeddedStruct, collectionChild := mgt.getEmbeddedDocumentStruct(currentFieldName, collection)

		// if recursive call use ConvertToSnakeCaseField of embeddedStruct (convertToSnakeCaseField param)
		// otherwise use ConvertToSnakeCaseField of Collection (MGTool)

		switch {
		case typeField.Name == collection.IDFieldName || (len(collection.IDFieldName) < 1 && typeField.Name == "Id"):
			if isUpdateFunc {
				continue
			}
			idValue := reflectCurrentFieldValue.Interface()
			if collection.AutoIncrementId {
				idValue, err = mgt.IncrementIdGenerator(collection.Name)
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Error when increase _id, details: %v", err))
					return
				}
			}
			fields["_id"] = idValue

		case utils.IsStringSliceContains(collection.StringDynamicField, currentFieldName) && reflectCurrentFieldValue.Kind() == reflect.String:
			if isUpdateFunc && !isUpdateAll {
				continue
			}

			valueOut := map[string]interface{}{}
			fieldValue := reflectCurrentFieldValue.Interface().(string)
			if (fieldValue == `""` || fieldValue == "") && currentFieldName == "utmData" {
				fields[currentFieldName] = map[string]interface{}{}
			} else if fieldValue != "" && fieldValue != `""` {
				err := json.Unmarshal([]byte(reflectCurrentFieldValue.Interface().(string)), &valueOut)
				if err == nil {
					fields[currentFieldName] = valueOut
				} else {
					valueOut := make([]interface{}, 0)
					err := json.Unmarshal([]byte(reflectCurrentFieldValue.Interface().(string)), &valueOut)
					if err == nil {
						fields[currentFieldName] = valueOut
					} else {
						logger.BkLog.Warnf("getCreateUpdateFields - Cannot convert StringDynamicField (%v : %v) to map[string]interface{}, details: %v",
							currentFieldName, fieldValue, err)
					}
				}
			}

		case utils.IsStringSliceContains(collection.DateTimeFields, currentFieldName):
			if isUpdateFunc && !isUpdateAll {
				continue
			}
			if utils.IsStringSliceContains(collection.UpdatedAtFields, currentFieldName) {
				fields[currentFieldName] = time.Now()
			} else {
				if reflectCurrentFieldValue.Int() > 0 {
					fields[currentFieldName] = time.Unix(reflectCurrentFieldValue.Int(), 0)
				}
			}

		case embeddedStruct != nil:

			if reflectCurrentFieldValue.Kind() == reflect.Slice {
				if isUpdateAll {
					valueOut := []interface{}{}
					for i := 0; i < reflectCurrentFieldValue.Len(); i++ {
						refData := reflectCurrentFieldValue.Index(i)
						if refData.IsNil() {
							// TODO PROCESS SLICE FOR VALUE OF ELEMENT IS NULL
							valueOut = append(valueOut, defaultStructEmbedValue)
						} else {
							tmp, _ := mgt.getCreateUpdateFields(refData.Interface(), isUpdateFunc, collectionChild,
								updateFieldName, isUpdateAll, moreIncludes)
							valueOut = append(valueOut, mgt.fixDataUpdateSlice(tmp))
						}
					}
					fields[currentFieldName] = valueOut
				} else {
					for i := 0; i < reflectCurrentFieldValue.Len(); i++ {
						refData := reflectCurrentFieldValue.Index(i)
						if refData.IsNil() {
							// TODO PROCESS SLICE FOR VALUE OF ELEMENT IS NULL
							//fields[fmt.Sprintf("%s.%d", currentFieldName, i)] = defaultStructEmbedValue
						} else {
							tmp, _ := mgt.getCreateUpdateFields(refData.Interface(), isUpdateFunc, collectionChild,
								updateFieldName, isUpdateAll, moreIncludes)
							mgt.updateJsonFieldForUpdateOrCreateMap(fields, tmp, currentFieldName, strconv.Itoa(i), jsonTag, isUpdateFunc, isUpdateAll)
						}
					}
				}

				//valueOut = mgt.updateJsonFieldForUpdateOrCreate(fields, valueOut, currentFieldName, jsonTag, isUpdateFunc, shouldCheckChildField)
			} else if reflectCurrentFieldValue.Kind() == reflect.Ptr {
				a := fmt.Sprintf("%v", reflectCurrentFieldValue.Interface())
				if a == "<nil>" || reflectCurrentFieldValue.IsNil() {
					fields = mgt.updateJsonFieldForUpdateOrCreate(fields, nil, currentFieldName, jsonTag, isUpdateFunc, isUpdateAll)
				} else {
					if isUpdateAll {
						updateField, _ := mgt.getCreateUpdateFields(reflectCurrentFieldValue.Interface(), isUpdateFunc, collectionChild,
							updateFieldName, isUpdateAll, moreIncludes)
						if updateField == nil {
							fields[currentFieldName] = defaultStructEmbedValue
						} else {
							fields[currentFieldName] = updateField
						}
					} else {
						updateField, _ := mgt.getCreateUpdateFields(reflectCurrentFieldValue.Interface(), isUpdateFunc, collectionChild,
							updateFieldName, isUpdateAll, moreIncludes)
						fields = mgt.updateJsonFieldForUpdateOrCreate(fields, updateField, currentFieldName, jsonTag, isUpdateFunc, isUpdateAll)
					}

				}

			} else if reflectCurrentFieldValue.Kind() == reflect.Map {
				if isUpdateAll {
					valueOut := map[string]interface{}{}
					keys := reflectCurrentFieldValue.MapKeys()
					for _, key := range keys {
						refData := reflectCurrentFieldValue.MapIndex(key)
						if refData.IsNil() {
							// TODO PROCESS VALUE OF MAP FOR KEY IS NULL
						} else {
							tmp, _ := mgt.getCreateUpdateFields(refData.Interface(), isUpdateFunc, collectionChild,
								updateFieldName, isUpdateAll, moreIncludes)
							valueOut[key.Interface().(string)] = tmp
						}
					}
					fields[currentFieldName] = valueOut
				} else {
					valueOut := map[string]interface{}{}
					keys := reflectCurrentFieldValue.MapKeys()
					for _, key := range keys {
						refData := reflectCurrentFieldValue.MapIndex(key)
						if refData.IsNil() {
							// TODO PROCESS VALUE OF MAP FOR KEY IS NULL
						} else {
							tmp, _ := mgt.getCreateUpdateFields(refData.Interface(), isUpdateFunc, collectionChild,
								updateFieldName, isUpdateAll, moreIncludes)
							mgt.updateJsonFieldForUpdateOrCreate(valueOut, tmp, key.Interface().(string), jsonTag, isUpdateFunc, isUpdateAll)
						}
					}
					mgt.updateJsonFieldForUpdateOrCreate(fields, valueOut, currentFieldName, jsonTag, isUpdateFunc, isUpdateAll)
				}
			}

		default:
			if isUpdateFunc && !isUpdateAll {
				continue
			}
			if (reflectCurrentFieldValue.Kind() == reflect.Ptr || reflectCurrentFieldValue.Kind() == reflect.Map) &&
				reflectCurrentFieldValue.IsNil() {
				if isUpdateFunc {
					fields[currentFieldName] = defaultStructEmbedValue
				}
			} else {
				fields[currentFieldName] = reflectCurrentFieldValue.Interface()
			}
		}
	}
	return
}

func (mgt MGTool) getPushFields(protoObj interface{}, fieldName string, collection *CollectionInfo) (bson.M, error) {
	fields := make(bson.M)
	embeddedStruct, _ := mgt.getEmbeddedDocumentStruct(fieldName, collection)
	if embeddedStruct != nil {
		tmp, err := mgt.getCreateFields(protoObj)
		if err != nil {
			return tmp, err
		}
		fields[fieldName] = tmp
	} else if _, ok := protoObj.(proto.Message); ok {
		fields[fieldName] = mgt.getFields(protoObj)
	} else {
		fields[fieldName] = protoObj
	}
	return fields, nil
}

func (mgt MGTool) getFields(protoObj interface{}) bson.M {
	fields := make(bson.M)
	reflectValue := reflect.ValueOf(protoObj).Elem()
	numField := reflectValue.NumField()

	for i := 0; i < numField; i++ {
		jsonTag := utils.StringSlice(reflectValue.Type().Field(i).Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		reflectCurrentFieldValue := reflectValue.Field(i)

		if reflectCurrentFieldValue.Interface() == nil || reflectCurrentFieldValue.Interface() == 0 {
			continue
		}

		typeField := reflectValue.Type().Field(i)
		if utils.IsIgnoreThisField(typeField.Name) {
			continue
		}
		currentFieldName := strings.ToLower(string(typeField.Name[0])) + string(typeField.Name[1:])

		fields[currentFieldName] = reflectCurrentFieldValue.Interface()
	}
	return fields
}

// updateJsonFieldForUpdateOrCreate --
func (mgt MGTool) updateJsonFieldForUpdateOrCreateMap(baseFields bson.M, updateField bson.M,
	currentFieldName string, keyOfMap string, jsonTag string, isUpdateFunc bool, isUpdateAll bool) bson.M {

	// if update data is null. set currentField by default struct
	if updateField == nil {
		if !isUpdateFunc {
			baseFields[currentFieldName] = defaultStructEmbedValue
		}
		return baseFields
	}
	// in create case: only set value for parent field
	if !isUpdateFunc {
		baseFields[currentFieldName] = updateField
		return baseFields
	}
	for k, v := range updateField {
		var fieldUpdate string

		if keyOfMap == "" {
			fieldUpdate = fmt.Sprintf("%s.%s", currentFieldName, k)
		} else {
			fieldUpdate = fmt.Sprintf("%s.%s.%s", currentFieldName, keyOfMap, k)
		}
		baseFields[fieldUpdate] = v
	}
	return baseFields
}

// updateJsonFieldForUpdateOrCreate --
func (mgt MGTool) updateJsonFieldForUpdateOrCreate(baseFields bson.M, updateField bson.M,
	currentFieldName string, jsonTag string, isUpdateFunc bool, isUpdateAll bool) bson.M {
	return mgt.updateJsonFieldForUpdateOrCreateMap(baseFields, updateField, currentFieldName, "", jsonTag, isUpdateFunc, isUpdateAll)
}

func (mgt MGTool) getIterFromQuery(query mgo_access_layout.QueryAL, size int) mgo_access_layout.IterAL {
	if queryMgo, ok := query.(*mgo.Query); ok {
		return queryMgo.Batch(size).Iter()
	}
	if queryMock, ok := query.(*mgo_mock.Query); ok {
		return queryMock.Batch(size).Iter()
	}
	return nil
}

func (mgt MGTool) getIterFromPipe(pipe mgo_access_layout.PipeAL, size int) mgo_access_layout.IterAL {
	if pipeMgo, ok := pipe.(*mgo.Pipe); ok {
		return pipeMgo.Batch(size).Iter()
	}
	if pipeMock, ok := pipe.(*mgo_mock.Pipe); ok {
		return pipeMock.Batch(size).Iter()
	}
	return nil
}

// reflect helpers
func baseType(t reflect.Type, expected reflect.Kind) (reflect.Type, error) {
	t = Deref(t)
	if t.Kind() != expected {
		return nil, fmt.Errorf("expected %s but got %s", expected, t.Kind())
	}
	return t, nil
}

func (mgt MGTool) fixDataUpdateSlice(changes map[string]interface{}) bson.M {
	tmp := changes
	//var isCheck = make(map[string]bool)
	for k, v := range tmp {
		listKey := strings.Split(k, ".")
		if len(listKey) > 1 && strings.Index(k, k) == 0 {
			var split = strings.Split(k, ".")
			if len(split) < 2 {
				continue
			}
			tmp = mgt.changeValue(tmp, v, k, split[0], 0)
			delete(tmp, k)
		}
	}
	return tmp
}

func (mgt MGTool) fixChangeData(change map[string]interface{}, fieldName string) (map[string]interface{}, error) {
	tmp := change
	if change["$set"] != nil {
		var ok bool
		if tmp, ok = change["$set"].(bson.M); !ok {
			if tmp, ok = change["$set"].(map[string]interface{}); !ok {
				return change, errors.New("can't type assertion change set to bson.M - type: " + reflect.TypeOf(change["$set"]).String())
			}
		}
	}
	for k, v := range tmp {
		listKey := strings.Split(k, ".")
		if len(listKey) > 1 && strings.Index(k, fieldName) == 0 {
			tmp = mgt.changeValue(tmp, v, k, fieldName, 0)
			delete(tmp, k)
		}
	}
	change["$set"] = tmp
	return change, nil
}

func (mgt MGTool) changeValue(rootValue map[string]interface{}, value interface{}, key string, fieldName string, recursive int) map[string]interface{} {
	if recursive > 0 {
		keySP := strings.Split(key, ".")
		if len(keySP) <= 1 {
			rootValue[key] = value
			return rootValue
		}
		fieldName = keySP[0]
	}
	if value == nil {
		rootValue[fieldName] = struct{}{}
		return rootValue
	}
	var canAssertMap bool
	var newRootV = make(map[string]interface{})
	if rootValue[fieldName] == nil {
		rootValue[fieldName] = make(map[string]interface{})
	} else if _, canAssertMap = rootValue[fieldName].(map[string]interface{}); canAssertMap {
		newRootV = rootValue[fieldName].(map[string]interface{})
	} else if _, canAssertMap = rootValue[fieldName].(bson.M); canAssertMap {
		newRootV = rootValue[fieldName].(bson.M)
	}
	keyChild := strings.Replace(key, fieldName+".", "", 1)
	rootValue[fieldName] = mgt.changeValue(newRootV, value, keyChild, fieldName, recursive+1)
	return rootValue
}
