package client

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/pkg/errors"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	bep3 "github.com/e-money/bep3/module"
	"github.com/e-money/client/keys"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc"
)

const (
	feeDenom    = "ungm"
	restSrv     = "http://localhost:1317"
	grpcSrv     = "127.0.0.1:9090"
	jsonContent = "application/json"
	defGasLimit = 100_000
)

var (
	EmoneyTrxCoinFee = sdk.NewCoin(feeDenom, sdk.NewInt(250))
	eMoneyFee        = sdk.NewCoins(EmoneyTrxCoinFee)
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
		grpcSrv,
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

func (c *Client) Send(fromAddr, fromKeyName, to string, keybase keyring.Keyring,
	amount sdk.Coins) (*sdk.TxResponse, error) {
	msg := &banktypes.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   to,
		Amount:      amount,
	}

	txRes, err := c.PostTxRest(fromAddr, fromKeyName, keybase, msg)
	if err != nil {
		return nil, err
	}

	fmt.Println(txRes.Info, txRes.Code, txRes.TxHash, txRes.RawLog)

	return txRes, nil
}

func (c *Client) SendGrpc(fromAddr, fromKeyName, to string, keybase keyring.Keyring,
	amount sdk.Coins) (*sdk.TxResponse, error) {
	msg := &banktypes.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   to,
		Amount:      amount,
	}

	txRes, err := c.PostTxGrpc(fromAddr, fromKeyName, keybase, msg)
	if err != nil {
		return nil, err
	}

	fmt.Println(txRes.Info, txRes.Code, txRes.TxHash, txRes.RawLog)

	return txRes, nil
}

func (c *Client) getLegacyTx(
	signerKeyName string, keybase keyring.Keyring, accNum, sequence uint64,
	fee sdk.Coins, msg sdk.Msg,
) ([]byte, error) {
	chainID, err := c.GetChainID()
	if err != nil {
		return nil, err
	}

	// prepare txBuilder with msg
	txBuilder := c.LegacyTxCfg.NewTxBuilder()
	txBuilder.SetMsgs(msg)
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(defGasLimit)
	txBuilder.SetMemo("ByClient")

	// setup txFactory
	txFactory := clienttx.Factory{}.
		WithChainID(chainID).
		WithKeybase(keybase).
		WithTxConfig(c.LegacyTxCfg).
		WithSignMode(signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON).
		WithAccountNumber(accNum).
		WithSequence(sequence)

	// offline mode requires sequence account number filled-in,
	if err := authclient.SignTx(txFactory, client.Context{}, signerKeyName,
		txBuilder, true, true); err != nil {

		return nil, err
	}

	// Protobuf encoded legacy signed Trx
	txBytes, err := c.ProtoTxCfg.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}

func (c *Client) getTx(
	signerKeyName string, keybase keyring.Keyring, accNum, sequence uint64,
	fee sdk.Coins, msg sdk.Msg,
) ([]byte, error) {
	chainID, err := c.GetChainID()
	if err != nil {
		return nil, err
	}

	// prepare txBuilder with msg
	txBuilder := c.ProtoTxCfg.NewTxBuilder()
	txBuilder.SetMsgs(msg)
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(defGasLimit)
	txBuilder.SetMemo("ByClient")

	// setup txFactory
	txFactory := clienttx.Factory{}.
		WithChainID(chainID).
		WithKeybase(keybase).
		WithTxConfig(c.ProtoTxCfg).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithAccountNumber(accNum).
		WithSequence(sequence)

	// offline mode requires sequence account number filled-in,
	if err := authclient.SignTx(txFactory, client.Context{}, signerKeyName,
		txBuilder, true, true); err != nil {

		return nil, err
	}

	txBytes, err := c.ProtoTxCfg.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}

func (c *Client) PostTxRest(signerAddr, signerKeyname string, keybase keyring.Keyring, msg sdk.Msg) (*sdk.TxResponse, error) {
	actNum, seq, err := c.getAccountNumSeq(signerAddr)
	if err != nil {
		return nil, err
	}

	txBytes, err := c.getTx(signerKeyname, keybase, actNum, seq, eMoneyFee, msg)
	if err != nil {
		return nil, err
	}

	return c.broadcastRest(context.Background(), txBytes)
}

func (c *Client) PostTxGrpc(signerAddr, signerKeyname string, keybase keyring.Keyring, msg sdk.Msg) (*sdk.TxResponse, error) {
	actNum, seq, err := c.getAccountNumSeq(signerAddr)
	if err != nil {
		return nil, err
	}

	txBytes, err := c.getTx(signerKeyname, keybase, actNum, seq, eMoneyFee, msg)
	if err != nil {
		return nil, err
	}

	return c.broadcastGrpc(context.Background(), txBytes)
}

// broadcastGrpc Broadcast the tx via gRPC. We create a new client for the
// Protobuf Tx service.
func (c *Client) broadcastGrpc(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	grpcClient := tx.NewServiceClient(c.grpcConn)
	// We then call the BroadcastTx method on this client.
	grpcRes, err := grpcClient.BroadcastTx(
		ctx,
	&tx.BroadcastTxRequest{
			Mode:    tx.BroadcastMode_BROADCAST_MODE_BLOCK,
			TxBytes: txBytes,
		},
	)
	if err != nil {
		return nil, err
	}

	return grpcRes.TxResponse, nil
}

func (c *Client) broadcastRest(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	res, err := rest.PostRequest(
		fmt.Sprintf("%s/cosmos/tx/v1beta1/txs", restSrv), jsonContent, txBytes,
	)
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(res), "error") {
		return nil, errors.New(string(res))
	}

	var txRes sdk.TxResponse
	if err := c.Amino.UnmarshalJSON(res, &txRes); err != nil {
		return nil, err
	}

	return &txRes, nil
}

func (c *Client) getAccountNumSeq(addr string) (accountNumber, accountSeq uint64, errRet error) {
	accountNumber = 0
	accountSeq = 0
	errRet = nil

	account, err := c.GetAccount(addr)
	if err != nil {
		return accountNumber, accountSeq, err
	}

	accountNumber = account.GetAccountNumber()
	accountSeq = account.GetSequence()

	return accountNumber, accountSeq, errRet
}

// Broadcast sends a message to the e-Money blockchain as a transaction.
// This pays no transaction fees.
func (c *Client) Broadcast(m sdk.Msg, syncType tx.BroadcastMode) (*tx.BroadcastTxResponse, error) {

	return c.BroadcastWithFee(m, eMoneyFee, syncType)
}

// BroadcastWithFee sends a message to the Cosmos blockchain as a transaction,
// paying the specified transaction fee.
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
	return c.HTTP.BroadcastTxCommit(context.Background(), tx)
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
	return c.HTTP.BroadcastTxSync(context.Background(), tx)
}
