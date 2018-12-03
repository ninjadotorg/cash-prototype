package lvdb

import (
	"encoding/binary"
	"sort"

	"github.com/ninjadotorg/constant/blockchain"
	"github.com/ninjadotorg/constant/database"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (db *db) AddVoteDCBBoard(StartedBlockInt int32, VoterPubKey string, CandidatePubKey string, amount uint64) error {
	StartedBlock := uint32(StartedBlockInt)
	//add to sum amount of vote token to this candidate
	key := db.GetKey(string(VoteDCBBoardSumPrefix), string(StartedBlock)+CandidatePubKey)
	ok, err := db.hasValue(key)
	if err != nil {
		return err
	}
	if !ok {
		zeroInBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(zeroInBytes, uint64(0))
		db.put(key, zeroInBytes)
	}

	currentVoteInBytes, err := db.lvdb.Get(key, nil)
	currentVote := binary.LittleEndian.Uint64(currentVoteInBytes)
	newVote := currentVote + amount

	newVoteInBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(newVoteInBytes, newVote)
	err = db.put(key, newVoteInBytes)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}

	// add to count amount of vote to this candidate
	key = db.GetKey(string(VoteDCBBoardCountPrefix), string(StartedBlock)+CandidatePubKey)
	currentCountInBytes, err := db.lvdb.Get(key, nil)
	if err != nil {
		return err
	}
	currentCount := binary.LittleEndian.Uint32(currentCountInBytes)
	newCount := currentCount + 1
	newCountInByte := make([]byte, 4)
	binary.LittleEndian.PutUint32(newCountInByte, newCount)
	err = db.put(key, newCountInByte)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}

	// add to list voter new voter base on count as index
	key = db.GetKey(string(VoteDCBBoardListPrefix), string(currentCount)+string(StartedBlock)+CandidatePubKey)
	amountInByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountInByte, amount)
	valueInByte := append([]byte(VoterPubKey), amountInByte...)
	err = db.put(key, valueInByte)

	return nil
}

func (db *db) AddVoteGOVBoard(StartedBlock uint32, VoterPubKey string, CandidatePubKey string, amount uint64) error {
	//add to sum amount of vote token to this candidate
	key := db.GetKey(string(VoteGOVBoardSumPrefix), string(StartedBlock)+CandidatePubKey)
	ok, err := db.hasValue(key)
	if err != nil {
		return err
	}
	if !ok {
		zeroInBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(zeroInBytes, uint64(0))
		db.put(key, zeroInBytes)
	}

	currentVoteInBytes, err := db.lvdb.Get(key, nil)
	currentVote := binary.LittleEndian.Uint64(currentVoteInBytes)
	newVote := currentVote + amount

	newVoteInBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(newVoteInBytes, newVote)
	err = db.put(key, newVoteInBytes)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}

	// add to count amount of vote to this candidate
	key = db.GetKey(string(VoteGOVBoardCountPrefix), string(StartedBlock)+CandidatePubKey)
	currentCountInBytes, err := db.lvdb.Get(key, nil)
	if err != nil {
		return err
	}
	currentCount := binary.LittleEndian.Uint32(currentCountInBytes)
	newCount := currentCount + 1
	newCountInByte := make([]byte, 4)
	binary.LittleEndian.PutUint32(newCountInByte, newCount)
	err = db.put(key, newCountInByte)
	if err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.put"))
	}

	// add to list voter new voter base on count as index
	key = db.GetKey(string(VoteGOVBoardListPrefix), string(StartedBlock)+CandidatePubKey+string(currentCount))
	err = db.put(key, []byte(VoterPubKey))

	return nil
}

type CandidateElement struct {
	PubKey     []byte
	VoteAmount uint64
}

type CandidateList []CandidateElement

func (A CandidateList) Len() int {
	return len(A)
}
func (A CandidateList) Swap(i, j int) {
	A[i], A[j] = A[j], A[i]
}
func (A CandidateList) Less(i, j int) bool {
	return A[i].VoteAmount < A[j].VoteAmount
}

func (db *db) GetTopMostVoteDCBGovernor(number uint32, StartedBlock uint32) (CandidateList, error) {
	var candidateList CandidateList
	//use prefix  as in file lvdb/block.go FetchChain
	prefix := db.GetKey(string(VoteDCBBoardSumPrefix), string(StartedBlock))
	iter := db.lvdb.NewIterator(util.BytesPrefix(prefix), nil)
	for iter.Next() {
		key := db.reverseGetKey(string(VoteDCBBoardSumPrefix), iter.Key()).(string)
		pubKey := key[len(string(StartedBlock)+"#"):]
		value := binary.LittleEndian.Uint64(iter.Value())
		candidateList = append(candidateList, CandidateElement{pubKey, value})
	}
	sort.Sort(candidateList)
	if len(candidateList) < blockchain.NumberOfDCBGovernors {
		return nil, database.NewDatabaseError(database.NotEnoughCandidateDCB, errors.Errorf("not enough DCB Candidate"))
	}

	return candidateList[len(candidateList)-blockchain.NumberOfDCBGovernors:], nil
}

func (db *db) GetTopMostVoteGOVGovernor(number uint32, StartedBlock uint32) (CandidateList, error) {
	var candidateList CandidateList
	//use prefix  as in file lvdb/block.go FetchChain
	prefix := db.GetKey(string(VoteGOVBoardSumPrefix), string(StartedBlock))

	iter := db.lvdb.NewIterator(util.BytesPrefix(prefix), nil)
	for iter.Next() {
		key := db.reverseGetKey(string(VoteGOVBoardSumPrefix), iter.Key()).(string)
		pubKey := key[len(string(StartedBlock)+"#"):]
		value := binary.LittleEndian.Uint64(iter.Value())
		candidateList = append(candidateList, CandidateElement{pubKey, value})
	}
	sort.Sort(candidateList)
	if len(candidateList) < blockchain.NumberOfGOVGovernors {
		return nil, database.NewDatabaseError(database.NotEnoughCandidateGOV, errors.Errorf("not enough GOV Candidate"))
	}

	return candidateList[len(candidateList)-blockchain.NumberOfGOVGovernors:], nil
}

func (db *db) NewIterator(slice *util.Range, ro *opt.ReadOptions) iterator.Iterator {
	return db.lvdb.NewIterator(slice, ro)
}
