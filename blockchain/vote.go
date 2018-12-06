package blockchain

import (
	"encoding/binary"
	"encoding/json"
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database/lvdb"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/peer"
	"github.com/ninjadotorg/constant/privacy-protocol"
	"github.com/ninjadotorg/constant/transaction"
	"github.com/syndtr/goleveldb/leveldb/util"

	"time"
)

//Todo: @0xjackalope count by database
func (blockgen *BlkTmplGenerator) createAcceptConstitutionTxDecs(
	chainID byte,
	ConstitutionHelper ConstitutionHelper,
) (*metadata.TxDesc, error) {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock

	// count vote from lastConstitution.StartedBlockHeight to Bestblock height
	CountVote := make(map[common.Hash]int64)
	Transaction := make(map[common.Hash]*metadata.Transaction)
	for blockHeight := ConstitutionHelper.GetStartedBlockHeight(blockgen, chainID); blockHeight < BestBlock.Header.Height; blockHeight += 1 {
		//retrieve block from block's height
		hashBlock, err := blockgen.chain.config.DataBase.GetBlockByIndex(blockHeight, chainID)
		if err != nil {
			return nil, err
		}
		blockBytes, err := blockgen.chain.config.DataBase.FetchBlock(hashBlock)
		if err != nil {
			return nil, err
		}
		block := Block{}
		err = json.Unmarshal(blockBytes, &block)
		if err != nil {
			return nil, err
		}
		//count vote of this block
		for _, tx := range block.Transactions {
			_, exist := CountVote[*tx.Hash()]
			if ConstitutionHelper.CheckSubmitProposalType(tx) {
				if exist {
					return nil, err
				}
				CountVote[*tx.Hash()] = 0
				Transaction[*tx.Hash()] = &tx
			} else {
				if ConstitutionHelper.CheckVotingProposalType(tx) {
					if !exist {
						return nil, err
					}
					CountVote[*tx.Hash()] += int64(ConstitutionHelper.GetAmountVoteToken(tx))
				}
			}
		}
	}

	// get transaction and create transaction desc
	var maxVote int64
	var res common.Hash
	for key, value := range CountVote {
		if value > maxVote {
			maxVote = value
			res = key
		}
	}

	acceptedSubmitProposalTransaction := ConstitutionHelper.TxAcceptProposal(*Transaction[res])

	AcceptedTransactionDesc := metadata.TxDesc{
		Tx:     acceptedSubmitProposalTransaction,
		Added:  time.Now(),
		Height: BestBlock.Header.Height,
		Fee:    0,
	}
	return &AcceptedTransactionDesc, nil
}

func (blockgen *BlkTmplGenerator) createSingleSendDCBVoteTokenTx(chainID string,pubKey []byte, amount uint64) (metadata.Transaction, error) {

	paymentAddress := privacy.PaymentAddress{
		Pk: pubKey,
	}
	txTokenVout := transaction.TxTokenVout{
		Value : amount,
		PaymentAddress: paymentAddress,
	}
	txTokenData := transaction.TxTokenData{
		Type: transaction.InitVoteDCBToken,
		Amount: amount,
		PropertyID: VoteDCBTokenID,
		Vins: []transaction.TxTokenVin{},
		Vouts: [] transaction.TxTokenVout{txTokenVout},
	}
	sendDCBVoteTokenTransaction :=  transaction.TxSendInitDCBVoteToken{
		TxCustomToken: transaction.TxCustomToken{
			TxTokenData: txTokenData,
		},
	}
	return &sendDCBVoteTokenTransaction, nil
}

func getAmountOfVoteToken(sumAmount uint64, voteAmount uint64) uint64 {
	return voteAmount * common.SumOfVoteDCBToken / sumAmount
}

func (blockgen *BlkTmplGenerator) CreateSendDCBVoteTokenToGovernorTx(chainID string, newDCBList lvdb.CandidateList, sumAmountDCB uint64) ([]metadata.Transaction) {
	var SendVoteTx []metadata.Transaction
	var newTx metadata.Transaction
	for i := 0; i <= NumberOfDCBGovernors; i++ {
		newTx, _ = blockgen.createSingleSendDCBVoteTokenTx(chainID, newDCBList[i].PubKey, getAmountOfVoteToken(sumAmountDCB, newDCBList[i].VoteAmount))
		SendVoteTx = append(SendVoteTx, newTx)
	}
	return SendVoteTx
}

func (blockgen *BlkTmplGenerator) createAcceptDCBBoardTx(DCBBoardPubKeys []string, sumOfVote uint64) transaction.TxAcceptDCBBoard{
	return transaction.TxAcceptDCBBoard{
		DCBBoardPubKeys: DCBBoardPubKeys,
		StartAmountDCBToken:sumOfVote,
	}
}

func (blockgen *BlkTmplGenerator) createAcceptGOVBoardTx(GOVBoardPubKeys []string, sumOfVote uint64) transaction.TxAcceptGOVBoard{
	return transaction.TxAcceptGOVBoard{
		GOVBoardPubKeys: GOVBoardPubKeys,
		StartAmountGOVToken:sumOfVote,
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


// startblock, pubkey, index
func parseVoteDCBBoardListKey(key []byte) (int32, []byte, uint32){
	// Todo xxx
	startedBlock := binary.LittleEndian.I key[:4]
	key
}

func parseVoteDCBBoardListValue(value []byte) ([]byte, uint64){
	// Todo xxx
}

func createSingleSendDCBVoteTokenFail(pubKey []byte, amount uint64) metadata.Transaction{
	paymentAddress := privacy.PaymentAddress{
		Pk: pubKey,
	}
	txTokenVout :=  transaction.TxTokenVout{
		Value: amount,
		PaymentAddress: paymentAddress,
	}
	newTx := transaction.TxCustomToken{
		TxTokenData:transaction.TxTokenData{
			Type: transaction.SendBackDCBTokenVoteFail,
			Amount:amount,
			PropertyID:DCBTokenID,
			Vins: []transaction.TxTokenVin{},
			Vouts: []transaction.TxTokenVout{txTokenVout},
		},
	}
	return &newTx
}

func (blockgen *BlkTmplGenerator) CreateSendBackDCBTokenAfterVoteInPool(chainID byte, newDCBList lvdb.CandidateList) []transaction.Transaction{
	setOfNewDCB := make(map[string]bool, 0)
	for _, i := range(newDCBList) {
		setOfNewDCB[string(i.PubKey)] = true
	}
	currentHeight := blockgen.chain.BestState[chainID].Height
	db := blockgen.chain.config.DataBase
	begin := db.GetKey(string(lvdb.VoteDCBBoardListPrefix), string(0))
	end := db.GetKey(string(lvdb.VoteDCBBoardListPrefix), currentHeight)
	searchRange := util.Range{begin, end}

	iter := blockgen.chain.config.DataBase.NewIterator( &searchRange, nil)
	listNewTx := make([]metadata.Transaction,0)
	for iter.Next() {
		key := iter.Key()
		startedBlock, PubKey, _ := parseVoteDCBBoardListKey(key)
		value := iter.Value()
		senderPubkey, amountOfDCBToken := parseVoteDCBBoardListValue(value)
		_, found := setOfNewDCB[string(PubKey)]
		if startedBlock < currentHeight || !found{
			listNewTx = append(listNewTx,createSingleSendDCBVoteTokenFail(senderPubkey, amountOfDCBToken))
		}
	}
	return listNewTx
}
