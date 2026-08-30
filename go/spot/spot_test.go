package spot

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"testing"

	onchainSui "github.com/k4k3ru-hub/onchain/go/sui"
)

type testSimulator struct {
	result  *onchainSui.SimulationResult
	request onchainSui.SimulationRequest
}

// SimulateTransaction returns the configured simulation result.
//
// Version:
//   - 2026-08-31: Added.
func (s *testSimulator) SimulateTransaction(_ context.Context, request onchainSui.SimulationRequest) (*onchainSui.SimulationResult, error) {
	s.request = request
	return s.result, nil
}

func TestPoolQuoteAndFlashSwap(t *testing.T) {
	poolAddress, _ := onchainSui.ParseAddress("0x9")
	object := &onchainSui.Object{Address: poolAddress, Version: 3, Move: &onchainSui.MoveObject{Type: "0x1::pool::Pool<0x2::sui::SUI, 0x3::usdc::USDC>", JSON: json.RawMessage(`{"coin_a":"100","coin_b":"200","current_sqrt_price":"10","liquidity":"300","fee_rate":"100","is_paused":false}`)}}
	pool, err := ParsePool(object)
	if err != nil {
		t.Fatalf("ParsePool() returned an unexpected error: %v", err)
	}
	value := make([]byte, 135)
	binary.LittleEndian.PutUint64(value[2:10], 100)
	binary.LittleEndian.PutUint64(value[18:26], 99)
	binary.LittleEndian.PutUint64(value[42:50], 1)
	value[74] = 11
	simulator := &testSimulator{result: &onchainSui.SimulationResult{CommandResults: []onchainSui.SimulationCommandResult{{ReturnValues: []onchainSui.CommandOutput{{BCS: value}}}}}}
	quoter, err := NewQuoter(testDeployment(), simulator)
	if err != nil {
		t.Fatalf("NewQuoter() returned an unexpected error: %v", err)
	}
	sender, _ := onchainSui.ParseAddress("0xa")
	quote, err := quoter.QuoteExactInput(context.Background(), QuoteExactInputParams{Sender: sender, Pool: *pool, AmountIn: 100, A2B: true, SqrtPriceLimit: big.NewInt(5)})
	if err != nil {
		t.Fatalf("QuoteExactInput() returned an unexpected error: %v", err)
	}
	if quote.AmountOut != 99 || simulator.request.Transaction.Commands[0].MoveCall.Function != "calculate_swap_results" {
		t.Fatalf("quote=%+v transaction=%+v", quote, simulator.request.Transaction)
	}
	builder := onchainSui.NewProgrammableTransactionBuilder()
	flash, err := AppendFlashSwap(builder, testDeployment(), *pool, true, 100, big.NewInt(5))
	if err != nil {
		t.Fatalf("AppendFlashSwap() returned an unexpected error: %v", err)
	}
	if err := AppendRepayFlashSwap(builder, testDeployment(), *pool, flash, flash.BalanceA, flash.BalanceB); err != nil {
		t.Fatalf("AppendRepayFlashSwap() returned an unexpected error: %v", err)
	}
	tx, err := builder.Build()
	if err != nil || len(tx.Commands) != 2 {
		t.Fatalf("Build()=%+v, %v", tx, err)
	}
}

func testDeployment() Deployment {
	packageAddress, _ := onchainSui.ParseAddress("0x1")
	config, _ := onchainSui.ParseAddress("0x2")
	clock, _ := onchainSui.ParseAddress("0x6")
	return Deployment{Package: packageAddress, PublishedAt: packageAddress, GlobalConfig: onchainSui.ObjectInput{Address: config, Version: 1}, Clock: onchainSui.ObjectInput{Address: clock, Version: 1}, PoolModule: "pool", EventsModule: "events"}
}
