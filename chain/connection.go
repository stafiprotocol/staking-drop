package chain

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stafiprotocol/chainbridge/utils/crypto/sr25519"
	"github.com/stafiprotocol/chainbridge/utils/keystore"
	"github.com/stafiprotocol/go-substrate-rpc-client/client"
)

type Connection struct {
	client               *client.GsrpcClient
	log                  client.Logger
	blockstoreUseAddress string
}

func NewConnection(cfgOption *ConfigOption, log client.Logger) (*Connection, error) {
	fmt.Printf("Will open stafihub wallet from <%s>. \nPlease ", cfgOption.KeystorePath)
	kp, blockstoreUseAddress, err := keystore.KeypairFromAddressV2(cfgOption.Account, keystore.SubChain, cfgOption.KeystorePath, false)
	if err != nil {
		return nil, err
	}
	krp := kp.(*sr25519.Keypair).AsKeyringPair()
	stop := make(chan int)
	gsc, err := client.NewGsrpcClient(client.ChainTypeStafi, cfgOption.Endpoint, cfgOption.TypesPath, client.AddressTypeAccountId, krp, log, stop)
	if err != nil {
		return nil, errors.Wrap(err, "NewGsrpcClient")
	}

	c := Connection{
		client:               gsc,
		log:                  log,
		blockstoreUseAddress: blockstoreUseAddress,
	}
	return &c, nil
}

func (c *Connection) BlockStoreUseAddress() string {
	return c.blockstoreUseAddress
}
