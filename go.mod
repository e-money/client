module github.com/e-money/client

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.42.3
	github.com/e-money/bep3 v0.2.2-0.20210314111415-cb3754badb4e
	github.com/e-money/em-ledger v0.9.3-0.20210311115650-b3ce4016e552
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.8
	google.golang.org/grpc v1.35.0
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace github.com/e-money/bep3 => ../bep3

// https://docs.cosmos.network/v0.41/core/grpc_rest.html#grpc-server
replace google.golang.org/grpc => google.golang.org/grpc v1.33.2