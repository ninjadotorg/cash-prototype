package rpcserver

import (
	"encoding/hex"
	"encoding/json"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/rpcserver/jsonresult"
	"github.com/ninjadotorg/constant/transaction"
)

func (self RpcServer) buildRawSealLv3VoteDCBProposalTransaction(
	params interface{},
) (*transaction.Tx, error) {
	tx, err := self.buildRawTransaction(params)
	arrayParams := common.InterfaceSlice(params)
	voteInfo := arrayParams[len(arrayParams)-4]
	firstPubKey := arrayParams[len(arrayParams)-3] // firstPubKey is pubkey of itself
	secondPubKey := arrayParams[len(arrayParams)-2]
	thirdPubKey := arrayParams[len(arrayParams)-1]
	Seal3Data := common.Encrypt(common.Encrypt(common.Encrypt(voteInfo, thirdPubKey), secondPubKey), firstPubKey)
	tx.Metadata = metadata.NewSealedLv3DCBBallotMetadata(
		map[string]interface{}{
			"SealedBallot": Seal3Data,
			"LockerPubKey": [][]byte{firstPubKey.([]byte), secondPubKey.([]byte), thirdPubKey.([]byte)},
		})
	return tx, err
}

func (self RpcServer) handleCreateRawSealLv3VoteDCBProposalTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, error) {
	tx, err := self.buildRawSealLv3VoteDCBProposalTransaction(params)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewRPCError(ErrUnexpected, err)
	}

	byteArrays, err := json.Marshal(tx)
	if err != nil {
		Logger.log.Error(err)
		return nil, NewRPCError(ErrUnexpected, err)
	}
	hexData := hex.EncodeToString(byteArrays)
	result := jsonresult.CreateTransactionResult{
		HexData: hexData,
	}
	return result, nil
}

func (self RpcServer) handleCreateAndSendSealLv3VoteDCBProposalTransaction(params interface{}, closeChan <-chan struct{}) (interface{}, error) {
	data, err := self.handleCreateRawSealLv3VoteDCBProposalTransaction(params, closeChan)
	if err != nil {
		return nil, err
	}
	tx := data.(jsonresult.CreateTransactionResult)
	hexStrOfTx := tx.HexData
	if err != nil {
		return nil, err
	}
	newParam := make([]interface{}, 0)
	newParam = append(newParam, hexStrOfTx)
	txId, err := self.handleSendRawTransaction(newParam, closeChan)
	return txId, err
}
