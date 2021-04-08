package client_test

import (
	"encoding/hex"
	"fmt"
	bep3types "github.com/e-money/bep3/module/types"
	"github.com/tendermint/tendermint/libs/bytes"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bep3 "github.com/e-money/bep3/module"
	emc "github.com/e-money/client"
	"github.com/e-money/client/keys"
	emtypes "github.com/e-money/em-ledger/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	denom          = "ungm"
	grpcSrv        = "127.0.0.1:9090"
	rpcAddr        = "tcp://localhost:26657"
	defGasLimit    = 100_000
	defNetId       = "localnet_reuse"
	signerIdx      = 0
	recipientIndex = 1
)

type testsSuite struct {
	conn       *grpc.ClientConn
	ProtoTxCfg client.TxConfig
}

var (
	eMoneyUserAddrs []string
	signerStrIdx    = strconv.Itoa(signerIdx)
	randomKeyNumber []byte

	tests testsSuite

	// min amnt == 50001 (fee + min amount (1))
	swapAmnt = sdk.NewInt64Coin(denom, 51000)
	outCoins   = sdk.NewCoins(swapAmnt)

	emoneyUserMnemonics = []string{
		"document weekend believe whip diesel earth hope elder quiz pact assist quarter public deal height pulp roof organ animal health month holiday front pencil",
		"play witness auto coast domain win tiny dress glare bamboo rent mule delay exact arctic vacuum laptop hidden siren sudden six tired fragile penalty",
		"treat ocean valid motor life marble syrup lady nephew grain cherry remember lion boil flock outside cupboard column dad rare build nut hip ostrich",
		"rice short length buddy zero snake picture enough steak admit balance garage exit crazy cloud this sweet virus can aunt embrace picnic stick wheel",
		"census museum crew rude tower vapor mule rib weasel faith page cushion rain inherit much cram that blanket occur region track hub zero topple",
		"flavor print loyal canyon expand salmon century field say frequent human dinosaur frame claim bridge affair web way direct win become merry crash frequent",
		"very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food",
	}
)

func TestMain(m *testing.M) {
	sdkConfig := sdk.GetConfig()

	emtypes.SetNetworkPrefix(sdkConfig)

	for idx, m := range emoneyUserMnemonics {
		keyManager, err := keys.NewMnemonicKeyManager(m, strconv.Itoa(idx))
		if err != nil {
			panic(err)
		}

		eMoneyUserAddrs = append(eMoneyUserAddrs, keyManager.GetAddr().String())
	}

	// Create a connection to the gRPC server.
	var err error
	tests.conn, err = grpc.Dial(
		grpcSrv,
		grpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// genSwapKeys generates new swap hash (id) by random number, timestamp.
func genSwapKeys() (timestamp int64, rndNum []byte, rndHash []byte, err error) {
	timestamp = time.Now().Unix()

	rndNum, err = bep3.GenerateSecureRandomNumber()
	if err != nil {
		return 0, nil, nil, fmt.Errorf(
			"couldn't generate random number: %w", err,
		)
	}

	rndHash = bep3.CalculateRandomHash(rndNum, timestamp)

	return timestamp, rndNum, rndHash, err
}

func TestSwapClaim(t *testing.T) {
	var tenMinutes int64 = 10

	balanceSignerBefSwap,
	balanceRecBefSwap,
	height,
	c,
	randomNumber,
	swapID,
	swap := createOutgoingSwapTx(t, tenMinutes)

	balanceSignerAfterSwap, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[signerIdx], denom,
	)
	require.NoError(t, err)

	balanceRecAfterSwap, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[recipientIndex], denom,
	)
	require.NoError(t, err)

	claimTrx(
		t, c, swapID, randomNumber, err, swap, height, balanceSignerAfterSwap,
		balanceRecAfterSwap, balanceSignerBefSwap, balanceRecBefSwap,
	)
}

func TestExpirationRefund(t *testing.T) {
	var (
		oneMinute int64 = 1 // minute
		err       error
	)

	balanceSignerBefSwap,
	balanceRecBefSwap,
	height,
	c,
	randomNumber,
	swapID,
	swap := createOutgoingSwapTx(t, oneMinute)

	ticker := time.NewTicker(2 * time.Second)
	timeout := time.After(time.Duration(oneMinute) * time.Minute)

	var secs int64 = 0
	for {
		select {
		case <-timeout:
			ticker.Stop()
			t.Log()
			goto checkExpiration
		case <-ticker.C:
			secs += 2
			fmt.Printf(" ...%d secs left", (oneMinute*60)-secs)
		}
	}

checkExpiration:
	incChainHeight(t, c, height)
	swap, err = c.GetSwapByID(swapID)
	require.NoError(t, err)
	require.NotNil(t, swap)
	require.Equal(t, swap.Status, bep3.Expired)

	balanceSignerAfterSwap, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[signerIdx], denom,
	)
	require.NoError(t, err)

	balanceRecAfterSwap, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[recipientIndex], denom,
	)
	require.NoError(t, err)

	/*
	 * Swap Trx Balance assertions after expiration
	 * same as before expiration
	 */

	require.True(
		t, balanceSignerBefSwap.Equal(
			balanceSignerAfterSwap.Add(swapAmnt).Add(emc.EmoneyTrxCoinFee),
		),
	)

	require.True(t, balanceRecBefSwap.Equal(balanceRecAfterSwap))

	refundTrx(
		t, c, swapID, randomNumber, err, swap, height, balanceSignerAfterSwap,
		balanceRecAfterSwap, balanceSignerBefSwap, balanceRecBefSwap,
	)
}

func claimTrx(
	t *testing.T, c *emc.Client, swapID bytes.HexBytes,
	randomNumber bytes.HexBytes, err error, swap *bep3types.AtomicSwap,
	height int64, balanceSignerAfterSwap *sdk.Coin,
	balanceRecAfterSwap *sdk.Coin, balanceSignerBefSwap *sdk.Coin,
	balanceRecBefSwap *sdk.Coin,
) {
	claimMsg := bep3.NewMsgClaimAtomicSwap(
		c.Keybase.GetAddr(), swapID, randomNumber,
	)

	// ------- Claim Trx
	resp, err := c.PostTxGrpc(
		swap.Sender,
		signerStrIdx,
		c.Keybase.Keyring,
		claimMsg,
	)
	require.NoError(t, err)
	require.True(t, resp.Code == 0)
	require.True(t, len(resp.TxHash) != 0, "claim trx hash should not be empty")

	incChainHeight(t, c, height)

	balanceSignerAfterClaim, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[signerIdx], denom,
	)
	require.NoError(t, err)

	balanceRecAfterClaim, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[recipientIndex], denom,
	)
	require.NoError(t, err)
	t.Log("receiver's balance after claim:", balanceRecAfterClaim)

	t.Log(
		"signer's balance after claim", balanceSignerAfterClaim, "diff:",
		balanceSignerAfterSwap.Sub(*balanceSignerAfterClaim),
	)
	t.Logf("diff (before swap - claimed): %s",
		balanceSignerBefSwap.Sub(*balanceSignerAfterClaim),
	)

	/*
	 * Claim Trx Balance assertions
	 */

	swapTrxFee := sdk.NewCoin(emc.EmoneyTrxCoinFee.GetDenom(), emc.EmoneyTrxCoinFee.Amount)
	doubleTrxFees := sdk.NewCoin(denom, swapTrxFee.Amount.MulRaw(2))

	// swap amount deducted and keep fee
	require.True(
		t, balanceSignerAfterClaim.Equal(
			balanceSignerBefSwap.
				Sub(swapAmnt).
				Sub(doubleTrxFees),
		),
	)

	// no diff for deputy
	require.True(t, balanceRecBefSwap.Equal(balanceRecAfterSwap))
}

func refundTrx(
	t *testing.T, c *emc.Client, swapID bytes.HexBytes,
	randomNumber bytes.HexBytes, err error, swap *bep3types.AtomicSwap,
	height int64, balanceSignerAfterSwap *sdk.Coin,
	balanceRecAfterSwap *sdk.Coin, balanceSignerBefSwap *sdk.Coin,
	balanceRecBefSwap *sdk.Coin,
) {
	refundMsg := bep3.NewMsgRefundAtomicSwap(
		c.Keybase.GetAddr(),
		swapID,
	)

	// ------- Refund Trx
	resp, err := c.PostTxGrpc(
		swap.Sender,
		signerStrIdx,
		c.Keybase.Keyring,
		refundMsg,
	)
	require.NoError(t, err)
	require.True(t, resp.Code == 0)
	require.True(
		t, len(resp.TxHash) != 0, "refund trx hash should not be empty",
	)

	incChainHeight(t, c, height)

	balanceSignerAfterRefund, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[signerIdx], denom,
	)
	require.NoError(t, err)

	balanceRecAfterRefund, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[recipientIndex], denom,
	)
	require.NoError(t, err)
	t.Log("receiver's balance after refund", balanceRecAfterRefund)
	t.Log("signer's balance after refund", balanceSignerAfterRefund)
	t.Logf("diff (refunded - swap): %s",
		balanceSignerAfterRefund.Sub(*balanceSignerAfterSwap),
	)
	t.Logf("diff (before swap - refunded): %s",
		balanceSignerBefSwap.Sub(*balanceSignerAfterRefund),
	)

	/*
	 * Refund Trx Balance assertions
	 */

	swapTrxFee := sdk.NewCoin(emc.EmoneyTrxCoinFee.GetDenom(), emc.EmoneyTrxCoinFee.Amount)
	doubleTrxFees := sdk.NewCoin(denom, swapTrxFee.Amount.MulRaw(2))

	// swap amount deducted and keep fee
	require.True(
		t, balanceSignerAfterRefund.Equal(
			balanceSignerBefSwap.
				Sub(doubleTrxFees),
		),
	)

	// no diff for deputy
	require.True(t, balanceRecBefSwap.Equal(balanceRecAfterSwap))
}

func createOutgoingSwapTx(
	t *testing.T, timespanMin int64,
) (
	balanceSignerBefSwap, balanceRecBefSwap *sdk.Coin,
	height int64,
	c *emc.Client,
	randomNumber,
	swapID bytes.HexBytes,
	swap *bep3types.AtomicSwap) {

	var err error

	c = emc.NewClient(
		emoneyUserMnemonics[signerIdx], signerStrIdx, rpcAddr,
	)
	t.Log("signer:", c.Keybase.GetAddr().String())

	height, err = c.GetHeight()
	require.NoError(t, err)

	t.Log("Creating swapMsg at height:", height)

	balanceSignerBefSwap, err = c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[signerIdx], denom,
	)
	require.NoError(t, err)
	coin := sdk.NewCoin(denom, sdk.NewInt(100_000))
	require.Truef(
		t, balanceSignerBefSwap.IsGTE(coin),
		"%s not enough balanceSignerBefSwap to cover swapping fees",
		c.Keybase.GetAddr().String(),
	)
	t.Log("signer's balance before swap", balanceSignerBefSwap.String())

	recipientKB, err := keys.NewMnemonicKeyManager(
		emoneyUserMnemonics[recipientIndex], strconv.Itoa(recipientIndex),
	)
	t.Log("receiver:", recipientKB.GetAddr().String())
	require.NoError(t, err)

	balanceRecBefSwap, err = c.GetDenomBalanceGRPC(
		recipientKB.GetAddr().String(), denom,
	)
	require.NoError(t, err)
	t.Log("receiver's balance before swap", balanceRecBefSwap.String())

	tm, randomNumber, hash, err := genSwapKeys()
	require.NoError(t, err)
	t.Log("timestamp:", time.Unix(tm, 0))
	t.Log("random number:", hex.EncodeToString(randomNumber))
	t.Log("hash:", hex.EncodeToString(hash))

	swapID, err = c.CalcSwapId(
		hash, eMoneyUserAddrs[signerIdx], "0x"+eMoneyUserAddrs[signerIdx],
	)
	require.NoError(t, err)

	t.Log("swapID:", hex.EncodeToString(swapID))

	require.True(
		t, len(randomNumber) > 0, "round number should not be empty",
	)
	require.Len(t, hash, 32, "hash length should be 32")

	swapMsg := bep3.NewMsgCreateAtomicSwap(
		c.Keybase.GetAddr().String(),        // User creating swapMsg trx.
		recipientKB.GetAddr().String(),      // NGM To or bep2Addr of Deputy of source chain.
		"0x"+recipientKB.GetAddr().String(), // NGM destChainRecipientAddr: Claimant or funds' intended user at dest chain.
		"0x"+c.Keybase.GetAddr().String(),   // NGM destChainDeputyAddr: Deputy of destination chain.
		hash,
		tm,
		outCoins, // NGM Amount
		timespanMin,
	)

	// ------- Create Swap Trx
	resp, err := c.PostTxGrpc(
		c.Keybase.GetAddr().String(), signerStrIdx,
		c.Keybase.Keyring, swapMsg,
	)
	require.NoError(t, err)
	require.True(t, resp.Code == 0)

	require.True(
		t, len(resp.TxHash) != 0, "swapMsg trx hash should not be empty",
	)

	height = incChainHeight(t, c, height)

	swap, err = c.GetSwapByID(swapID)
	require.NoError(t, err)
	require.NotNil(t, swap)
	require.Equal(t, swap.Status, bep3.Open)

	balanceSignerAfterSwap, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[signerIdx], denom,
	)
	require.NoError(t, err)

	t.Log(
		"signer's balance after swap:", balanceSignerAfterSwap,
		"diff: neg(-)", balanceSignerBefSwap.Sub(*balanceSignerAfterSwap),
	)

	balanceRecAfterSwap, err := c.GetDenomBalanceGRPC(
		eMoneyUserAddrs[recipientIndex], denom,
	)
	require.NoError(t, err)

	t.Log(
		"receiver's balance after swap:", balanceRecAfterSwap,
		"diff:", balanceRecBefSwap.Sub(*balanceRecAfterSwap),
	)

	/*
	 * Swap Trx Balance assertions
	 */

	swapTrxFee := sdk.NewCoin(emc.EmoneyTrxCoinFee.GetDenom(), emc.EmoneyTrxCoinFee.Amount)
	require.True(
		t, balanceSignerBefSwap.Equal(
			balanceSignerAfterSwap.Add(swapAmnt).Add(swapTrxFee),
		),
	)

	require.True(t, balanceRecBefSwap.Equal(balanceRecAfterSwap))

	return balanceSignerBefSwap,
		balanceRecBefSwap,
		height,
		c,
		randomNumber,
		swapID,
		swap
}

func incChainHeight(t *testing.T, c *emc.Client, height int64) int64 {
	// Will not exhaust it in most cases
	var timeOutDur = 8 * time.Second

	newHeight, err := c.WaitForHeight(
		height+1, timeOutDur,
	)
	require.NoError(t, err)
	require.True(t, newHeight > height)

	return newHeight
}
