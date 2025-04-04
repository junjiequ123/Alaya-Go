// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// This file contains some shares testing functionality, common to  multiple
// different files and modules being tested.

package eth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
	"sort"
	"sync"
	"testing"

	"github.com/AlayaNetwork/Alaya-Go/core/rawdb"

	"github.com/AlayaNetwork/Alaya-Go/consensus"
	"github.com/AlayaNetwork/Alaya-Go/x/xcom"

	"github.com/AlayaNetwork/Alaya-Go/common"
	"github.com/AlayaNetwork/Alaya-Go/core"
	"github.com/AlayaNetwork/Alaya-Go/core/types"
	"github.com/AlayaNetwork/Alaya-Go/crypto"
	"github.com/AlayaNetwork/Alaya-Go/eth/downloader"
	"github.com/AlayaNetwork/Alaya-Go/ethdb"
	"github.com/AlayaNetwork/Alaya-Go/event"
	"github.com/AlayaNetwork/Alaya-Go/p2p"
	"github.com/AlayaNetwork/Alaya-Go/p2p/discover"
	"github.com/AlayaNetwork/Alaya-Go/params"
	_ "github.com/AlayaNetwork/Alaya-Go/x/xcom"
)

var (
	testBankKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBank       = crypto.PubkeyToAddress(testBankKey.PublicKey)
)

// newTestProtocolManager creates a new protocol manager for testing purposes,
// with the given number of blocks already known, and potential notification
// channels for different events.
func newTestProtocolManager(mode downloader.SyncMode, blocks int, generator func(int, *core.BlockGen), newtx chan<- []*types.Transaction) (*ProtocolManager, ethdb.Database, error) {
	xcom.GetEc(xcom.DefaultTestNet)
	var (
		evmux = new(event.TypeMux)
		//	engine = cbft.New(params.GrapeChainConfig.Cbft, evmux, nil)
		engine = consensus.NewFaker()
		db     = rawdb.NewMemoryDatabase()
		gspec  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc:  core.GenesisAlloc{testBank: {Balance: big.NewInt(1000000)}},
		}
		genesis = gspec.MustCommit(db)

		//blockchain, _ = core.NewBlockChain(db, nil, gspec.Config, engine, vm.Config{}, nil)
	)
	engine.InsertChain(genesis)
	//cache := core.NewBlockChainCache(blockchain)
	//
	//engine.SetBlockChainCache(cache)
	//
	//txpool := core.NewTxPool(core.DefaultTxPoolConfig, gspec.Config, core.NewTxPoolBlockChain(cache))
	//
	//engine.Start(blockchain, txpool, cbft.NewStaticAgency([]discover.Node{}))
	//chain, _ := core.GenerateChain(gspec.Config, genesis, engine, db, blocks, generator)
	chain := core.GenerateBlockChain2(gspec.Config, genesis, engine, db, blocks, generator)
	//if _, err := blockchain.InsertChain(chain); err != nil {
	//	panic(err)
	//}
	pm, err := NewProtocolManager(gspec.Config, mode, DefaultConfig.NetworkId, evmux, &testTxPool{added: newtx}, engine, chain, db, 1)
	if err != nil {
		return nil, nil, err
	}
	pm.Start(1000)
	return pm, db, nil
}

// newTestProtocolManagerMust creates a new protocol manager for testing purposes,
// with the given number of blocks already known, and potential notification
// channels for different events. In case of an error, the constructor force-
// fails the test.
func newTestProtocolManagerMust(t *testing.T, mode downloader.SyncMode, blocks int, generator func(int, *core.BlockGen), newtx chan<- []*types.Transaction) (*ProtocolManager, ethdb.Database) {
	pm, db, err := newTestProtocolManager(mode, blocks, generator, newtx)
	if err != nil {
		t.Fatalf("Failed to create protocol manager: %v", err)
	}
	return pm, db
}

// testTxPool is a fake, helper transaction pool for testing purposes
type testTxPool struct {
	txFeed event.Feed
	pool   []*types.Transaction        // Collection of all transactions
	added  chan<- []*types.Transaction // Notification channel for new transactions

	lock sync.RWMutex // Protects the transaction pool
}

// Has returns an indicator whether txpool has a transaction
// cached with the given hash.
func (p *testTxPool) Has(hash common.Hash) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, tx := range p.pool {
		if tx.Hash() == hash {
			return true
		}
	}
	return false
}

// Get retrieves the transaction from local txpool with given
// tx hash.
func (p *testTxPool) Get(hash common.Hash) *types.Transaction {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, tx := range p.pool {
		if tx.Hash() == hash {
			return tx
		}
	}
	return nil
}

// AddRemotes appends a batch of transactions to the pool, and notifies any
// listeners if the addition channel is non nil
func (p *testTxPool) AddRemotes(txs []*types.Transaction) []error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.pool = append(p.pool, txs...)
	if p.added != nil {
		p.added <- txs
	}
	return make([]error, len(txs))
}

// Pending returns all the transactions known to the pool
func (p *testTxPool) Pending() (map[common.Address]types.Transactions, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	batches := make(map[common.Address]types.Transactions)
	for _, tx := range p.pool {
		from, _ := types.Sender(types.NewEIP155Signer(new(big.Int)), tx)
		batches[from] = append(batches[from], tx)
	}
	for _, batch := range batches {
		sort.Sort(types.TxByNonce(batch))
	}
	return batches, nil
}

func (p *testTxPool) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return p.txFeed.Subscribe(ch)
}

// newTestTransaction create a new dummy transaction.
func newTestTransaction(from *ecdsa.PrivateKey, nonce uint64, datasize int) *types.Transaction {
	tx := types.NewTransaction(nonce, common.Address{}, big.NewInt(0), 100000, big.NewInt(0), make([]byte, datasize))
	tx, _ = types.SignTx(tx, types.NewEIP155Signer(new(big.Int)), from)
	return tx
}

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
	*peer
}

// newTestPeer creates a new peer registered at the given protocol manager.
func newTestPeer(name string, version int, pm *ProtocolManager, shake bool) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Start the peer on a new thread
	var id discover.NodeID
	rand.Read(id[:])
	peer := pm.newPeer(version, p2p.NewPeer(id, name, nil), net)

	// Start the peer on a new thread
	errc := make(chan error, 1)
	go func() { errc <- pm.runPeer(peer) }()
	tp := &testPeer{app: app, net: net, peer: peer}
	// Execute any implicitly requested handshakes and return
	if shake {
		var (
			genesis = pm.blockchain.Genesis()
			head    = pm.blockchain.CurrentHeader()
		)
		tp.handshake(nil, head.Number, head.Hash(), genesis.Hash())
	}
	return tp, errc
}

// handshake simulates a trivial handshake that expects the same state from the
// remote side as we are simulating locally.
func (p *testPeer) handshake(t *testing.T, bn *big.Int, head common.Hash, genesis common.Hash) {
	msg := &statusData{
		ProtocolVersion: uint32(p.version),
		NetworkId:       DefaultConfig.NetworkId,
		CurrentBlock:    head,
		GenesisBlock:    genesis,
		BN:              bn,
	}
	if err := p2p.ExpectMsg(p.app, StatusMsg, msg); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p.app, StatusMsg, msg); err != nil {
		t.Fatalf("status send: %v", err)
	}
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.app.Close()
}
