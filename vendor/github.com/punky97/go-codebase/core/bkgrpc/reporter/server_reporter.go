// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.
package reporter

import (
	"github.com/punky97/go-codebase/core/bkgrpc/common"
	"time"

	"google.golang.org/grpc/codes"
)

type grpcType string

const (
	Unary        grpcType = "unary"
	ClientStream grpcType = "client_stream"
	ServerStream grpcType = "server_stream"
	BidiStream   grpcType = "bidi_stream"
)

type serverReporter struct {
	metrics          *ServerMetrics
	rpcType          grpcType
	serviceName      string
	methodName       string
	endpoint         string
	reportEndpointMt *EndpointMetrics
	alertEndPointMt  *EndpointMetrics
	startTime        time.Time
}

func newServerReporter(m *ServerMetrics, rpcType grpcType, fullMethod string) *serverReporter {
	r := &serverReporter{
		metrics: m,
		rpcType: rpcType,
	}
	r.startTime = time.Now()
	r.serviceName, r.methodName = common.SplitMethodName(fullMethod)
	r.endpoint = fullMethod

	r.reportEndpointMt = r.metrics.reportMetrics.GetOrCreateEndpointMetrics(r.endpoint)
	r.alertEndPointMt = r.metrics.alertMetrics.GetOrCreateEndpointMetrics(r.endpoint)

	r.reportEndpointMt.Call()
	r.alertEndPointMt.Call()

	return r
}

func (r *serverReporter) ReceivedMessage(size int) {
	r.reportEndpointMt.ReceivedMessage(size)
	r.alertEndPointMt.ReceivedMessage(size)
}

func (r *serverReporter) SentMessage(size int) {
	r.reportEndpointMt.SentMessage(size, r.startTime)
	r.alertEndPointMt.SentMessage(size, r.startTime)
}

func (r *serverReporter) Handled(code codes.Code) {
	r.reportEndpointMt.Handled(code)
	r.alertEndPointMt.Handled(code)
}
