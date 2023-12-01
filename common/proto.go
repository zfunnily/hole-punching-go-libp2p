package common

const version = "1.0.0"
const (
	GET_PEER      = "/relay/getpeer/" + version
	ON_GET_PEER   = "/chat/getpeer/" + version
	SEND_PEERS    = "/relay/sendpeers/" + version
	FIRST_P2P_MSG = "/chat/first/p2p/msg/" + version
	START_P2P_MSG = "/chat/start/p2p/msg/" + version
	RELAY_MSG     = "/relay/relaymsg/" + version
)
