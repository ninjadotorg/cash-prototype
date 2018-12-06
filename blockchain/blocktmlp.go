package blockchain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/privacy-protocol"
	"github.com/ninjadotorg/constant/transaction"
	"github.com/ninjadotorg/constant/wallet"
)

type BlkTmplGenerator struct {
	txPool      TxPool
	chain       *BlockChain
	rewardAgent RewardAgent
}

type ConstitutionHelper interface {
	GetStartedBlockHeight(generator *BlkTmplGenerator, chainID byte) int32
	CheckSubmitProposalType(tx metadata.Transaction) bool
	CheckVotingProposalType(tx metadata.Transaction) bool
	GetAmountVoteToken(tx metadata.Transaction) uint32
	TxAcceptProposal(originTx metadata.Transaction) metadata.Transaction
}

// txPool represents a source of transactions to consider for inclusion in
// new blocks.
//
// The interface contract requires that all of these methods are safe for
// concurrent access with respect to the source.
type TxPool interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	LastUpdated() time.Time

	// MiningDescs returns a slice of mining descriptors for all the
	// transactions in the source pool.
	MiningDescs() []*metadata.TxDesc

	// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
	HaveTransaction(hash *common.Hash) bool

	// RemoveTx remove tx from tx resource
	RemoveTx(tx metadata.Transaction) error

	//CheckTransactionFee
	// CheckTransactionFee(tx metadata.Transaction) (uint64, error)

	// Check tx validate by it self
	// ValidateTxByItSelf(tx metadata.Transaction) bool
}

type RewardAgent interface {
	GetBasicSalary(chainID byte) uint64
	GetSalaryPerTx(chainID byte) uint64
}

func (self BlkTmplGenerator) Init(txPool TxPool, chain *BlockChain, rewardAgent RewardAgent) (*BlkTmplGenerator, error) {
	return &BlkTmplGenerator{
		txPool:      txPool,
		chain:       chain,
		rewardAgent: rewardAgent,
	}, nil
}

func (blockgen *BlkTmplGenerator) NewBlockTemplate(payToAddress privacy.PaymentAddress, chainID byte) (*Block, error) {

	prevBlock := blockgen.chain.BestState[chainID].BestBlock
	prevBlockHash := blockgen.chain.BestState[chainID].BestBlock.Hash()
	prevCmTree := blockgen.chain.BestState[chainID].CmTree.MakeCopy()
	sourceTxns := blockgen.txPool.MiningDescs()

	var txsToAdd []metadata.Transaction
	var txToRemove []metadata.Transaction
	var buySellReqTxs []metadata.Transaction
	var txTokenVouts map[*common.Hash]*transaction.TxTokenVout
	bondsSold := uint64(0)
	incomeFromBonds := uint64(0)
	totalFee := uint64(0)
	buyBackCoins := uint64(0)

	// Get salary per tx
	salaryPerTx := blockgen.rewardAgent.GetSalaryPerTx(chainID)
	// Get basic salary on block
	basicSalary := blockgen.rewardAgent.GetBasicSalary(chainID)

	// Check if it is the case we need to apply a new proposal
	// 1. newNW < lastNW * 0.9
	// 2. current block height == last Constitution start time + last Constitution execute duration
	if blockgen.neededNewDCBConstitution(chainID) {
		tx, err := blockgen.createAcceptConstitutionTxDecs(chainID, DCBConstitutionHelper{})
		if err != nil {
			Logger.log.Error(err)
			return nil, err
		}
		sourceTxns = append(sourceTxns, tx)
	}
	if blockgen.neededNewGovConstitution(chainID) {
		tx, err := blockgen.createAcceptConstitutionTxDecs(chainID, GOVConstitutionHelper{})
		if err != nil {
			Logger.log.Error(err)
			return nil, err
		}
		sourceTxns = append(sourceTxns, tx)
	}

	if len(sourceTxns) < common.MinTxsInBlock {
		// if len of sourceTxns < MinTxsInBlock -> wait for more transactions
		Logger.log.Info("not enough transactions. Wait for more...")
		<-time.Tick(common.MinBlockWaitTime * time.Second)
		sourceTxns = blockgen.txPool.MiningDescs()
		if len(sourceTxns) == 0 {
			<-time.Tick(common.MaxBlockWaitTime * time.Second)
			sourceTxns = blockgen.txPool.MiningDescs()
			if len(sourceTxns) == 0 {
				// return nil, errors.New("No Tx")
				Logger.log.Info("Creating empty block...")
				goto concludeBlock
			}
		}
	}

	for _, txDesc := range sourceTxns {
		tx := txDesc.Tx
		txChainID, _ := common.GetTxSenderChain(tx.GetSenderAddrLastByte())
		if txChainID != chainID {
			continue
		}
		// ValidateTransaction vote and propose transaction

		// if !blockgen.txPool.ValidateTxByItSelf(tx) {
		if !tx.ValidateTxByItself(blockgen.chain) {
			txToRemove = append(txToRemove, metadata.Transaction(tx))
			continue
		}

		if tx.GetType() == common.TxBuyFromGOVRequest {
			income, soldAmt, addable := blockgen.checkBuyFromGOVReqTx(chainID, tx, bondsSold)
			if !addable {
				txToRemove = append(txToRemove, tx)
				continue
			}
			bondsSold += soldAmt
			incomeFromBonds += income
			buySellReqTxs = append(buySellReqTxs, tx)
		}

		if tx.GetType() == common.TxBuyBackRequest {
			txTokenVout, buyBackReqTxID, addable := blockgen.checkBuyBackReqTx(chainID, tx, buyBackCoins)
			if !addable {
				txToRemove = append(txToRemove, tx)
				continue
			}
			buyBackCoins += txTokenVout.Value * txTokenVout.BuySellResponse.BuyBackInfo.BuyBackPrice
			txTokenVouts[buyBackReqTxID] = txTokenVout
		}

		totalFee += tx.GetTxFee()
		txsToAdd = append(txsToAdd, tx)
		if len(txsToAdd) == common.MaxTxsInBlock {
			break
		}
	}

	// check len of txs in block
	if len(txsToAdd) == 0 {
		// return nil, errors.New("no transaction available for this chain")
		Logger.log.Info("Creating empty block...")
	}

concludeBlock:
	rt := prevBlock.Header.MerkleRootCommitments.CloneBytes()
	blockHeight := prevBlock.Header.Height + 1

	// TODO(@0xbunyip): cap #tx to common.MaxTxsInBlock
	// Process dividend payout for DCB if needed
	bankDivTxs, bankPayoutAmount, err := blockgen.processBankDividend(rt, chainID, blockHeight)
	if err != nil {
		return nil, err
	}
	for _, tx := range bankDivTxs {
		txsToAdd = append(txsToAdd, tx)
	}

	// Process dividend payout for GOV if needed
	govDivTxs, govPayoutAmount, err := blockgen.processGovDividend(rt, chainID, blockHeight)
	if err != nil {
		return nil, err
	}
	for _, tx := range govDivTxs {
		txsToAdd = append(txsToAdd, tx)
	}

	// Process crowdsale for DCB
	dcbSaleTxs, removableTxs, err := blockgen.processCrowdsale(sourceTxns, rt, chainID)
	if err != nil {
		return nil, err
	}
	for _, tx := range dcbSaleTxs {
		txsToAdd = append(txsToAdd, tx)
	}
	for _, tx := range removableTxs {
		txToRemove = append(txToRemove, tx)
	}

	// Get blocksalary fund from txs
	salaryFundAdd := uint64(0)
	salaryMULTP := uint64(0) //salary multiplier
	for _, blockTx := range txsToAdd {
		if blockTx.GetTxFee() > 0 {
			salaryMULTP++
		}
	}

	// ------------------------ HOW to GET salary on a block-------------------
	// total salary = tx * (salary per tx) + (basic salary on block)
	// ------------------------------------------------------------------------
	totalSalary := salaryMULTP*salaryPerTx + basicSalary
	// create salary tx to pay constant for block producer
	salaryTx, err := transaction.CreateTxSalary(totalSalary, &payToAddress, rt, chainID)
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	// create buy/sell response txs to distribute bonds/govs to requesters
	buySellResTxs := blockgen.buildBuySellResponsesTx(
		common.TxBuyFromGOVResponse,
		buySellReqTxs,
		blockgen.chain.BestState[0].BestBlock.Header.GOVConstitution.GOVParams.SellingBonds,
	)
	// create buy-back response txs to distribute constants to buy-back requesters
	buyBackResTxs, err := blockgen.buildBuyBackResponsesTx(common.TxBuyBackResponse, txTokenVouts, chainID)
	// create refund txs
	currentSalaryFund := prevBlock.Header.SalaryFund
	remainingFund := currentSalaryFund + totalFee + salaryFundAdd + incomeFromBonds - (totalSalary + buyBackCoins)
	refundTxs, totalRefundAmt := blockgen.buildRefundTxs(chainID, remainingFund)

	// Get loan payment amount to add to DCB fund
	loanPaymentAmount, unlockTxs, removableTxs := blockgen.processLoan(sourceTxns, rt, chainID)
	for _, tx := range removableTxs {
		txToRemove = append(txToRemove, tx)
	}

	coinbases := []metadata.Transaction{salaryTx}
	for _, tx := range unlockTxs {
		coinbases = append(coinbases, tx)
	}
	for _, resTx := range buySellResTxs {
		coinbases = append(coinbases, resTx)
	}
	for _, resTx := range buyBackResTxs {
		coinbases = append(coinbases, resTx)
	}
	for _, refundTx := range refundTxs {
		coinbases = append(coinbases, refundTx)
	}
	txsToAdd = append(coinbases, txsToAdd...)

	for _, tx := range txToRemove {
		blockgen.txPool.RemoveTx(tx)
	}

	// Check for final balance of DCB and GOV
	if currentSalaryFund+totalFee+salaryFundAdd+incomeFromBonds < totalSalary+govPayoutAmount+buyBackCoins+totalRefundAmt {
		return nil, fmt.Errorf("Gov fund is not enough for salary and dividend payout")
	}

	currentBankFund := prevBlock.Header.BankFund
	if currentBankFund < bankPayoutAmount { // Can't spend loan payment just received in this block
		return nil, fmt.Errorf("Bank fund is not enough for dividend payout")
	}

	merkleRoots := Merkle{}.BuildMerkleTreeStore(txsToAdd)
	merkleRoot := merkleRoots[len(merkleRoots)-1]

	block := Block{
		Transactions: make([]metadata.Transaction, 0),
	}

	block.Header = BlockHeader{
		Height:                prevBlock.Header.Height + 1,
		Version:               BlockVersion,
		PrevBlockHash:         *prevBlockHash,
		MerkleRoot:            *merkleRoot,
		MerkleRootCommitments: common.Hash{},
		Timestamp:             time.Now().Unix(),
		BlockCommitteeSigs:    make([]string, common.TotalValidators),
		Committee:             make([]string, common.TotalValidators),
		ChainID:               chainID,
		SalaryFund:            currentSalaryFund + incomeFromBonds + totalFee + salaryFundAdd - totalSalary - govPayoutAmount - buyBackCoins - totalRefundAmt,
		BankFund:              prevBlock.Header.BankFund + loanPaymentAmount - bankPayoutAmount,
		GOVConstitution:       prevBlock.Header.GOVConstitution, // TODO: need get from gov-params tx
		DCBConstitution:       prevBlock.Header.DCBConstitution, // TODO: need get from dcb-params tx
	}
	if block.Header.GOVConstitution.GOVParams.SellingBonds != nil {
		block.Header.GOVConstitution.GOVParams.SellingBonds.BondsToSell -= bondsSold
	}

	if int32(block.Header.DCBGovernor.EndBlock) == block.Header.Height {
		newBoardList, _ := blockgen.chain.config.DataBase.GetTopMostVoteDCBGovernor(NumberOfDCBGovernors)
		sort.Sort(newBoardList)
		sumOfVote := uint64(0)
		var newDCBBoardPubKey []string
		for _, i := range newBoardList {
			newDCBBoardPubKey = append(newDCBBoardPubKey, i.PubKey)
			sumOfVote += i.VoteAmount
		}

		// prepend send votetoken to txstoadd
		txsToAdd = append([]transaction.Transaction{blockgen.createAcceptDCBBoardTx(newDCBBoardPubKey, sumOfVote)}, txsToAdd...)
		txsToAdd = append(blockgen.CreateSendDCBVoteTokenToGovernorTx(newBoardList, sumOfVote), txsToAdd...)

		// Todo @0xjackalope: send reward to board and delete them from database before send back token to new board

		txsAddtoPool := blockgen.CreateSendBackDCBTokenAfterVoteInPool(block.Header.DCBGovernor.DCBBoardPubKeys, block.Header.DCBGovernor.StartAmountDCBToken)
		//xxx add to pool
	}

	for _, tx := range txsToAdd {
		if err := block.AddTransaction(tx); err != nil {
			return nil, err
		}
		// Handle if this transaction change something in block header
		if tx.GetType() == common.TxAcceptDCBProposal {
			block.updateDCBConstitution(tx, blockgen)
		}
		if tx.GetType() == common.TxAcceptGOVProposal {
			block.updateGOVConstitution(tx, blockgen)
		}
		if tx.GetType() == common.TxAcceptDCBBoard {
			block.UpdateDCBBoard(tx)
		}
		if tx.GetType() == common.TxAcceptGOVBoard {
			block.UpdateGOVBoard(tx)
		}
	}

	// Add new commitments to merkle tree and save the root
	newTree := prevCmTree
	err = UpdateMerkleTreeForBlock(newTree, &block)
	if err != nil {
		Logger.log.Error(err)
		return nil, err
	}
	rt = newTree.GetRoot(common.IncMerkleTreeHeight)
	copy(block.Header.MerkleRootCommitments[:], rt)

	//update the latest AgentDataPoints to block
	// block.AgentDataPoints = agentDataPoints
	return &block, nil
}

func GetOracleDCBNationalWelfare() int32 {
	fmt.Print("Get national welfare. It is constant now. Need to change !!!")
	return 1234
}
func GetOracleGOVNationalWelfare() int32 {
	fmt.Print("Get national welfare. It is constant now. Need to change !!!")
	return 1234
}

//1. Current National welfare (NW)  < lastNW * 0.9 (Emergency case)
//2. Block height == last constitution start time + last constitution window
func (blockgen *BlkTmplGenerator) neededNewDCBConstitution(chainID byte) bool {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastDCBConstitution := BestBlock.Header.DCBConstitution
	if GetOracleDCBNationalWelfare() < lastDCBConstitution.CurrentDCBNationalWelfare*ThresholdRatioOfDCBCrisis/100 ||
		BestBlock.Header.Height+1 == lastDCBConstitution.StartedBlockHeight+lastDCBConstitution.ExecuteDuration {
		return true
	}
	return false
}
func (blockgen *BlkTmplGenerator) neededNewGovConstitution(chainID byte) bool {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastGovConstitution := BestBlock.Header.GOVConstitution
	if GetOracleGOVNationalWelfare() < lastGovConstitution.CurrentGOVNationalWelfare*ThresholdRatioOfGovCrisis/100 ||
		BestBlock.Header.Height+1 == lastGovConstitution.StartedBlockHeight+lastGovConstitution.ExecuteDuration {
		return true
	}
	return false
}

func (blockgen *BlkTmplGenerator) createRequestConstitutionTxDecs(
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

func (blockgen *BlkTmplGenerator) processDividend(
	rt []byte,
	chainID byte,
	proposal *transaction.PayoutProposal,
	blockHeight int32,
) ([]*transaction.TxDividendPayout, uint64, error) {
	payoutAmount := uint64(0)

	// TODO(@0xbunyip): how to execute payout dividend proposal
	dividendTxs := []*transaction.TxDividendPayout{}
	if false && chainID == 0 && blockHeight%transaction.PayoutFrequency == 0 { // only chain 0 process dividend proposals
		totalTokenSupply, tokenHolders, amounts, err := blockgen.chain.GetAmountPerAccount(proposal)
		if err != nil {
			return nil, 0, err
		}

		infos := []transaction.DividendInfo{}
		// Build tx to pay dividend to each holder
		for i, holder := range tokenHolders {
			// TODO(@0xbunyip): holder here is Pk only, change to use both Pk and Pkenc
			holderAddr, err := hex.DecodeString(holder)
			if err != nil {
				return nil, 0, err
			}
			holderAddress := (&privacy.PaymentAddress{}).FromBytes(holderAddr)
			info := transaction.DividendInfo{
				TokenHolder: *holderAddress,
				Amount:      amounts[i] / totalTokenSupply,
			}
			payoutAmount += info.Amount
			infos = append(infos, info)

			if len(infos) > transaction.MaxDivTxsPerBlock {
				break // Pay dividend to only some token holders in this block
			}
		}

		dividendTxs, err = transaction.BuildDividendTxs(infos, rt, chainID, proposal)
		if err != nil {
			return nil, 0, err
		}
	}
	return dividendTxs, payoutAmount, nil
}

func (blockgen *BlkTmplGenerator) processBankDividend(rt []byte, chainID byte, blockHeight int32) ([]*transaction.TxDividendPayout, uint64, error) {
	tokenID := &common.Hash{} // TODO(@0xbunyip): hard-code tokenID of BANK token and get proposal
	proposal := &transaction.PayoutProposal{
		TokenID: tokenID,
	}
	return blockgen.processDividend(rt, chainID, proposal, blockHeight)
}

func (blockgen *BlkTmplGenerator) processGovDividend(rt []byte, chainID byte, blockHeight int32) ([]*transaction.TxDividendPayout, uint64, error) {
	tokenID := &common.Hash{} // TODO(@0xbunyip): hard-code tokenID of GOV token and get proposal
	proposal := &transaction.PayoutProposal{
		TokenID: tokenID,
	}
	return blockgen.processDividend(rt, chainID, proposal, blockHeight)
}

func buildSingleBuySellResponseTx(
	buySellReqTx *transaction.TxBuySellRequest,
	sellingBondsParam *SellingBonds,
) transaction.TxTokenVout {
	buyBackInfo := &transaction.BuyBackInfo{
		Maturity:     sellingBondsParam.Maturity,
		BuyBackPrice: sellingBondsParam.BuyBackPrice,
	}

	bondID := fmt.Sprintf("%s%s%s", sellingBondsParam.Maturity, sellingBondsParam.BuyBackPrice, sellingBondsParam.StartSellingAt)
	additionalSuffix := make([]byte, 24-len(bondID))
	bondIDBytes := append([]byte(bondID), additionalSuffix...)
	buySellResponse := &transaction.BuySellResponse{
		BuyBackInfo: buyBackInfo,
		BondID:      bondIDBytes,
	}
	return transaction.TxTokenVout{
		Value:           buySellReqTx.Amount,
		PaymentAddress:  buySellReqTx.PaymentAddress,
		BuySellResponse: buySellResponse,
	}
}

func (blockgen *BlkTmplGenerator) checkBuyFromGOVReqTx(
	chainID byte,
	tx metadata.Transaction,
	bondsSold uint64,
) (uint64, uint64, bool) {
	prevBlock := blockgen.chain.BestState[chainID].BestBlock
	sellingBondsParams := prevBlock.Header.GOVConstitution.GOVParams.SellingBonds
	if uint32(prevBlock.Header.Height)+1 > sellingBondsParams.StartSellingAt+sellingBondsParams.SellingWithin {
		return 0, 0, false
	}

	reqTx, ok := tx.(*transaction.TxBuySellRequest)
	if !ok {
		return 0, 0, false
	}
	if bondsSold+reqTx.Amount > sellingBondsParams.BondsToSell { // run out of bonds for selling
		return 0, 0, false
	}
	return reqTx.Amount * reqTx.BuyPrice, reqTx.Amount, true
}

// buildBuySellResponsesTx
// the tx is to distribute tokens (bond, gov, ...) to token requesters
func (blockgen *BlkTmplGenerator) buildBuySellResponsesTx(
	coinbaseTxType string,
	buySellReqTxs []metadata.Transaction,
	sellingBondsParam *SellingBonds,
) []*transaction.TxCustomToken {
	if len(buySellReqTxs) == 0 {
		return nil
	}
	var resTxs []*transaction.TxCustomToken
	for _, reqTx := range buySellReqTxs {
		tx, _ := reqTx.(*transaction.TxBuySellRequest)
		txTokenVout := buildSingleBuySellResponseTx(tx, sellingBondsParam)
		bondIDBytes := txTokenVout.BuySellResponse.BondID
		var propertyID [common.HashSize]byte
		copy(propertyID[:], append(BondTokenID[0:8], bondIDBytes...))
		txTokenData := transaction.TxTokenData{
			Type:       transaction.CustomTokenInit,
			Amount:     tx.Amount,
			PropertyID: common.Hash(propertyID),
			Vins:       []transaction.TxTokenVin{},
			Vouts:      []transaction.TxTokenVout{txTokenVout},
			// PropertyName:   "",
			// PropertySymbol: coinbaseTxType,
		}
		resTx := &transaction.TxCustomToken{
			TxTokenData: txTokenData,
		}
		resTx.Type = coinbaseTxType
		resTx.RequestedTxID = tx.Hash()
		resTxs = append(resTxs, resTx)
	}
	return resTxs
}

func (blockgen *BlkTmplGenerator) checkBuyBackReqTx(
	chainID byte,
	tx metadata.Transaction,
	buyBackConsts uint64,
) (*transaction.TxTokenVout, *common.Hash, bool) {
	txBuyBackReq, ok := tx.(*transaction.TxBuyBackRequest)
	if !ok {
		return nil, nil, false
	}
	_, _, _, fromTx, err := blockgen.chain.GetTransactionByHash(txBuyBackReq.BuyBackFromTxID)
	if err != nil {
		Logger.log.Error(err)
		return nil, nil, false
	}
	customTokenTx, ok := fromTx.(*transaction.TxCustomToken)
	if !ok {
		return nil, nil, false
	}
	vout := customTokenTx.TxTokenData.Vouts[txBuyBackReq.VoutIndex]
	buyBackInfo := vout.BuySellResponse.BuyBackInfo
	if buyBackInfo == nil {
		Logger.log.Error("Missing buy-back info")
		return nil, nil, false
	}
	prevBlock := blockgen.chain.BestState[chainID].BestBlock

	if buyBackInfo.StartSellingAt+buyBackInfo.Maturity > uint32(prevBlock.Header.Height)+1 {
		Logger.log.Error("The token is not overdued yet.")
		return nil, nil, false
	}
	// check remaining constants in GOV fun is enough or not
	buyBackAmount := vout.Value * vout.BuySellResponse.BuyBackInfo.BuyBackPrice
	if buyBackConsts+buyBackAmount > prevBlock.Header.SalaryFund {
		return nil, nil, false
	}
	return &vout, tx.Hash(), true
}

func (blockgen *BlkTmplGenerator) buildBuyBackResponsesTx(
	coinbaseTxType string,
	txTokenReqVouts map[*common.Hash]*transaction.TxTokenVout,
	chainID byte,
) ([]*transaction.Tx, error) {
	if len(txTokenReqVouts) == 0 {
		return []*transaction.Tx{}, nil
	}

	prevBlock := blockgen.chain.BestState[chainID].BestBlock
	rt := prevBlock.Header.MerkleRootCommitments.CloneBytes()
	var buyBackResTxs []*transaction.Tx
	for buyBackReqTxID, txTokenReqVout := range txTokenReqVouts {
		buyBackAmount := txTokenReqVout.Value * txTokenReqVout.BuySellResponse.BuyBackInfo.BuyBackPrice
		buyBackResTx, err := transaction.CreateTxSalary(buyBackAmount, &txTokenReqVout.PaymentAddress, rt, chainID)
		if err != nil {
			return []*transaction.Tx{}, err
		}
		buyBackResTx.RequestedTxID = buyBackReqTxID
		buyBackResTx.Type = coinbaseTxType
		buyBackResTxs = append(buyBackResTxs, buyBackResTx)
	}
	return buyBackResTxs, nil
}

func calculateAmountOfRefundTxs(
	addresses []*privacy.PaymentAddress,
	estimatedRefundAmt uint64,
	remainingFund uint64,
	rt []byte,
	chainID byte,
) ([]*transaction.Tx, uint64) {
	amt := uint64(0)
	if estimatedRefundAmt <= remainingFund {
		amt = estimatedRefundAmt
	} else {
		amt = remainingFund
	}
	actualRefundAmt := amt / uint64(len(addresses))
	var refundTxs []*transaction.Tx
	for i := 0; i < len(addresses); i++ {
		addr := addresses[i]
		refundTx, err := transaction.CreateTxSalary(actualRefundAmt, addr, rt, chainID)
		if err != nil {
			Logger.log.Error(err)
			continue
		}
		refundTxs = append(refundTxs, refundTx)
	}
	return refundTxs, amt
}

func (blockgen *BlkTmplGenerator) buildRefundTxs(
	chainID byte,
	remainingFund uint64,
) ([]*transaction.Tx, uint64) {
	if remainingFund <= 0 {
		Logger.log.Info("GOV fund is not enough for refund.")
		return []*transaction.Tx{}, 0
	}
	prevBlock := blockgen.chain.BestState[chainID].BestBlock
	header := prevBlock.Header
	govParams := header.GOVConstitution.GOVParams
	refundInfo := govParams.RefundInfo
	if refundInfo == nil {
		Logger.log.Info("Refund info is not existed.")
		return []*transaction.Tx{}, 0
	}
	lookbackBlockHeight := header.Height - common.RefundPeriod
	if lookbackBlockHeight < 0 {
		return []*transaction.Tx{}, 0
	}
	lookbackBlock, err := blockgen.chain.GetBlockByBlockHeight(lookbackBlockHeight, chainID)
	if err != nil {
		Logger.log.Error(err)
		return []*transaction.Tx{}, 0
	}
	var addresses []*privacy.PaymentAddress
	estimatedRefundAmt := uint64(0)
	for _, tx := range lookbackBlock.Transactions {
		if tx.GetType() != common.TxNormalType {
			continue
		}
		lookbackTx, ok := tx.(*transaction.Tx)
		if !ok {
			continue
		}
		addr, txValue := lookbackTx.CalculateTxValue()
		if addr == nil || txValue > refundInfo.ThresholdToLargeTx {
			continue
		}
		addresses = append(addresses, addr)
		estimatedRefundAmt += refundInfo.RefundAmount
	}
	if len(addresses) == 0 {
		return []*transaction.Tx{}, 0
	}
	refundTxs, totalRefundAmt := calculateAmountOfRefundTxs(
		addresses,
		estimatedRefundAmt,
		remainingFund,
		prevBlock.Header.MerkleRootCommitments.CloneBytes(),
		chainID,
	)
	return refundTxs, totalRefundAmt
}

func (blockgen *BlkTmplGenerator) processCrowdsale(sourceTxns []*metadata.TxDesc, rt []byte, chainID byte) ([]*transaction.TxBuySellDCBResponse, []metadata.Transaction, error) {
	txsToRemove := []metadata.Transaction{}
	txsResponse := []*transaction.TxBuySellDCBResponse{}
	// Get unspent bond tx to spend if needed
	accountDCB, _ := wallet.Base58CheckDeserialize(common.DCBAddress)
	keySet := accountDCB.KeySet

	tokenID := &common.Hash{} // TODO(@0xbunyip): hard code bond token id here
	unspentTxTokenOuts, err := blockgen.chain.GetUnspentTxCustomTokenVout(keySet, tokenID)
	if err != nil {
		unspentTxTokenOuts = []transaction.TxTokenVout{}
	}
	for _, txDesc := range sourceTxns {
		if txDesc.Tx.GetType() != common.TxBuySellDCBRequest {
			continue
		}

		tx, ok := (txDesc.Tx).(*transaction.TxBuySellRequest)
		if !ok {
			txsToRemove = append(txsToRemove, tx)
		}

		// Create corresponding response to send selling asset
		// Get buying and selling asset from current sale
		saleData, err := blockgen.chain.config.DataBase.LoadCrowdsaleData(tx.SaleID)
		if err != nil {
			txsToRemove = append(txsToRemove, tx)
			continue
		}

		// Get price for asset bond
		bondPrices := blockgen.chain.BestState[chainID].BestBlock.Header.Oracle.Bonds
		if bytes.Equal(saleData.SellingAsset, ConstantID[:]) {
			txResponse, err := transaction.BuildResponseForCoin(tx, saleData.SellingAsset, rt, chainID, bondPrices, tx.SaleID, common.DCBAddress)
			if err != nil {
				txsToRemove = append(txsToRemove, tx)
			} else {
				txsResponse = append(txsResponse, txResponse)
			}
		} else if bytes.Equal(saleData.SellingAsset[:8], BondTokenID[:8]) {
			// Get unspent token UTXO to send to user
			txResponse := &transaction.TxBuySellDCBResponse{}
			txResponse, unspentTxTokenOuts, err = transaction.BuildResponseForBond(tx, saleData.SellingAsset, rt, chainID, bondPrices, unspentTxTokenOuts, tx.SaleID, common.DCBAddress)
			if err != nil {
				txsToRemove = append(txsToRemove, tx)
			} else {
				txsResponse = append(txsResponse, txResponse)
			}
		} else {
			txsToRemove = append(txsToRemove, tx)
		}
	}
	return txsResponse, txsToRemove, nil
}

func (blockgen *BlkTmplGenerator) processLoan(sourceTxns []*metadata.TxDesc, rt []byte, chainID byte) (uint64, []*transaction.TxLoanUnlock, []metadata.Transaction) {
	amount := uint64(0)
	loanUnlockTxs := []*transaction.TxLoanUnlock{}
	removableTxs := []metadata.Transaction{}
	for _, txDesc := range sourceTxns {
		if txDesc.Tx.GetType() == common.TxLoanPayment {
			tx := (txDesc.Tx).(*transaction.TxLoanPayment)
			_, _, _, err := blockgen.chain.config.DataBase.GetLoanPayment(tx.LoanID)
			if err != nil {
				removableTxs = append(removableTxs, tx)
				continue
			}
			paymentAmount := uint64(0)
			for _, desc := range tx.Descs {
				for _, note := range desc.Note {
					paymentAmount += note.Value
				}
			}
			if !tx.PayPrinciple { // Only keep interest
				amount += paymentAmount
			}
		} else if txDesc.Tx.GetType() == common.TxLoanWithdraw {
			tx := txDesc.Tx.(*transaction.TxLoanWithdraw)
			meta, err := blockgen.chain.getLoanRequestMeta(tx.LoanID)
			if err != nil {
				removableTxs = append(removableTxs, tx)
				continue
			}
			pks := [][]byte{meta.ReceiveAddress.Pk[:], make([]byte, 33)}
			tks := [][]byte{meta.ReceiveAddress.Tk[:], make([]byte, 33)}
			amounts := []uint64{meta.LoanAmount, 0}
			txNormal, err := transaction.BuildCoinbaseTx(pks, tks, amounts, rt, chainID, common.TxLoanUnlock)
			if err != nil {
				removableTxs = append(removableTxs, tx)
				continue
			}
			txUnlock := &transaction.TxLoanUnlock{
				Tx:     *txNormal,
				LoanID: make([]byte, len(tx.LoanID)),
			}
			copy(txUnlock.LoanID, tx.LoanID)
			loanUnlockTxs = append(loanUnlockTxs, txUnlock)
		}
	}
	return amount, loanUnlockTxs, removableTxs
}
