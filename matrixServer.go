package main

//服务器向pc机发送消息
import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	hooks "matrix/hook"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ServerConfig struct {
	Addr           string `json:"addr"`
	MaxCommand     int    `json:"max_command"`
	Duration       int64  `json:"duration"`
	MaxSendTimes   int    `json:"max_send_times"`
	ReportDuration int    `json:"report_duration"`
}

var configPath = "matrixConfig.json"
var ConfigInfo ServerConfig
var quit chan os.Signal

type MotorResult struct {
	success bool
	lock    *sync.RWMutex
}
type Command struct {
	id       string
	content  string
	duration int64
}
type MotorServer struct {
	command       map[string]chan *Command
	commandLock   *sync.RWMutex
	numberOfMotor int64
	connections   int64
	ctx           context.Context
	cancel        context.CancelFunc
	startTime     time.Time
	motorPowerOn  int64
}

func NewMotorServer() *MotorServer {
	m := new(MotorServer)
	m.command = make(map[string]chan *Command, ConfigInfo.MaxCommand)
	m.commandLock = new(sync.RWMutex)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.startTime = time.Now()
	return m
}

var motorServer *MotorServer

func parse() {
	conf, err := os.Open(configPath)
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = json.NewDecoder(conf).Decode(&ConfigInfo)
	if err != nil {
		log.Fatalf(err.Error())
	}
}
func initLog() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.AddHook(hooks.NewContextHook())
	f, err := os.Create("server.log")
	if err != nil {
		panic("创建日志文件失败")
	}
	log.SetOutput(f)
}
func init() {
	parse()
	initLog()
	motorServer = NewMotorServer()
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()
	quit = make(chan os.Signal)
	listener, err := net.Listen("tcp", ConfigInfo.Addr)
	if err != nil {
		log.Fatalf("%s listen error: %s", ConfigInfo.Addr, err.Error())
	}
	log.Infof("start to listen, addr is: %s", ConfigInfo.Addr)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("%s accept error: %s", ConfigInfo.Addr, err.Error())
				return
			} else {
				atomic.AddInt64(&motorServer.connections, 1)
				log.Debugf("%d connections", motorServer.connections)
				go handlerConn(conn)
			}
		}
	}()
	signal.Notify(quit)
	select {
	case <-quit:
	case <-motorServer.ctx.Done():
	}
	motorServer.cancel()
	log.Println("start closing server...")
	if listener != nil {
		listener.Close()
	}
	time.Sleep(5 * time.Second)
	log.Println("closed server")
}
func handlerConn(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	ctx, cancel := context.WithCancel(motorServer.ctx)
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := writeData(conn, "sid")
				if err != nil {
					log.Errorf("connection is closed,error: %s,remote addr: %s", err.Error(), conn.RemoteAddr())
					conn.Close()
					atomic.AddInt64(&motorServer.connections, -1)
					return
				}
				time.Sleep(10 * time.Second)
			}
		}
	}()

	r := bufio.NewReader(conn)
	go func() {
		for {
			recs, err := readData(r)
			if err != nil {
				log.Errorf("connection is closed,error: %s,remote addr: %s", err.Error(), conn.RemoteAddr())
				conn.Close()
				atomic.AddInt64(&motorServer.connections, -1)
				return
			}
			log.Infof("receive %s from %s", recs, conn.RemoteAddr())
			r, err := regexp.Compile("<[0-9]+>")
			if err != nil {
				log.Errorf("regex compile error: %s", err.Error())
				conn.Close()
				atomic.AddInt64(&motorServer.connections, -1)
				return
			}
			data := r.FindString(recs)
			if data != "" {
				s := data[1 : len(data)-1]
				if id, err := strconv.Atoi(s); err != nil {
					log.Errorf("response to sid error: %s,response is %s", err.Error(), s)
					continue
				} else {
					cancel()
					if id > 0 {
						atomic.AddInt64(&motorServer.numberOfMotor, 1)
						log.Debugf("motor %s has connected, there have %d connected motors", s, motorServer.numberOfMotor)
						r := new(MotorResult)
						r.lock = new(sync.RWMutex)
						c := make(chan *Command, ConfigInfo.MaxCommand)
						motorServer.commandLock.Lock()
						motorServer.command[s] = c
						motorServer.commandLock.Unlock()
						go handleMotor(conn, s, r, c)
					} else {
						go handlerUnity(conn)
					}
					return
				}
			} else if strings.Contains(recs, "<admin>") {
				cancel()
				go handlerAdmin(conn)
				return
			}
		}
	}()
	<-motorServer.ctx.Done()
}
func handlerAdmin(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	defer func() {
		conn.Close()
		atomic.AddInt64(&motorServer.connections, -1)
	}()
	r := bufio.NewReader(conn)
	ctx, cancel := context.WithCancel(motorServer.ctx)
	defer cancel()
	go func() {
		for {
			data, err := readData(r)
			if err != nil {
				log.Errorf("read data error: %s", err.Error())
				cancel()
				return
			} else if strings.Contains(data, "quit") {
				motorServer.cancel()
			}

		}
	}()
	<-ctx.Done()
}
func handlerUnity(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	defer func() {
		conn.Close()
		atomic.AddInt64(&motorServer.connections, -1)
	}()
	ctx, cancel := context.WithCancel(motorServer.ctx)
	defer cancel()
	r := bufio.NewReader(conn)
	go func() {
		for {
			data, err := readData(r)
			if err != nil {
				log.Errorf("read data error: %s", err.Error())
				cancel()
				return
			} else {
				cmd := strings.Split(data, ",")
				if len(cmd) < 2 {
					log.Errorf("incorrect data: %s", data)
				} else {
					log.Infof("receive command: %s", data)
					m := Command{
						id:      cmd[0],
						content: cmd[1],
					}
					if len(cmd) > 2 {
						d, err := strconv.ParseInt(cmd[2], 10, 64)
						if err != nil {
							log.Errorf("error: %s", err.Error())
						} else {
							m.duration = d
						}
					}
					motorServer.commandLock.RLock()
					c, ok := motorServer.command[m.id]
					motorServer.commandLock.RUnlock()
					if ok {
						select {
						case c <- &m:
						default:
							log.Infof("command channel has filled", m.id)
						}
					} else {
						log.Infof("motor %s doesn't exist", m.id)
					}
				}
			}
		}

	}()
	go func() {
		ticker := time.NewTicker(time.Duration(ConfigInfo.ReportDuration) * time.Second)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				motorServer.commandLock.RLock()
				motor := make([]string, 0, len(motorServer.command))
				for k := range motorServer.command {
					motor = append(motor, k)
				}
				motorServer.commandLock.RUnlock()
				writeData(conn, strings.Join(motor, ","))
			}
		}
	}()
	<-ctx.Done()

}
func handleMotor(c net.Conn, id string, result *MotorResult, command chan *Command) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	defer func() {
		c.Close()
		atomic.AddInt64(&motorServer.connections, -1)
		atomic.AddInt64(&motorServer.numberOfMotor, -1)
		log.Debugf("close connection with %s, there have %d connected motors", id, motorServer.numberOfMotor)
		close(command)
		motorServer.commandLock.Lock()
		delete(motorServer.command, id)
		motorServer.commandLock.Unlock()
	}()
	ctx, cancel := context.WithCancel(motorServer.ctx)
	defer cancel()
	go func() {
		for m := range command {
			cmd := fmt.Sprintf("#%d %s", time.Now().Sub(motorServer.startTime).Milliseconds()+motorServer.motorPowerOn+m.duration, m.content)
			if sendCommand(c, m.id, cmd, result, 0) != nil {
				cancel()
				return
			}
		}
	}()
	go func() {
		reg, err := regexp.Compile("<[0-9]+>")
		if err != nil {
			log.Errorf("regexp compile error: %s", err.Error())
			return
		}
		for {
			r := bufio.NewReader(c)
			data, err := readData(r)
			if err != nil {
				log.Errorf("read motor: %s error: %s", id, err.Error())
				cancel()
				return
			} else {
				log.Infof("receive data: %s from %s", data, id)
				s := reg.FindString(data)
				if s != "" {
					s := s[1 : len(s)-2]
					d, err := strconv.ParseInt(s, 10, 64)
					if err != nil {
						log.Errorf("%s parse to int error: %s", s, err.Error())
						continue
					} else {
						//更新时间
						motorServer.motorPowerOn = d
						motorServer.startTime = time.Now()
					}
				}
				if data == "<OK>\r" {
					result.lock.Lock()
					result.success = true
					result.lock.Unlock()
				}
			}

		}
	}()
	if id == "480" {
		go func() {
			ticker := time.NewTicker(time.Hour * 6)
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					if sendCommand(c, id, "gtime", result, 0) != nil {
						cancel()
					}
				}
			}
		}()
	}
	<-ctx.Done()
}

func sendCommand(conn net.Conn, id string, cmd string, result *MotorResult, times int) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	if times > ConfigInfo.MaxSendTimes {
		log.Infof("failed to send %s to %s, have send for %d times", cmd, id, times)
		return nil
	}
	err := writeMotor(conn, id, cmd, result)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ConfigInfo.Duration)*time.Millisecond)
	defer cancel()
	flag := false
	for {
		select {
		case <-ctx.Done():
			return sendCommand(conn, id, cmd, result, times+1)
		default:
			result.lock.RLock()
			flag = result.success
			result.lock.RUnlock()
			if flag {
				log.Infof("success to send %s to %s, have send for %d times", cmd, id, times+1)
				return nil
			}
		}
	}
}
func readData(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], err
}
func writeData(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + "\n")
	return conn.Write(buffer.Bytes())
}
func writeMotor(c net.Conn, id string, cmd string, result *MotorResult) error {
	result.lock.Lock()
	result.success = false
	result.lock.Unlock()
	_, err := writeData(c, cmd)
	if err != nil {
		log.Errorf("failed to send command :%s to motor: %s", cmd, id)
		return err
	} else {
		log.Infof("send %s to motor %s", cmd, id)
		return nil
	}
}
