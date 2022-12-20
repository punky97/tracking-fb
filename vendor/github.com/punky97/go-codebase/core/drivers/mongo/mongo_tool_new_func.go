package mongo

import (
	"github.com/punky97/go-codebase/core/drivers/mongo/mgo_access_layout"
	"github.com/punky97/go-codebase/core/logger"
	"context"
	"errors"
	"fmt"
	"reflect"
)

func (mgt MGTool) toSlice(result []map[string]interface{}, dest interface{}) error {
	value := reflect.ValueOf(dest)
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

	for _, data := range result {
		vp := mgt.initData(base)
		err := mgt.Scan(data, vp.Interface())
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error while scan, details: %v", err))
		}
		if isPtr {
			direct.Set(reflect.Append(direct, vp))
		} else {
			direct.Set(reflect.Append(direct, reflect.Indirect(vp)))
		}
	}
	return nil
}

func (mgt MGTool) GetAll(queries interface{}, dest interface{}, limit, offset int, sort []string, fields ...string) (err error) {
	value := reflect.ValueOf(dest)
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
	base := Deref(slice.Elem())

	query := mgt.getQuery(queries, mgt.initData(base).Interface(), limit, offset, sort, fields...)

	result := make([]map[string]interface{}, 0)
	err = query.All(&result)
	if err != nil {
		return err
	}
	return mgt.toSlice(result, dest)
}

func (mgt MGTool) GetAllPipe(queries interface{}, dest interface{}, limit, offset int, sort []string, fields ...string) (err error) {
	value := reflect.ValueOf(dest)
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
	base := Deref(slice.Elem())

	query := mgt.getQuery(queries, mgt.initData(base).Interface(), limit, offset, sort, fields...)

	result := make([]map[string]interface{}, 0)
	err = query.All(&result)
	return mgt.toSlice(result, dest)
}

func (mgt MGTool) GetAllMap(queries interface{}, dest []map[string]interface{}, limit, offset int, sort []string, fields ...string) (err error) {
	query := mgt.getQuery(queries, nil, limit, offset, sort, fields...)
	err = query.All(dest)
	return err
}

func (mgt MGTool) GetOneMap(queries interface{}, dest map[string]interface{}, limit, offset int, sort []string, fields ...string) (err error) {
	query := mgt.getQuery(queries, nil, limit, offset, sort, fields...)
	err = query.One(dest)
	return err
}

func (mgt MGTool) GetOne(queries interface{}, dest interface{}, offset int, sort []string, fields ...string) (err error) {
	destType := reflect.TypeOf(dest)
	if Deref(destType.Elem()).Kind() != reflect.Struct || destType.Kind() != reflect.Ptr {
		return errors.New("dest must pointer of struct")
	}
	query := mgt.getQuery(queries, dest, 0, offset, sort, fields...)
	result := make(map[string]interface{})
	err = query.One(&result)
	if err != nil {
		return err
	}
	err = mgt.Scan(result, dest)
	if err != nil {
		return errors.New("error when scan from obj - " + err.Error())
	}
	return nil
}

func (mgt MGTool) initData(dataType reflect.Type) reflect.Value {
	base := Deref(dataType)
	return reflect.New(base)
}

func (mgt MGTool) StreamOption(destType reflect.Type, query mgo_access_layout.QueryAL, pipe mgo_access_layout.PipeAL, size int,
	breakChan chan bool, ctx context.Context) (
	chDestO chan interface{}, chErrorO chan error, err error) {
	if query == nil && pipe == nil {
		return nil, nil, errors.New("query or pipe must be not null")
	}
	var iter mgo_access_layout.IterAL

	if query != nil {
		iter = mgt.getIterFromQuery(query, size)
	} else if pipe != nil {
		iter = mgt.getIterFromPipe(pipe, size)
	}
	chDestO = make(chan interface{}, 5)
	chErrorO = make(chan error, 5)
	go func(chDest chan interface{}, errorChan chan error) {
		defer func() {
			iter.Close()
			chDest <- nil
		}()
		var isMap = destType.Kind() == reflect.Map
		var isSlice = destType.Kind() == reflect.Slice
		if isSlice {
			destType = destType.Elem()
		}
		var slice = reflect.MakeSlice(reflect.SliceOf(destType), 0, 0)
		isPtr := destType.Kind() == reflect.Ptr
		base := Deref(destType)

		dataOrigin := make(map[string]interface{})

		for iter.Next(dataOrigin) {
			data := make(map[string]interface{})
			for k, v := range dataOrigin {
				data[k] = v
			}
			if isMap {
				chDest <- data
				continue
			}
			var vp = reflect.New(base)
			err := mgt.Scan(data, vp.Interface())
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
		if slice.Len() > 0 {
			chDest <- slice.Interface()
		}
	}(chDestO, chErrorO)
	return
}

// StreamOne - select data from mongodb with stream process
// args:
//    - dest: dest type you want to response
//    - query: sql query
//    - ctx: context of stream. when ctx done then close connection and channel
//    - execute: func execute data return of stream. If this func return error != nil, stream will be close
func (mgt MGTool) StreamOne(dest interface{}, query mgo_access_layout.QueryAL, pipe mgo_access_layout.PipeAL, size int,
	ctx context.Context, execute StreamExecute) error {
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Struct && Deref(destType).Kind() != reflect.Struct {
		return errors.New("dest must be struct or ptr of struct")
	}
	breakChan := make(chan bool, 1)
	dataChan, errChan, err := mgt.StreamOption(destType, query, pipe, size, breakChan, ctx)
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

// StreamOne - select data from mongodb with stream process
// args:
//    - dest: dest type you want to response
//    - query: sql query
//    - ctx: context of stream. when ctx done then close connection and channel
//    - execute: func execute data return of stream. If this func return error != nil, stream will be close
func (mgt MGTool) StreamOneMap(dest interface{}, query mgo_access_layout.QueryAL, pipe mgo_access_layout.PipeAL, size int,
	ctx context.Context, execute StreamExecute) error {
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Map {
		return errors.New("dest must be map")
	}
	breakChan := make(chan bool, 1)
	dataChan, errChan, err := mgt.StreamOption(destType, query, pipe, size, breakChan, ctx)
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

// StreamSlice - select data from mongodb with stream process
// args:
//    - dest: dest type you want to response
//    - query: sql query
//    - size: size of slice you want to rev
//    - ctx: context of stream. when ctx done then close connection and channel
//    - execute: func execute data return of stream. If this func return error != nil, stream will be close
func (st MGTool) StreamSlice(dest interface{}, query mgo_access_layout.QueryAL, pipe mgo_access_layout.PipeAL, size int,
	ctx context.Context, execute StreamExecute) error {
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Slice && Deref(destType.Elem()).Kind() != reflect.Struct {
		return errors.New("dest must be slice of struct or slice ptr of struct")
	}
	breakChan := make(chan bool, 1)
	dataChan, errChan, err := st.StreamOption(destType, query, pipe, size, breakChan, ctx)
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
