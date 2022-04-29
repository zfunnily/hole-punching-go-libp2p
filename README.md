## go-libp2p p2p传输
解决在NAT后面的两个节点之间进行通信

## 下载
```shell
$ git clone https://github.com/zfunnily/hole-punching-go-libp2p.git hole-punching
```
## 使用方法
编译 & 启动
```shell
$ cd hole-punching && go mod tidy
$ ./build.sh
```
启动中继服务
```shell
./relay -sp 3001
2021/11/08 15:10:01 Run './chat -relay /ip4/172.16.3.205/tcp/3001/p2p/QmVJoYJC447ZVqQxWfWUecFAXPFd6QbT9mQvHuew3jaCBd' on another console.
2021/11/08 15:10:01 You can replace 192.168.0.100 with public IP as well.
2021/11/08 15:10:01 Waiting for incoming connection
```
在窗口A启动chat 连上中继
```shell
./chat -relay /ip4/172.16.3.205/tcp/3001/p2p/QmVJoYJC447ZVqQxWfWUecFAXPFd6QbT9mQvHuew3jaCBd
2021/11/08 15:10:09 start ConnectNodeByRelay.... QmWtaMSNakQ8x5LUf4TpMb9JQy3PPSTPS4yNjRFyxAmaom /ip4/172.16.3.205/tcp/56309
2021/11/08 15:10:09 receive onFirstP2PMsg:  Hello Im QmT2qBsSWkTHPHkgfd6aUZQiF7j2ArmWF9NwQ8R62Wixrw
```
在窗口B启动chat 连上中继
```shell
./chat -relay /ip4/172.16.3.205/tcp/3001/p2p/QmVJoYJC447ZVqQxWfWUecFAXPFd6QbT9mQvHuew3jaCBd
2021/11/08 15:10:09 start ConnectNodeByRelay.... QmT2qBsSWkTHPHkgfd6aUZQiF7j2ArmWF9NwQ8R62Wixrw /ip4/172.16.3.205/tcp/56306
2021/11/08 15:10:09 receive onFirstP2PMsg:  Hello Im QmWtaMSNakQ8x5LUf4TpMb9JQy3PPSTPS4yNjRFyxAmaom
2021/11/08 15:10:14 Cmdp2p start ....
2021/11/08 15:10:14 opening p2p chat stream
2021/11/08 15:10:14 [INFO] p2p chat connected!
```

* A和B连上了中继后，中继服务会返回对方的地址 s.Conn.RemoteMultiaddr()
* A和B接收到了地址后，发送了第一条消息`FirstP2PMsg`后，则过了5秒后直连对方节点`n.HoleService.DirectConnect(peerInfo.ID)`，代码可见,(自动p2p打洞).
* A和B可以进行p2p沟通。可以试着把relay-server关闭，然后继续发送消息。可见消息能发送成功。

这里p2p的打洞流程参考 `coordination.go`:
```golang
func (hs *Service) handleNewStream(s network.Stream) {
// Check directionality of the underlying connection.
// Peer A receives an inbound connection from peer B.
// Peer A opens a new hole punch stream to peer B.
// Peer B receives this stream, calling this function.
// Peer B sees the underlying connection as an outbound connection.
}
```

## 更改日志
- 22/04/29 增加一个ubuntu下可执行的relay-server


博客地址：https://zfunnily.github.io/2021/11/gop2pfour/
