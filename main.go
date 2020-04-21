package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"matrix/dao"
	"matrix/handler"
	"matrix/util"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var level = []log.Level{log.TraceLevel, log.DebugLevel, log.InfoLevel, log.WarnLevel, log.ErrorLevel, log.FatalLevel, log.PanicLevel}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	log.SetOutput(os.Stdout)
	if util.ConfigInfo.Service.LogLevel < len(level) && util.ConfigInfo.Service.LogLevel >= 0 {
		log.SetLevel(level[util.ConfigInfo.Service.LogLevel])
	} else {
		log.SetLevel(log.InfoLevel)
	}
	f, err := os.Create("matrix.log")
	if err != nil {
		panic("创建日志文件失败")
	}
	log.SetOutput(f)
	dao.InitDB()
}
func main() {
	r := handler.InitRouter()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", util.ConfigInfo.Service.Port),
		Handler: r,
	}
	go func() {
		// 服务连接
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server Shutdown: %s", err)
		os.Exit(1)
	}
	fmt.Println("Server exiting")
}
