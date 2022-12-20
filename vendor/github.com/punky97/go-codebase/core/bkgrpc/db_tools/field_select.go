package db_tools

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"github.com/punky97/go-codebase/core/utils"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
	"strconv"
)

func UnaryClientInterceptor(useDefaultLimit, useDefaultFieldSelect bool) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		fieldSelectOption, options := filterCallOptionFieldSelects(opts)
		appliedOption := appliedFieldSelectOption(fieldSelectOption)
		ctx = updateMetadata(ctx, useDefaultLimit, useDefaultFieldSelect, appliedOption)
		return invoker(ctx, method, req, reply, cc, options...)
	}
}

func StreamClientInterceptor(isDefaultLimit, isDefaultFieldSelect bool) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		fieldSelectOption, options := filterCallOptionFieldSelects(opts)
		appliedOption := appliedFieldSelectOption(fieldSelectOption)
		ctx = updateMetadata(ctx, isDefaultLimit, isDefaultFieldSelect, appliedOption)
		return streamer(ctx, desc, cc, method, options...)
	}
}

func updateMetadata(ctx context.Context, useDefaultLimit, useServiceDefaultField bool, option *options) context.Context {
	header := metautils.ExtractOutgoing(ctx).Clone()

	// limit
	if option.limit > 0 {
		header.Set(utils.LimitHeader, strconv.FormatInt(option.limit, 10))
	} else if useDefaultLimit {
		header.Set(utils.DefaultLimitHeader, strconv.FormatBool(useDefaultLimit))
	}

	// offset
	if option.offset > 0 {
		header.Set(utils.OffsetHeader, strconv.FormatInt(option.offset, 10))
	}

	// select fields
	if option.useAllField {
		header.Set(utils.UseAllFieldHeader, strconv.FormatBool(option.useAllField))
	} else {
		if len(option.ignoreFields) > 0 {
			for _, f := range option.ignoreFields {
				header.Add(utils.UseAllWithoutFieldHeader, f)
			}
		} else if len(option.fields) > 0 {
			for _, f := range option.fields {
				header.Add(utils.FieldSelectHeader, f)
			}
		} else if useServiceDefaultField {
			header.Set(utils.DefaultSelectHeader, cast.ToString(useServiceDefaultField))
		}
	}

	// force sql source
	if option.sqlSource != "" {
		header.Add(meta.XCtxData, meta.CtxSqlDB+"="+option.sqlSource)
	}

	return header.ToOutgoing(ctx)
}
