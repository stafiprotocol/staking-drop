package chain

import (
	"fmt"
	"math/big"
	"time"

	"github.com/stafiprotocol/chainbridge/utils/blockstore"
	"github.com/stafiprotocol/go-substrate-rpc-client/client"
	"github.com/stafiprotocol/go-substrate-rpc-client/submodel"
)

var (
	BlockRetryInterval = time.Second * 6
	BlockRetryLimit    = 50
	BlockConfirmNumber = uint64(3)
)

type Listener struct {
	startBlock uint64
	blockstore blockstore.Blockstorer
	conn       *Connection
	dropInfos  map[submodel.RSymbol]DropInfo //rsymbol -> drop info
	log        client.Logger
	stopChan   <-chan struct{}
	sysErrChan chan<- error
}

func NewListener(dropInfos map[submodel.RSymbol]DropInfo, startBlock uint64, bs blockstore.Blockstorer, conn *Connection, log client.Logger, stopChan <-chan struct{}, sysErr chan<- error) *Listener {
	return &Listener{
		startBlock: startBlock,
		blockstore: bs,
		conn:       conn,
		dropInfos:  dropInfos,
		log:        log,
		stopChan:   stopChan,
		sysErrChan: sysErr,
	}
}

func (l *Listener) start() error {
	latestBlk, err := l.conn.client.GetLatestBlockNumber()
	if err != nil {
		return err
	}

	if latestBlk < l.startBlock {
		return fmt.Errorf("starting block (%d) is greater than latest known block (%d)", l.startBlock, latestBlk)
	}
	go func() {
		err := l.pollBlocks()
		if err != nil {
			l.log.Error("Polling blocks failed", "err", err)
			l.sysErrChan <- err
		}
	}()

	l.log.Info("listener start pollBlocks")
	return nil
}

func (l *Listener) pollBlocks() error {
	var willDealBlock = l.startBlock
	var retry = BlockRetryLimit
	for {
		select {
		case <-l.stopChan:
			l.log.Info("pollBlocks receive stop chan, will stop")
			return nil
		default:
			if retry <= 0 {
				return fmt.Errorf("pollBlocks reach retry limit ")
			}

			latestBlk, err := l.conn.client.GetLatestBlockNumber()
			if err != nil {
				l.log.Error("Failed to fetch latest blockNumber", "err", err)
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}
			// Sleep if the block we want comes after the most recently finalized block
			if willDealBlock+BlockConfirmNumber > latestBlk {
				time.Sleep(BlockRetryInterval)
				continue
			}
			err = l.processBlockEvents(willDealBlock)
			if err != nil {
				l.log.Error("Failed to process events in block", "block", willDealBlock, "err", err)
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}

			// Write to blockstore
			err = l.blockstore.StoreBlock(new(big.Int).SetUint64(willDealBlock))
			if err != nil {
				l.log.Error("Failed to write to blockstore", "err", err)
			}
			if willDealBlock%1000 == 0 {
				l.log.Info("Have dealed block ", "height", willDealBlock)
			}
			willDealBlock++

			retry = BlockRetryLimit
		}
	}
}
