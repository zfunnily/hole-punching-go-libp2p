//relay-server
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	relayv1 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv1/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"
	"log"
	"sync"
)

const RelayProtocolRequest = "/relay/relayreq/1.0.0"
const RelayProtocolResponse = "/relay/relayrsp/1.0.0"

type RelayProtocol struct {
	Server *RelayServer
}

func NewRelayProtocol(server *RelayServer)*RelayProtocol {
	protocol := &RelayProtocol{ Server: server}
	server.SetStreamHandler(RelayProtocolRequest, protocol.onReq)
	server.SetStreamHandler(RelayProtocolResponse, protocol.onRsp)
	return protocol
}

//返回peers节点ID
func (p *RelayProtocol)onReq(s network.Stream) {
	p.Server.Mutex.Lock()
	defer p.Server.Mutex.Unlock()
	remotePeerID := s.Conn().RemotePeer()
	p.Server.Peers[remotePeerID]=s.Conn().RemoteMultiaddr()

	localAddrs := s.Conn().LocalMultiaddr()
	remoteAddrs := s.Conn().RemoteMultiaddr()

	log.Printf("a new Stream relay req, remotePeerID: %s; \nremoteAddr: %s, localAddr: %s\n",  remotePeerID.Pretty(), remoteAddrs.String(), localAddrs.String())

	if len(p.Server.Peers) != 2 {
		return
	}

	var rsp = ""
	for k, v := range p.Server.Peers {
		rsp = rsp + k.Pretty() +  "=" + v.String() + "\n"
	}
	for k, _ := range p.Server.Peers {
		log.Println("start send peer: " , s.Conn().RemotePeer().Pretty())
		p.Server.sendMessage(k, RelayProtocolResponse, rsp)
		delete(p.Server.Peers, k)
	}
}

func (p *RelayProtocol)onRsp(s network.Stream) {
	log.Println("a new Stream relay rsp")
}

type RelayServer struct {
	host.Host
	*RelayProtocol
	Ctx context.Context
	HoleServer *holepunch.Service

	//记录peers
	Mutex *sync.Mutex
	Peers map[peer.ID]multiaddr.Multiaddr
}

func NewRelayServer(host host.Host) *RelayServer {
	relayServer := &RelayServer{ Host:host, Ctx: context.Background(), Mutex: new(sync.Mutex), Peers: make(map[peer.ID]multiaddr.Multiaddr)}
	relayServer.RelayProtocol = NewRelayProtocol(relayServer)
	return relayServer
}

func (n *RelayServer) sendMessage(id peer.ID, p protocol.ID, data string) bool {
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
func makeRelayHost(port int) (*RelayServer,error) {
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	h, err:= libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.DisableRelay(),
		libp2p.EnableHolePunching(),
	)
	_, err = relayv1.NewRelay(h)
	if err != nil {
		log.Printf("Failed to instantiate h2 relay: %v", err)
		return nil, err
	}
	return NewRelayServer(h), err
}

func main() {
	sourcePort := flag.Int("sp", 0, "Source port number")
	flag.Parse()

	if *sourcePort == 0 {
		fmt.Println("Please Use -sp port")
		return
	}
	flag.Parse()

	serverHost, err := makeRelayHost(*sourcePort)
	if err != nil {
		log.Printf("Failed to create relay: %v", err)
		return
	}

	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", serverHost.ID().Pretty()))
	addr := serverHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	log.Printf("Run './chat -relay %s' on another console.\n",  fullAddr)
	log.Println("You can replace 192.168.0.100 with public IP as well.")
	log.Println("Waiting for incoming connection")
	log.Println()
	<- serverHost.Ctx.Done()
}