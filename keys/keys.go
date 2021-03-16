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

// KeyManager is an interface for common methods on KeyManagers
type KeyManager interface {
	GetPrivKey() cryptotypes.PrivKey
	GetAddr() sdk.AccAddress
	Sign(legacytx.StdSignMsg, *codec.LegacyAmino) ([]byte, error)
}

type keyManager struct {
	Account  keyring.Info
	privKey  cryptotypes.PrivKey
	addr     sdk.AccAddress
	mnemonic string
}

// NewMnemonicKeyManager creates a new KeyManager from a mnenomic
func NewMnemonicKeyManager(mnemonic string) (KeyManager, error) {
	hdPath := sdk.FullFundraiserPath
	prvKey, err := GenPrvKey(mnemonic, defaultBIP39Passphrase, hdPath, hd.Secp256k1)
	keybase, err := keyring.New(
		appName, keyring.BackendMemory, hdPath, nil,
	)
	if err != nil {
		return nil, err
	}
	memInfo, err :=
		keybase.NewAccount("",
			mnemonic, defaultBIP39Passphrase, hdPath, hd.Secp256k1)
	if err != nil {
		return nil, err
	}

	k := keyManager{
		Account:  memInfo,
		privKey:  prvKey,
		addr:     memInfo.GetAddress(),
		mnemonic: mnemonic,
	}
	return &k, err
}

func GenPrvKey(mnemonic, bip39Passwd, hdPath string, algo keyring.SignatureAlgo) (cryptotypes.PrivKey, error) {
	// create master key and derive first key for keyring
	derivedPriv, err := hd.Secp256k1.Derive()(mnemonic, bip39Passwd, hdPath)
	if err != nil {
		return nil, err
	}

	return hd.Secp256k1.Generate()(derivedPriv), nil
}

func (m *keyManager) GetPrivKey() cryptotypes.PrivKey {
	return m.privKey
}

func (m *keyManager) GetAddr() sdk.AccAddress {
	return m.addr
}

// Sign signs a standard msg and marshals the result to bytes
func (m *keyManager) Sign(stdMsg legacytx.StdSignMsg, cdc *codec.LegacyAmino) ([]byte, error) {
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

func (m *keyManager) makeSignature(msg legacytx.StdSignMsg) (sig legacytx.StdSignature, err error) {
	sigBytes, err := m.privKey.Sign(msg.Bytes())
	if err != nil {
		return
	}

	return legacytx.StdSignature{
		PubKey:    m.privKey.PubKey(),
		Signature: sigBytes,
	}, nil
}
