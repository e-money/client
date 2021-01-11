package keys

import (
	"testing"

	"github.com/e-money/em-ledger/types"
	"github.com/stretchr/testify/require"
)

const (
	TestMnenomic     = "play witness auto coast domain win tiny dress glare bamboo rent mule delay exact arctic vacuum laptop hidden siren sudden six tired fragile penalty"
	TestExpectedAddr = "emoney1lagqmceycrfpkyu7y6ayrk6jyvru5mkrezacpw"
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
