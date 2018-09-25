package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ninjadotorg/cash-prototype/consensus/ppos"

	"github.com/ninjadotorg/cash-prototype/cashec"

	"crypto/tls"
	"os"
	"strconv"

	peer2 "github.com/libp2p/go-libp2p-peer"
	"github.com/ninjadotorg/cash-prototype/addrmanager"
	"github.com/ninjadotorg/cash-prototype/blockchain"
	"github.com/ninjadotorg/cash-prototype/common"
	"github.com/ninjadotorg/cash-prototype/connmanager"
	"github.com/ninjadotorg/cash-prototype/database"
	"github.com/ninjadotorg/cash-prototype/mempool"
	"github.com/ninjadotorg/cash-prototype/netsync"
	"github.com/ninjadotorg/cash-prototype/peer"
	"github.com/ninjadotorg/cash-prototype/rpcserver"
	"github.com/ninjadotorg/cash-prototype/transaction"
	"github.com/ninjadotorg/cash-prototype/wallet"
	"github.com/ninjadotorg/cash-prototype/wire"
)

const (
	defaultNumberOfTargetOutbound = 8
	defaultNumberOfTargetInbound  = 8
)

// onionAddr implements the net.Addr interface and represents a tor address.
type onionAddr struct {
	addr string
}

type Server struct {
	started     int32
	startupTime int64

	donePeers chan *peer.Peer
	quit      chan struct{}
	newPeers  chan *peer.Peer

	chainParams *blockchain.Params
	ConnManager *connmanager.ConnManager
	BlockChain  *blockchain.BlockChain
	Db          database.DB
	RpcServer   *rpcserver.RpcServer
	MemPool     *mempool.TxPool
	WaitGroup   sync.WaitGroup
	// Miner       *miner.Miner
	NetSync     *netsync.NetSync
	AddrManager *addrmanager.AddrManager
	Wallet      *wallet.Wallet

	// The fee estimator keeps track of how long transactions are left in
	// the mempool before they are mined into blocks.
	FeeEstimator map[byte]*mempool.FeeEstimator

	ConsensusEngine *ppos.Engine
}

// setupRPCListeners returns a slice of listeners that are configured for use
// with the RPC server depending on the configuration settings for listen
// addresses and TLS.
func (self Server) setupRPCListeners() ([]net.Listener, error) {
	// Setup TLS if not disabled.
	listenFunc := net.Listen
	if !cfg.DisableTLS {
		Logger.log.Info("Disable TLS for RPC is false")
		// Generate the TLS cert and key file if both don't already
		// exist.
		if !fileExists(cfg.RPCKey) && !fileExists(cfg.RPCCert) {
			err := rpcserver.GenCertPair(cfg.RPCCert, cfg.RPCKey)
			if err != nil {
				return nil, err
			}
		}
		keypair, err := tls.LoadX509KeyPair(cfg.RPCCert, cfg.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	} else {
		Logger.log.Info("Disable TLS for RPC is true")
	}

	netAddrs, err := common.ParseListeners(cfg.RPCListeners, "tcp")
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			log.Printf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

func (self *Server) NewServer(listenAddrs []string, db database.DB, chainParams *blockchain.Params, interrupt <-chan struct{}) error {

	// Init data for Server
	self.chainParams = chainParams
	self.quit = make(chan struct{})
	self.donePeers = make(chan *peer.Peer)
	self.newPeers = make(chan *peer.Peer)
	self.Db = db

	var err error

	// Create a new block chain instance with the appropriate configuration.9
	self.BlockChain = &blockchain.BlockChain{}
	err = self.BlockChain.Init(&blockchain.Config{
		ChainParams: self.chainParams,
		DataBase:    self.Db,
		Interrupt:   interrupt,
	})
	if err != nil {
		return err
	}

	// Search for a FeeEstimator state in the database. If none can be found
	// or if it cannot be loaded, create a new one.
	self.FeeEstimator = make(map[byte]*mempool.FeeEstimator)
	for _, bestState := range self.BlockChain.BestState {
		chainId := bestState.BestBlock.Header.ChainID
		feeEstimatorData, err := self.Db.GetFeeEstimator(chainId)
		if err == nil && len(feeEstimatorData) > 0 {
			feeEstimator, err := mempool.RestoreFeeEstimator(feeEstimatorData)
			if err != nil {
				Logger.log.Errorf("Failed to restore fee estimator %v", err)
			} else {
				self.FeeEstimator[chainId] = feeEstimator
			}
		}
	}

	// If no feeEstimator has been found, or if the one that has been found
	// is behind somehow, create a new one and start over.
	// if self.FeeEstimator == nil || self.FeeEstimator.LastKnownHeight() != self.BlockChain.BestState.BestBlock.Height {
	for _, bestState := range self.BlockChain.BestState {
		chainId := bestState.BestBlock.Header.ChainID
		if self.FeeEstimator[chainId] == nil {
			self.FeeEstimator[chainId] = mempool.NewFeeEstimator(
				mempool.DefaultEstimateFeeMaxRollback,
				mempool.DefaultEstimateFeeMinRegisteredBlocks)
		}
	}

	// create mempool tx
	self.MemPool = &mempool.TxPool{}
	self.MemPool.Init(&mempool.Config{
		Policy: mempool.Policy{
			MaxTxVersion: transaction.TxVersion + 1,
		},
		BlockChain:   self.BlockChain,
		ChainParams:  chainParams,
		FeeEstimator: self.FeeEstimator,
	})

	self.AddrManager = addrmanager.New(cfg.DataDir, nil)

	// blockTemplateGenerator := mining.NewBlkTmplGenerator(self.MemPool, self.BlockChain)

	// self.Miner = miner.New(&miner.Config{
	// 	ChainParams:            self.chainParams,
	// 	BlockTemplateGenerator: blockTemplateGenerator,
	// 	MiningAddrs:            cfg.MiningAddrs,
	// 	Chain:                  self.BlockChain,
	// 	Server:                 self,
	// })

	self.ConsensusEngine = ppos.New(&ppos.Config{
		ChainParams: self.chainParams,
		BlockChain:  self.BlockChain,
		MemPool:     self.MemPool,
		Server:      self,
	})

	// Init Net Sync manager to process messages
	self.NetSync, err = netsync.NetSync{}.New(&netsync.NetSyncConfig{
		BlockChain:   self.BlockChain,
		ChainParam:   chainParams,
		MemPool:      self.MemPool,
		Server:       self,
		Consensus:    self.ConsensusEngine,
		FeeEstimator: self.FeeEstimator,
	})
	if err != nil {
		return err
	}

	var peers []*peer.Peer
	if !cfg.DisableListen {
		var err error
		peers, err = self.InitListenerPeers(self.AddrManager, listenAddrs)
		if err != nil {
			return err
		}
	}

	// Create a connection manager.
	targetOutbound := defaultNumberOfTargetOutbound
	if cfg.MaxOutPeers > targetOutbound {
		targetOutbound = cfg.MaxOutPeers
	}
	targetInbound := defaultNumberOfTargetInbound
	if cfg.MaxInPeers > targetInbound {
		targetInbound = cfg.MaxInPeers
	}

	connManager, err := connmanager.ConnManager{}.New(&connmanager.Config{
		OnInboundAccept:      self.InboundPeerConnected,
		OnOutboundConnection: self.OutboundPeerConnected,
		ListenerPeers:        peers,
		TargetOutbound:       uint32(targetOutbound),
		TargetInbound:        uint32(targetInbound),
		DiscoverPeers:        cfg.DiscoverPeers,
	})
	if err != nil {
		return err
	}
	self.ConnManager = connManager

	// Start up persistent peers.
	permanentPeers := cfg.ConnectPeers
	if len(permanentPeers) == 0 {
		permanentPeers = cfg.AddPeers
	}

	for _, addr := range permanentPeers {
		go self.ConnManager.Connect(addr, "")
	}

	if !cfg.DisableRPC {
		// Setup listeners for the configured RPC listen addresses and
		// TLS settings.
		rpcListeners, err := self.setupRPCListeners()
		if err != nil {
			return err
		}
		if len(rpcListeners) == 0 {
			return errors.New("RPCS: No valid listen address")
		}

		rpcConfig := rpcserver.RpcServerConfig{
			Listenters:     rpcListeners,
			RPCQuirks:      cfg.RPCQuirks,
			RPCMaxClients:  cfg.RPCMaxClients,
			ChainParams:    chainParams,
			BlockChain:     self.BlockChain,
			TxMemPool:      self.MemPool,
			Server:         self,
			Wallet:         self.Wallet,
			ConnMgr:        self.ConnManager,
			AddrMgr:        self.AddrManager,
			RPCUser:        cfg.RPCUser,
			RPCPass:        cfg.RPCPass,
			RPCLimitUser:   cfg.RPCLimitUser,
			RPCLimitPass:   cfg.RPCLimitPass,
			DisableAuth:    cfg.RPCDisableAuth,
			IsGenerateNode: cfg.Generate,
			FeeEstimator:   self.FeeEstimator,
		}
		self.RpcServer = &rpcserver.RpcServer{}
		err = self.RpcServer.Init(&rpcConfig)
		if err != nil {
			return err
		}

		// Signal process shutdown when the RPC server requests it.
		go func() {
			<-self.RpcServer.RequestedProcessShutdown()
			shutdownRequestChannel <- struct{}{}
		}()
	}

	return nil
}

// InboundPeerConnected is invoked by the connection manager when a new
// inbound connection is established.
func (self *Server) InboundPeerConnected(peerConn *peer.PeerConn) {
	Logger.log.Info("inbound connected")
}

// outboundPeerConnected is invoked by the connection manager when a new
// outbound connection is established.  It initializes a new outbound server
// peer instance, associates it with the relevant state such as the connection
// request instance and the connection itself, and finally notifies the address
// manager of the attempt.
func (self *Server) OutboundPeerConnected(peerConn *peer.PeerConn) {
	Logger.log.Info("Outbound PEER connected with PEER ID - " + peerConn.PeerID.String())
	// TODO:
	// call address manager to process new outbound peer
	// push message version
	// if message version is compatible -> add outbound peer to address manager
	//for _, listen := range self.ConnManager.Config.ListenerPeers {
	//	listen.NegotiateOutboundProtocol(peer)
	//}
	//go self.peerDoneHandler(peer)
	//
	//msgNew, err := wire.MakeEmptyMessage(wire.CmdGetBlocks)
	//msgNew.(*wire.MessageGetBlocks).LastBlockHash = *self.BlockChain.BestState.BestBlock.Hash()
	//msgNew.(*wire.MessageGetBlocks).SenderID = self.ConnManager.Config.ListenerPeers[0].PeerID
	//if err != nil {
	//	return
	//}
	//self.ConnManager.Config.ListenerPeers[0].QueueMessageWithEncoding(msgNew, nil)

	// push message version
	msg, err := wire.MakeEmptyMessage(wire.CmdVersion)
	msg.(*wire.MessageVersion).Timestamp = time.Unix(time.Now().Unix(), 0)
	msg.(*wire.MessageVersion).LocalAddress = peerConn.ListenerPeer.ListeningAddress
	msg.(*wire.MessageVersion).RawLocalAddress = peerConn.ListenerPeer.RawAddress
	msg.(*wire.MessageVersion).LocalPeerId = peerConn.ListenerPeer.PeerID
	msg.(*wire.MessageVersion).RemoteAddress = peerConn.ListenerPeer.ListeningAddress
	msg.(*wire.MessageVersion).RawRemoteAddress = peerConn.ListenerPeer.RawAddress
	msg.(*wire.MessageVersion).RemotePeerId = peerConn.ListenerPeer.PeerID
	msg.(*wire.MessageVersion).LastBlock = 0
	msg.(*wire.MessageVersion).ProtocolVersion = 1
	// Validate Public Key from SealerPrvKey
	if peerConn.ListenerPeer.Config.SealerPrvKey != "" {
		sealKey, err := base64.StdEncoding.DecodeString(peerConn.ListenerPeer.Config.SealerPrvKey)
		if err != nil {
			Logger.log.Critical("Invalid sealer's private key")
			return
		}
		keySet := &cashec.KeySet{}
		keySet.ImportFromPrivateKeyByte(sealKey)
		msg.(*wire.MessageVersion).PublicKey = base64.StdEncoding.EncodeToString(keySet.SealerKeyPair.PublicKey)
	}

	if err != nil {
		return
	}
	var dc chan<- struct{}
	peerConn.QueueMessageWithEncoding(msg, dc)
}

// peerDoneHandler handles peer disconnects by notifiying the server that it's
// done along with other performing other desirable cleanup.
func (self *Server) peerDoneHandler(peer *peer.Peer) {
	//peer.WaitForDisconnect()
	self.donePeers <- peer
}

// WaitForShutdown blocks until the main listener and peer handlers are stopped.
func (self Server) WaitForShutdown() {
	self.WaitGroup.Wait()
}

// Stop gracefully shuts down the connection manager.
func (self Server) Stop() error {
	// stop connection manager
	self.ConnManager.Stop()

	// Shutdown the RPC server if it's not disabled.
	if !cfg.DisableRPC && self.RpcServer != nil {
		self.RpcServer.Stop()
	}

	// Save fee estimator in the db
	for chainId, feeEstimator := range self.FeeEstimator {
		feeEstimatorData := feeEstimator.Save()
		if len(feeEstimatorData) > 0 {
			err := self.Db.StoreFeeEstimator(feeEstimatorData, chainId)
			if err != nil {
				Logger.log.Errorf("Can't save fee estimator data: %v", err)
			} else {
				Logger.log.Info("Save fee estimator data")
			}
		}
	}

	// self.Miner.Stop()

	self.ConsensusEngine.Stop()

	// Signal the remaining goroutines to quit.
	close(self.quit)
	return nil
}

// peerHandler is used to handle peer operations such as adding and removing
// peers to and from the server, banning peers, and broadcasting messages to
// peers.  It must be run in a goroutine.
func (self Server) peerHandler() {
	// Start the address manager and sync manager, both of which are needed
	// by peers.  This is done here since their lifecycle is closely tied
	// to this handler and rather than adding more channels to sychronize
	// things, it's easier and slightly faster to simply start and stop them
	// in this handler.
	self.AddrManager.Start()
	self.NetSync.Start()

	Logger.log.Info("Start peer handler")

	if !cfg.DisableDNSSeed {
		// TODO load peer from seed DNS
		// add to address manager
		//self.AddrManager.AddAddresses(make([]*peer.Peer, 0))

		self.ConnManager.SeedFromDNS(self.chainParams.DNSSeeds, func(addrs []string) {
			// Bitcoind uses a lookup of the dns seeder here. This
			// is rather strange since the values looked up by the
			// DNS seed lookups will vary quite a lot.
			// to replicate this behaviour we put all addresses as
			// having come from the first one.
			self.AddrManager.AddAddressesStr(addrs)
		})
	}

	if len(cfg.ConnectPeers) == 0 {
		// TODO connect with peer in file
		for _, addr := range self.AddrManager.AddressCache() {
			go self.ConnManager.Connect(addr.RawAddress, addr.PublicKey)
		}
	}

	go self.ConnManager.Start()

out:
	for {
		select {
		case p := <-self.donePeers:
			self.handleDonePeerMsg(p)
		case p := <-self.newPeers:
			self.handleAddPeerMsg(p)
		case <-self.quit:
			{
				// Disconnect all peers on server shutdown.
				//state.forAllPeers(func(sp *serverPeer) {
				//	srvrLog.Tracef("Shutdown peer %s", sp)
				//	sp.Disconnect()
				//})
				break out
			}
		}
	}
	self.NetSync.Stop()
	self.AddrManager.Stop()
	self.ConnManager.Stop()
}

// Start begins accepting connections from peers.
func (self Server) Start() {
	// Already started?
	if atomic.AddInt32(&self.started, 1) != 1 {
		return
	}

	Logger.log.Info("Starting server")
	// Server startup time. Used for the uptime command for uptime calculation.
	self.startupTime = time.Now().Unix()

	// Start the peer handler which in turn starts the address and block
	// managers.
	self.WaitGroup.Add(1)

	go self.peerHandler()

	if !cfg.DisableRPC && self.RpcServer != nil {
		self.WaitGroup.Add(1)

		// Start the rebroadcastHandler, which ensures user tx received by
		// the RPC server are rebroadcast until being included in a block.
		//go self.rebroadcastHandler()

		self.RpcServer.Start()
	}

	// //creat mining
	// if cfg.Generate == true && (len(cfg.MiningAddrs) > 0) {
	// 	self.Miner.Start()
	// }
	self.ConsensusEngine.Start()
	if cfg.Generate == true && (len(cfg.SealerPrvKey) > 0) {
		self.ConsensusEngine.StartSealer(cfg.SealerPrvKey)
	}
}

// initListeners initializes the configured net listeners and adds any bound
// addresses to the address manager. Returns the listeners and a NAT interface,
// which is non-nil if UPnP is in use.
func (self *Server) InitListenerPeers(amgr *addrmanager.AddrManager, listenAddrs []string) ([]*peer.Peer, error) {
	netAddrs, err := common.ParseListeners(listenAddrs, "ip")
	if err != nil {
		return nil, err
	}

	kc := KeyCache{}
	kc.Load(filepath.Join(cfg.DataDir, "kc.json"))

	peers := make([]*peer.Peer, 0, len(netAddrs))
	for _, addr := range netAddrs {
		seed := int64(0)
		seedC, _ := strconv.ParseInt(os.Getenv("NODE_SEED"), 10, 64)
		if seedC == 0 {
			key := fmt.Sprintf("%s_seed", addr.String())
			seedT := kc.Get(key)
			if seedT == nil {
				seed = time.Now().UnixNano()
				kc.Set(key, seed)
			} else {
				seed = int64(seedT.(float64))
			}
		} else {
			seed = seedC
		}
		peer, err := peer.Peer{
			Seed:             seed,
			ListeningAddress: addr,
			Config:           *self.NewPeerConfig(),
			PeerConns:        make(map[string]*peer.PeerConn),
			PendingPeers:     make(map[string]*peer.Peer),
		}.NewPeer()
		if err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	kc.Save()

	return peers, nil
}

/**
// newPeerConfig returns the configuration for the listening Peer.
*/
func (self *Server) NewPeerConfig() *peer.Config {
	return &peer.Config{
		MessageListeners: peer.MessageListeners{
			OnBlock:     self.OnBlock,
			OnTx:        self.OnTx,
			OnVersion:   self.OnVersion,
			OnGetBlocks: self.OnGetBlocks,
			OnVerAck:    self.OnVerAck,
			OnGetAddr:   self.OnGetAddr,
			OnAddr:      self.OnAddr,

			//ppos
			OnRequestSign:   self.OnRequestSign,
			OnInvalidBlock:  self.OnInvalidBlock,
			OnBlockSig:      self.OnBlockSig,
			OnGetChainState: self.OnGetChainState,
			OnChainState:    self.OnChainState,
		},
		SealerPrvKey: cfg.SealerPrvKey,
	}
}

// OnBlock is invoked when a peer receives a block message.  It
// blocks until the coin block has been fully processed.
func (self *Server) OnBlock(p *peer.PeerConn,
	msg *wire.MessageBlock) {
	Logger.log.Info("Receive a new block")
	var txProcessed chan struct{}
	self.NetSync.QueueBlock(nil, msg, txProcessed)
	//<-txProcessed
}

func (self *Server) OnGetBlocks(_ *peer.PeerConn, msg *wire.MessageGetBlocks) {
	Logger.log.Info("Receive a get-block message")
	var txProcessed chan struct{}
	self.NetSync.QueueGetBlock(nil, msg, txProcessed)
	//<-txProcessed
}

// OnTx is invoked when a peer receives a tx message.  It blocks
// until the transaction has been fully processed.  Unlock the block
// handler this does not serialize all transactions through a single thread
// transactions don't rely on the previous one in a linear fashion like blocks.
func (self Server) OnTx(peer *peer.PeerConn,
	msg *wire.MessageTx) {
	Logger.log.Info("Receive a new transaction")
	var txProcessed chan struct{}
	self.NetSync.QueueTx(nil, msg, txProcessed)
	//<-txProcessed
}

/**
// OnVersion is invoked when a peer receives a version message
// and is used to negotiate the protocol version details as well as kick start
// the communications.
*/
func (self *Server) OnVersion(peerConn *peer.PeerConn, msg *wire.MessageVersion) {
	remotePeer := &peer.Peer{
		ListeningAddress: msg.LocalAddress,
		RawAddress:       msg.RawLocalAddress,
		PeerID:           msg.LocalPeerId,
		PublicKey:        msg.PublicKey,
	}

	if msg.PublicKey != "" {
		peerConn.Peer.PublicKey = msg.PublicKey
	}

	self.newPeers <- remotePeer
	// TODO check version message
	valid := false

	if msg.ProtocolVersion == 1 {
		valid = true
	}
	//

	// if version message is ok -> add to addManager
	//self.AddrManager.Good(remotePeer)

	// TODO push message again for remote peer
	//var dc chan<- struct{}
	//for _, listen := range self.ConnManager.Config.ListenerPeers {
	//	msg, err := wire.MakeEmptyMessage(wire.CmdVerack)
	//	if err != nil {
	//		continue
	//	}
	//	listen.QueueMessageWithEncoding(msg, dc)
	//}

	msgV, err := wire.MakeEmptyMessage(wire.CmdVerack)
	if err != nil {
		return
	}

	msgV.(*wire.MessageVerAck).Valid = valid

	peerConn.QueueMessageWithEncoding(msgV, nil)

	//	push version message again
	if !peerConn.VerAckReceived() {
		msgS, err := wire.MakeEmptyMessage(wire.CmdVersion)
		msgS.(*wire.MessageVersion).Timestamp = time.Unix(time.Now().Unix(), 0)
		msgS.(*wire.MessageVersion).LocalAddress = peerConn.ListenerPeer.ListeningAddress
		msgS.(*wire.MessageVersion).RawLocalAddress = peerConn.ListenerPeer.RawAddress
		msgS.(*wire.MessageVersion).LocalPeerId = peerConn.ListenerPeer.PeerID
		msgS.(*wire.MessageVersion).RemoteAddress = peerConn.ListenerPeer.ListeningAddress
		msgS.(*wire.MessageVersion).RawRemoteAddress = peerConn.ListenerPeer.RawAddress
		msgS.(*wire.MessageVersion).RemotePeerId = peerConn.ListenerPeer.PeerID
		msgS.(*wire.MessageVersion).LastBlock = 0
		msgS.(*wire.MessageVersion).ProtocolVersion = 1
		// Validate Public Key from SealerPrvKey
		if peerConn.ListenerPeer.Config.SealerPrvKey != "" {
			keyPair := &cashec.KeyPair{}
			keyPair.Import(peerConn.ListenerPeer.Config.SealerPrvKey)
			msgS.(*wire.MessageVersion).PublicKey = string(keyPair.PublicKey)
		}
		if err != nil {
			return
		}
		peerConn.QueueMessageWithEncoding(msgS, nil)
	}
}

/**
OnVerAck is invoked when a peer receives a version acknowlege message
*/
func (self *Server) OnVerAck(peerConn *peer.PeerConn, msg *wire.MessageVerAck) {
	// TODO for onverack message
	log.Printf("Receive verack message")

	if msg.Valid {
		peerConn.VerValid = true

		if peerConn.IsOutbound {
			self.AddrManager.Good(peerConn.Peer)
		}

		// send message for get addr
		msgS, err := wire.MakeEmptyMessage(wire.CmdGetAddr)
		if err != nil {
			return
		}
		var dc chan<- struct{}
		peerConn.QueueMessageWithEncoding(msgS, dc)

		//	broadcast addr to all peer
		for _, listen := range self.ConnManager.ListeningPeers {
			msgS, err := wire.MakeEmptyMessage(wire.CmdAddr)
			if err != nil {
				return
			}

			rawPeers := []wire.RawPeer{}
			peers := self.AddrManager.AddressCache()
			for _, peer := range peers {
				if peerConn.PeerID.Pretty() != self.ConnManager.GetPeerId(peer.RawAddress) {
					rawPeers = append(rawPeers, wire.RawPeer{peer.RawAddress, peer.PublicKey})
				}
			}
			msgS.(*wire.MessageAddr).RawPeers = rawPeers
			var doneChan chan<- struct{}
			for _, _peerConn := range listen.PeerConns {
				go _peerConn.QueueMessageWithEncoding(msgS, doneChan)
			}
		}

		// send message get blocks

		//msgNew, err := wire.MakeEmptyMessage(wire.CmdGetBlocks)
		//msgNew.(*wire.MessageGetBlocks).LastBlockHash = *self.BlockChain.BestState.BestBlockHash
		//println(peerConn.ListenerPeer.PeerId.String())
		//msgNew.(*wire.MessageGetBlocks).SenderID = peerConn.ListenerPeer.PeerId.String()
		//if err != nil {
		//	return
		//}
		//peerConn.QueueMessageWithEncoding(msgNew, nil)
	} else {
		peerConn.VerValid = true
	}

}

func (self *Server) OnGetAddr(peerConn *peer.PeerConn, msg *wire.MessageGetAddr) {
	// TODO for ongetaddr message
	log.Printf("Receive getaddr message")

	// send message for addr
	msgS, err := wire.MakeEmptyMessage(wire.CmdAddr)
	if err != nil {
		return
	}

	addresses := []string{}
	peers := self.AddrManager.AddressCache()
	for _, peer := range peers {
		if peerConn.PeerID.Pretty() != self.ConnManager.GetPeerId(peer.RawAddress) {
			addresses = append(addresses, peer.RawAddress)
		}
	}

	rawPeers := []wire.RawPeer{}
	for _, peer := range peers {
		if peerConn.PeerID.Pretty() != self.ConnManager.GetPeerId(peer.RawAddress) {
			rawPeers = append(rawPeers, wire.RawPeer{peer.RawAddress, peer.PublicKey})
		}
	}
	msgS.(*wire.MessageAddr).RawPeers = rawPeers
	var dc chan<- struct{}
	peerConn.QueueMessageWithEncoding(msgS, dc)
}

func (self *Server) OnAddr(peerConn *peer.PeerConn, msg *wire.MessageAddr) {
	// TODO for onaddr message
	log.Printf("Receive addr message", msg.RawPeers)
	//for _, rawPeer := range msg.RawPeers {
	//	for _, listen := range self.ConnManager.ListeningPeers {
	//		for _, _peerConn := range listen.PeerConns {
	//			if _peerConn.PeerID.Pretty() != self.ConnManager.GetPeerId(rawPeer.RawAddress) {
	//				go self.ConnManager.Connect(rawPeer.RawAddress, rawPeer.PublicKey)
	//			}
	//		}
	//	}
	//}
}

func (self *Server) OnRequestSign(_ *peer.PeerConn, msg *wire.MessageRequestSign) {
	Logger.log.Info("Receive a requestsign")
	var txProcessed chan struct{}
	self.NetSync.QueueMessage(nil, msg, txProcessed)
}

func (self *Server) OnInvalidBlock(_ *peer.PeerConn, msg *wire.MessageInvalidBlock) {
	Logger.log.Info("Receive a invalidblock", msg)
	var txProcessed chan struct{}
	self.NetSync.QueueMessage(nil, msg, txProcessed)
}

func (self *Server) OnBlockSig(_ *peer.PeerConn, msg *wire.MessageBlockSig) {
	Logger.log.Info("Receive a BlockSig")
	var txProcessed chan struct{}
	self.NetSync.QueueMessage(nil, msg, txProcessed)
}

func (self *Server) OnGetChainState(_ *peer.PeerConn, msg *wire.MessageGetChainState) {
	Logger.log.Info("Receive a getchainstate")
	var txProcessed chan struct{}
	self.NetSync.QueueMessage(nil, msg, txProcessed)
}

func (self *Server) OnChainState(_ *peer.PeerConn, msg *wire.MessageChainState) {
	Logger.log.Info("Receive a chainstate")
	var txProcessed chan struct{}
	self.NetSync.QueueMessage(nil, msg, txProcessed)
}

func (self *Server) GetPeerIdsFromPublicKey(pubKey string) []peer2.ID {
	result := []peer2.ID{}

	for _, listener := range self.ConnManager.Config.ListenerPeers {
		for _, peerConn := range listener.PeerConns {
			// Logger.log.Info("Test PeerConn", peerConn.Peer.PublicKey)
			if peerConn.Peer.PublicKey == pubKey {
				exist := false
				for _, item := range result {
					if item.Pretty() == peerConn.Peer.PeerID.Pretty() {
						exist = true
					}
				}

				if !exist {
					result = append(result, peerConn.Peer.PeerID)
				}
			}
		}
	}

	return result
}

/**
PushMessageToAll broadcast msg
*/
func (self *Server) PushMessageToAll(msg wire.Message) error {
	Logger.log.Info("Push msg to all")
	var dc chan<- struct{}
	for index := 0; index < len(self.ConnManager.Config.ListenerPeers); index++ {
		Logger.log.Info("Pushed 1")
		msg.SetSenderID(self.ConnManager.Config.ListenerPeers[index].PeerID)
		Logger.log.Info("Pushed 2")
		self.ConnManager.Config.ListenerPeers[index].QueueMessageWithEncoding(msg, dc)
		Logger.log.Info("Pushed 3")
	}
	return nil
}

/**
PushMessageToPeer push msg to peer
*/
func (self *Server) PushMessageToPeer(msg wire.Message, peerId peer2.ID) error {
	Logger.log.Info("Push msg to ", peerId)
	var dc chan<- struct{}
	for index := 0; index < len(self.ConnManager.Config.ListenerPeers); index++ {
		peerConn, exist := self.ConnManager.Config.ListenerPeers[index].PeerConns[peerId.String()]
		if exist {
			msg.SetSenderID(self.ConnManager.Config.ListenerPeers[index].PeerID)
			peerConn.QueueMessageWithEncoding(msg, dc)
			Logger.log.Info("Pushed")
			return nil
		} else {
			fmt.Println()
			Logger.log.Critical("Peer not exist!")
			fmt.Println()
		}
	}
	return errors.New("Peer not found")
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It is
// invoked from the peerHandler goroutine.
func (self *Server) handleDonePeerMsg(sp *peer.Peer) {
	//self.AddrManager.
	// TODO
}

// handleAddPeerMsg deals with adding new peers.  It is invoked from the
// peerHandler goroutine.
func (self *Server) handleAddPeerMsg(peer *peer.Peer) bool {
	if peer == nil {
		return false
	}

	// TODO:
	return true
}

// /**
// UpdateChain - Update chain with received block
// */
// func (self *Server) UpdateChain(block *blockchain.Block) {
// 	// save block
// 	self.BlockChain.StoreBlock(block)
// 	self.FeeEstimator.RegisterBlock(block)

// 	// Update commitments merkle tree
// 	tree := self.BlockChain.BestState.CmTree
// 	blockchain.UpdateMerkleTreeForBlock(tree, block)

// 	// save best state
//  	numTxns := uint64(len(block.Transactions))
// 	totalTxns := self.BlockChain.BestState.TotalTxns + numTxns
// 	newBestState := &blockchain.BestState{}
// 	newBestState.Init(block, 0, 0, numTxns, totalTxns, time.Unix(block.Header.Timestamp, 0), tree)
// 	self.BlockChain.BestState = newBestState
// 	self.BlockChain.StoreBestState()

// 	// save index of block
// 	self.BlockChain.StoreBlockIndex(block)
// }

func (self *Server) GetChainState() error {
	Logger.log.Info("Send a GetChainState")
	var dc chan<- struct{}
	for _, listener := range self.ConnManager.Config.ListenerPeers {
		msg, err := wire.MakeEmptyMessage(wire.CmdGetChainState)
		if err != nil {
			return err
		}
		msg.SetSenderID(listener.PeerID)
		Logger.log.Info("Send a GetChainState ", msg)
		listener.QueueMessageWithEncoding(msg, dc)
	}
	return nil
}
