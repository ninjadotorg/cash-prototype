package metadata

import (
	"github.com/ninjadotorg/constant/common"
)

type VoteDCBBoardMetadata struct {
	CandidatePubKey string

	MetadataBase
}

func NewVoteDCBBoardMetadata(voteDCBBoardMetadata map[string]interface{}) *VoteDCBBoardMetadata {
	return &VoteDCBBoardMetadata{
		CandidatePubKey: voteDCBBoardMetadata["candidatePubKey"].(string),
	}
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) ValidateTxWithBlockChain(txr Transaction, bcr BlockchainRetriever, chainID byte) (bool, error) {
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

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) ValidateSanityData() (bool, bool, error) {
	return true, true, nil
}

func (voteDCBBoardMetadata *VoteDCBBoardMetadata) ValidateMetadataByItself() bool {
	return true
}

type VoteGOVBoardMetadata struct {
	CandidatePubKey string

	MetadataBase
}

func NewVoteGOVBoardMetadata(voteGOVBoardMetadata map[string]interface{}) *VoteGOVBoardMetadata {
	return &VoteGOVBoardMetadata{
		CandidatePubKey: voteGOVBoardMetadata["candidatePubKey"].(string),
	}
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) ValidateTxWithBlockChain(txr Transaction, bcr BlockchainRetriever, chainID byte) (bool, error) {
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

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) ValidateSanityData() (bool, bool, error) {
	return true, true, nil
}

func (voteGOVBoardMetadata *VoteGOVBoardMetadata) ValidateMetadataByItself() bool {
	return true
}
