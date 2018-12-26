package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ninjadotorg/constant/blockchain/btc/btcapi"

	"github.com/ninjadotorg/constant/common"
)

/*
Insert new block into beaconchain
1. Verify Block
	1.1 Verify Block (block height, parent block,...)
	1.2 Validate Block after process (root hash, random number,...)
2. Update: Process block
	2.1 Process BestStateBeacon
	2.2 Store BestStateBeacon
3. Store Block
*/

// func (self *BlockChain) ConnectBlockBeacon(block *BeaconBlock) error {
// 	self.chainLock.Lock()
// 	defer self.chainLock.Unlock()
// 	blockHash := block.Hash().String()

// 	Logger.log.Infof("Insert block %+v to Blockchain", blockHash)

// 	//===================Verify============================
// 	Logger.log.Infof("Verify Pre-Process block %+v to Blockchain", blockHash)

// 	err := self.VerifyPreProcessingBeaconBlock(block)
// 	if err != nil {
// 		Logger.log.Error("Error update best state for block", block, "in beacon chain")
// 		return NewBlockChainError(UnExpectedError, err)
// 	}

// 	//===================Post-Verify == Validation============================
// 	Logger.log.Infof("Verify Post-Process block %+v to Blockchain", blockHash)
// 	err = self.VerifyPostProcessingBeaconBlock(block)
// 	if err != nil {
// 		Logger.log.Error("Error Verify Post-Processing block", block, "in beacon chain")
// 		return NewBlockChainError(UnExpectedError, err)
// 	}

// 	//===================Process============================
// 	Logger.log.Infof("Process block %+v", blockHash)

// 	Logger.log.Infof("Process BeaconBestState block %+v", blockHash)
// 	// Process best state or not and store beststate
// 	err = self.BestState.Beacon.Update(block)
// 	if err != nil {
// 		Logger.log.Error("Error update best state for block", block, "in beacon chain")
// 		return NewBlockChainError(UnExpectedError, err)
// 	}
// 	//===================Store Block and BestState in cache======================
// 	return nil
// }

// Verify Block and Snapshot beststate + block if valid
func (self *BlockChain) MaybeAcceptBeaconBlock(block *BeaconBlock) (string, error) {
	self.chainLock.Lock()
	defer self.chainLock.Unlock()
	//========Verify block only
	if err := self.VerifyPreProcessingBeaconBlock(block); err != nil {
		return "", err
	}
	//========Verify block with previous best state
	// Get Beststate of previous block == previous best state
	// Clone best state value into new variable
	beaconBestState := BestStateBeacon{}
	// check with current final best state
	if strings.Compare(self.BestState.Beacon.BestBlockHash.String(), block.Header.PrevBlockHash.String()) == 0 {
		tempMarshal, err := json.Marshal(self.BestState.Beacon)
		if err != nil {
			return "", NewBlockChainError(UnmashallJsonBlockError, err)
		}
		json.Unmarshal(tempMarshal, beaconBestState)
	} else {
		// check with current cache best state
		var err error
		beaconBestState, err = self.GetMaybeAcceptBeaconBestState(block.Header.PrevBlockHash.String())
		if err != nil {
			return "", err
		}
	}
	// if no match best state found then block is unknown
	if reflect.DeepEqual(beaconBestState, BestStateBeacon{}) {
		return "", NewBlockChainError(BeaconError, errors.New("Beacon Block does not match with any Beacon State in cache or in Database"))
	}
	// Verify block with previous best state
	if err := beaconBestState.VerifyBestStateWithBeaconBlock(block); err != nil {
		return "", err
	}
	//========Update best state with new block
	if err := beaconBestState.Update(block); err != nil {
		return "", err
	}
	//========Post verififcation: verify new beaconstate with corresponding block
	if err := beaconBestState.VerifyPostProcessingBeaconBlock(block); err != nil {
		return "", err
	}

	//========Store new Beaconblock and new Beacon bestState in cache
	_, err := self.StoreMaybeAcceptBeaconBeststate(beaconBestState)
	if err != nil {
		return "", err
	}
	keyBL, err := self.StoreMaybeAcceptBeaconBlock(*block)
	if err != nil {
		return "", err
	}

	//=========Accept previous if new block is valid
	if err := self.AcceptBeaconBlock(&block.Header.PrevBlockHash); err != nil {
		return "", err
	}
	return keyBL, nil
}

//Store block & state offcial
func (self *BlockChain) AcceptBeaconBlock(blockHash *common.Hash) error {
	self.chainLock.Lock()
	defer self.chainLock.Unlock()
	// This function make sure if stored block at height 91, then best state height at 90
	beaconBlock, err := self.GetMaybeAcceptBeaconBlock(blockHash.String())
	if err != nil {
		return err
	}
	err = self.BestState.Beacon.Update(&beaconBlock)
	if err != nil {
		return err
	}
	// beaconBestState, err := self.GetMaybeAcceptBeaconBestState(blockHash.String())
	// if err != nil {
	// 	return err
	// }
	// if !reflect.DeepEqual(beaconBestState, self.BestState.Beacon) {
	// 	Logger.log.Error("Current best state and stored block %+v are not compatible", blockHash)
	// 	return NewBlockChainError(BeaconError, errors.New("Current best state and stored block are not compatible"))
	// }
	//===================Store Block============================
	Logger.log.Infof("Store Beacon block %+v", blockHash)
	if err := self.config.DataBase.StoreBeaconBlock(beaconBlock); err != nil {
		Logger.log.Error("Error store beacon block", blockHash, "in beacon chain")
		return err
	}

	//===================Store State============================
	Logger.log.Infof("Store BeaconBestState block %+v", blockHash)
	//Process stored block with current best state

	if err := self.config.DataBase.StoreBeaconBestState(self.BestState.Beacon); err != nil {
		Logger.log.Error("Error Store best state for block", blockHash, "in beacon chain")
		return NewBlockChainError(UnExpectedError, err)
	}
	Logger.log.Infof("Accepted block %+v", blockHash)
	return nil
}

func (self *BlockChain) VerifyPreProcessingBeaconBlock(block *BeaconBlock) error {
	/* Verify Pre-prosessing data
	This function DOES NOT verify new block with best state
	DO NOT USE THIS with GENESIS BLOCK
	- Sanity
	- Signature
		+ Producer validity
		+ Signature of producer
		+ agg signature
	- version
	- parent hash
	- Height = parent hash + 1
	- Epoch = blockHeight % Epoch ? Parent Epoch + 1
	- Timestamp can not excess some limit
	- Instruction hash
	- ShardStateHash
	*/
	//TODO: Verify sanity, signature
	// Verify version
	if block.Header.Version != VERSION {
		return NewBlockChainError(VersionError, errors.New("Version should be :"+strconv.Itoa(VERSION)))
	}
	prevBlockHash := block.Header.PrevBlockHash
	// Verify parent hash exist or not
	parentBlock, err := self.config.DataBase.FetchBeaconBlock(&prevBlockHash)
	if err != nil {
		return NewBlockChainError(DBError, err)
	}
	parentBlockInterface := NewBeaconBlock()
	json.Unmarshal(parentBlock, &parentBlockInterface)
	// Verify block height with parent block
	if parentBlockInterface.Header.Height+1 != block.Header.Height {
		return NewBlockChainError(BlockHeightError, errors.New("Block height of new block should be :"+strconv.Itoa(int(block.Header.Height+1))))
	}
	// Verify epoch with parent block
	if block.Header.Height%EPOCH == 0 && parentBlockInterface.Header.Epoch != block.Header.Epoch-1 {
		return NewBlockChainError(EpochError, errors.New("Block height and Epoch is not compatiable"))
	}
	// Verify timestamp with parent block
	if block.Header.Timestamp <= parentBlockInterface.Header.Timestamp {
		return NewBlockChainError(TimestampError, errors.New("Timestamp of new block can't equal to parent block"))
	}

	tempShardStateArr := []common.Hash{}
	for _, hashes := range block.Body.ShardState {
		tempShardStateArr = append(tempShardStateArr, hashes...)
	}
	if !VerifyHashFromHashArray(tempShardStateArr, block.Header.ShardStateHash) {
		return NewBlockChainError(ShardStateHashError, errors.New("Shard state hash is not correct"))
	}

	tempInstructionArr := []string{}
	for _, strs := range block.Body.Instructions {
		tempInstructionArr = append(tempInstructionArr, strs...)
	}
	if !VerifyHashFromStringArray(tempInstructionArr, block.Header.InstructionHash) {
		return NewBlockChainError(InstructionHashError, errors.New("Instruction hash is not correct"))
	}
	return nil
}
func (self *BestStateBeacon) VerifyBestStateWithBeaconBlock(block *BeaconBlock) error {
	/*
			This function will verify the validation of a block with some best state in cache or current best state
		// - Producer
		// - Has parent hash is some beststate block hash
		// - Height
		// - Epoch
	*/
	// Get beacon state of this block
	// For example, new blockHeight is 91 then beacon state of this block must have height 90
	// OR new block has previous has is beacon best block hash

	// TODO: Verify producer
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.BeaconHeight+1 != block.Header.Height {
		return NewBlockChainError(BlockHeightError, errors.New("Block height of new block should be :"+strconv.Itoa(int(block.Header.Height+1))))
	}
	if block.Header.Height%EPOCH == 0 && self.BeaconEpoch+1 != block.Header.Epoch {
		return NewBlockChainError(EpochError, errors.New("Block height and Epoch is not compatiable"))
	}
	if block.Header.Height%EPOCH != 0 && self.BeaconEpoch != block.Header.Epoch {
		return NewBlockChainError(EpochError, errors.New("Block height and Epoch is not compatiable"))
	}
	return nil
}
func (self *BestStateBeacon) VerifyPostProcessingBeaconBlock(block *BeaconBlock) error {
	/* Verify Post-processing data
	- Validator root: BeaconCommittee + BeaconPendingValidator
	- Beacon Candidate root: CandidateBeaconWaitingForCurrentRandom + CandidateBeaconWaitingForNextRandom
	- Shard Candidate root: CandidateShardWaitingForCurrentRandom + CandidateShardWaitingForNextRandom
	- Shard Validator root: ShardCommittee + ShardPendingValidator
	- Random number if have in instruction
	*/
	self.lock.Lock()
	defer self.lock.Unlock()

	var (
		strs []string
		isOk bool
	)

	strs = append(strs, self.BeaconCommittee...)
	strs = append(strs, self.BeaconPendingValidator...)
	isOk = VerifyHashFromStringArray(strs, block.Header.ValidatorsRoot)
	if !isOk {
		return NewBlockChainError(HashError, errors.New("Error verify Validator root"))
	}

	strs = []string{}
	strs = append(strs, self.CandidateBeaconWaitingForCurrentRandom...)
	strs = append(strs, self.CandidateBeaconWaitingForNextRandom...)
	isOk = VerifyHashFromStringArray(strs, block.Header.BeaconCandidateRoot)
	if !isOk {
		return NewBlockChainError(HashError, errors.New("Error verify Beacon Candidate root"))
	}

	strs = []string{}
	strs = append(strs, self.CandidateShardWaitingForCurrentRandom...)
	strs = append(strs, self.CandidateShardWaitingForNextRandom...)
	isOk = VerifyHashFromStringArray(strs, block.Header.ShardCandidateRoot)
	if !isOk {
		return NewBlockChainError(HashError, errors.New("Error verify Shard Candidate root"))
	}

	isOk = VerifyHashFromMapByteString(self.ShardCommittee, self.ShardPendingValidator, block.Header.ShardValidatorsRoot)
	if !isOk {
		return NewBlockChainError(HashError, errors.New("Error verify shard validator root"))
	}

	instructions := block.Body.Instructions

	for _, l := range instructions {
		if l[0] == "random" {
			temp, err := strconv.Atoi(l[1])
			if err != nil {
				Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
				return NewBlockChainError(UnExpectedError, err)
			}
			isOk, err = btcapi.VerifyNonceWithTimestamp(self.CurrentRandomTimeStamp, int64(temp))
			if err != nil {
				Logger.log.Error("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
				return NewBlockChainError(UnExpectedError, err)
			}
			if !isOk {
				return NewBlockChainError(RandomError, errors.New("Error verify random number"))
			}
		}
	}
	return nil
}
func (self *BestStateBeacon) Update(newBlock *BeaconBlock) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	if newBlock == nil {
		return errors.New("Null pointer")
	}
	// signal of random parameter from beacon block
	randomFlag := false
	// update BestShardHash, BestBlock, BestBlockHash
	self.BestBlockHash = *newBlock.Hash()
	self.BestBlock = newBlock
	self.BeaconEpoch = newBlock.Header.Epoch
	self.BeaconHeight = newBlock.Header.Height

	shardState := newBlock.Body.ShardState
	for idx, l := range shardState {
		self.BestShardHash[idx] = l[len(l)-1]
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
					Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
					return NewBlockChainError(UnExpectedError, err)
				}
				shardID := byte(temp)
				// delete in public key out of sharding pending validator list
				self.ShardPendingValidator[shardID], err = RemoveValidator(self.ShardPendingValidator[shardID], inPubkeys)
				if err != nil {
					Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
					return NewBlockChainError(UnExpectedError, err)
				}
				// delete out public key out of current committees
				self.ShardCommittee[shardID], err = RemoveValidator(self.ShardPendingValidator[shardID], outPubkeys)
				if err != nil {
					Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
					return NewBlockChainError(UnExpectedError, err)
				}
				// append in public key to committees
				self.ShardCommittee[shardID] = append(self.ShardCommittee[shardID], inPubkeys...)

				// TODO: Check new list with root hash received from block
			} else if l[3] == "beacon" {
				var err error
				self.BeaconPendingValidator, err = RemoveValidator(self.BeaconPendingValidator, inPubkeys)
				if err != nil {
					Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
					return NewBlockChainError(UnExpectedError, err)
				}
				self.BeaconCommittee, err = RemoveValidator(self.BeaconCommittee, outPubkeys)
				if err != nil {
					Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
					return NewBlockChainError(UnExpectedError, err)
				}
				self.BeaconCommittee = append(self.BeaconCommittee, inPubkeys...)
				// TODO: Check new list with root hash received from block
			}
		}
		// ["random" "{nonce}" "{blockheight}" "{timestamp}" "{bitcoinTimestamp}"]
		if l[0] == "random" {
			//TODO: Verify nonce is from a right block
			temp, err := strconv.Atoi(l[1])
			if err != nil {
				Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
				return NewBlockChainError(UnExpectedError, err)
			}
			self.CurrentRandomNumber = int64(temp)
			randomFlag = true
		}
	}
	// Update candidate
	// get staking candidate list and store
	newBeaconCandidate, newShardCandidate := GetStakingCandidate(*newBlock)
	// store new staking candidate
	self.CandidateBeaconWaitingForNextRandom = append(self.CandidateBeaconWaitingForNextRandom, newBeaconCandidate...)
	self.CandidateShardWaitingForNextRandom = append(self.CandidateShardWaitingForNextRandom, newShardCandidate...)

	if self.BeaconHeight%EPOCH == 0 && self.BeaconHeight != 0 {
		self.IsGetRandomNUmber = false
		// Begin of each epoch
	} else if self.BeaconHeight%EPOCH < RANDOM_TIME {
		// Before get random from bitcoin

	} else if self.BeaconHeight%EPOCH >= RANDOM_TIME {
		// After get random from bitcoin
		if self.BeaconHeight%EPOCH == RANDOM_TIME {
			// snapshot candidate list
			self.CandidateShardWaitingForCurrentRandom = self.CandidateShardWaitingForNextRandom
			self.CandidateBeaconWaitingForCurrentRandom = self.CandidateBeaconWaitingForNextRandom

			// reset candidate list
			self.CandidateShardWaitingForNextRandom = []string{}
			self.CandidateBeaconWaitingForNextRandom = []string{}

			self.CurrentRandomTimeStamp = newBlock.Header.Timestamp
		}
		// if get new random number
		// Assign candidate to shard
		// assign CandidateShardWaitingForCurrentRandom to ShardPendingValidator with CurrentRandom
		if randomFlag {
			self.IsGetRandomNUmber = true
			err := AssignValidatorShard(self.ShardPendingValidator, self.CandidateShardWaitingForCurrentRandom, self.CurrentRandomNumber)
			if err != nil {
				Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
				return NewBlockChainError(UnExpectedError, err)
			}
			// delete CandidateShardWaitingForCurrentRandom list
			self.CandidateShardWaitingForCurrentRandom = []string{}

			/// Shuffle candidate
			// shuffle CandidateBeaconWaitingForCurrentRandom with current random number
			newBeaconPendingValidator, err := ShuffleCandidate(self.CandidateBeaconWaitingForCurrentRandom, self.CurrentRandomNumber)
			if err != nil {
				Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
				return NewBlockChainError(UnExpectedError, err)
			}
			self.CandidateBeaconWaitingForCurrentRandom = []string{}
			self.BeaconPendingValidator = append(self.BeaconPendingValidator, newBeaconPendingValidator...)
			if err != nil {
				return err
			}
		}
	} else if self.BeaconHeight%EPOCH == EPOCH-1 {
		// At the end of each epoch, eg: block 199, 399, 599 with epoch is 200
		// Swap pending validator in committees, pop some of public key in committees out
		// ONLY SWAP FOR BEACON
		// SHARD WILL SWAP ITSELF
		var (
			beaconSwapedCommittees []string
			beaconNextCommittees   []string
			err                    error
		)
		self.BeaconPendingValidator, self.BeaconCommittee, beaconSwapedCommittees, beaconNextCommittees, err = SwapValidator(self.BeaconPendingValidator, self.BeaconCommittee, OFFSET)
		Logger.log.Infof("Swaped out committees %+v", beaconSwapedCommittees)
		Logger.log.Infof("Nextcommittees %+v", beaconNextCommittees)
		if err != nil {
			Logger.log.Errorf("Blockchain Error %+v", NewBlockChainError(UnExpectedError, err))
			return NewBlockChainError(UnExpectedError, err)
		}
	}
	return nil
}

//===================================Util for Beacon=============================
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
func SwapValidator(pendingValidators []string, currentValidators []string, offset int) ([]string, []string, []string, []string, error) {
	if offset == 0 {
		return nil, pendingValidators, currentValidators, nil, errors.New("Can't not swap 0 validator")
	}
	// if number of pending validator is less or equal than offset, set offset equal to number of pending validator
	if offset > len(pendingValidators) {
		offset = len(pendingValidators)
	}
	// do nothing
	if offset == 0 {
		return nil, pendingValidators, currentValidators, nil, errors.New("No pending validator for swapping")
	}
	if offset > len(currentValidators) {
		return nil, pendingValidators, currentValidators, nil, errors.New("Trying to swap too many validator")
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
	return pendingValidators, currentValidators, swapValidator, tempValidators, nil
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

func VerifyRootHashFromStringArray(strs1 []string, strs2 []string, hash common.Hash) error {
	var (
		tempMerkle Merkle
		merkleTree []*common.Hash
		hashArrays []*common.Hash
		// merkleRoot *common.Hash
	)

	hashes1, err := common.ConvertArrayStringToArrayHash(strs1)
	if err != nil {
		Logger.log.Errorf("Error converting from string array to hash array %+v", err)
		return err
	}

	hashes2, err := common.ConvertArrayStringToArrayHash(strs2)
	if err != nil {
		Logger.log.Errorf("Error converting from string array to hash array %+v", err)
		return err
	}
	hashArrays = append(hashArrays, hashes1...)
	hashArrays = append(hashArrays, hashes2...)

	merkleTree = tempMerkle.BuildMerkleTreeOfHashs(hashArrays)
	if tempMerkle.VerifyMerkleRootOfHashs(merkleTree, &hash) == false {
		err = NewBlockChainError(UnExpectedError, errors.New("Error verify merkle root"))
		Logger.log.Errorf("Error in VerifyRootHashFromStringArray %+v", err)
		return err
	}
	return nil
}

func GenerateHashFromStringArray(strs []string) (common.Hash, error) {
	var (
		hash common.Hash
		buf  bytes.Buffer
	)
	for _, value := range strs {
		buf.WriteString(value)
	}
	temp := sha256.Sum256(buf.Bytes())
	if err := hash.SetBytes(temp[:]); err != nil {
		return common.Hash{}, NewBlockChainError(HashError, err)
	}
	return hash, nil
}

func GenerateHashFromHashArray(hashes []common.Hash) (common.Hash, error) {
	strs := []string{}
	for _, value := range hashes {
		str := value.String()
		strs = append(strs, str)
	}
	return GenerateHashFromStringArray(strs)
}

//TODO: implement
func GenerateHashFromMapByteString(maps1 map[byte][]string, maps2 map[byte][]string) (common.Hash, error) {
	return common.Hash{}, nil
}
func VerifyHashFromHashArray(hashes []common.Hash, hash common.Hash) bool {
	strs := []string{}
	for _, value := range hashes {
		str := value.String()
		strs = append(strs, str)
	}
	return VerifyHashFromStringArray(strs, hash)
}

func VerifyHashFromStringArray(strs []string, hash common.Hash) bool {
	res, err := GenerateHashFromStringArray(strs)
	if err != nil {
		return false
	}
	if bytes.Compare(res.GetBytes(), hash.GetBytes()) != 0 {
		return true
	}
	return false
}

//TODO: implement
func VerifyHashFromMapByteString(maps1 map[byte][]string, maps2 map[byte][]string, hash common.Hash) bool {
	return true
}