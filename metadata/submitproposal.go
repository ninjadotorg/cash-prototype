package metadata

import (
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/voting"
)

type SubmitDCBProposalMetadata struct {
	DCBVotingParams voting.DCBVotingParams
	ExecuteDuration int32
	Explanation     string

	MetadataBase
}

//calling from rpc function
func NewSubmitDCBProposalMetadataFromJson(jsonData map[string]interface{}) *SubmitDCBProposalMetadata {
	submitDCBProposalMetadata := SubmitDCBProposalMetadata{
		DCBVotingParams: voting.DCBVotingParams{
			SaleData: &voting.SaleData{
				SaleID:       []byte(jsonData["SaleID"].(string)),
				BuyingAsset:  []byte(jsonData["BuyingAsset"].(string)),
				SellingAsset: []byte(jsonData["SellingAsset"].(string)),
				EndBlock:     int32(jsonData["EndBlock"].(float64)),
			},
		},
		ExecuteDuration: int32(jsonData["ExecuteDuration"].(float64)),
		Explanation:     jsonData["Explanation"].(string),
		MetadataBase: MetadataBase{
			Type: SubmitDCBProposalMeta,
		},
	}
	return &submitDCBProposalMetadata
}

func (submitDCBProposalMetadata *SubmitDCBProposalMetadata) Hash() *common.Hash {
	record := string(common.ToBytes(submitDCBProposalMetadata.DCBVotingParams.Hash()))
	record += string(submitDCBProposalMetadata.ExecuteDuration)
	record += submitDCBProposalMetadata.Explanation
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (submitDCBProposalMetadata *SubmitDCBProposalMetadata) ValidateTxWithBlockChain(Transaction, BlockchainRetriever, byte) (bool, error) {
	return true, nil
}

func (submitDCBProposalMetadata *SubmitDCBProposalMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	if !submitDCBProposalMetadata.DCBVotingParams.ValidateSanityData() {
		return true, false, nil
	}
	if submitDCBProposalMetadata.ExecuteDuration < common.MinimumBlockOfProposalDuration ||
		submitDCBProposalMetadata.ExecuteDuration > common.MaximumBlockOfProposalDuration {
		return true, false, nil
	}
	if len(submitDCBProposalMetadata.Explanation) > common.MaximumProposalExplainationLength {
		return true, false, nil
	}
	return true, true, nil
}

func (submitDCBProposalMetadata *SubmitDCBProposalMetadata) ValidateMetadataByItself() bool {
	return true
}

type SubmitGOVProposalMetadata struct {
	GOVVotingParams voting.GOVVotingParams
	ExecuteDuration int32
	Explaination    string

	MetadataBase
}

//calling from rpc function
func NewSubmitGOVProposalMetadataFromJson(jsonData map[string]interface{}) *SubmitGOVProposalMetadata {
	submitGOVProposalMetadata := SubmitGOVProposalMetadata{
		GOVVotingParams: voting.GOVVotingParams{
			SalaryPerTx: uint64(jsonData["SalaryPerTx"].(float64)),
			BasicSalary: uint64(jsonData["BasicSalary"].(float64)),
			TxFee:       uint64(jsonData["TxFee"].(float64)),
			SellingBonds: &voting.SellingBonds{
				BondsToSell:    uint64(jsonData["BondsToSell"].(float64)),
				BondPrice:      uint64(jsonData["BondPrice"].(float64)),
				Maturity:       uint32(jsonData["Maturity"].(float64)),
				BuyBackPrice:   uint64(jsonData["BuyBackPrice"].(float64)),
				StartSellingAt: uint32(jsonData["StartSellingAt"].(float64)),
				SellingWithin:  uint32(jsonData["SellingWithin"].(float64)),
			},
			RefundInfo: &voting.RefundInfo{
				ThresholdToLargeTx: uint64(jsonData["ThresholdToLargeTx"].(float64)),
				RefundAmount:       uint64(jsonData["RefundAmount"].(float64)),
			},
		},
		ExecuteDuration: int32(jsonData["ExecuteDuration"].(float64)),
		Explaination:    string(jsonData["Explaination"].(string)),
		MetadataBase: MetadataBase{
			Type: SubmitGOVProposalMeta,
		},
	}
	return &submitGOVProposalMetadata
}

func (submitGOVProposalMetadata *SubmitGOVProposalMetadata) Hash() *common.Hash {
	record := string(common.ToBytes(submitGOVProposalMetadata.GOVVotingParams.Hash()))
	record += string(submitGOVProposalMetadata.ExecuteDuration)
	record += submitGOVProposalMetadata.Explaination
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (submitGOVProposalMetadata *SubmitGOVProposalMetadata) ValidateTxWithBlockChain(Transaction, BlockchainRetriever, byte) (bool, error) {
	return true, nil
}

func (submitGOVProposalMetadata *SubmitGOVProposalMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	if !submitGOVProposalMetadata.GOVVotingParams.ValidateSanityData() {
		return true, false, nil
	}
	if submitGOVProposalMetadata.ExecuteDuration < common.MinimumBlockOfProposalDuration ||
		submitGOVProposalMetadata.ExecuteDuration > common.MaximumBlockOfProposalDuration {
		return true, false, nil
	}
	if len(submitGOVProposalMetadata.Explaination) > common.MaximumProposalExplainationLength {
		return true, false, nil
	}
	return true, true, nil
}

func (submitGOVProposalMetadata *SubmitGOVProposalMetadata) ValidateMetadataByItself() bool {
	return true
}
