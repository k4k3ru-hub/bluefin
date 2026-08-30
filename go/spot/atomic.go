package spot

import (
	"fmt"
	"math/big"

	onchainSui "github.com/k4k3ru-hub/onchain/go/sui"
)

type FlashSwap struct {
	Config   onchainSui.Argument
	Pool     onchainSui.Argument
	BalanceA onchainSui.Argument
	BalanceB onchainSui.Argument
	Receipt  onchainSui.Argument
}

// AppendFlashSwap appends a Bluefin Spot flash swap to a programmable transaction.
//
// Parameters:
//   - builder: Programmable transaction builder.
//   - deployment: Bluefin deployment.
//   - pool: Bluefin pool.
//   - a2b: Swap direction.
//   - amount: Exact input amount.
//   - sqrtPriceLimit: Q64.64 price limit.
//
// Returns:
//   - Flash-swap arguments.
//   - Validation error.
//
// Version:
//   - 2026-08-30: Added.
func AppendFlashSwap(builder *onchainSui.ProgrammableTransactionBuilder, deployment Deployment, pool Pool, a2b bool, amount uint64, sqrtPriceLimit *big.Int) (FlashSwap, error) {
	if builder == nil {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: builder=null")
	}
	if err := deployment.Validate(); err != nil {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: %w", err)
	}
	if pool.Address.IsZero() || pool.InitialVersion == 0 || amount == 0 || sqrtPriceLimit == nil || sqrtPriceLimit.Sign() <= 0 || sqrtPriceLimit.BitLen() > 128 {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: parameters=invalid")
	}
	clock := deployment.Clock
	clock.Mutable = false
	clockArg, err := builder.Object(onchainSui.InputKindShared, clock)
	if err != nil {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: %w", err)
	}
	config := deployment.GlobalConfig
	config.Mutable = false
	configArg, err := builder.Object(onchainSui.InputKindShared, config)
	if err != nil {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: %w", err)
	}
	poolArg, err := builder.Object(onchainSui.InputKindShared, onchainSui.ObjectInput{Address: pool.Address, Version: pool.InitialVersion, Mutable: true})
	if err != nil {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: %w", err)
	}
	direction, _ := builder.Pure(bcsBool(a2b))
	exact, _ := builder.Pure(bcsBool(true))
	amountArg, _ := builder.Pure(bcsUint64(amount))
	limit, _ := builder.Pure(bcsUint128(sqrtPriceLimit))
	result, err := builder.MoveCall(onchainSui.MoveCall{Package: deployment.PublishedAt, Module: deployment.PoolModule, Function: "flash_swap", TypeArguments: []string{pool.CoinTypeA, pool.CoinTypeB}, Arguments: []onchainSui.Argument{clockArg, configArg, poolArg, direction, exact, amountArg, limit}})
	if err != nil {
		return FlashSwap{}, fmt.Errorf("failed to append bluefin spot flash swap: %w", err)
	}
	balanceA, _ := onchainSui.NestedResult(result, 0)
	balanceB, _ := onchainSui.NestedResult(result, 1)
	receipt, _ := onchainSui.NestedResult(result, 2)
	return FlashSwap{Config: configArg, Pool: poolArg, BalanceA: balanceA, BalanceB: balanceB, Receipt: receipt}, nil
}

// AppendRepayFlashSwap appends Bluefin Spot flash-swap repayment.
//
// Parameters:
//   - builder: Programmable transaction builder.
//   - deployment: Bluefin deployment.
//   - pool: Bluefin pool.
//   - flashSwap: Original flash-swap arguments.
//   - balanceA: Repayment balance A.
//   - balanceB: Repayment balance B.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-30: Added.
func AppendRepayFlashSwap(builder *onchainSui.ProgrammableTransactionBuilder, deployment Deployment, pool Pool, flashSwap FlashSwap, balanceA, balanceB onchainSui.Argument) error {
	if builder == nil {
		return fmt.Errorf("failed to append bluefin spot flash swap repayment: builder=null")
	}
	_, err := builder.MoveCall(onchainSui.MoveCall{Package: deployment.PublishedAt, Module: deployment.PoolModule, Function: "repay_flash_swap", TypeArguments: []string{pool.CoinTypeA, pool.CoinTypeB}, Arguments: []onchainSui.Argument{flashSwap.Config, flashSwap.Pool, balanceA, balanceB, flashSwap.Receipt}})
	if err != nil {
		return fmt.Errorf("failed to append bluefin spot flash swap repayment: %w", err)
	}
	return nil
}
