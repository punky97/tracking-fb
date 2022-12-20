package utils

import (
	"context"
	"time"
)

// GetContext get normal context
func GetContext() context.Context {
	return context.Background()
}

// GetContextWithTimeout2 --
func GetContextWithTimeout2(timeout time.Duration) (context.Context, func()) {
	return GetContextWithTimeout(context.Background(), timeout)
}

// GetContextWithTimeout --
func GetContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, func()) {
	return context.WithTimeout(parent, timeout)
}

// GetContextWithValue --
func GetContextWithValue(parent context.Context, key interface{}, value interface{}) context.Context {
	return context.WithValue(parent, key, value)
}

// simpleCtxKey --
type simpleCtxKey string

// GetContextWithValue2 --
func GetContextWithValue2(parent context.Context, key string, value interface{}) context.Context {
	return context.WithValue(parent, simpleCtxKey(key), value)
}

// StringLength --
const StringLength = 20

// GetContextWithRandomKV --
func GetContextWithRandomKV(parent context.Context) context.Context {
	k, _ := GenerateRandomString(StringLength)
	v, _ := GenerateRandomString(StringLength)
	return GetContextWithValue2(parent, k, v)
}

// GetContextWithRandomV --
func GetContextWithRandomV(parent context.Context, key string) context.Context {
	v, _ := GenerateRandomString(StringLength)
	return GetContextWithValue2(parent, key, v)
}

// GetContextWithRandomK --
func GetContextWithRandomK(parent context.Context, val interface{}) context.Context {
	k, _ := GenerateRandomString(StringLength)
	return GetContextWithValue2(parent, k, val)
}

// GetContextWithCancel --
func GetContextWithCancel(parent context.Context) (context.Context, func()) {
	return context.WithCancel(parent)
}

// GetContextWithCancelSafe --
func GetContextWithCancelSafe(parent context.Context) (context.Context, func()) {
	p := GetContextWithRandomKV(parent)
	return context.WithCancel(p)
}

// GetContextWithCancelSafe2 --
func GetContextWithCancelSafe2() (context.Context, func()) {
	p := GetContextWithRandomKV(context.Background())
	return context.WithCancel(p)
}

var dataKey = simpleCtxKey("data-key")

func GetCtxData(ctx context.Context, k string) string {
	if val := ctx.Value(dataKey); val != nil {
		if valMap, ok := val.(map[string]string); ok {
			return valMap[k]
		}
	}

	return ""
}

func SetCtxData(ctx context.Context, k string, v string) {
	if val := ctx.Value(dataKey); val != nil {
		if valMap, ok := val.(map[string]string); ok {
			valMap[k] = v
		}
	}
}

func NewDataContext(ctx context.Context) context.Context {
	if val := ctx.Value(dataKey); val != nil {
		return ctx
	}
	return context.WithValue(ctx, dataKey, make(map[string]string))
}
