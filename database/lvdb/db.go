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
	VoteDCBBoardSumPrefix     = []byte("votedcbsumboard-")
	VoteGOVBoardSumPrefix     = []byte("votegovsumboard-")
	VoteDCBBoardCountPrefix   = []byte("votedcbcountboard-")
	VoteGOVBoardCountPrefix   = []byte("votegovcountboard-")
	VoteDCBBoardListPrefix    = []byte("votedcblistboard-")
	VoteGOVBoardListPrefix    = []byte("votegovlistboard-")
	loanRequestPostfix        = []byte("-req")
	loanResponsePostfix       = []byte("-res")
	rewared                   = []byte("reward")
	unreward                  = []byte("unreward")
	spent                     = []byte("spent")
	unspent                   = []byte("unspent")
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
	case string(voteDCBBoardPrefix):
		postfix := []byte(key.(string))
		dbkey = append(voteDCBBoardPrefix, postfix...)
	case string(voteGOVBoardPrefix):
		postfix := []byte(key.(string))
		dbkey = append(voteGOVBoardPrefix, postfix...)
	}
	return dbkey
}

// get real PubKey from dbkey
func (db db) reverseGetKey(keyType string, dbkey []byte) interface{} {
	var key interface{}
	switch keyType {
	case string(voteDCBBoardPrefix):
		key = string(dbkey[len(voteDCBBoardPrefix):])
	case string(voteGOVBoardPrefix):
		key = string(dbkey[len(voteGOVBoardPrefix):])
	}
	return key
}
