package blockchain

import (
	"github.com/ninjadotorg/constant/blockchain/params"
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/metadata"
	"github.com/ninjadotorg/constant/transaction"
	"github.com/ninjadotorg/constant/voting"
)

type ConstitutionInfo struct {
	StartedBlockHeight uint32
	ExecuteDuration    uint32
	ProposalTXID       common.Hash
}

type GOVConstitution struct {
	ConstitutionInfo
	CurrentGOVNationalWelfare int32
	GOVParams                 params.GOVParams
}

func (dcbConstitution DCBConstitution) GetEndedBlockHeight() uint32 {
	return dcbConstitution.StartedBlockHeight + dcbConstitution.ExecuteDuration
}

func (govConstitution GOVConstitution) GetEndedBlockHeight() uint32 {
	return govConstitution.StartedBlockHeight + govConstitution.ExecuteDuration
}

type DCBConstitution struct {
	ConstitutionInfo
	CurrentDCBNationalWelfare int32
	DCBParams                 params.DCBParams
}

type DCBConstitutionHelper struct{}
type GOVConstitutionHelper struct{}

func (DCBConstitutionHelper) GetEndedBlockHeight(blockgen *BlkTmplGenerator, chainID byte) uint32 {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastDCBConstitution := BestBlock.Header.DCBConstitution
	return lastDCBConstitution.StartedBlockHeight + lastDCBConstitution.ExecuteDuration
}

func (GOVConstitutionHelper) GetEndedBlockHeight(blockgen *BlkTmplGenerator, chainID byte) uint32 {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastGOVConstitution := BestBlock.Header.GOVConstitution
	return lastGOVConstitution.StartedBlockHeight + lastGOVConstitution.ExecuteDuration
}

func (DCBConstitutionHelper) GetStartedNormalVote(blockgen *BlkTmplGenerator, chainID byte) uint32 {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastDCBConstitution := BestBlock.Header.DCBConstitution
	return lastDCBConstitution.StartedBlockHeight - common.EncryptionPhaseDuration
}

func (DCBConstitutionHelper) CheckSubmitProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.SubmitDCBProposalMeta
}

func (DCBConstitutionHelper) CheckVotingProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.VoteDCBProposalMeta
}

func (DCBConstitutionHelper) GetAmountVoteToken(tx metadata.Transaction) uint64 {
	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
}

func (GOVConstitutionHelper) GetStartedNormalVote(blockgen *BlkTmplGenerator, chainID byte) uint32 {
	BestBlock := blockgen.chain.BestState[chainID].BestBlock
	lastGOVConstitution := BestBlock.Header.GOVConstitution
	return lastGOVConstitution.StartedBlockHeight - common.EncryptionPhaseDuration
}

func (GOVConstitutionHelper) CheckSubmitProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.SubmitGOVProposalMeta
}

func (GOVConstitutionHelper) CheckVotingProposalType(tx metadata.Transaction) bool {
	return tx.GetMetadataType() == metadata.VoteGOVProposalMeta
}

func (GOVConstitutionHelper) GetAmountVoteToken(tx metadata.Transaction) uint64 {
	return tx.(*transaction.TxCustomToken).GetAmountOfVote()
}

func (DCBConstitutionHelper) TxAcceptProposal(txId *common.Hash, voter voting.Voter) metadata.Transaction {
	acceptTx := transaction.Tx{
		Metadata: metadata.NewAcceptDCBProposalMetadata(*txId, voter),
	}
	return &acceptTx
}

func (GOVConstitutionHelper) TxAcceptProposal(txId *common.Hash, voter voting.Voter) metadata.Transaction {
	acceptTx := transaction.Tx{
		Metadata: metadata.NewAcceptGOVProposalMetadata(*txId, voter),
	}
	return &acceptTx
}

func (DCBConstitutionHelper) GetLowerCaseBoardType() string {
	return "dcb"
}

func (GOVConstitutionHelper) GetLowerCaseBoardType() string {
	return "gov"
}

func (DCBConstitutionHelper) CreatePunishDecryptTx(data map[string]interface{}) metadata.Metadata {
	return metadata.NewPunishDCBDecryptMetadata(data)
}

func (GOVConstitutionHelper) CreatePunishDecryptTx(data map[string]interface{}) metadata.Metadata {
	return metadata.NewPunishGOVDecryptMetadata(data)
}

func (DCBConstitutionHelper) GetSealerPubKey(tx metadata.Transaction) [][]byte {
	meta := tx.GetMetadata().(*metadata.SealedLv3DCBBallotMetadata)
	return meta.LockerPubKeys
}

func (GOVConstitutionHelper) GetSealerPubKey(tx metadata.Transaction) [][]byte {
	meta := tx.GetMetadata().(*metadata.SealedLv3GOVBallotMetadata)
	return meta.LockerPubKey
}
