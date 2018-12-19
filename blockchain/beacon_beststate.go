package blockchain

import (
	"crypto/sha256"
	"errors"
	"github.com/ninjadotorg/constant/common"
	"sort"
	"strconv"
	"strings"
)

// BestState houses information about the current best block and other info
// related to the state of the main chain as it exists from the point of view of
// the current best block.
//
// The BestSnapshot method can be used to obtain access to this information
// in a concurrent safe manner and the data will not be changed out from under
// the caller when chain state changes occur as the function name implies.
// However, the returned snapshot must be treated as immutable since it is
// shared by all callers.
type BestStateBeacon struct {
	BestBlockHash common.Hash  // The hash of the block.
	BestBlock     *BeaconBlock // The block.
	BestShardHash []common.Hash

	BeaconHeight uint64

	BeaconCommittee        []string
	BeaconPendingCommittee []string

	// assigned candidate
	CandidateShardWaitingForCurrentRandom  []string
	CandidateBeaconWaitingForCurrentRandom []string

	// assigned candidate
	CandidateShardWaitingForNextRandom  []string
	CandidateBeaconWaitingForNextRandom []string

	// validator of shards
	ShardCommittee map[byte][]string
	// pending validator of shards
	ShardPendingValidator map[byte][]string

	// UnassignBeaconCandidate []string
	// UnassignShardCandidate  []string

	CurrentRandomNumber int64
	NextRandomNumber    int64

	Params map[string]string
}

/*
Init create a beststate data from block and commitment tree
*/
// #1 - block
// #2 - commitment merkle tree
func (self *BestStateBeacon) Init(block *BeaconBlock) {
	self.BestBlock = block
	self.BestBlockHash = *block.Hash()
}

func (self *BestStateBeacon) Update(newBlock *BeaconBlock) error {
	//TODO:
	/*
		//variable
		// swap 3 validators each time
		const offset = int(3)
		// shardSwapValidator := make(map[byte][]string)
		beaconSwapValidator := []string{}

		// update BestShardHash, BestBlock, BestBlockHash
		self.BestBlockHash = *newBlock.Hash()
		self.BestBlock = newBlock
		shardState := newBlock.Body.ShardState
		for idx, l := range shardState {
			self.BestShardHash[idx] = l[len(l)-1]
		}



			// Assign Validator
			// Shuffle candidate + validator for beacon
			if self.BeaconHeight%200 == 1 {
				newBeaconNode, newShardNode := GetStakingCandidate(newBlock)
				self.UnassignBeaconCandidate = append(self.UnassignBeaconCandidate, newBeaconNode...)
				self.UnassignShardCandidate = append(self.UnassignShardCandidate, newShardNode...)

				/// Shuffle candidate for shard
				// assign UnassignShardCandidate to ShardPendingValidator with CurrentRandom this shard
				err := AssignValidatorShard(self.ShardPendingValidator, self.UnassignShardCandidate, self.CurrentRandomNumber)
				// reset self.UnassignShardCandidate
				self.UnassignShardCandidate = []string{}

				if err != nil {
					return self, err
				}
				// for i := 0; i < 256; i++ {
				// 	shardID := byte(i)
				// 	//swap validator for each shard
				// 	self.ShardPendingValidator[shardID], self.ShardValidator[shardID], shardSwapValidator[shardID], err = SwapValidator(self.ShardPendingValidator[shardID], self.ShardValidator[shardID], offset)
				// 	if err != nil {
				// 		return self, err
				// 	}
				// }
				// ShuffleCandidate
				shuffleBeaconCandidate, err := ShuffleCandidate(self.UnassignBeaconCandidate, self.CurrentRandomNumber)
				if err != nil {
					return self, err
				}
				// append new candidate to pending validator in beacon
				self.BeaconPendingCandidate = append(self.BeaconPendingCandidate, shuffleBeaconCandidate...)
				// reset UnassignBeaconCandidate
				self.UnassignBeaconCandidate = []string{}
				//swap validator in beacon
				self.BeaconPendingCandidate, self.BeaconCandidate, beaconSwapValidator, err = SwapValidator(self.BeaconPendingCandidate, self.BeaconCandidate, offset)
				if err != nil {
					return self, err
				}
				// update random number for new epoch
				fmt.Println(beaconSwapValidator)
				self.CurrentRandomNumber = self.NextRandomNumber
			} else {
				// GetStakingCandidate -> UnassignCandidate
				newBeaconNode, newShardNode := GetStakingCandidate(newBlock)
				self.UnassignBeaconCandidate = append(self.UnassignBeaconCandidate, newBeaconNode...)
				self.UnassignShardCandidate = append(self.UnassignShardCandidate, newShardNode...)
			}

			// update param
			instructions := newBlock.Body.Instructions

			for _, l := range instructions {
				if l[0] == "set" {
					self.Params[l[1]] = l[2]
				}
				if l[0] == "del" {
					delete(self.Params, l[1])
				}
				if l[0] == "swap" {
					//TODO: remove from candidate list
					// format
					// ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,...") "shard" "shardID"]
					// ["swap" "inPubkey1,inPubkey2,..." "outPupkey1, outPubkey2,...") "beacon"]
					inPubkeys := strings.Split(l[1], ",")
					outPubkeys := strings.Split(l[2], ",")
					if l[3] == "shard" {
						temp, err := strconv.Atoi(l[4])
						if err != nil {
							return self, nil
						}
						shardID := byte(temp)
						self.ShardPendingValidator[shardID], err = RemoveValidator(self.ShardPendingValidator[shardID], inPubkeys)
						if err != nil {
							return self, nil
						}
						self.ShardValidator[shardID], err = RemoveValidator(self.ShardPendingValidator[shardID], outPubkeys)
						if err != nil {
							return self, nil
						}
						self.ShardValidator[shardID] = append(self.ShardValidator[shardID], inPubkeys...)
					} else if l[3] == "beacon" {
						var err error
						self.BeaconPendingCandidate, err = RemoveValidator(self.BeaconPendingCandidate, inPubkeys)
						if err != nil {
							return self, nil
						}
						self.BeaconCandidate, err = RemoveValidator(self.BeaconPendingCandidate, outPubkeys)
						if err != nil {
							return self, nil
						}
						self.BeaconCandidate = append(self.BeaconCandidate, inPubkeys...)
					}
				}
			}
	*/
	return nil
}

func GetStakingCandidate(beaconBlock BeaconBlock) (beacon []string, shard []string) {

	beaconBlockBody := beaconBlock.Body
	for _, v := range beaconBlockBody.Instructions {
		if v[0] == "assign" && v[2] == "beacon" {
			beacon = strings.Split(v[1], ",")
		}
		if v[0] == "assign" && v[2] == "shard" {
			shard = strings.Split(v[1], ",")
		}
	}

	return beacon, shard
}

// Assumption:
// validator and candidate public key encode as base58 string
// assume that candidates are already been checked
// Check validation of candidate in transaction
func AssignValidator(candidates []string, rand int64) (map[byte][]string, error) {
	pendingValidators := make(map[byte][]string)
	for _, candidate := range candidates {
		shardID := calculateHash(candidate, rand)
		pendingValidators[shardID] = append(pendingValidators[shardID], candidate)
	}
	return pendingValidators, nil
}

// AssignValidatorShard, param for better convenice than AssignValidator
func AssignValidatorShard(currentCandidates map[byte][]string, shardCandidates []string, rand int64) error {
	for _, candidate := range shardCandidates {
		shardID := calculateHash(candidate, rand)
		currentCandidates[shardID] = append(currentCandidates[shardID], candidate)
	}
	return nil
}

func VerifyValidator(candidate string, rand int64, shardID byte) (bool, error) {
	res := calculateHash(candidate, rand)
	if shardID == res {
		return true, nil
	} else {
		return false, nil
	}
}

// Formula ShardID: LSB[hash(candidatePubKey+randomNumber)]
// Last byte of hash(candidatePubKey+randomNumber)
func calculateHash(candidate string, rand int64) (shardID byte) {
	seed := candidate + strconv.Itoa(int(rand))
	hash := sha256.Sum256([]byte(seed))
	// fmt.Println("Candidate public key", candidate)
	// fmt.Println("Hash of candidate serialized pubkey and random number", hash)
	// fmt.Printf("\"%d\",\n", hash[len(hash)-1])
	// fmt.Println("Shard to be assign", hash[len(hash)-1])
	shardID = hash[len(hash)-1]
	return shardID
}

// consider these list as queue structure
// unqueue a number of validator out of currentValidators list
// enqueue a number of validator into currentValidators list <=> unqueue a number of validator out of pendingValidators list
// return value: #1 remaining pendingValidators, #2 new currentValidators # swap validator
func SwapValidator(pendingValidators []string, currentValidators []string, offset int) ([]string, []string, []string, error) {
	if offset == 0 {
		return pendingValidators, currentValidators, nil, errors.New("Can't not swap 0 validator")
	}
	// if number of pending validator is less or equal than offset, set offset equal to number of pending validator
	if offset > len(pendingValidators) {
		offset = len(pendingValidators)
	}
	// do nothing
	if offset == 0 {
		return pendingValidators, currentValidators, nil, errors.New("No pending validator for swapping")
	}
	if offset > len(currentValidators) {
		return pendingValidators, currentValidators, nil, errors.New("Trying to swap too many validator")
	}
	swapValidator := currentValidators[:offset]
	// unqueue validator with index from 0 to offset-1 from currentValidators list
	currentValidators = currentValidators[offset:]
	// unqueue validator with index from 0 to offset-1 from currentValidators list
	tempValidators := pendingValidators[:offset]
	// save new pending validators list
	pendingValidators = pendingValidators[offset:]

	// enqueue new validator to the remaning of current validators list
	currentValidators = append(currentValidators, tempValidators...)
	return pendingValidators, currentValidators, swapValidator, nil
}

// return: #param1: validator list after remove
// in parameter: #param1: list of full validator
// in parameter: #param2: list of removed validator
// removed validators list must be a subset of full validator list and it must be first in the list
func RemoveValidator(validators []string, removedValidators []string) ([]string, error) {
	// if number of pending validator is less or equal than offset, set offset equal to number of pending validator
	if len(removedValidators) > len(validators) {
		return validators, errors.New("Trying to remove too many validators")
	}

	for index, validator := range removedValidators {
		if strings.Compare(validators[index], validator) == 0 {
			validators = validators[1:]
		} else {
			return validators, errors.New("Remove Validator with Wrong Format")
		}
	}
	return validators, nil
}

func ShuffleCandidate(candidates []string, rand int64) ([]string, error) {
	hashes := []string{}
	m := make(map[string]string)
	sortedCandidate := []string{}
	for _, candidate := range candidates {
		seed := candidate + strconv.Itoa(int(rand))
		hash := sha256.Sum256([]byte(seed))
		hashes = append(hashes, string(hash[:32]))
		m[string(hash[:32])] = candidate
	}
	sort.Strings(hashes)
	for _, candidate := range hashes {
		sortedCandidate = append(sortedCandidate, candidate)
	}
	return hashes, nil
}