package client

import (
	"context"
	"errors"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tendermint/go-amino"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bep3 "github.com/e-money/bep3/module"
)

// GetSwapByID gets an atomic swap on e-Money by ID
func (c *Client) GetSwapByID(swapID tmbytes.HexBytes) (swap bep3.AtomicSwap, err error) {
	params := bep3.NewQueryAtomicSwapByID(swapID)
	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}

	path := "custom/bep3/swap"

	result, err := c.ABCIQuery(path, bz)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}

	err = c.Cdc.UnmarshalJSON(result, &swap)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}
	return swap, nil
}
// GetAccount gets the account associated with an address on e-Money by gRPC
func (c *Client) GetAccountGrpc(addr string) (acc authtypes.AccountI, err error) {
	// TODO test approach
	q := authtypes.NewQueryClient(c.grpcConn)
	res, err := q.Account(context.Background(),
		&authtypes.QueryAccountRequest{Address: addr})
	if err != nil {
		return nil, err
	}

	interfaceRegistry := types.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	var resAccount authtypes.AccountI
	if err := marshaler.UnpackAny(res.Account, &resAccount); err != nil {
		return nil, errors.New("did not retrieve the account")
	}

	return resAccount, nil
}

// GetAccount gets the account associated with an address on e-Money
func (c *Client) GetAccount(addr string) (account authtypes.BaseAccount, err error) {
	res, err := rest.GetRequest("http://localhost:1317/cosmos/auth/v1beta1/accounts/"+addr)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	var resAccount authtypes.BaseAccount
	if err:=c.Cdc.UnmarshalJSON(res, resAccount); err != nil {
		return authtypes.BaseAccount{}, err
	}

	return resAccount, err
}

// GetDenomBalance gets the balance associated with an address on e-Money by gRPC
func (c *Client) GetDenomBalanceGRPC(addr string) (*sdk.Coin, error) {
	q := banktypes.NewQueryClient(c.grpcConn)
	res, err := q.Balance(
		context.Background(),
		&banktypes.QueryBalanceRequest{
			Address: addr,
		})
	if err != nil {
		return nil, err
	}

	return res.Balance, nil
}

// GetAccount gets the account associated with an address on e-Money
func (c *Client) GetDenomBalance(addr string) (*sdk.Coin, error) {
	res, err := rest.GetRequest("http://localhost:1317/cosmos/auth/v1beta1/accounts/"+addr)
	if err != nil {
		return nil, err
	}

	var balance sdk.Coin
	if err:=c.Cdc.UnmarshalJSON(res, &balance); err != nil {
		return nil, err
	}

	return &balance, err
}

// GetAccount gets the account associated with an address on e-Money
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	err = c.Cdc.UnmarshalJSON(result, &acc)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	return acc, err
}

func (c *Client) GetChainID() (string, error) {
	result, err := c.HTTP.Status()
	if err != nil {
		return "", err
	}
	return result.NodeInfo.Network, nil
}

// ABCIQuery sends a query to e-Money node
func (c *Client) ABCIQuery(path string, data tmbytes.HexBytes) ([]byte, error) {
	if err := ValidateABCIQuery(path, data); err != nil {
		return []byte{}, err
	}

	result, err := c.HTTP.ABCIQuery(context.Background(), path, data)
	if err != nil {
		return []byte{}, err
	}

	resp := result.Response
	if !resp.IsOK() {
		return []byte{}, errors.New(resp.Log)
	}

	value := result.Response.GetValue()
	if len(value) == 0 {
		return []byte{}, nil
	}

	return value, nil
}
