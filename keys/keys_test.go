package keys

import (
	"fmt"
	"testing"

	"github.com/e-money/em-ledger/types"
	"github.com/stretchr/testify/require"
)

func TestNewMnemonicKeyManager(t *testing.T) {
	types.ConfigureSDK()

	tests := []struct {
		name         string
		mnenomic     string
		expectedAddr string
		expectpass   bool
	}{
		{
			name:         "first",
			mnenomic:     "play witness auto coast domain win tiny dress glare bamboo rent mule delay exact arctic vacuum laptop hidden siren sudden six tired fragile penalty",
			expectedAddr: "emoney1lagqmceycrfpkyu7y6ayrk6jyvru5mkrezacpw",
			expectpass:   true,
		},
		{
			name:         "second",
			mnenomic:     "equip town gesture square tomorrow volume nephew minute witness beef rich gadget actress egg sing secret pole winter alarm law today check violin uncover",
			expectedAddr: "emoney15qdefkmwswysgg4qxgqpqr35k3m49pkxeyrqj5",
			expectpass:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keyManager, err := NewMnemonicKeyManager(tc.mnenomic)

			if tc.expectpass {
				require.Nil(t, err)

				// Confirm correct address
				addr := keyManager.GetAddr()
				fmt.Println(tc.name, "addr:", addr)
				require.Equal(t, tc.expectedAddr, addr.String())
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
