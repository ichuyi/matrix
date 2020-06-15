package socket

//服务器向pc机发送消息
import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"matrix/util"
	"net"
	"sync"
	"time"
)

type doFunc func(string) (bool, string)
type TcpServer struct {
	Message  chan string
	client   []net.Conn
	lock     *sync.Mutex
	duration time.Duration
	timer    *time.Timer
	doFunc   doFunc
}

func NewServer(addr string, duration int, do doFunc) *TcpServer {
	server := new(TcpServer)
	server.Message = make(chan string, util.ConfigInfo.Socket.MaxMessage)
	server.client = make([]net.Conn, 0)
	server.lock = new(sync.Mutex)
	server.doFunc = do
	if duration > 0 {
		server.duration = time.Duration(duration) * time.Millisecond
		server.timer = time.NewTimer(server.duration)
	}
	go func() {
		defer close(server.Message)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("%s listen error: %s", addr, err.Error())
		}
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("%s accept error: %s", addr, err.Error())
			} else {
				server.lock.Lock()
				server.client = append(server.client, conn)
				server.lock.Unlock()
				log.Infof("%s success connect to %s", addr, conn.RemoteAddr())
				log.Infof("%s there have %d connections", addr, len(server.client))
			}
		}
	}()
	go func() {
		for m := range server.Message {
			if ok, message := server.doFunc(m); ok {
				if server.timer != nil {
					<-server.timer.C
					server.timer.Reset(server.duration)
				}
				for i := 0; i < len(server.client); {
					_, err := write(server.client[i], message+util.ConfigInfo.Socket.Delimiter)
					if err != nil {
						log.Errorf("%s socket send message failed,error: %s", addr, err.Error())
						log.Infof("%s close connection between :%s", addr, server.client[i].RemoteAddr())
						_ = server.client[i].Close()
						server.lock.Lock()
						server.client = append(server.client[0:i], server.client[i+1:]...)
						server.lock.Unlock()
						log.Infof("%s current there have %d connections", addr, len(server.client))
					} else {
						log.Infof("%s socket send message successfully to: %s", addr, server.client[i].RemoteAddr())
						i++
					}
				}
			}
		}
	}()
	return server
}

//func handleConn(conn net.Conn) {
//	for {
//		str, err := read(conn)
//		if err != nil {
//			if err == io.EOF {
//				log.Info("conn close")
//			} else {
//				log.Error("read error: %s", err.Error())
//			}
//			break
//		}
//		fmt.Println(str)
//		if str == util.ConfigInfo.Socket.Confirm {
//			var message string
//			if len(Message) > 0 {
//				message, _ = <-Message
//			} else {
//				message = ""
//			}
//			_, err := write(conn, message+util.ConfigInfo.Socket.Delimiter)
//			if err != nil {
//				log.Errorf("socket send message: %s,error: %s", message, err.Error())
//			} else {
//				log.Info("socket send message success")
//			}
//		}
//	}
//}
//
//func read(conn net.Conn) (string, error) {
//	readByte := make([]byte, 1)
//	var buffer bytes.Buffer
//	for {
//		_, err := conn.Read(readByte)
//		if err != nil {
//			return "", err
//		}
//		if string(readByte) == util.ConfigInfo.Socket.Delimiter {
//			break
//		}
//		buffer.Write(readByte)
//	}
//	return buffer.String(), nil
//
//}
func write(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + util.ConfigInfo.Socket.Delimiter)
	return conn.Write(buffer.Bytes())
}
