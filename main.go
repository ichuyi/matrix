package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"matrix/handler"
	"matrix/util"
	"net/http"
	"os"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}
func main() {
	r := handler.InitRouter()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", util.ConfigInfo.Service.Port),
		Handler: r,
	}
	server.ListenAndServe()
}
