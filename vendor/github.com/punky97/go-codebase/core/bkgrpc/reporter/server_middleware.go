package reporter

import "google.golang.org/grpc"

var (
	defaultServerMetrics *ServerMetrics
)

func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	defaultServerMetrics = NewServerMetrics(serviceName)
	return defaultServerMetrics.UnaryServerInterceptor()
}

func StartServerCollector() {
	if defaultServerMetrics != nil {
		defaultServerMetrics.Start()
	}
}

func StopServerCollector() {
	if defaultServerMetrics != nil {
		defaultServerMetrics.Close()
	}
}
