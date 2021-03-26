# Http Client for Cosmos SDK

This is a fork of the [Kava](https://www.kava.io/) [Client](https://github.com/kava-labs/go-sdk/client) implementation.

This repo's main purpose is to decouple the client Cosmos implementation from the larger Kava sdk and make it easier to use in other Cosmos SDK based blockchains such as e-Money's.

## Components 

This Go SDK includes the following components:
- client: sends transactions and queries to the Cosmos blockchain
- bep3: msgs and types from the bep3 blockchain required for complete codec registration
- keys: management of private keys and account recovery from mnenomic phrase based on Bitcoin bip39 standard.

### Client

To initialize a new client we'll need to set up the codec and pass it into the constructor

```go
// Required imports
import (
	"github.com/e-money/client"
)
    
// Set up cosmos prefixes and codec
config := sdk.GetConfig()
emledger.SetBech32AddressPrefixes(config)
cdc := emledger.MakeCodec()

// Initialize new client and set codec
client := client.NewClient(cdc, mnemonic, emledger.Bip44CoinType, rpcAddr, networkTestnet)
client.Keybase.SetCodec(cdc)
```

Let's use our new client to query the e-Money blockchain for information about an account

```go
address := "emoney1wxenlc8xpeq6jxmdklznzfam6svsnms9m7vhap"
addr, err := sdk.AccAddressFromBech32(address)
if err != nil {
    panic(err)
}

acc, err := client.GetAccount(addr)
if err != nil {
    panic(err)
}

fmt.Println("Account:", acc)
```

### Keys

Client uses the keys package for signing transactions, but keys can also be used standalone. The following example shows how to create a new key manager from a mnemonic phrase according to the Bitcoin bip39.

```go
// Required imports
import (
    app "github.com/e-money/client"
    "github.com/emoney/client/keys"
)

// Create a new mnemonic key manager
mnemonic := "secret words that unlock your address"
keybase, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
if err != nil {
    fmt.Println(err)
}
```

## Version compatibility

The go-sdk is compatible with other libraries that use different versions of Tendermint and the Cosmos SDK that employ the amino serialization. Specifically it uses the tendermint v0.33.3 and the cosmos-sdk v0.39.1 dependencies:
- github.com/cosmos/cosmos-sdk v0.39.1
- github.com/tendermint/tendermint v0.33.8
