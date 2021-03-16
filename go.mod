module github.com/e-money/client

go 1.15

require (
	github.com/bartekn/go-bip39 v0.0.0-20171116152956-a05967ea095d // indirect
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/cosmos/cosmos-sdk v0.42.1
	github.com/cosmos/go-bip39 v1.0.0
	github.com/e-money/bep3 v0.2.2-0.20210314111415-cb3754badb4e
	github.com/e-money/em-ledger v0.9.3-0.20210311115650-b3ce4016e552
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/iavl v0.14.0 // indirect
	github.com/tendermint/tendermint v0.34.8
	google.golang.org/grpc v1.35.0 // indirect
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
