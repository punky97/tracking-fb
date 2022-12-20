package mgo_mock

import (
	"github.com/punky97/go-codebase/core/utils"
	"encoding/json"
	"errors"
	"github.com/globalsign/mgo"
	"reflect"
)

type Session struct {
	queries []*QueryExpect
	inserts []*InsertCondition
	updates []*UpdateCondition
	upserts []*UpsertCondition
	removes []*RemoveCondition
}

func (s *Session) Close() {
	s.queries = make([]*QueryExpect, 0)
	s.inserts = make([]*InsertCondition, 0)
	s.updates = make([]*UpdateCondition, 0)
	s.upserts = make([]*UpsertCondition, 0)
	s.removes = make([]*RemoveCondition, 0)
}
func (*Session) Refresh() {

}
func (*Session) SetBatch(n int) {

}

func (s *Session) DB(name string) *Database {
	return &Database{Session: s}
}

type ExpectQueryResult struct {
	changeInfo *mgo.ChangeInfo
	result     interface{}
	err        error
}

type QueryApply struct {
	change *mgo.Change
	result *ExpectQueryResult
}

type QueryExpect struct {
	collectionName string
	query          interface{}
	limit          *int
	offset         *int
	sort           *[]string
	fields         []string
	result         *ExpectQueryResult
	resultApply    []*QueryApply
	absoluteMatch  bool
}

type UpdateCondition struct {
	collectionName string
	selector       interface{}
	update         interface{}
	changeInfo     *mgo.ChangeInfo
	err            error
}

type InsertCondition struct {
	collectionName string
	doc            interface{}
	err            error
}

type UpsertCondition struct {
	collectionName string
	selector       interface{}
	update         interface{}
	changeInfo     *mgo.ChangeInfo
	err            error
}

type RemoveCondition struct {
	collectionName string
	selector       interface{}
	changeInfo     *mgo.ChangeInfo
	err            error
}

func (q *QueryExpect) WithFields(fields ...string) *QueryExpect {
	q.fields = fields
	return q
}
func (q *QueryExpect) WithLimitOffset(limit, offset int) *QueryExpect {
	q.limit = &limit
	q.offset = &offset
	return q
}
func (q *QueryExpect) WithSort(sort []string) *QueryExpect {
	q.sort = &sort
	return q
}
func (q *QueryExpect) WithApply(change mgo.Change) *QueryApply {
	queryApply := &QueryApply{change: &change, result: &ExpectQueryResult{}}
	q.resultApply = append(q.resultApply, queryApply)
	return queryApply
}
func (q *QueryApply) WillReturnResult(res interface{}) *QueryApply {
	q.result.result = res
	return q
}

func (q *QueryApply) WillReturnChangeInfo(changeInfo *mgo.ChangeInfo) *QueryApply {
	q.result.changeInfo = changeInfo
	return q
}

func (q *QueryApply) WillReturnChangeError(err error) *QueryApply {
	q.result.err = err
	return q
}

func (q *QueryExpect) WillReturnResult(res interface{}) *QueryExpect {
	q.result = &ExpectQueryResult{result: res}
	return q
}
func (q *QueryExpect) WillReturnError(err error) *QueryExpect {
	q.result = &ExpectQueryResult{err: err}
	return q
}
func (q *QueryExpect) AbsoluteAccurateQuery() *QueryExpect {
	q.absoluteMatch = true
	return q
}

func (q *QueryExpect) WithQuery(queries interface{}) *QueryExpect {
	q.query = queries
	q.absoluteMatch = true
	return q
}
func (q *QueryExpect) WithContainsQuery(queries interface{}) *QueryExpect {
	q.query = queries
	q.absoluteMatch = false
	return q
}
func (q *QueryExpect) Match(realQuery QueryReal) (bool, interface{}, error) {
	if q.collectionName != realQuery.collectionName {
		return false, nil, errors.New("collection name is not match")
	}
	if q.limit != nil && *q.limit != realQuery.limit {
		return false, nil, errors.New("limit query not match")
	}
	if q.offset != nil && *q.offset != realQuery.offset {
		return false, nil, errors.New("offset query not match")
	}
	if q.sort != nil {
		if len(*q.sort) != len(realQuery.sort) {
			return false, nil, errors.New("len sort not match")
		}
		for i, sortField := range *q.sort {
			if sortField != realQuery.sort[i] {
				return false, nil, errors.New("field sort not match")
			}
		}
	}
	if len(q.fields) > 0 {
		if len(q.fields) != len(realQuery.fields) {
			return false, nil, errors.New("len field not match")
		}
		if len(utils.SubStringArrays(q.fields, realQuery.fields)) > 0 {
			return false, nil, errors.New("field not match")
		}
		if len(utils.SubStringArrays(realQuery.fields, q.fields)) > 0 {
			return false, nil, errors.New("field not match")
		}
	}
	if q.query != nil {
		if !compareInteface(q.query, realQuery.query, q.absoluteMatch) {
			return false, nil, errors.New("real query is not map")
		}
	}
	if q.result == nil {
		return true, nil, nil
	}
	return true, q.result.result, q.result.err
}

func (q *QueryExpect) matchApply(change mgo.Change) (bool, interface{}, *mgo.ChangeInfo, error) {
	if len(q.resultApply) < 1 {
		return false, nil, nil, errors.New("do not has any apply for match")
	}
	for _, apply := range q.resultApply {
		if reflect.DeepEqual(*apply.change, change) {
			return true, apply.result.result, apply.result.changeInfo, apply.result.err
		}
	}
	return false, nil, nil, errors.New("do not match any apply")
}

func matchMap(expect, src map[string]interface{}) bool {
	for k, v := range expect {
		if data, keyExist := src[k]; keyExist {
			if !compareInteface(v, data, false) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (s *Session) ExpectQuery(collectionName string) *QueryExpect {
	query := &QueryExpect{
		collectionName: collectionName,
	}
	s.queries = append(s.queries, query)
	return query
}

func (s *Session) ExpectInsert(collectionName string) *InsertCondition {
	insert := &InsertCondition{
		collectionName: collectionName,
	}
	s.inserts = append(s.inserts, insert)
	return insert

}

func (s *Session) ExpectUpsert(collectionName string) *UpsertCondition {
	upsert := &UpsertCondition{
		collectionName: collectionName,
	}
	s.upserts = append(s.upserts, upsert)
	return upsert

}
func (s *Session) ExpectRemove(collectionName string) *RemoveCondition {
	remove := &RemoveCondition{
		collectionName: collectionName,
	}
	s.removes = append(s.removes, remove)
	return remove

}
func (s *Session) ExpectUpdate(collectionName string) *UpdateCondition {
	upsert := &UpdateCondition{
		collectionName: collectionName,
	}
	s.updates = append(s.updates, upsert)
	return upsert

}
func (u *UpdateCondition) WithSelector(doc interface{}) *UpdateCondition {
	u.selector = doc
	return u

}
func (u *UpdateCondition) WithUpdate(update interface{}) *UpdateCondition {
	u.update = update
	return u
}

func (u *UpdateCondition) WillReturnError(err error) *UpdateCondition {
	u.err = err
	return u
}
func (u *UpdateCondition) WillReturnChangeInfo(changeInfo *mgo.ChangeInfo) *UpdateCondition {
	u.changeInfo = changeInfo
	return u
}

func (u *UpsertCondition) WithSelector(selector interface{}) *UpsertCondition {
	u.selector = selector
	return u

}
func (u *UpsertCondition) WithUpdate(update interface{}) *UpsertCondition {
	u.update = update
	return u
}

func (u *UpsertCondition) WillReturnError(err error) *UpsertCondition {
	u.err = err
	return u
}
func (u *UpsertCondition) WillReturnChangeInfo(changeInfo *mgo.ChangeInfo) *UpsertCondition {
	u.changeInfo = changeInfo
	return u
}
func (u *RemoveCondition) WithSelector(doc interface{}) *RemoveCondition {
	u.selector = doc
	return u

}
func (u *RemoveCondition) WillReturnError(err error) *RemoveCondition {
	u.err = err
	return u
}
func (u *RemoveCondition) WillReturnChangeInfo(changeInfo *mgo.ChangeInfo) *RemoveCondition {
	u.changeInfo = changeInfo
	return u
}

func (u *InsertCondition) WithDocument(doc interface{}) *InsertCondition {
	u.doc = doc
	return u

}
func (u *InsertCondition) WillReturnError(err error) *InsertCondition {
	u.err = err
	return u
}

func (u *UpdateCondition) match(doc, update interface{}) (match bool, changeInfo *mgo.ChangeInfo, err error) {
	if u.selector != nil && !compareInteface(u.selector, doc, false) {
		return false, nil, errors.New("selector not match")
	}
	if u.update != nil && update != nil && !compareInteface(u.update, update, false) {
		return false, nil, errors.New("update not match")
	}
	return true, u.changeInfo, u.err
}

func (u *InsertCondition) match(doc interface{}) (match bool, err error) {
	if u.doc != nil && !compareInteface(u.doc, doc, false) {
		return false, errors.New("selector not match")
	}
	return true, u.err
}

func (u *UpsertCondition) match(doc, update interface{}) (match bool, changeInfo *mgo.ChangeInfo, err error) {
	if u.selector != nil && !compareInteface(u.selector, doc, false) {
		return false, nil, errors.New("selector not match")
	}
	if u.update != nil && update != nil && !compareInteface(u.update, update, false) {
		return false, nil, errors.New("update not match")
	}
	return true, u.changeInfo, u.err
}

func (u *RemoveCondition) match(doc interface{}) (match bool, changeInfo *mgo.ChangeInfo, err error) {
	if u.selector != nil && !compareInteface(u.selector, doc, false) {
		return false, nil, errors.New("selector not match")
	}
	return true, u.changeInfo, u.err
}

func (s *Session) matchUpdate(doc interface{}, change interface{}) (bool, error) {
	match, _, errDoc := matchUpdateAll(s.updates, doc, nil)
	return match, errDoc
}

func (s *Session) matchUpdateAll(doc interface{}, change interface{}) (bool, *mgo.ChangeInfo, error) {
	return matchUpdateAll(s.updates, doc, nil)
}

func (s *Session) matchInsert(docs ...interface{}) error {
	var err error
	if len(docs) < 1 {
		return nil
	}
	for _, doc := range docs {
		if match, errDoc := matchInsertAll(s.inserts, doc); !match {
			err = errDoc
			break
		}
	}
	return err
}

func (s *Session) matchUpserts(doc interface{}, change interface{}) (bool, *mgo.ChangeInfo, error) {
	return matchUpsertAll(s.upserts, doc, change)
}

func (s *Session) matchRemove(doc interface{}, change interface{}) (bool, *mgo.ChangeInfo, error) {
	return matchRemoveAll(s.removes, doc)
}

func matchUpdateAll(changeConditions []*UpdateCondition, doc, update interface{}) (bool, *mgo.ChangeInfo, error) {
	var errRes error
	for _, changeInfo := range changeConditions {
		if match, change, err := changeInfo.match(doc, update); match {
			return match, change, err
		} else {
			errRes = err
		}
	}
	return false, nil, errRes
}

func matchInsertAll(changeConditions []*InsertCondition, doc interface{}) (bool, error) {
	var errRes error
	if len(changeConditions) < 1 {
		return false, errors.New("len of insert expect equal 0")
	}
	for _, changeInfo := range changeConditions {
		if match, err := changeInfo.match(doc); match {
			return match, err
		} else {
			errRes = err
		}
	}
	return false, errRes
}

func matchUpsertAll(changeConditions []*UpsertCondition, selector, update interface{}) (bool, *mgo.ChangeInfo, error) {
	var errRes error
	if len(changeConditions) < 1 {
		return false, nil, errors.New("len of upsert expect equal 0")
	}
	for _, changeInfo := range changeConditions {
		if match, changeInfo, err := changeInfo.match(selector, update); match {
			return match, changeInfo, err
		} else {
			errRes = err
		}
	}
	return false, nil, errRes
}

func matchRemoveAll(changeConditions []*RemoveCondition, selector interface{}) (bool, *mgo.ChangeInfo, error) {
	var errRes error
	if len(changeConditions) < 1 {
		return false, nil, errors.New("len of upsert expect equal 0")
	}
	for _, changeInfo := range changeConditions {
		if match, changeInfo, err := changeInfo.match(selector); match {
			return match, changeInfo, err
		} else {
			errRes = err
		}
	}
	return false, nil, errRes
}

func compareInteface(expect, real interface{}, absolute bool) bool {
	if !absolute {
		if expectMap, errConvertExpect := convertToMap(expect); errConvertExpect == nil {
			if realMap, errConvertReal := convertToMap(real); errConvertReal == nil {
				return matchMap(realMap, expectMap)
			}
		}
	}
	strS1, _ := json.Marshal(expect)
	strS2, _ := json.Marshal(real)
	return string(strS1) == string(strS2)
}

func convertToMap(data interface{}) (map[string]interface{}, error) {
	dataByte, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	res := make(map[string]interface{})
	err = json.Unmarshal(dataByte, &res)
	return res, err
}
