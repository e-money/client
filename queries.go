package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	bep3 "github.com/e-money/bep3/module"
	bep3types "github.com/e-money/bep3/module/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

// GetHeight gets the current block height
func (c *Client) GetHeight() (int64, error) {
	status, err := c.HTTP.Status(context.Background())
	if err != nil {
		return 0, err
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// WaitForHeight waits till the chain reaches the requested height
// or times out whichever occurs first.
func (c *Client) WaitForHeight(
	requestedHeight int64, expirationSpan time.Duration,
) (int64, error) {
	ticker := time.NewTicker(750 * time.Millisecond)
	timeout := time.After(expirationSpan)

	var blockHeight int64 = -1

	for {
		select {
		case <-timeout:
			ticker.Stop()
			return blockHeight, fmt.Errorf(
				"timeout at height %d, before reaching height:%d",
				blockHeight,
				requestedHeight,
			)
		case <-ticker.C:
			height, err := c.GetHeight()
			if err != nil {
				return -1, err
			}
			if height >= requestedHeight {
				return height, nil
			}
		}
	}
}

// CalcSwapId calculates the swap ID for a given random number hash, sender,
// and senderOtherChain
func (c *Client) CalcSwapId(
	randomNumberHash tmbytes.HexBytes, sender string, senderOtherChain string,
) ([]byte, error) {
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {

		return nil, err
	}

	return bep3.CalculateSwapID(
		randomNumberHash[:], senderAddr, senderOtherChain,
	), nil
}

// GetSwapByID gets an atomic swap on e-Money by ID
func (c *Client) GetSwapByID(swapID tmbytes.HexBytes) (swap *bep3types.AtomicSwap, err error) {
	q := bep3types.NewQueryClient(c.grpcConn)
	res, err := q.Swap(
		context.Background(),
		&bep3types.QuerySwapRequest{
			SwapID: swapID,
		},
	)
	if err != nil {
		return nil, err
	}

	return &res.Swap, nil
}

// GetAccount gets the account associated with an address on e-Money by gRPC
func (c *Client) GetAccountGrpc(addr string) (acc authtypes.AccountI, err error) {
	// TODO test approach
	q := authtypes.NewQueryClient(c.grpcConn)
	res, err := q.Account(
		context.Background(),
		&authtypes.QueryAccountRequest{Address: addr},
	)
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

func (c *Client) GetAccount(addr string) (authtypes.AccountI, error) {
	url := fmt.Sprintf("%s/auth/accounts/%s", restSrv, addr)

	resp, err := rest.GetRequest(url)
	if err != nil {
		return nil, err
	}

	bz, err := rest.ParseResponseWithHeight(c.Amino, resp)
	if err != nil {
		return nil, err
	}

	var acc authtypes.AccountI
	err = c.Amino.UnmarshalJSON(bz, &acc)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

// GetBaseAccount gets the account associated with an address on e-Money
func (c *Client) GetBaseAccount(addr string) (account authtypes.BaseAccount, err error) {
	res, err := rest.GetRequest("http://localhost:1317/cosmos/auth/v1beta1/accounts/" + addr)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	var resAccount authtypes.BaseAccount
	if err := c.Amino.UnmarshalJSON(res, resAccount); err != nil {
		return authtypes.BaseAccount{}, err
	}

	return resAccount, err
}

// GetDenomBalance gets the balance associated with an address on e-Money by gRPC
func (c *Client) GetDenomBalanceGRPC(addr, denom string) (*sdk.Coin, error) {
	q := banktypes.NewQueryClient(c.grpcConn)
	res, err := q.Balance(
		context.Background(),
		&banktypes.QueryBalanceRequest{
			Address: addr,
			Denom:   denom,
		},
	)
	if err != nil {
		return nil, err
	}

	return res.Balance, nil
}

// GetAccount gets the account associated with an address on e-Money
func (c *Client) GetDenomBalance(addr, denom string) (*sdk.Coin, error) {
	res, err := rest.
		GetRequest(
			fmt.Sprintf(
				"http://localhost:1317/cosmos/auth/v1beta1/accounts/%s/%s",
				addr, denom,
			),
	)
	if err != nil {
		return nil, err
	}

	var balance sdk.Coin
	if err := c.Amino.UnmarshalJSON(res, &balance); err != nil {
		return nil, err
	}

	return &balance, err
}

// GetAccount gets the account associated with an address on e-Money
func (c *Client) GetBalances(addr string) (*sdk.Coins, error) {
	res, err := rest.GetRequest("http://localhost:1317/cosmos/bank/v1beta1/balances/" + addr)
	if err != nil {
		return nil, err
	}

	var balances sdk.Coins
	if err := c.Amino.UnmarshalJSON(res, &balances); err != nil {
		return nil, err
	}

	return &balances, err
}

func (c *Client) GetChainID() (string, error) {
	result, err := c.HTTP.Status(context.Background())
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
