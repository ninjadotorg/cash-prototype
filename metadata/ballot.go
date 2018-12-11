package metadata

import (
	"github.com/ninjadotorg/constant/common"
)

//abtract class
type SealedDCBBallotMetadata struct {
	SealedBallot []byte
	LockerPubkey [][]byte

	MetadataBase
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) Hash() *common.Hash {
	record := string(sealDCBBallotMetadata.SealedBallot)
	for _, i := range sealDCBBallotMetadata.LockerPubkey {
		record += string(i)
	}
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte) (bool, error) {
	//Validate these pubkeys are in board
	dcbBoardPubKeys := bcr.GetDCBBoardPubKeys()
	for _, j := range sealDCBBallotMetadata.LockerPubkey {
		exist := false
		for _, i := range dcbBoardPubKeys {
			if common.ByteEqual(i, j) {
				exist = true
				break
			}
		}
		if !exist {
			return false, nil
		}
	}
	return true, nil
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	for _, i := range sealDCBBallotMetadata.LockerPubkey {
		if len(i) != common.HashSize {
			return true, false, nil
		}
	}
	return true, true, nil
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(sealDCBBallotMetadata.LockerPubkey); index1++ {
		pub1 := sealDCBBallotMetadata.LockerPubkey[index1]
		for index2 := index1 + 1; index2 < len(sealDCBBallotMetadata.LockerPubkey); index2++ {
			pub2 := sealDCBBallotMetadata.LockerPubkey[index2]
			if !common.ByteEqual(pub1, pub2) {
				return false
			}
		}
	}
	return true
}

type SealedLv1DCBBallotMetadata struct {
	SealedDCBBallotMetadata
	PointerToLv2Ballot *common.Hash
}

func (sealedLv1DCBBallotMetadata *SealedLv1DCBBallotMetadata) Hash() *common.Hash {
	record := string(common.ToBytes(sealedLv1DCBBallotMetadata.SealedDCBBallotMetadata.Hash()))
	record += string(common.ToBytes(sealedLv1DCBBallotMetadata.PointerToLv2Ballot))
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealedLv1DCBBallotMetadata *SealedLv1DCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte) (bool, error) {
	//Check base seal metadata
	ok, err := sealedLv1DCBBallotMetadata.SealedDCBBallotMetadata.ValidateTxWithBlockChain(tx, bcr, chainID)
	if err != nil || !ok {
		return ok, err
	}

	//Check precede transaction type
	_, _, _, lv2Tx, _ := bcr.GetTransactionByHash(sealedLv1DCBBallotMetadata.PointerToLv2Ballot)
	if lv2Tx.GetMetadataType() != SealedLv2DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv2 := lv2Tx.GetMetadata().(*SealedLv2DCBBallotMetadata)
	for i := 0; i < len(sealedLv1DCBBallotMetadata.LockerPubkey); i++ {
		if !common.ByteEqual(sealedLv1DCBBallotMetadata.LockerPubkey[i], metaLv2.LockerPubkey[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(sealedLv1DCBBallotMetadata.SealedBallot, common.Encrypt(metaLv2.SealedBallot, metaLv2.LockerPubkey[1]).([]byte)) {
		return false, nil
	}
	return true, nil
}

type SealedLv2DCBBallotMetadata struct {
	SealedDCBBallotMetadata
	PointerToLv3Ballot *common.Hash
}

func (sealedLv2DCBBallotMetadata *SealedLv2DCBBallotMetadata) Hash() *common.Hash {
	record := string(common.ToBytes(sealedLv2DCBBallotMetadata.SealedDCBBallotMetadata.Hash()))
	record += string(common.ToBytes(sealedLv2DCBBallotMetadata.PointerToLv3Ballot))
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealedLv2DCBBallotMetadata *SealedLv2DCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte) (bool, error) {
	//Check base seal metadata
	ok, err := sealedLv2DCBBallotMetadata.SealedDCBBallotMetadata.ValidateTxWithBlockChain(tx, bcr, chainID)
	if err != nil || !ok {
		return ok, err
	}

	//Check precede transaction type
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(sealedLv2DCBBallotMetadata.PointerToLv3Ballot)
	if lv3Tx.GetMetadataType() != SealedLv3DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv3 := lv3Tx.GetMetadata().(*SealedLv3DCBBallotMetadata)
	for i := 0; i < len(sealedLv2DCBBallotMetadata.LockerPubkey); i++ {
		if !common.ByteEqual(sealedLv2DCBBallotMetadata.LockerPubkey[i], metaLv3.LockerPubkey[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(sealedLv2DCBBallotMetadata.SealedBallot, common.Encrypt(metaLv3.SealedBallot, metaLv3.LockerPubkey[2]).([]byte)) {
		return false, nil
	}
	return true, nil
}

type SealedLv3DCBBallotMetadata struct {
	SealedDCBBallotMetadata
}

func NewSealedLv3DCBBallotMetadata(data map[string]interface{}) *SealedLv3DCBBallotMetadata {
	return &SealedLv3DCBBallotMetadata{
		SealedDCBBallotMetadata: SealedDCBBallotMetadata{
			SealedBallot: data["SealedBallot"].([]byte),
			LockerPubkey: data["LockerPubKey"].([][]byte),
			MetadataBase: MetadataBase{
				Type: SealedLv3DCBBallotMeta,
			},
		},
	}
}

type NormalDCBBallotMetadata struct {
	Ballot             []byte
	LockerPubkey       [][]byte
	PointerToLv1Ballot *common.Hash
}

func (normalDCBBallotMetadata *NormalDCBBallotMetadata) Hash() *common.Hash {
	record := string(normalDCBBallotMetadata.Ballot)
	for _, i := range normalDCBBallotMetadata.LockerPubkey {
		record += string(i)
	}
	record += string(common.ToBytes(normalDCBBallotMetadata.PointerToLv1Ballot))
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (normalDCBBallotMetadata *NormalDCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte) (bool, error) {
	//Validate these pubkeys are in board
	dcbBoardPubKeys := bcr.GetDCBBoardPubKeys()
	for _, j := range normalDCBBallotMetadata.LockerPubkey {
		exist := false
		for _, i := range dcbBoardPubKeys {
			if common.ByteEqual(i, j) {
				exist = true
				break
			}
		}
		if !exist {
			return false, nil
		}
	}

	//Check precede transaction type
	_, _, _, lv1Tx, _ := bcr.GetTransactionByHash(normalDCBBallotMetadata.PointerToLv1Ballot)
	if lv1Tx.GetMetadataType() != SealedLv1DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv1 := lv1Tx.GetMetadata().(*SealedLv1DCBBallotMetadata)
	for i := 0; i < len(normalDCBBallotMetadata.LockerPubkey); i++ {
		if !common.ByteEqual(normalDCBBallotMetadata.LockerPubkey[i], metaLv1.LockerPubkey[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(normalDCBBallotMetadata.Ballot, common.Encrypt(metaLv1.SealedBallot, metaLv1.LockerPubkey[0]).([]byte)) {
		return false, nil
	}
	return true, nil
}
