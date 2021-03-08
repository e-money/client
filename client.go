package client

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	"google.golang.org/grpc"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/e-money/client/keys"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/cosmos/cosmos-sdk/codec/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"

)

// Client facilitates interaction with the cosmos blockchain
type Client struct {
	Network ChainNetwork
	HTTP    *rpcclient.HTTP
	Keybase keys.KeyManager
	Cdc     *codec.LegacyAmino
	Marshaller codec.BinaryMarshaler
	grpcConn *grpc.ClientConn
}

// NewClient creates a new cosmos sdk client
func NewClient(cdc *amino.Codec, mnemonic string, rpcAddr string, networkType ChainNetwork) *Client {
	// Set up HTTP client
	http, err := rpcclient.New(rpcAddr, "/websocket")
	if err != nil {
		panic(err)
	}
	http.Logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",
		grpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	// Set up key manager
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(fmt.Sprintf("new key manager from mnenomic err, err=%s", err.Error()))
	}

	return &Client{
		Network: networkType,
		HTTP:    http,
		Keybase: keyManager,
		Cdc:     codec.NewLegacyAmino(),
		Marshaller: codec.NewProtoCodec(types.NewInterfaceRegistry()),
		grpcConn: grpcConn,
	}
}

// Broadcast sends a message to the Kava blockchain as a transaction.
// This pays no transaction fees.
func (c *Client) Broadcast(m sdk.Msg, syncType SyncType) (*ctypes.ResultBroadcastTx, error) {
	fee := authtypes.NewStdFee(250000, nil)
	return c.BroadcastWithFee(m, fee, syncType)
}

// BroadcastWithFee sends a message to the Cosmos blockchain as a transaction, paying the specified transaction fee.
func (c *Client) BroadcastWithFee(m sdk.Msg, fee authtypes.StdFee, syncType SyncType) (*ctypes.ResultBroadcastTx, error) {
	signBz, err := c.sign(m, fee)
	if err != nil {
		return nil, err
	}
	switch syncType {
	case Async:
		return c.BroadcastTxAsync(signBz)
	case Sync:
		return c.BroadcastTxSync(signBz)
	case Commit:
		commitRes, err := c.BroadcastTxCommit(signBz)
		if err != nil {
			return nil, err
		}
		if commitRes.CheckTx.IsErr() {
			return &ctypes.ResultBroadcastTx{
				Code: commitRes.CheckTx.Code,
				Log:  commitRes.CheckTx.Log,
				Hash: commitRes.Hash,
				Data: commitRes.CheckTx.Data,
			}, nil
		}
		return &ctypes.ResultBroadcastTx{
			Code: commitRes.DeliverTx.Code,
			Log:  commitRes.DeliverTx.Log,
			Hash: commitRes.Hash,
			Data: commitRes.DeliverTx.Data,
		}, nil
	default:
		return nil, fmt.Errorf("unknown synctype")
	}
}

func (c *Client) sign(m sdk.Msg, fee authtypes.StdFee) ([]byte, error) {
	if c.Keybase == nil {
		return nil, fmt.Errorf("Keys are missing, must to set key")
	}

	chainID, err := c.GetChainID()
	if err != nil {
		return nil, fmt.Errorf("could not fetch chain id: %w", err)
	}

	signMsg := &authtypes.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: 0,
		Sequence:      0,
		Fee:           fee,
		Msgs:          []sdk.Msg{m},
		Memo:          "",
	}

	if signMsg.Sequence == 0 || signMsg.AccountNumber == 0 {
		fromAddr := c.Keybase.GetAddr()
		acc, err := c.GetAccount(fromAddr)
		if err != nil {
			return nil, err
		}

		if acc.Address.Empty() {
			return nil, fmt.Errorf("the signer account does not exist on kava")
		}

		signMsg.Sequence = acc.Sequence
		signMsg.AccountNumber = acc.AccountNumber
	}

	for _, m := range signMsg.Msgs {
		if err := m.ValidateBasic(); err != nil {
			return nil, err
		}
	}

	signedMsg, err := c.Keybase.Sign(*signMsg, c.Cdc)
	if err != nil {
		return nil, err
	}

	return signedMsg, nil
}

// BroadcastTxCommit sends a transaction using commit
func (c *Client) BroadcastTxCommit(tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	if err := ValidateTx(tx); err != nil {
		return nil, err
	}
	return c.HTTP.BroadcastTxCommit(tx)
}

// BroadcastTxAsync sends a transaction using async
func (c *Client) BroadcastTxAsync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	if err := ValidateTx(tx); err != nil {
		return nil, err
	}
	return c.HTTP.BroadcastTxAsync(tx)
}

// BroadcastTxSync sends a transaction using sync
func (c *Client) BroadcastTxSync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	if err := ValidateTx(tx); err != nil {
		return nil, err
	}
	return c.HTTP.BroadcastTxSync(tx)
}
