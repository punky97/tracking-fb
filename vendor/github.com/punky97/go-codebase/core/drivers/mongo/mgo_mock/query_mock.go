package mgo_mock

import (
	"github.com/punky97/go-codebase/core/logger"
	"encoding/json"
	"errors"
	"github.com/globalsign/mgo"
)

type Query struct {
	Session *Session
	query   *QueryReal
}

type QueryReal struct {
	collectionName string
	query          interface{}
	limit          int
	offset         int
	sort           []string
	fields         []string
	change         *mgo.Change
}

func (q *Query) One(result interface{}) (err error) {
	var errRes error
	if len(q.Session.queries) < 1 {
		return errors.New("not has any queries expect")
	}
	for _, query := range q.Session.queries {
		if match, res, err := query.Match(*q.query); match {
			if res == nil {
				errRes = errors.New("result is null")
				continue
			}
			if err != nil {
				return err
			}

			dataB, err := json.Marshal(res)
			if err != nil {
				return err
			}
			return json.Unmarshal(dataB, result)
		} else {
			errRes = err
		}
	}
	return errRes
}

func (q *Query) All(result interface{}) error {
	return q.One(result)
}

func (q *Query) Apply(change mgo.Change, result interface{}) (*mgo.ChangeInfo, error) {
	var errRes error
	if len(q.Session.queries) < 1 {
		return nil, errors.New("not has any queries expect")
	}
	for _, query := range q.Session.queries {
		if match, _, errMatch := query.Match(*q.query); match {
			if matchChange, res, changeInfo, errMatchApply := query.matchApply(change); matchChange {
				if errMatchApply != nil {
					return changeInfo, errMatchApply
				}
				dataB, err := json.Marshal(res)
				if err != nil {
					return changeInfo, err
				}
				return changeInfo, json.Unmarshal(dataB, &result)
			} else {
				errRes = errMatchApply
			}
		} else {
			errRes = errMatch
		}
	}
	return nil, errRes
}

func (q *Query) Batch(result interface{}) *Query {
	return q
}

func (q *Query) Iter() *Iter {
	data := make([]interface{}, 0)
	err := q.One(data)
	if err != nil {
		logger.BkLog.Error("Error when query: %v", err)
	}
	return &Iter{
		Result: data,
	}
}

func (q *Query) Limit(n int) *Query {
	q.query.limit = n
	return q
}
func (q *Query) Skip(n int) *Query {
	q.query.offset = n
	return q
}
func (q *Query) Sort(field string) *Query {
	q.query.sort = append(q.query.sort, field)
	return q
}
func (q *Query) Select(fields map[string]int8) *Query {
	fieldStr := make([]string, 0)
	for field := range fields {
		fieldStr = append(fieldStr, field)
	}
	q.query.fields = fieldStr
	return q
}
