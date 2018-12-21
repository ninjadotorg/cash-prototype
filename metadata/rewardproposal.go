package metadata

import (
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
)

type RewardProposalMetadata struct {
	PubKey []byte
	Prize  uint32
	MetadataBase
}

func NewRewardProposalMetadata(pubKey []byte, prize uint32) *RewardProposalMetadata {
	return &RewardProposalMetadata{
		PubKey:       pubKey,
		Prize:        prize,
		MetadataBase: *NewMetadataBase(RewardProposalMeta),
	}
}

func (rewardProposalMetadata *RewardProposalMetadata) Hash() *common.Hash {
	record := string(rewardProposalMetadata.PubKey)
	record += common.Uint32ToString(rewardProposalMetadata.Prize)
	record += string(rewardProposalMetadata.MetadataBase.Hash()[:])
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (rewardProposalMetadata *RewardProposalMetadata) ValidateTxWithBlockChain(Transaction, BlockchainRetriever, byte, database.DatabaseInterface) (bool, error) {
	return true, nil
}

func (rewardProposalMetadata *RewardProposalMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	return true, true, nil
}

func (rewardProposalMetadata *RewardProposalMetadata) ValidateMetadataByItself() bool {
	return true
}
