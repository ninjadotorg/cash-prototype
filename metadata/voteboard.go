package metadata

import (
	"github.com/ninjadotorg/constant/common"
)

type VoteDCBBoardMetadata struct {
	CandidatePubKey string

	Metadata
}

func NewVoteDCBBoardMetadata(voteDCBBoardMetadata map[string]interface{}) *VoteDCBBoardMetadata {
	return &VoteDCBBoardMetadata{
		CandidatePubKey: voteDCBBoardMetadata["candidatePubKey"].(string),
	}
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) Validate() error {
	return nil
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) Process() error {
	return nil
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) CheckTransactionFee(tr TxRetriever, minFee uint64) bool {
	return true
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) ValidateTxWithBlockChain(bcr BlockchainRetriever, chainID byte) (bool, error) {
	return true, nil
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) GetType() int {
	return VoteDCBBoardMeta
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) Hash() *common.Hash {
	record := voteDCBBoardMetadata.CandidatePubKey
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

type VoteGOVBoardMetadata struct {
	CandidatePubKey string

	Metadata
}

func NewVoteGOVBoardMetadata(voteGOVBoardMetadata map[string]interface{}) *VoteGOVBoardMetadata {
	return &VoteGOVBoardMetadata{
		CandidatePubKey: voteGOVBoardMetadata["candidatePubKey"].(string),
	}
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) Validate() error {
	return nil
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) Process() error {
	return nil
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) CheckTransactionFee(tr TxRetriever, minFee uint64) bool {
	return true
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) ValidateTxWithBlockChain(bcr BlockchainRetriever, chainID byte) (bool, error) {
	return true, nil
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) GetType() int {
	return VoteGOVBoardMeta
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) Hash() *common.Hash {
	record := voteGOVBoardMetadata.CandidatePubKey
	hash := common.DoubleHashH([]byte(record))
	return &hash
}
