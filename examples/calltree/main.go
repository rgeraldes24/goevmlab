// Copyright 2019 Martin Holst Swende
// This file is part of the goevmlab library.
//
// The library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the goevmlab library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	common2 "github.com/rgeraldes24/goevmlab/common"
	"github.com/rgeraldes24/goevmlab/ops"
	"github.com/rgeraldes24/goevmlab/program"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/state"
	"github.com/theQRL/go-zond/core/vm"
	"github.com/theQRL/go-zond/core/vm/runtime"
	"github.com/theQRL/go-zond/params"
)

type dumbTracer struct {
	common2.BasicTracer
	counter uint64
}

func (d *dumbTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	fmt.Printf("captureStart\n")
	fmt.Printf("	from: %v\n", from.Hex())
	fmt.Printf("	to: %v\n", to.Hex())
}

func (d *dumbTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	fmt.Printf("\nCaptureEnd\n")
	fmt.Printf("Counter: %d\n", d.counter)
}

func (d *dumbTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if op == vm.CALL {
		if depth == 1 {
			fmt.Println("")
		} else {
			d.counter++
		}
		if depth < 2 {
			fmt.Printf("(%d: %d)", depth, gas)
		}
	}
}

func main() {

	if err := program.RunProgram(runit); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func runit() error {
	a := program.NewProgram()
	b := program.NewProgram()

	aAddr := common.HexToAddress("0xff0a")
	bAddr := common.HexToAddress("0xff0b")

	dest := a.Jumpdest()
	a.Call(nil, bAddr, nil, 0, 0, 0, 0)
	a.Jump(dest)

	// The self-call can be done a bit more clever, gas-wise

	b.Op(ops.PC)      // get zero on stack (out size)
	b.Op(ops.DUP1)    // out offset
	b.Op(ops.DUP1)    // insize
	b.Op(ops.DUP1)    // inoffset
	b.Op(ops.DUP1)    // value
	b.Op(ops.ADDRESS) // address
	b.Op(ops.GAS)     // Gas
	b.Op(ops.CALL)

	alloc := make(core.GenesisAlloc)
	alloc[aAddr] = core.GenesisAccount{
		Nonce:   0,
		Code:    a.Bytecode(),
		Balance: big.NewInt(0xffffffff),
	}
	alloc[bAddr] = core.GenesisAccount{
		Nonce:   0,
		Code:    b.Bytecode(),
		Balance: big.NewInt(0xffffffff),
	}
	//-------------

	outp, err := json.MarshalIndent(alloc, "", " ")
	if err != nil {
		fmt.Printf("error : %v", err)
		os.Exit(1)
	}
	fmt.Printf("output \n%v\n", string(outp))
	//----------
	var (
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		sender     = common.BytesToAddress([]byte("sender"))
	)
	for addr, acc := range alloc {
		statedb.CreateAccount(addr)
		statedb.SetCode(addr, acc.Code)
		statedb.SetNonce(addr, acc.Nonce)
		if acc.Balance != nil {
			statedb.SetBalance(addr, acc.Balance)
		}
	}
	statedb.CreateAccount(sender)
	var vmConf vm.Config
	if false {
		vmConf = vm.Config{
			Tracer: &dumbTracer{},
		}
	}
	runtimeConfig := runtime.Config{
		Origin:      sender,
		State:       statedb,
		GasLimit:    10000000,
		BlockNumber: new(big.Int).SetUint64(1),
		ChainConfig: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
		EVMConfig: vmConf,
	}
	// Diagnose it
	t0 := time.Now()
	_, _, _ = runtime.Call(aAddr, nil, &runtimeConfig)
	t1 := time.Since(t0)
	fmt.Printf("Time elapsed: %v\n", t1)
	t0 = time.Now()
	_, _, err = runtime.Call(aAddr, nil, &runtimeConfig)
	t1 = time.Since(t0)
	fmt.Printf("Time elapsed: %v\n", t1)
	return err
}
