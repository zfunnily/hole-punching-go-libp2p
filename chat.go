//chat.go
package main

import (
	"bufio"
	"container/list"
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const RelayProtocolRequest = "/relay/relayreq/1.0.0"
const RelayProtocolResponse = "/relay/relayrsp/1.0.0"
const FirstP2PMsg = "/First/P2PMsg/1.0.0"
const P2PMsg = "/Start/P2PMsg/1.0.0"
const RelayMsg = "/relay/relaymsg/1.0.0"


type NodeProtocol struct {
	node *Node
}

func NewNodeProtocol(server *Node)*NodeProtocol {
	protocol := &NodeProtocol{ node: server}
	server.SetStreamHandler(RelayProtocolResponse, protocol.onRsp)
	server.SetStreamHandler(FirstP2PMsg, protocol.onFirstP2PMsg)
	server.SetStreamHandler(P2PMsg, protocol.onP2PMsg)
	server.SetStreamHandler(RelayMsg, protocol.onRelayMsg)
	return protocol
}

//获得对方peerID
func (p *NodeProtocol)onRsp(s network.Stream) {
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		log.Println(err.Error())
		return
	}
	peers := strings.Split(string(buf), "\n")
	var addrs []string
	for  _,v := range peers {
		tmp := strings.Split(string(v), "=")
		if tmp[0] != p.node.ID().Pretty() {
			addrs = tmp
			break
		}
	}
	log.Printf("a new Stream Relay rsp,addrs.len %d, peerInfo : %s\n", len(addrs) , buf)
	if len(addrs) <= 1 { return}
	p.node.ConnectNode(addrs[0], addrs[1])
}

func (p* NodeProtocol) onFirstP2PMsg(s network.Stream)  {
	buf, _ := ioutil.ReadAll(s)
	log.Println("receive onFirstP2PMsg: ", string(buf))
	time.Sleep(5*time.Second)
	p.node.CmdP2P()
}

func (p* NodeProtocol) onP2PMsg(s network.Stream)  {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go p.node.writeData(rw)
	go p.node.readData(rw)
}

func (p* NodeProtocol) onRelayMsg(s network.Stream)  {
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	go p.node.writeData(rw)
	go p.node.readData(rw)
}

type Node struct {
	host.Host
	HoleService *holepunch.Service
	*NodeProtocol

	chatP2PAddr *peer.AddrInfo
	Ctx context.Context
	//记录RelayServer
	ServerAddrInfo *peer.AddrInfo
	//记录peers
	Mutex *sync.Mutex
	Peers *list.List
}

func NewNode(host host.Host, hps *holepunch.Service) *Node {
	relayServer := &Node{ Host: host, HoleService: hps, chatP2PAddr: new(peer.AddrInfo), Ctx: context.Background(), Mutex: new(sync.Mutex), Peers: new(list.List)}
	relayServer.NodeProtocol = NewNodeProtocol(relayServer)
	return relayServer
}

func (n *Node)readData(rw *bufio.ReadWriter) {
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

func (n *Node)writeData(rw *bufio.ReadWriter) {
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

func getPeerInfoByDest(relay string) (*peer.AddrInfo, error) {
	relayAddr, err := multiaddr.NewMultiaddr(relay)
	if err != nil {
		return nil , err
	}
	pid, err := relayAddr.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return nil , err
	}
	relayPeerID, err := peer.Decode(pid)
	if err != nil {
		return nil , err
	}

	relayPeerAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", pid))
	relayAddress := relayAddr.Decapsulate(relayPeerAddr)
	peerInfo := &peer.AddrInfo{
		ID:    relayPeerID,
		Addrs: []multiaddr.Multiaddr{relayAddress},
	}
	return peerInfo, err
}

func (n *Node)ConnectRelay(relay string) error {
	peerInfo ,err := getPeerInfoByDest(relay)
	if err != nil {
		return err
	}
	n.ServerAddrInfo = peerInfo
	return n.Connect(n.Ctx, *peerInfo)
}

func (n* Node)ConnectNode(nodePeerID string, nodeAddr string) {
	log.Println("start ConnectNodeByRelay....", nodePeerID, nodeAddr)
	peerID, err := peer.Decode(nodePeerID)
	if err != nil {
		log.Println(err.Error())
		return
	}
	n.Network().(*swarm.Swarm).Backoff().Clear(peerID)

	var address = nodeAddr +"/p2p/" + n.ServerAddrInfo.ID.Pretty()  + "/p2p-circuit/p2p/" + peerID.Pretty()
	//var address = nodeAddr +"/p2p/" + peerID.Pretty() + "/p2p-circuit"
	//var address = nodeAddr +"/p2p/" + peerID.Pretty()
	//var address = "/p2p/"+ peerID.Pretty()
	chatTargetAddr, err := multiaddr.NewMultiaddr(address)
	if err != nil {
		log.Println("chatTargetAddr is err")
		log.Println(err.Error())
		return
	}

	chatTargetAddrInfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: []multiaddr.Multiaddr{chatTargetAddr},
	}

	if err := n.Connect(n.Ctx, chatTargetAddrInfo); err != nil {
		log.Println("chatTargetAddr connect err")
		log.Println(err.Error())
		return
	}

	n.chatP2PAddr = &chatTargetAddrInfo
	n.sendMessage(chatTargetAddrInfo.ID, FirstP2PMsg, "Hello Im " + chatTargetAddrInfo.ID.Pretty())
	//n.CmdRelay()
}

func (n *Node)CmdP2P()  {
	log.Println("Cmdp2p start ....")
	peerInfo := n.chatP2PAddr

	n.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.ConnectedAddrTTL)
	err := n.HoleService.DirectConnect(peerInfo.ID)
	if err != nil {
		log.Println("DirectConnect error: ")
		log.Println(err.Error())
		return
	}

	s, err := n.NewStream(n.Ctx, peerInfo.ID, P2PMsg)
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

func (n *Node)CmdRelay()  {
	log.Println("CmdRelay start....")
	peerInfo := n.chatP2PAddr
	s, err := n.NewStream(n.Ctx, peerInfo.ID, RelayMsg)
	if err != nil {
		log.Println(err.Error())
		return
	}

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	log.Println("opening chat stream")
	log.Println("[INFO] chat connected!")
	n.sendMessage(peerInfo.ID, RelayMsg, "Hello this is relay msg")

	go n.writeData(rw)
	go n.readData(rw)
}

func (n *Node) Login()  {
	if n.sendMessage(n.ServerAddrInfo.ID, RelayProtocolRequest, "login") {
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

func AddHolePunchService( h host.Host) *holepunch.Service {
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
	return hps
}

func makeRandomNode(relay string) *Node {
	peerInfo ,err := getPeerInfoByDest(relay)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	//priv, _ := generateIdentity(0)
	h, err := libp2p.New(
		libp2p.ListenAddrs(multiaddr.StringCast("/ip4/127.0.0.1/tcp/0/")),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelay(),
		libp2p.ForceReachabilityPrivate(),
		libp2p.StaticRelays([]peer.AddrInfo{*peerInfo}),
		libp2p.EnableHolePunching(),
	)

	if err != nil {
		log.Println(err.Error())
		return nil
	}
	hps := AddHolePunchService(h)
	if hps == nil { return nil}
	return NewNode(h, hps)
}

func main() {
	relay := flag.String("relay", "", "relay addrs")
	flag.Parse()

	if *relay == "" {
		fmt.Println("Please Use -relay ")
		return
	}
	flag.Parse()

	node := makeRandomNode(*relay)
	if err := node.ConnectRelay(*relay); err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("Connect relay success Next Login...")
	node.Login()

	log.Println("Im ", node.ID())
	select {
	case <-node.Ctx.Done():
		log.Println(node.Ctx.Err().Error())
	}
}