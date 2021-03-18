package keys

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
)

const (
	defaultBIP39Passphrase = ""
	appName                = "e-Money-client"
)

type KeyManager struct {
	Keyring  keyring.Keyring
	Account  keyring.Info
	privKey  cryptotypes.PrivKey
	addr     sdk.AccAddress
	mnemonic string
}

// NewMnemonicKeyManager creates a new KeyManager from a mnenomic
func NewMnemonicKeyManager(mnemonic, accountName string) (*KeyManager, error) {
	hdPath := sdk.FullFundraiserPath
	prvKey, err := GenPrvKeyByMnemonic(mnemonic, defaultBIP39Passphrase, hdPath, hd.Secp256k1)
	keybase, err := keyring.New(
		appName, keyring.BackendMemory, hdPath, nil,
	)
	if err != nil {
		return nil, err
	}
	memInfo, err :=
		keybase.NewAccount(accountName,
			mnemonic, defaultBIP39Passphrase, hdPath, hd.Secp256k1)
	if err != nil {
		return nil, err
	}

	k := KeyManager{
		Keyring:  keybase,
		Account:  memInfo,
		privKey:  prvKey,
		addr:     memInfo.GetAddress(),
		mnemonic: mnemonic,
	}
	return &k, err
}

func GenPrvKeyByMnemonic(mnemonic, bip39Passwd, hdPath string, algo keyring.SignatureAlgo) (cryptotypes.PrivKey, error) {
	// create master key and derive first key for keyring
	derivedPriv, err := hd.Secp256k1.Derive()(mnemonic, bip39Passwd, hdPath)
	if err != nil {
		return nil, err
	}

	return hd.Secp256k1.Generate()(derivedPriv), nil
}

func (m *KeyManager) GetPrivKey() cryptotypes.PrivKey {
	return m.privKey
}

func (m *KeyManager) GetAddr() sdk.AccAddress {
	return m.addr
}

// Sign signs a standard msg and marshals the result to bytes
// TODO evaluate for removal
func (m *KeyManager) Sign(stdMsg legacytx.StdSignMsg, cdc *codec.LegacyAmino) ([]byte, error) {
	sig, err := m.makeSignature(stdMsg)
	if err != nil {
		return nil, err
	}

	newTx := legacytx.NewStdTx(stdMsg.Msgs, stdMsg.Fee,
		[]legacytx.StdSignature{sig}, stdMsg.Memo)

	bz, err := cdc.MarshalBinaryLengthPrefixed(&newTx)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

// TODO evaluate for removal
func (m *KeyManager) makeSignature(msg legacytx.StdSignMsg) (sig legacytx.StdSignature, err error) {
	sigBytes, err := m.privKey.Sign(msg.Bytes())
	if err != nil {
		return
	}

	return legacytx.StdSignature{
		PubKey:    m.privKey.PubKey(),
		Signature: sigBytes,
	}, nil
}
