package keys

import (
	"testing"

	"github.com/e-money/em-ledger/types"
	"github.com/stretchr/testify/require"
)

const (
	TestMnenomic     = "equip town gesture square tomorrow volume nephew minute witness beef rich gadget actress egg sing secret pole winter alarm law today check violin uncover"
	TestExpectedAddr = "emoney1se7u38fjc4jd88tacwvzwyggujl52ydn02uzy7"
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
			keyManager, err := NewMnemonicKeyManager(tc.mnenomic, tc.coinID)

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
