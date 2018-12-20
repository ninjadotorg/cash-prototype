package blockchain

import (
	"encoding/binary"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
	"github.com/ninjadotorg/constant/database/lvdb"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/privacy-protocol"
	"github.com/ninjadotorg/constant/transaction"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type voter struct {
}

func (blockgen *BlkTmplGenerator) createAcceptConstitutionAndPunishTx(
	chainID byte,
	ConstitutionHelper ConstitutionHelper,
) ([]metadata.Transaction, error) {
	resTx := make([]metadata.Transaction, 0)
	SumVote := make(map[common.Hash]uint64)
	CountVote := make(map[common.Hash]uint32)
	VoteTable := make(map[common.Hash]map[string]int32)
	BestBlock := blockgen.chain.BestState[chainID].BestBlock

	db := blockgen.chain.config.DataBase
	boardType := ConstitutionHelper.GetLowerCaseBoardType()
	begin := lvdb.GetThreePhraseCryptoSealerKey(boardType, 0, nil)
	end := lvdb.GetThreePhraseCryptoSealerKey(boardType, ConstitutionHelper.GetEndedBlockHeight(blockgen, chainID), nil)

	searchrange := util.Range{
		Start: begin,
		Limit: end,
	}
	iter := db.NewIterator(&searchrange, nil)
	rightStartedBlock := BestBlock.Header.Height + 1
	for iter.Next() {
		key := iter.Key()
		_, startedBlock, transactionID, err := lvdb.ParseKeyThreePhraseCryptoSealer(key)
		if err != nil {
			return nil, err
		}
		if startedBlock != uint32(rightStartedBlock) {
			//@todo 0xjackalope delete all relevant thing
			db.Delete(key)
			continue
		}
		//Punish owner if he don't send decrypted message
		keyOwner := lvdb.GetThreePhraseCryptoOwnerKey(boardType, startedBlock, transactionID)
		valueOwnerInByte, err := db.Get(keyOwner)
		if err != nil {
			return nil, err
		}
		valueOwner, err := lvdb.ParseValueThreePhraseCryptoOwner(valueOwnerInByte)
		if err != nil {
			return nil, err
		}

		_, _, _, lv3Tx, _ := blockgen.chain.GetTransactionByHash(transactionID)
		sealerPubKeyList := ConstitutionHelper.GetSealerPubKey(lv3Tx)
		if valueOwner != 1 {
			newTx := transaction.Tx{
				Metadata: ConstitutionHelper.CreatePunishDecryptTx(map[string]interface{}{
					"pubKey": sealerPubKeyList[0],
				}),
			}
			resTx = append(resTx, &newTx)
		}
		//Punish sealer if he don't send decrypted message
		keySealer := lvdb.GetThreePhraseCryptoSealerKey(boardType, startedBlock, transactionID)
		valueSealerInByte, err := db.Get(keySealer)
		if err != nil {
			return nil, err
		}
		valueSealer := binary.LittleEndian.Uint32(valueSealerInByte)
		if valueSealer != 3 {
			//Count number of time she don't send encrypted message if number==2 create punish transaction
			newTx := transaction.Tx{
				Metadata: ConstitutionHelper.CreatePunishDecryptTx(map[string]interface{}{
					"pubKey": sealerPubKeyList[valueSealer],
				}),
			}
			resTx = append(resTx, &newTx)
		}

		//Accumulate count vote
		voter := sealerPubKeyList[0]
		keyVote := lvdb.GetThreePhraseVoteValueKey(boardType, startedBlock, transactionID)
		valueVote, err := db.Get(keyVote)
		if err != nil {
			return nil, err
		}
		txId, voteAmount, err := lvdb.ParseValueThreePhraseVoteValue(valueVote)
		if err != nil {
			return nil, err
		}

		SumVote[*txId] += uint64(voteAmount)
		if VoteTable[*txId] == nil {
			VoteTable[*txId] = make(map[string]int32)
		}
		VoteTable[*txId][string(voter)] += voteAmount
		CountVote[*txId] += 1
	}

	bestVoter := voter{}
	// Get most vote proposal
	for txId, listVoter := range VoteTable {

		for voterPubKey, amount := range listVoter {
			if db.GetAmountVoteToken(boardType, voterPubKey) < amount {
				listVoter[voterPubKey] = 0
				// can change listvoter because it is a pointer
				continue
			}

		}
	}
	maxVoteAmount := uint64(0)
	maxVoteCount := uint32(0)
	bestTxId := common.Hash{}
	for txId, voteAmount := range SumVote {
		if voteAmount > maxVoteAmount ||
			(voteAmount == maxVoteAmount && CountVote[txId] > maxVoteCount) ||
			(voteAmount == maxVoteAmount && CountVote[txId] == maxVoteCount && string(txId.GetBytes()) > string(bestTxId.GetBytes())) {
			maxVoteAmount = voteAmount
			maxVoteCount = CountVote[txId]
			bestTxId = txId
		}
	}

	x := make(map[int]map[int]int)
	x[0][1] = 4

	acceptedSubmitProposalTransaction := ConstitutionHelper.TxAcceptProposal(&bestTxId)

	resTx = append(resTx, acceptedSubmitProposalTransaction)

	return resTx, nil
}

func (blockgen *BlkTmplGenerator) createSingleSendDCBVoteTokenTx(chainID byte, pubKey []byte, amount uint64) (metadata.Transaction, error) {
	data := map[string]interface{}{
		"Amount":         amount,
		"ReceiverPubkey": pubKey,
	}
	sendDCBVoteTokenTransaction := transaction.Tx{
		Metadata: metadata.NewSendInitDCBVoteTokenMetadata(data),
	}
	return &sendDCBVoteTokenTransaction, nil
}

func (blockgen *BlkTmplGenerator) createSingleSendGOVVoteTokenTx(chainID byte, pubKey []byte, amount uint64) (metadata.Transaction, error) {
	data := map[string]interface{}{
		"Amount":         amount,
		"ReceiverPubkey": pubKey,
	}
	sendGOVVoteTokenTransaction := transaction.Tx{
		Metadata: metadata.NewSendInitGOVVoteTokenMetadata(data),
	}
	return &sendGOVVoteTokenTransaction, nil
}

func getAmountOfVoteToken(sumAmount uint64, voteAmount uint64) uint64 {
	return voteAmount * common.SumOfVoteDCBToken / sumAmount
}

func (blockgen *BlkTmplGenerator) CreateSendDCBVoteTokenToGovernorTx(chainID byte, newDCBList database.CandidateList, sumAmountDCB uint64) []metadata.Transaction {
	var SendVoteTx []metadata.Transaction
	var newTx metadata.Transaction
	for i := 0; i <= common.NumberOfDCBGovernors; i++ {
		newTx, _ = blockgen.createSingleSendDCBVoteTokenTx(chainID, newDCBList[i].PubKey, getAmountOfVoteToken(sumAmountDCB, newDCBList[i].VoteAmount))
		SendVoteTx = append(SendVoteTx, newTx)
	}
	return SendVoteTx
}

func (blockgen *BlkTmplGenerator) CreateSendGOVVoteTokenToGovernorTx(chainID byte, newGOVList database.CandidateList, sumAmountGOV uint64) []metadata.Transaction {
	var SendVoteTx []metadata.Transaction
	var newTx metadata.Transaction
	for i := 0; i <= common.NumberOfGOVGovernors; i++ {
		newTx, _ = blockgen.createSingleSendGOVVoteTokenTx(chainID, newGOVList[i].PubKey, getAmountOfVoteToken(sumAmountGOV, newGOVList[i].VoteAmount))
		SendVoteTx = append(SendVoteTx, newTx)
	}
	return SendVoteTx
}

func (blockgen *BlkTmplGenerator) createAcceptDCBBoardTx(DCBBoardPubKeys [][]byte, sumOfVote uint64) metadata.Transaction {
	return &transaction.Tx{
		Metadata: &metadata.AcceptDCBBoardMetadata{
			DCBBoardPubKeys:     DCBBoardPubKeys,
			StartAmountDCBToken: sumOfVote,
		},
	}
}

func (blockgen *BlkTmplGenerator) createAcceptGOVBoardTx(DCBBoardPubKeys [][]byte, sumOfVote uint64) metadata.Transaction {
	return &transaction.Tx{
		Metadata: &metadata.AcceptGOVBoardMetadata{
			GOVBoardPubKeys:     DCBBoardPubKeys,
			StartAmountGOVToken: sumOfVote,
		},
	}
}

func (block *Block) UpdateDCBBoard(thisTx metadata.Transaction) error {
	tx := thisTx.(transaction.TxAcceptDCBBoard)
	block.Header.DCBGovernor.DCBBoardPubKeys = tx.DCBBoardPubKeys
	block.Header.DCBGovernor.StartedBlock = uint32(block.Header.Height)
	block.Header.DCBGovernor.EndBlock = block.Header.DCBGovernor.StartedBlock + common.DurationOfTermDCB
	block.Header.DCBGovernor.StartAmountDCBToken = tx.StartAmountDCBToken
	return nil
}

func (block *Block) UpdateGOVBoard(thisTx metadata.Transaction) error {
	tx := thisTx.(transaction.TxAcceptGOVBoard)
	block.Header.GOVGovernor.GOVBoardPubKeys = tx.GOVBoardPubKeys
	block.Header.GOVGovernor.StartedBlock = uint32(block.Header.Height)
	block.Header.GOVGovernor.EndBlock = block.Header.GOVGovernor.StartedBlock + common.DurationOfTermGOV
	block.Header.GOVGovernor.StartAmountGOVToken = tx.StartAmountGOVToken
	return nil
}

func parseVoteDCBBoardListValue(value []byte) ([]byte, uint64) {
	voterPubKey := value[:common.PubKeyLength]
	amount := binary.LittleEndian.Uint64(value[common.PubKeyLength:])
	return voterPubKey, amount
}

func parseVoteGOVBoardListValue(value []byte) ([]byte, uint64) {
	voterPubKey := value[:common.PubKeyLength]
	amount := binary.LittleEndian.Uint64(value[common.PubKeyLength:])
	return voterPubKey, amount
}

func createSingleSendDCBVoteTokenFail(pubKey []byte, amount uint64) metadata.Transaction {
	paymentAddress := privacy.PaymentAddress{
		Pk: pubKey,
	}
	txTokenVout := transaction.TxTokenVout{
		Value:          amount,
		PaymentAddress: paymentAddress,
	}
	newTx := transaction.TxCustomToken{
		TxTokenData: transaction.TxTokenData{
			Type:       transaction.SendBackDCBTokenVoteFail,
			Amount:     amount,
			PropertyID: common.DCBTokenID,
			Vins:       []transaction.TxTokenVin{},
			Vouts:      []transaction.TxTokenVout{txTokenVout},
		},
	}
	return &newTx
}

func createSingleSendGOVVoteTokenFail(pubKey []byte, amount uint64) metadata.Transaction {
	paymentAddress := privacy.PaymentAddress{
		Pk: pubKey,
	}
	txTokenVout := transaction.TxTokenVout{
		Value:          amount,
		PaymentAddress: paymentAddress,
	}
	newTx := transaction.TxCustomToken{
		TxTokenData: transaction.TxTokenData{
			Type:       transaction.SendBackGOVTokenVoteFail,
			Amount:     amount,
			PropertyID: common.GOVTokenID,
			Vins:       []transaction.TxTokenVin{},
			Vouts:      []transaction.TxTokenVout{txTokenVout},
		},
	}
	return &newTx
}

//Send back vote token to voters who have vote to lose candidate
func (blockgen *BlkTmplGenerator) CreateSendBackDCBTokenAfterVoteFail(chainID byte, newDCBList [][]byte) []metadata.Transaction {
	setOfNewDCB := make(map[string]bool, 0)
	for _, i := range newDCBList {
		setOfNewDCB[string(i)] = true
	}
	currentHeight := blockgen.chain.BestState[chainID].Height
	db := blockgen.chain.config.DataBase
	begin := db.GetKey(string(blockgen.chain.config.DataBase.GetVoteDCBBoardListPrefix()), string(0))
	end := db.GetKey(string(blockgen.chain.config.DataBase.GetVoteDCBBoardListPrefix()), currentHeight)
	searchRange := util.Range{
		Start: begin,
		Limit: end,
	}

	iter := blockgen.chain.config.DataBase.NewIterator(&searchRange, nil)
	listNewTx := make([]metadata.Transaction, 0)
	for iter.Next() {
		key := iter.Key()
		startedBlock, PubKey, _, _ := lvdb.ParseKeyVoteDCBBoardList(key)
		value := iter.Value()
		senderPubkey, amountOfDCBToken := parseVoteDCBBoardListValue(value)
		_, found := setOfNewDCB[string(PubKey)]
		if startedBlock < uint32(currentHeight) || !found {
			listNewTx = append(listNewTx, createSingleSendDCBVoteTokenFail(senderPubkey, amountOfDCBToken))
		}
	}
	return listNewTx
}

func (blockgen *BlkTmplGenerator) CreateSendBackGOVTokenAfterVoteFail(chainID byte, newGOVList [][]byte) []metadata.Transaction {
	setOfNewGOV := make(map[string]bool, 0)
	for _, i := range newGOVList {
		setOfNewGOV[string(i)] = true
	}
	currentHeight := blockgen.chain.BestState[chainID].Height
	db := blockgen.chain.config.DataBase
	begin := db.GetKey(string(blockgen.chain.config.DataBase.GetVoteGOVBoardListPrefix()), string(0))
	end := db.GetKey(string(blockgen.chain.config.DataBase.GetVoteGOVBoardListPrefix()), currentHeight)
	searchRange := util.Range{
		Start: begin,
		Limit: end,
	}

	iter := blockgen.chain.config.DataBase.NewIterator(&searchRange, nil)
	listNewTx := make([]metadata.Transaction, 0)
	for iter.Next() {
		key := iter.Key()
		startedBlock, PubKey, _, _ := lvdb.ParseKeyVoteGOVBoardList(key)
		value := iter.Value()
		senderPubkey, amountOfGOVToken := parseVoteGOVBoardListValue(value)
		_, found := setOfNewGOV[string(PubKey)]
		if startedBlock < uint32(currentHeight) || !found {
			listNewTx = append(listNewTx, createSingleSendGOVVoteTokenFail(senderPubkey, amountOfGOVToken))
		}
	}
	return listNewTx
}
