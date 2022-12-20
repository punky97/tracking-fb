package db_tools

import "google.golang.org/grpc"

func WithFieldSelect(fields ...string) CallOptionFieldSelect {
	return CallOptionFieldSelect{
		applyFunc: func(opt *options) {
			if len(fields) > 0 {
				opt.fields = fields
				// clear ignore fields
				opt.ignoreFields = nil
			}
		},
	}
}

func UseAllField() CallOptionFieldSelect {
	return CallOptionFieldSelect{
		applyFunc: func(opt *options) {
			opt.useAllField = true
		},
	}
}
func UseAllWithoutField(ignoreFields ...string) CallOptionFieldSelect {
	return CallOptionFieldSelect{
		applyFunc: func(opt *options) {
			if len(ignoreFields) > 0 {
				opt.ignoreFields = ignoreFields
				// clear select field
				opt.fields = nil
			}
		},
	}
}

func SetOffsetLimit(offset int64, limit int64) CallOptionFieldSelect {
	return CallOptionFieldSelect{
		applyFunc: func(opt *options) {
			if limit > 0 {
				opt.limit = limit
			}

			if offset > 0 {
				opt.offset = offset
			}
		},
	}
}

func WithSqlMaster() CallOptionFieldSelect {
	return CallOptionFieldSelect{
		applyFunc: func(opt *options) {
			opt.sqlSource = "master"
		},
	}
}

type CallOptionFieldSelect struct {
	grpc.EmptyCallOption // make sure we implement private after() and before() fields so we don't panic.
	//opt                  *options
	applyFunc func(*options)
}

type options struct {
	fields       []string
	ignoreFields []string
	useAllField  bool
	limit        int64
	offset       int64
	sqlSource    string
}

func filterCallOptionFieldSelects(callOptions []grpc.CallOption) (option []CallOptionFieldSelect, others []grpc.CallOption) {
	for _, opt := range callOptions {
		if co, ok := opt.(CallOptionFieldSelect); ok {
			option = append(option, co)
		} else {
			others = append(others, opt)
		}
	}

	return option, others
}

func appliedFieldSelectOption(callOptions []CallOptionFieldSelect) *options {
	if len(callOptions) == 0 {
		return &options{}
	}

	optCopy := &options{}
	for _, f := range callOptions {
		f.applyFunc(optCopy)
	}
	return optCopy
}
