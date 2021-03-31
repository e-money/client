package client_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	emtypes "github.com/e-money/em-ledger/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bep3 "github.com/e-money/bep3/module"
	emc "github.com/e-money/client"
	"github.com/e-money/client/keys"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	denom       = "ungm"
	grpcSrv     = "127.0.0.1:9090"
	rpcAddr     = "tcp://localhost:26657"
	defGasLimit = 100_000
	defNetId    = "localnet_reuse"
	signerIdx   = 1
	recipientIndex = 0
)

type testsSuite struct {
	conn *grpc.ClientConn
	ProtoTxCfg  client.TxConfig
}

var (
	eMoneyUserAddrs []string
	signerStrIdx    = strconv.Itoa(signerIdx)
	randomKeyNumber []byte

	tests testsSuite

	emoneyUserMnemonics = []string{
		"play witness auto coast domain win tiny dress glare bamboo rent mule delay exact arctic vacuum laptop hidden siren sudden six tired fragile penalty",		"document weekend believe whip diesel earth hope elder quiz pact assist quarter public deal height pulp roof organ animal health month holiday front pencil",
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
		return 0, nil, nil, fmt.Errorf("couldn't generate random number: %w", err)
	}

	rndHash = bep3.CalculateRandomHash(rndNum, timestamp)

	return timestamp, rndNum, rndHash, err
}

func TestSwap(t *testing.T) {
	signerClient := emc.NewClient(emoneyUserMnemonics[signerIdx], signerStrIdx, rpcAddr)

	balanceSender, err := signerClient.GetDenomBalanceGRPC(eMoneyUserAddrs[signerIdx], eMoneyUserAddrs[signerIdx])
	require.NoError(t, err)
	coin := sdk.NewCoin(denom, sdk.NewInt(100_000))
	require.Truef(t, balanceSender.IsGTE(coin), "%s not enough balanceSender to cover swapping fees", signerClient.Keybase.GetAddr().String())

	recipientKB, err := keys.NewMnemonicKeyManager(emoneyUserMnemonics[recipientIndex], strconv.Itoa(recipientIndex))
	require.NoError(t, err)
	balanceRec, err := signerClient.GetDenomBalanceGRPC(recipientKB.GetAddr().String(), denom)
	require.NoError(t, err)
	t.Log("receiver's balance", balanceRec)

	swapAmnt := sdk.NewInt64Coin(denom, 100)
	outCoins := sdk.NewCoins(swapAmnt)

	tm, rndNum, hash, err := genSwapKeys()
	require.NoError(t, err)

	randomKeyNumber = rndNum
	require.True(t, len(randomKeyNumber)>0, "round number should not be empty")
	require.Len(t, hash, 32, "hash length should be 32")

	createMsg := bep3.NewMsgCreateAtomicSwap(
		signerClient.Keybase.GetAddr(),        // User creating swap trx.
		recipientKB.GetAddr(),             // NGM To or bep2Addr of Deputy of source chain.
		"0x"+signerClient.Keybase.GetAddr().String(), // NGM destChainRecipientAddr: Claimant or funds' intended user at dest chain.
		"0x"+recipientKB.GetAddr().String(),    // NGM destChainDeputyAddr: Deputy of destination chain.
		hash[:],
		tm,
		outCoins, // NGM Amount
		600,      // 10 min (10 * 60 sec)
	)

	resp, err := signerClient.PostTxGrpc(signerClient.Keybase.GetAddr().String(), signerStrIdx, signerClient.Keybase.Keyring, createMsg)
	require.NoError(t, err)

	require.True(t, len(resp.TxHash) != 0, "swap trx hash should not be empty")

	balanceRecAfter, err := signerClient.GetDenomBalanceGRPC(recipientKB.GetAddr().String(), denom)
	require.NoError(t, err)
	t.Log("receiver's balance", balanceRecAfter)

	require.Equal(t, balanceRec.Add(swapAmnt), balanceRecAfter, "balance before + swap not equal to balance after")
}

