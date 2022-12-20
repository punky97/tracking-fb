package main

import (
	"github.com/punky97/go-codebase/core/apimono"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/spf13/viper"
	"net/http"
	"tracking-fb/cmd/api/tracking-api/app"
	"tracking-fb/pkg/codename"
)

var version string

func main() {
	bkMicroApp, hs := apimono.NewAPIApp(
		codename.TrackingApi.CodeName,
		codename.TrackingApi.HTTPBasePath,
		version,
		onClose,
	)

	logger.BkLog.Infof("Mysql: %v", viper.GetString("mysql.host"))

	// init server
	app.NewServer(bkMicroApp)

	// register micro app to service discovery (consul)
	defer onClose(bkMicroApp)

	// run api
	err := bkMicroApp.RunAPI(hs)
	if err != nil {
		if err.Error() == http.ErrServerClosed.Error() {
			logger.BkLog.Info(http.ErrServerClosed.Error())
		} else {
			logger.BkLog.Errorf("HTTP server closed with error: %v", err)
		}
	}
}

func onClose(bkMicroApp *apimono.App) {
	app.OnClose()
	bkMicroApp.Shutdown()
}
