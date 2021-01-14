package keys

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/e-money/em-ledger/types"
	"github.com/stretchr/testify/require"
)

const (
	TestMnenomic     = "play witness auto coast domain win tiny dress glare bamboo rent mule delay exact arctic vacuum laptop hidden siren sudden six tired fragile penalty"
	TestExpectedAddr = "emoney1lagqmceycrfpkyu7y6ayrk6jyvru5mkrezacpw"
	TestKavaMnenomic = "equip town gesture square tomorrow volume nephew minute witness beef rich gadget actress egg sing secret pole winter alarm law today check violin uncover"
	TestKavaCoinID   = 459
)

func TestNewMnemonicKeyManager(t *testing.T) {
	types.ConfigureSDK()

	tests := []struct {
		name       string
		mnenomic   string
		coinID     uint32
		expectpass bool
	}{
		{"normal", TestMnenomic, TestKavaCoinID, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keyManager, err := NewMnemonicKeyManager(tc.mnenomic)

			if tc.expectpass {
				require.Nil(t, err)

				// Confirm correct address
				addr := keyManager.GetAddr()
				require.Equal(t, TestExpectedAddr, addr.String())
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

const (
	Bech32KavaMainPrefix = "kava"
)

func SetBech32AddressPrefixes(config *sdk.Config) {
	config.SetBech32PrefixForAccount(Bech32KavaMainPrefix, Bech32KavaMainPrefix+sdk.PrefixPublic)
	config.SetBech32PrefixForValidator(Bech32KavaMainPrefix+sdk.PrefixValidator+sdk.PrefixOperator, Bech32KavaMainPrefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic)
	config.SetBech32PrefixForConsensusNode(Bech32KavaMainPrefix+sdk.PrefixValidator+sdk.PrefixConsensus, Bech32KavaMainPrefix+sdk.PrefixValidator+sdk.PrefixConsensus+sdk.PrefixPublic)
}

func TestNewKavaMnemonicKeyManager(t *testing.T) {
	tests := []struct {
		name       string
		mnenomic   string
		coinID     uint32
		expected   string
		expectpass bool
	}{
		{
			name: "normal", mnenomic: TestKavaMnenomic, coinID: TestKavaCoinID,
			// expected: "kava15qdefkmwswysgg4qxgqpqr35k3m49pkx2jdfnw",
			expected:   "kava1ffv7nhd3z6sych2qpqkk03ec6hzkmufy0r2s4c",
			expectpass: true,
		},
	}

	SetBech32AddressPrefixes(sdk.GetConfig())

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keyManager, err := NewKavaMnemonicKeyManager(tc.mnenomic, tc.coinID)

			if tc.expectpass {
				require.Nil(t, err)

				// Confirm correct address
				addr := keyManager.GetAddr()
				require.Equal(t, tc.expected, addr.String())
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
