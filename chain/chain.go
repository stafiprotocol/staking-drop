package chain

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/stafiprotocol/go-substrate-rpc-client/client"
)

var (
	ErrorTerminated = errors.New("terminated")
)

type Chain struct {
	conn        *Connection
	listener    *Listener // The listener of this chain
	stop        chan<- struct{}
	initialized bool
}

func NewChain() *Chain {
	return &Chain{}
}

func (c *Chain) Initialize(option *ConfigOption, logger client.Logger, sysErr chan<- error) error {
	stop := make(chan struct{})

	conn, err := NewConnection(option, logger)
	if err != nil {
		return err
	}

	bs, err := NewBlockstore(option.BlockstorePath, conn.BlockStoreUseAddress())
	if err != nil {
		return err
	}

	var startBlk uint64
	startBlk, err = StartBlock(bs, uint64(option.StartBlock))
	if err != nil {
		return err
	}

	for _, drop := range option.DropInfos {
		if drop.DropAmount.GreaterThan(decimal.NewFromInt(10e12)) {
			return fmt.Errorf("drop amount too big")
		}
	}

	l := NewListener(option.DropInfos, startBlk, bs, conn, logger, stop, sysErr)

	c.listener = l
	c.conn = conn
	c.initialized = true
	c.stop = stop
	return nil
}

func (c *Chain) Start() error {
	if !c.initialized {
		return fmt.Errorf("chain must be initialized with Initialize()")
	}
	return c.listener.start()
}

// stop will stop handler and listener
func (c *Chain) Stop() {
	close(c.stop)
}
