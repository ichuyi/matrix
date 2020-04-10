package main

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"io"
	"matrix/util"
	"net"
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
	f, err := os.Create("client.log")
	if err != nil {
		panic("创建日志文件失败")
	}
	log.SetOutput(f)
}
func main() {
	for i := 0; i < 5; i++ {
		go func() {
			conn, err := net.Dial("tcp", util.ConfigInfo.Socket.Laddr)
			if err != nil {
				log.Fatalf("socket connect err: %s", err.Error())
			}
			for {
				//_, err = write(conn, util.ConfigInfo.Socket.Confirm)
				//if err != nil {
				//	log.Errorf("socket send message: %s,error: %s", util.ConfigInfo.Socket.Confirm, err.Error())
				//	break
				//} else {
				//	//	log.Info("socket send message success")
				//}
				str, err := read(conn)
				if err != nil {
					if err == io.EOF {
						log.Info("conn close")
					} else {
						log.Error("read error: %s", err.Error())
					}
					break
				}
				if str != "" {
					log.Infof("receive:%s",str)
				}
			}
			conn.Close()
		}()
	}
	select {}
}
func write(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + util.ConfigInfo.Socket.Delimiter)
	return conn.Write(buffer.Bytes())
}
func read(conn net.Conn) (string, error) {
	readByte := make([]byte, 1)
	var buffer bytes.Buffer
	for {
		_, err := conn.Read(readByte)
		if err != nil {
			return "", err
		}
		if string(readByte) == util.ConfigInfo.Socket.Delimiter {
			break
		}
		buffer.Write(readByte)
	}
	return buffer.String(), nil

}
