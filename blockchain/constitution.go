package blockchain

import (
	"github.com/ninjadotorg/constant/blockchain/params"
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/transaction"
)

type ConstitutionInfo struct {
	StartedBlockHeight int32
	ExecuteDuration    int32
	ProposalTXID       *common.Hash
}

type GOVConstitution struct {
	ConstitutionInfo
	CurrentGOVNationalWelfare int32
	GOVParams                 GOVParams
}

type DCBConstitution struct {
	ConstitutionInfo
	CurrentDCBNationalWelfare int32
	DCBParams                 params.DCBParams
}

type DCBConstitutionHelper struct{}
type GOVConstitutionHelper struct{}

func (DCBConstitutionHelper) GetStartedBlockHeight(blockgen *BlkTmplGenerator, chainID byte) int32 {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastDCBConstitution := BestBlock.Header.DCBConstitution
	return lastDCBConstitution.StartedBlockHeight
}

func (DCBConstitutionHelper) CheckSubmitProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.SubmitDCBProposal
}

func (DCBConstitutionHelper) CheckVotingProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.VoteDCBProposal
}

func (DCBConstitutionHelper) GetAmountVoteToken(tx metadata.Transaction) uint64 {
	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
}

func (GOVConstitutionHelper) GetStartedBlockHeight(blockgen *BlkTmplGenerator, chainID byte) int32 {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastGOVConstitution := BestBlock.Header.GOVConstitution
	return lastGOVConstitution.StartedBlockHeight
}

func (GOVConstitutionHelper) CheckSubmitProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.SubmitGOVProposal
}

func (GOVConstitutionHelper) CheckVotingProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.VoteGOVProposal
}

func (GOVConstitutionHelper) GetAmountVoteToken(tx metadata.Transaction) uint64 {
	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
}

func (DCBConstitutionHelper) TxAcceptProposal(originTx metadata.Transaction) metadata.Transaction {
	acceptTx := transaction.Tx{
		Metadata: &metadata.AcceptDCBProposalMetadata{
			DCBProposalTXID: originTx.Hash(),
		},
	}
	return &acceptTx
}

func (GOVConstitutionHelper) TxAcceptProposal(originTx metadata.Transaction) metadata.Transaction {
	acceptTx := transaction.Tx{
		Metadata: &metadata.AcceptGOVProposalMetadata{
			GOVProposalTXID: originTx.Hash(),
		},
	}
	return &acceptTx
}
