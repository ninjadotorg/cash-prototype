package blockchain

import (
	"fmt"
	"strconv"
	"time"
)

type BeaconBlockGenerator struct{}

// @Hung: genesis should be build as configuration file like JSON
func (self *BeaconBlockGenerator) CreateBeaconGenesisBlock(
	version int,
	beaconNodes []string,
	shardNodes []string,
	icoParams IcoParams,
	salaryPerTx uint64,
	basicSalary uint64,
) *BlockV2 {

	loc, _ := time.LoadLocation("America/New_York")
	time := time.Date(2018, 8, 1, 0, 0, 0, 0, loc)

	//TODO: build param
	inst := [][]string{}
	// build validator beacon
	// test generate public key in utility/generateKeys
	// CHANGE preSelectBeaconNodeTestnetSerializedPubkey to beaconNodes in param
	// CHANGE preSelectShardNodeTestnetSerializedPubkey to shardNodes in param
	strBeacon := []string{"assign"}
	strBeacon = append(strBeacon, preSelectBeaconNodeTestnetSerializedPubkey...)
	strBeacon = append(strBeacon, "beacon")

	strShard := []string{"assign"}
	strShard = append(strShard, preSelectShardNodeTestnetSerializedPubkey...)
	strShard = append(strShard, "shard")
	inst = append(inst, strBeacon)
	inst = append(inst, strShard)
	// build network param
	inst = append(inst, []string{"set", "salaryPerTx", fmt.Sprintf("%v", salaryPerTx)})
	inst = append(inst, []string{"set", "basicSalary", fmt.Sprintf("%v", basicSalary)})
	inst = append(inst, []string{"set", "initialPaymentAddress", icoParams.InitialPaymentAddress})
	inst = append(inst, []string{"set", "initFundSalary", strconv.Itoa(int(icoParams.InitFundSalary))})
	inst = append(inst, []string{"set", "initialDCBToken", strconv.Itoa(int(icoParams.InitialDCBToken))})
	inst = append(inst, []string{"set", "initialCMBToken", strconv.Itoa(int(icoParams.InitialCMBToken))})
	inst = append(inst, []string{"set", "initialGOVToken", strconv.Itoa(int(icoParams.InitialGOVToken))})
	inst = append(inst, []string{"set", "initialBondToken", strconv.Itoa(int(icoParams.InitialBondToken))})

	body := &BeaconBlockBody{ShardState: nil, Instructions: nil}
	header := &BeaconBlockHeader{
		BlockHeaderGeneric: BlockHeaderGeneric{
			PrevBlockHash: nil,
			Timestamp:     time.Unix(),
			Height:        1,
			Version:       1,
		},
		DataHash: body.Hash(),
	}

	block := &BlockV2{
		AggregatedSig: nil,
		ProducerSig:   nil,
		ValidatorsIdx: nil,
		Type:          "beacon",
		Body:          body,
		Header:        header,
	}

	return block
}