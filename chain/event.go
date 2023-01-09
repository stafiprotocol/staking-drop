package chain

import (
	"fmt"
	"strings"

	"github.com/JFJun/go-substrate-crypto/ss58"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stafiprotocol/go-substrate-rpc-client/client"
	"github.com/stafiprotocol/go-substrate-rpc-client/config"
	"github.com/stafiprotocol/go-substrate-rpc-client/types"
)

func (l *Listener) processBlockEvents(currentBlock uint64) error {
	if currentBlock%100 == 0 {
		l.log.Debug("processEvents", "blockNum", currentBlock)
	}

	events, err := l.conn.client.GetEvents(currentBlock)
	if err != nil {
		return fmt.Errorf("client.GetBlockTxs failed: %s", err)
	}

	for _, evt := range events {
		switch {
		case evt.ModuleId == config.RTokenSeriesModuleId && evt.EventId == config.ExecuteBondAndSwapEventId:
			l.log.Trace("Handling ExecuteBondAndSwap", "block", currentBlock)
			eventData, err := client.ParseLiquidityBondAndSwapEvent(evt)
			if err != nil {
				return errors.Wrap(err, "LiquidityBondAndSwapEventData")
			}
			addrStr, err := ss58.Encode(eventData.AccountId[:], ss58.StafiPrefix)
			if err != nil {
				return fmt.Errorf("address: %s encode to ss58 err: %s", hexutil.Encode(eventData.AccountId[:]), err)
			}

			dropInfo, exist := l.dropInfos[eventData.Symbol]
			if !exist {
				l.log.Warn(fmt.Sprintf("user %s LiquidityBondAndSwap, but symbol %s not support drop, will skip",
					addrStr, eventData.Symbol))
				continue
			}

			mintAmount := decimal.NewFromBigInt(eventData.Amount.Int, 0)

			if mintAmount.GreaterThanOrEqual(dropInfo.MinBondAmount) {
				balance, err := l.conn.client.FreeBalance(eventData.AccountId[:])
				if err != nil {
					if !strings.Contains(err.Error(), "can not get accountInfo for account") {
						return errors.Wrap(err, "FreeBalance")
					}
				} else {
					if balance.Sign() > 0 {
						balanceDeci := decimal.NewFromBigInt(balance.Int, 0)
						l.log.Warn(fmt.Sprintf("user %s LiquidityBondAndSwap amount: %s, symbol: %s, but already have: %sfis, will skip",
							addrStr, mintAmount.String(), eventData.Symbol, balanceDeci.Div(decimal.NewFromInt(1e12)).String()))
						continue
					}
				}
				dropAmount := types.NewUCompact(dropInfo.DropAmount.BigInt())
				err = l.conn.client.SingleTransferTo(eventData.AccountId[:], dropAmount)
				if err != nil {
					return errors.Wrap(err, "SignAndSubmitTx")
				} else {

					l.log.Info(fmt.Sprintf("user %s LiquidityBondAndSwap amount: %s, denom: %s, drop amount: %sfis success",
						addrStr, mintAmount.String(), eventData.Symbol, dropInfo.DropAmount.Div(decimal.NewFromInt(1e12)).String()))
				}

			}

		}
	}
	return nil
}
