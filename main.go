package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"matrix/handler"
	"matrix/util"
	"net/http"
	"os"
	"os/signal"
	"time"
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
		fmt.Printf("Server Shutdown:", err)
		os.Exit(1)
	}
	fmt.Println("Server exiting")
}
