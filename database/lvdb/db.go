package lvdb

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
)

type db struct {
	lvdb *leveldb.DB
}

type hasher interface {
	Hash() *common.Hash
}

var (
	chainIDPrefix             = []byte("c")
	blockKeyPrefix            = []byte("b-")
	blockHeaderKeyPrefix      = []byte("bh-")
	blockKeyIdxPrefix         = []byte("i-")
	transactionKeyPrefix      = []byte("tx-")
	privateKeyPrefix          = []byte("prk-")
	nullifiersPrefix          = []byte("nullifiers-")
	commitmentsPrefix         = []byte("commitments-")
	bestBlockKey              = []byte("bestBlock")
	feeEstimator              = []byte("feeEstimator")
	splitter                  = []byte("-[-]-")
	tokenPrefix               = []byte("token-")
	tokenPaymentAddressPrefix = []byte("token-paymentaddress-")
	tokenInitPrefix           = []byte("token-init-")
	loanIDKeyPrefix           = []byte("loanID-")
	loanTxKeyPrefix           = []byte("loanTx-")
	loanRequestPostfix        = []byte("-req")
	loanResponsePostfix       = []byte("-res")
	rewared                   = []byte("reward")
	unreward                  = []byte("unreward")
	spent                     = []byte("spent")
	unspent                   = []byte("unspent")

	//vote prefix
	voteDCBBoardSumPrefix   = []byte("votedcbsumboard-")
	voteGOVBoardSumPrefix   = []byte("votegovsumboard-")
	voteDCBBoardCountPrefix = []byte("votedcbcountboard-")
	voteGOVBoardCountPrefix = []byte("votegovcountboard-")
	VoteDCBBoardListPrefix  = []byte("votedcblistboard-")
	VoteGOVBoardListPrefix  = []byte("votegovlistboard-")
)

func open(dbPath string) (database.DatabaseInterface, error) {
	lvdb, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, database.NewDatabaseError(database.OpenDbErr, errors.Wrapf(err, "levelvdb.OpenFile %s", dbPath))
	}
	return &db{lvdb: lvdb}, nil
}

func (db *db) Close() error {
	return errors.Wrap(db.lvdb.Close(), "db.lvdb.Close")
}

func (db *db) hasValue(key []byte) (bool, error) {
	ret, err := db.lvdb.Has(key, nil)
	if err != nil {
		return false, database.NewDatabaseError(database.NotExistValue, err)
	}
	return ret, nil
}

func (db *db) put(key, value []byte) error {
	if err := db.lvdb.Put(key, value, nil); err != nil {
		return database.NewDatabaseError(database.UnexpectedError, errors.Wrap(err, "db.lvdb.Put"))
	}
	return nil
}

func (db db) GetKey(keyType string, key interface{}) []byte {
	var dbkey []byte
	switch keyType {
	case string(blockKeyPrefix):
		dbkey = append(blockKeyPrefix, key.(*common.Hash)[:]...)
	case string(blockKeyIdxPrefix):
		dbkey = append(blockKeyIdxPrefix, key.(*common.Hash)[:]...)
	case string(nullifiersPrefix):
		dbkey = append(nullifiersPrefix, []byte(key.(string))...)
	case string(commitmentsPrefix):
		dbkey = append(commitmentsPrefix, []byte(key.(string))...)
	case string(tokenPrefix):
		dbkey = append(tokenPrefix, key.(*common.Hash)[:]...)
	case string(tokenInitPrefix):
		dbkey = append(tokenInitPrefix, key.(*common.Hash)[:]...)

	// Voting case
	case string(voteDCBBoardSumPrefix):
		postfix := []byte(key.(string))
		dbkey = append(voteDCBBoardSumPrefix, postfix...)
	case string(voteDCBBoardCountPrefix):
		postfix := []byte(key.(string))
		dbkey = append(voteDCBBoardCountPrefix, postfix...)
	case string(VoteDCBBoardListPrefix):
		postfix := []byte(key.(string))
		dbkey = append(VoteDCBBoardListPrefix, postfix...)
	case string(voteGOVBoardSumPrefix):
		postfix := []byte(key.(string))
		dbkey = append(voteGOVBoardSumPrefix, postfix...)
	case string(voteGOVBoardCountPrefix):
		postfix := []byte(key.(string))
		dbkey = append(voteGOVBoardCountPrefix, postfix...)
	case string(VoteGOVBoardListPrefix):
		postfix := []byte(key.(string))
		dbkey = append(VoteGOVBoardListPrefix, postfix...)
	}
	return dbkey
}

// get real PubKey from dbkey
func ReverseGetKey(keyType string, dbkey []byte) (interface{}, error) {
	var key interface{}
	switch keyType {
	case string(voteDCBBoardSumPrefix):
		key = string(dbkey[len(voteDCBBoardSumPrefix):])
	case string(voteDCBBoardCountPrefix):
		key = string(dbkey[len(voteDCBBoardCountPrefix):])
	case string(VoteDCBBoardListPrefix):
		key = string(dbkey[len(VoteDCBBoardListPrefix):])
	case string(voteGOVBoardSumPrefix):
		key = string(dbkey[len(voteGOVBoardSumPrefix):])
	case string(voteGOVBoardCountPrefix):
		key = string(dbkey[len(voteGOVBoardCountPrefix):])
	case string(VoteGOVBoardListPrefix):
		key = string(dbkey[len(VoteGOVBoardListPrefix):])
	default:
		return nil, errors.New("This keyType is not handled yet")
	}
	return key, nil
}
