package chain

import (
	"fmt"
	"strings"

	"github.com/JFJun/go-substrate-crypto/ss58"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/itering/scale.go/utiles"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stafiprotocol/go-substrate-rpc-client/config"
	"github.com/stafiprotocol/go-substrate-rpc-client/submodel"
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
			eventData, err := LiquidityBondAndSwapEventData(evt)
			if err != nil {
				return errors.Wrap(err, "LiquidityBondAndSwapEventData")
			}
			addrStr, err := ss58.Encode(eventData.AccountId[:], ss58.StafiPrefix)
			if err != nil {
				return fmt.Errorf("address: %s encode to ss58 err: %s", hexutil.Encode(eventData.AccountId[:]), err)
			}

			dropInfo, exist := l.dropInfos[eventData.Symbol]
			if !exist {
				l.log.Warn(fmt.Sprintf("user %s liquidity bond, but symbol %s not support drop, will skip",
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
						l.log.Warn(fmt.Sprintf("user %s stake amount: %s, symbol: %s, but already have: %sfis, will skip",
							addrStr, mintAmount.String(), eventData.Symbol, balanceDeci.Div(decimal.NewFromInt(1e12)).String()))
						continue
					}
				}
				dropAmount := types.NewUCompact(dropInfo.DropAmount.BigInt())
				err = l.conn.client.SingleTransferTo(eventData.AccountId[:], dropAmount)
				if err != nil {
					return errors.Wrap(err, "SignAndSubmitTx")
				} else {

					l.log.Info(fmt.Sprintf("user %s liquidity bond amount: %s, denom: %s, drop amount: %sfis success",
						addrStr, mintAmount.String(), eventData.Symbol, dropInfo.DropAmount.Div(decimal.NewFromInt(1e12)).String()))
				}

			}

		}
	}

	return nil
}

func LiquidityBondAndSwapEventData(evt *submodel.ChainEvent) (*EvtExecuteBondAndSwap, error) {
	if len(evt.Params) != 6 {
		return nil, fmt.Errorf("LiquidityBondEventData params number not right: %d, expected: 3", len(evt.Params))
	}
	accountId, err := parseAccountId(evt.Params[0].Value)
	if err != nil {
		return nil, fmt.Errorf("LiquidityBondEventData params[0] -> AccountId error: %s", err)
	}
	symbol, err := parseRsymbol(evt.Params[1].Value)
	if err != nil {
		return nil, fmt.Errorf("LiquidityBondEventData params[1] -> RSymbol error: %s", err)
	}
	bondId, err := parseHash(evt.Params[2].Value)
	if err != nil {
		return nil, fmt.Errorf("LiquidityBondEventData params[2] -> BondId error: %s", err)
	}
	amount, err := parseU128(evt.Params[3].Value)
	if err != nil {
		return nil, fmt.Errorf("LiquidityBondEventData params[3] -> BondId error: %s", err)
	}
	destRecipient, err := parseBytes(evt.Params[4].Value)
	if err != nil {
		return nil, fmt.Errorf("LiquidityBondEventData params[4] -> BondId error: %s", err)
	}
	destId, err := parseU8(evt.Params[5].Value)
	if err != nil {
		return nil, fmt.Errorf("LiquidityBondEventData params[5] -> BondId error: %s", err)
	}

	return &EvtExecuteBondAndSwap{
		AccountId:     accountId,
		Symbol:        symbol,
		BondId:        bondId,
		Amount:        amount,
		DestRecipient: types.NewBytes(destRecipient),
		DestId:        destId,
	}, nil
}

type EvtExecuteBondAndSwap struct {
	AccountId     types.AccountID
	Symbol        submodel.RSymbol
	BondId        types.Hash
	Amount        types.U128
	DestRecipient types.Bytes
	DestId        types.U8
}

var (
	ValueNotStringError      = errors.New("value not string")
	ValueNotFloatError       = errors.New("value not float64")
	ValueNotMapError         = errors.New("value not map")
	ValueNotU32              = errors.New("value not u32")
	ValueNotStringSliceError = errors.New("value not string slice")
)

func parseBytes(value interface{}) ([]byte, error) {
	val, ok := value.(string)
	if !ok {
		return nil, ValueNotStringError
	}

	bz, err := hexutil.Decode(utiles.AddHex(val))
	if err != nil {
		if err.Error() == hexutil.ErrSyntax.Error() {
			return []byte(val), nil
		}
		return nil, err
	}

	return bz, nil
}

func parseAccountId(value interface{}) (types.AccountID, error) {
	val, ok := value.(string)
	if !ok {
		return types.NewAccountID([]byte{}), ValueNotStringError
	}
	ac, err := hexutil.Decode(utiles.AddHex(val))
	if err != nil {
		return types.NewAccountID([]byte{}), err
	}

	return types.NewAccountID(ac), nil
}

func parseRsymbol(value interface{}) (submodel.RSymbol, error) {
	sym, ok := value.(string)
	if !ok {
		return submodel.RSymbol(""), ValueNotStringError
	}

	return submodel.RSymbol(sym), nil
}

func parseHash(value interface{}) (types.Hash, error) {
	val, ok := value.(string)
	if !ok {
		return types.NewHash([]byte{}), ValueNotStringError
	}

	hash, err := types.NewHashFromHexString(utiles.AddHex(val))
	if err != nil {
		return types.NewHash([]byte{}), err
	}

	return hash, err
}

func parseU128(value interface{}) (types.U128, error) {
	val, ok := value.(string)
	if !ok {
		return types.U128{}, ValueNotStringError
	}
	return types.NewU128(*utiles.U256(val)), nil
}
func parseU8(value interface{}) (types.U8, error) {
	val, ok := value.(float64)
	if !ok {
		return types.U8(0), ValueNotFloatError
	}
	return types.NewU8(uint8(val)), nil
}
