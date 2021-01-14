package client

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

// GetAccount gets the account associated with an address on e-Money
func (c *Client) GetAccount(addr sdk.AccAddress) (acc authtypes.BaseAccount, err error) {
	params := authtypes.NewQueryAccountParams(addr)
	bz, err := c.Cdc.MarshalJSON(params)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	path := fmt.Sprintf("custom/acc/account/%s", addr.String())

	result, err := c.ABCIQuery(path, bz)
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

// ABCIQuery sends a query to Kava
func (c *Client) ABCIQuery(path string, data tmbytes.HexBytes) ([]byte, error) {
	if err := ValidateABCIQuery(path, data); err != nil {
		return []byte{}, err
	}

	result, err := c.HTTP.ABCIQuery(path, data)
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
