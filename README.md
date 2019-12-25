# matrix
相关配置文件在config.json文件里，里面可以定义服务器监听的端口、MySQL数据库的host、port及其其他连接信息、矩阵的大小、接收矩阵数据的服务器的host、port以及存储消息的队列大小、消息边界

如果使用默认接收矩阵数据的服务器，则执行``go build -o server server.go``生成二进制文件并执行该文件，需要注意平台。同时需要执行``go build -o matrix main.go``生成后端的二进制文件，若使用默认服务器，则如下执行start.sh文件；否则只执行start.sh文件的第二行命令
```
git clone https://github.com/mhiwy/matrix.git
sh start.sh
```
要停止该服务请执行``kill -2 pid``