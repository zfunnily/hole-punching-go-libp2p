package node

import (
	"bufio"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
	"v1/common"
)

type Node struct {
	host.Host

	HoleService *holepunch.Service

	Ctx         context.Context
	chatP2PAddr *peer.AddrInfo

	//记录RelayServer
	ServerAddrInfo *peer.AddrInfo

	GetPeers    chan *peer.AddrInfo
	FirstP2PMsg chan struct{}
	StartP2PMsg chan struct{}
}

func NewNode(relay string) *Node {
	//peerInfo, err := GetPeerInfoByDest(relay)
	//if err != nil {
	//	log.Println(err.Error())
	//	return nil
	//}

	//priv, _ := generateIdentity(0)
	h, err := libp2p.New(
		libp2p.ListenAddrs(multiaddr.StringCast("/ip4/127.0.0.1/tcp/0"), multiaddr.StringCast("/ip6/::1/tcp/0")),
		libp2p.ForceReachabilityPrivate(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	ids, err := identify.NewIDService(h)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	hps, err := holepunch.NewService(h, ids)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	if hps == nil {
		return nil
	}

	n := &Node{
		Host:        h,
		HoleService: hps,
		chatP2PAddr: new(peer.AddrInfo),
		Ctx:         context.Background(),
		GetPeers:    make(chan *peer.AddrInfo),
		FirstP2PMsg: make(chan struct{}),
		StartP2PMsg: make(chan struct{}),
	}

	n.SetStreamHandler(common.ON_GET_PEER, n.onGetPeers)
	n.SetStreamHandler(common.FIRST_P2P_MSG, n.onFirstP2PMsg)
	n.SetStreamHandler(common.START_P2P_MSG, n.onP2PMsg)
	n.SetStreamHandler(common.RELAY_MSG, n.onRelayMsg)

	return n
}

func (n *Node) readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}
	}
}

func (n *Node) writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		rw.WriteString(fmt.Sprintf("%s\n", sendData))
		rw.Flush()
	}
}

// 获得对方peerID
func (n *Node) onGetPeers(s network.Stream) {
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		log.Println(err.Error())
		return
	}

	peers := strings.Split(string(buf), "\n")
	var addrs []string
	for _, v := range peers {
		tmp := strings.Split(string(v), "=")
		if tmp[0] != n.ID().String() {
			addrs = tmp
			break
		}
	}

	log.Printf("a new Stream Relay rsp,addrs.len %d, peerInfo : %s\n", len(addrs), buf)
	if len(addrs) <= 1 {
		return
	}

	peerID, err := peer.Decode(addrs[0])
	if err != nil {
		log.Println(err.Error())
		return
	}

	//var targetAddress = addrs[1] + "/p2p/" + n.ServerAddrInfo.ID.String() + "/p2p-circuit/p2p/" + peerID.String()
	//var address = nodeAddr +"/p2p/" + peerID.Pretty() + "/p2p-circuit"
	//var address = nodeAddr +"/p2p/" + peerID.Pretty()
	//var address = "/p2p/"+ peerID.Pretty()
	var targetAddress = addrs[1] + "/p2p/" + n.ServerAddrInfo.ID.String() + "/p2p-circuit/p2p/" + addrs[0]
	chatTargetAddr, err := multiaddr.NewMultiaddr(targetAddress)
	if err != nil {
		log.Println("chatTargetAddr is err")
		log.Println(err.Error())
		return
	}

	n.chatP2PAddr = &peer.AddrInfo{
		ID:    peerID,
		Addrs: []multiaddr.Multiaddr{chatTargetAddr},
	}

	//n.NoHolePunchIfDirectConnExists(n.chatP2PAddr)
	//n.EndToEndSimConnect(n.chatP2PAddr)
	n.CmdRelay(n.chatP2PAddr)

}

func (n *Node) onFirstP2PMsg(s network.Stream) {
	buf, _ := ioutil.ReadAll(s)
	log.Println("receive onFirstP2PMsg: ", string(buf))
	//p.node.sendMessage(p.node.chatP2PAddr.ID, holepunch.Protocol, "this is holepunchProtocol")
	//n.sendMessage(n.chatP2PAddr.ID, holepunch.Protocol, "Hello Im "+n.chatP2PAddr.ID.String())
	time.Sleep(5 * time.Second)

	n.FirstP2PMsg <- struct{}{}
}

func (n *Node) onP2PMsg(s network.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go n.writeData(rw)
	go n.readData(rw)
}

func (n *Node) onRelayMsg(s network.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go n.writeData(rw)
	go n.readData(rw)
}

func GetPeerInfoByDest(relay string) (*peer.AddrInfo, error) {
	relayAddr, err := multiaddr.NewMultiaddr(relay)
	if err != nil {
		return nil, err
	}
	pid, err := relayAddr.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return nil, err
	}
	relayPeerID, err := peer.Decode(pid)
	if err != nil {
		return nil, err
	}

	relayPeerAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", pid))
	relayAddress := relayAddr.Decapsulate(relayPeerAddr)
	peerInfo := &peer.AddrInfo{
		ID:    relayPeerID,
		Addrs: []multiaddr.Multiaddr{relayAddress},
	}
	return peerInfo, err
}

func (n *Node) ConnectRelay(relay string) error {
	peerInfo, err := GetPeerInfoByDest(relay)
	if err != nil {
		return err
	}

	n.ServerAddrInfo = peerInfo
	return n.Connect(n.Ctx, *peerInfo)
}

func (n *Node) Login() {
	if n.sendMessage(n.ServerAddrInfo.ID, common.GET_PEER, "login") {
		log.Printf("%s login to relay\n", n.ID())
	} else {
		log.Printf("%s connect to relay error\n", n.ID())
	}
}

func (n *Node) sendMessage(id peer.ID, p protocol.ID, data string) bool {
	s, err := n.NewStream(n.Ctx, id, p)
	if err != nil {
		log.Println("NewStream err")
		log.Println(err)
		return false
	}
	defer s.Close()

	s.Write([]byte(data))
	return true
}

func (n *Node) StartSendP2PMsg(chatTargetAddrInfo *peer.AddrInfo) {
	log.Println("start send p2p msg...")

	s, err := n.NewStream(n.Ctx, chatTargetAddrInfo.ID, common.START_P2P_MSG)
	if err != nil {
		log.Println("p2p NewStream err")
		log.Println(err)
		return
	}

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	log.Println("opening p2p chat stream")
	log.Println("[INFO] p2p chat connected!")

	rw.WriteString("Hello this is  msg")
	rw.Flush()
	go n.writeData(rw)
	go n.readData(rw)
}

// NoHolePunchIfDirectConnExists 对应 holepunch_test.go 函数 TestNoHolePunchIfDirectConnExists
func (n *Node) NoHolePunchIfDirectConnExists(target *peer.AddrInfo) {
	log.Println("NoHolePunchIfDirectConnExists....", target.String())

	if err := n.Connect(n.Ctx, *target); err != nil {
		log.Println("chatTargetAddr connect err")
		log.Println(err.Error())
		return
	}

	//n.sendMessage(chatTargetAddrInfo.ID, common.FIRST_P2P_MSG, "Hello Im "+chatTargetAddrInfo.ID.String())

	time.Sleep(50 * time.Millisecond)

	n.Network().ConnsToPeer(target.ID)

	log.Println("NoHolePunchIfDirectConnExists start direct connect")
	if err := n.HoleService.DirectConnect(target.ID); err != nil {
		log.Println("DirectConnect error:", err.Error())
		return
	}
	log.Println("NoHolePunchIfDirectConnExists  direct connect success")

	n.StartSendP2PMsg(target)
}

// DirectDialWorks 对应 holepunch_test.go 函数 TestDirectDialWorks
func (n *Node) DirectDialWorks(target *peer.AddrInfo) {
	log.Println("DirectDialWorks....", target.String())
	n.Peerstore().AddAddrs(target.ID, target.Addrs, peerstore.ConnectedAddrTTL)
	n.Network().ConnsToPeer(target.ID)

	log.Println("DirectDialWorks start direct connect")
	if err := n.HoleService.DirectConnect(target.ID); err != nil {
		log.Println("DirectConnect error:", err.Error())
		return
	}
	log.Println("DirectDialWorks  direct connect success")

	n.StartSendP2PMsg(target)
}

// EndToEndSimConnect 对应 holepunch_test.go 函数 TestEndToEndSimConnect
func (n *Node) EndToEndSimConnect(target *peer.AddrInfo) {
	log.Println("EndToEndSimConnect....", target.String())
	n.Peerstore().AddAddrs(target.ID, target.Addrs, peerstore.ConnectedAddrTTL)

	for _, c := range n.Network().ConnsToPeer(target.ID) {
		if _, err := c.RemoteMultiaddr().ValueForProtocol(multiaddr.P_CIRCUIT); err != nil {
			log.Println(err)
			return
		}
	}

	log.Println("EndToEndSimConnect start direct connect")
	if err := n.HoleService.DirectConnect(target.ID); err != nil {
		log.Println("DirectConnect error:", err.Error())
		return
	}
	log.Println("EndToEndSimConnect  direct connect success")

	n.StartSendP2PMsg(target)
}

func (n *Node) CmdRelay(target *peer.AddrInfo) {
	log.Println("CmdRelay start....")

	peerInfo := target
	s, err := n.NewStream(n.Ctx, peerInfo.ID, common.RELAY_MSG)
	if err != nil {
		log.Println(err.Error())
		return
	}

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	log.Println("opening chat stream")
	log.Println("[INFO] chat connected!")
	n.sendMessage(peerInfo.ID, common.RELAY_MSG, "Hello this is relay msg")

	go n.writeData(rw)
	go n.readData(rw)
}
