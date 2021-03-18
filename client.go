package client

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/e-money/client/keys"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc"
	"os"
)

const (
	feeDenom = "ungm"
	restSrv = "http://localhost:1317"
	jsonContent = "application/json"
)

type encodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         codec.Marshaler // Interface and default codec selection
	Proto             *codec.ProtoCodec
	Amino             *codec.LegacyAmino
}

func makeEncodingConfig() *encodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()
	proto := codec.NewProtoCodec(interfaceRegistry)

	return &encodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         proto,
		Amino:             cdc,
		Proto:             proto,
	}
}

func registerCdc() *encodingConfig {
	cfg := makeEncodingConfig()

	cryptocodec.RegisterCrypto(cfg.Amino)
	authtypes.RegisterLegacyAminoCodec(cfg.Amino)
	banktypes.RegisterLegacyAminoCodec(cfg.Amino)

	// register Tx, Msg
	sdk.RegisterLegacyAminoCodec(cfg.Amino)

	// register Bep3 types
	cfg.Amino.RegisterConcrete(&bep3.MsgCreateAtomicSwap{}, "MsgCreateAtomicSwap", nil)
	cfg.Amino.RegisterConcrete(&bep3.MsgRefundAtomicSwap{}, "MsgRefundAtomicSwap", nil)
	cfg.Amino.RegisterConcrete(&bep3.MsgClaimAtomicSwap{}, "MsgClaimAtomicSwap", nil)

	return cfg
}

// Client facilitates interaction with the cosmos blockchain
type Client struct {
	AccountName string
	Network     ChainNetwork
	HTTP        *rpcclient.HTTP
	Keybase     *keys.KeyManager
	Amino       *codec.LegacyAmino
	Marshaller  codec.BinaryMarshaler
	grpcConn    *grpc.ClientConn

	LegacyTxCfg *legacytx.StdTxConfig
	ProtoTxCfg  client.TxConfig
}

// NewClient creates a new cosmos sdk client
func NewClient(mnemonic, accountName, rpcAddr string) *Client {
	// Set up HTTP client
	http, err := rpcclient.New(rpcAddr, "/websocket")
	if err != nil {
		panic(err)
	}
	http.Logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	enc := registerCdc()

	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		"127.0.0.1:9090",
		grpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	// Set up key manager
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic, accountName)
	if err != nil {
		panic(fmt.Sprintf("new key manager from mnenomic err, err=%s", err.Error()))
	}

	return &Client{
		AccountName: accountName,
		HTTP:        http,
		Keybase:     keyManager,
		Amino:       enc.Amino,
		Marshaller:  enc.Marshaler,
		grpcConn:    grpcConn,
		LegacyTxCfg: &legacytx.StdTxConfig{Cdc: enc.Amino},
		ProtoTxCfg:  authtx.NewTxConfig(enc.Proto, authtx.DefaultSignModes),
	}
}

func (c *Client) GetAmino() *codec.LegacyAmino {
	return c.Amino
}

	}
}

// Broadcast sends a message to the e-Money blockchain as a transaction.
// This pays no transaction fees.
func (c *Client) Broadcast(m sdk.Msg, syncType tx.BroadcastMode) (*tx.BroadcastTxResponse, error) {

	fee := sdk.NewCoins(sdk.NewCoin(feeDenom, sdk.NewInt(250000)))
	return c.BroadcastWithFee(m, fee, syncType)
}

// BroadcastWithFee sends a message to the Cosmos blockchain as a transaction, paying the specified transaction fee.
func (c *Client) BroadcastWithFee(m sdk.Msg, fee sdk.Coins, syncType tx.BroadcastMode) (*tx.BroadcastTxResponse, error) {
	signBz, err := c.sign(m, fee)
	if err != nil {
		return nil, err
	}

	txBytes, err := c.LegacyTxCfg.TxEncoder()(signBz)
	if err != nil {
		return nil, err
	}

	txreq := tx.BroadcastTxRequest{
		Mode:    syncType,
		TxBytes: txBytes,
	}

	req, err := c.Amino.MarshalJSON(txreq)
	if err != nil {
		return nil, err
	}

	res, err := rest.PostRequest(
		fmt.Sprintf("%s/cosmos/tx/v1beta1/txs", restSrv), jsonContent, req)
	if err != nil {
		return nil, err
	}

	var result tx.BroadcastTxResponse
	if err = c.Amino.UnmarshalJSON(res, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) sign(m sdk.Msg, fee sdk.Coins) (sdk.Tx, error) {
	if c.Keybase == nil {
		return nil, fmt.Errorf("keys are missing, must to set key")
	}

	if err := m.ValidateBasic(); err != nil {
		return nil, err
	}

	txBuilder := c.LegacyTxCfg.NewTxBuilder()
	txBuilder.SetFeeAmount(fee)
	if err := txBuilder.SetMsgs(m); err != nil {
		return nil, err
	}
	txBuilder.SetMemo("ByClient")

	sig := signing.SignatureV2{
		PubKey: c.Keybase.GetPrivKey().PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
		},
	}

	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

// BroadcastTxCommit sends a transaction using commit
func (c *Client) BroadcastTxCommit(tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	if err := ValidateTx(tx); err != nil {
		return nil, err
	}
	return c.HTTP.BroadcastTxCommit(context.Background(),tx)
}

// BroadcastTxAsync sends a transaction using async
func (c *Client) BroadcastTxAsync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	if err := ValidateTx(tx); err != nil {
		return nil, err
	}
	return c.HTTP.BroadcastTxAsync(context.Background(), tx)
}

// BroadcastTxSync sends a transaction using sync
func (c *Client) BroadcastTxSync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	if err := ValidateTx(tx); err != nil {
		return nil, err
	}
	return c.HTTP.BroadcastTxSync(context.Background(),tx)
}
