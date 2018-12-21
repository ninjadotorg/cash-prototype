package metadata

import (
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
)

//abstract class
type SealedDCBBallotMetadata struct {
	SealedBallot  []byte
	LockerPubKeys [][]byte
}

func NewSealedDCBBallotMetadata(sealedBallot []byte, lockerPubKeys [][]byte) *SealedDCBBallotMetadata {
	return &SealedDCBBallotMetadata{
		SealedBallot:  sealedBallot,
		LockerPubKeys: lockerPubKeys,
	}
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) Hash() *common.Hash {
	record := string(sealDCBBallotMetadata.SealedBallot)
	for _, i := range sealDCBBallotMetadata.LockerPubKeys {
		record += string(i)
	}
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	//Validate these pubKeys are in board
	dcbBoardPubKeys := bcr.GetDCBBoardPubKeys()
	for _, j := range sealDCBBallotMetadata.LockerPubKeys {
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
	for _, i := range sealDCBBallotMetadata.LockerPubKeys {
		if len(i) != common.PubKeyLength {
			return true, false, nil
		}
	}
	return true, true, nil
}

func (sealDCBBallotMetadata *SealedDCBBallotMetadata) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(sealDCBBallotMetadata.LockerPubKeys); index1++ {
		pub1 := sealDCBBallotMetadata.LockerPubKeys[index1]
		for index2 := index1 + 1; index2 < len(sealDCBBallotMetadata.LockerPubKeys); index2++ {
			pub2 := sealDCBBallotMetadata.LockerPubKeys[index2]
			if !common.ByteEqual(pub1, pub2) {
				return false
			}
		}
	}
	return true
}

type SealedLv1DCBBallotMetadata struct {
	SealedDCBBallotMetadata
	PointerToLv2Ballot common.Hash
	PointerToLv3Ballot common.Hash
	MetadataBase
}

func NewSealedLv1DCBBallotMetadata(sealedBallot []byte, lockersPubKey [][]byte, pointerToLv2Ballot common.Hash, pointerToLv3Ballot common.Hash) *SealedLv1DCBBallotMetadata {
	return &SealedLv1DCBBallotMetadata{
		SealedDCBBallotMetadata: *NewSealedDCBBallotMetadata(sealedBallot, lockersPubKey),
		PointerToLv2Ballot:      pointerToLv2Ballot,
		PointerToLv3Ballot:      pointerToLv3Ballot,
		MetadataBase:            *NewMetadataBase(SealedLv1DCBBallotMeta),
	}
}

func (sealedLv1DCBBallotMetadata *SealedLv1DCBBallotMetadata) Hash() *common.Hash {
	record := string(common.ToBytes(*sealedLv1DCBBallotMetadata.SealedDCBBallotMetadata.Hash()))
	record += string(common.ToBytes(sealedLv1DCBBallotMetadata.PointerToLv2Ballot))
	record += string(common.ToBytes(sealedLv1DCBBallotMetadata.PointerToLv3Ballot))
	record += string(sealedLv1DCBBallotMetadata.MetadataBase.Hash()[:])
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealedLv1DCBBallotMetadata *SealedLv1DCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	//Check base seal metadata
	ok, err := sealedLv1DCBBallotMetadata.SealedDCBBallotMetadata.ValidateTxWithBlockChain(tx, bcr, chainID, db)
	if err != nil || !ok {
		return ok, err
	}

	//Check precede transaction type
	_, _, _, lv2Tx, _ := bcr.GetTransactionByHash(&sealedLv1DCBBallotMetadata.PointerToLv2Ballot)
	if lv2Tx.GetMetadataType() != SealedLv2DCBBallotMeta {
		return false, nil
	}
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&sealedLv1DCBBallotMetadata.PointerToLv3Ballot)
	if lv3Tx.GetMetadataType() != SealedLv3DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv2 := lv2Tx.GetMetadata().(*SealedLv2DCBBallotMetadata)
	for i := 0; i < len(sealedLv1DCBBallotMetadata.LockerPubKeys); i++ {
		if !common.ByteEqual(sealedLv1DCBBallotMetadata.LockerPubKeys[i], metaLv2.LockerPubKeys[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(sealedLv1DCBBallotMetadata.SealedBallot, common.Encrypt(metaLv2.SealedBallot, metaLv2.LockerPubKeys[1]).([]byte)) {
		return false, nil
	}
	return true, nil
}

type SealedLv2DCBBallotMetadata struct {
	SealedDCBBallotMetadata
	PointerToLv3Ballot common.Hash
	MetadataBase
}

func NewSealedLv2DCBBallotMetadata(sealedBallot []byte, lockerPubKeys [][]byte, pointerToLv3Ballot common.Hash) *SealedLv2DCBBallotMetadata {
	return &SealedLv2DCBBallotMetadata{
		SealedDCBBallotMetadata: *NewSealedDCBBallotMetadata(sealedBallot, lockerPubKeys),
		PointerToLv3Ballot:      pointerToLv3Ballot,
		MetadataBase:            *NewMetadataBase(SealedLv2DCBBallotMeta),
	}
}

func (sealedLv2DCBBallotMetadata *SealedLv2DCBBallotMetadata) Hash() *common.Hash {
	record := string(common.ToBytes(*sealedLv2DCBBallotMetadata.SealedDCBBallotMetadata.Hash()))
	record += string(common.ToBytes(sealedLv2DCBBallotMetadata.PointerToLv3Ballot))
	record += string(sealedLv2DCBBallotMetadata.MetadataBase.Hash()[:])
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (sealedLv2DCBBallotMetadata *SealedLv2DCBBallotMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	//Check base seal metadata
	ok, err := sealedLv2DCBBallotMetadata.SealedDCBBallotMetadata.ValidateTxWithBlockChain(tx, bcr, chainID, db)
	if err != nil || !ok {
		return ok, err
	}

	//Check precede transaction type
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&sealedLv2DCBBallotMetadata.PointerToLv3Ballot)
	if lv3Tx.GetMetadataType() != SealedLv3DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv3 := lv3Tx.GetMetadata().(*SealedLv3DCBBallotMetadata)
	for i := 0; i < len(sealedLv2DCBBallotMetadata.LockerPubKeys); i++ {
		if !common.ByteEqual(sealedLv2DCBBallotMetadata.LockerPubKeys[i], metaLv3.LockerPubKeys[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(sealedLv2DCBBallotMetadata.SealedBallot, common.Encrypt(metaLv3.SealedBallot, metaLv3.LockerPubKeys[2]).([]byte)) {
		return false, nil
	}
	return true, nil
}

type SealedLv3DCBBallotMetadata struct {
	SealedDCBBallotMetadata
	MetadataBase
}

func NewSealedLv3DCBBallotMetadata(sealedBallot []byte, lockerPubKeys [][]byte) *SealedLv3DCBBallotMetadata {
	return &SealedLv3DCBBallotMetadata{
		SealedDCBBallotMetadata: *NewSealedDCBBallotMetadata(sealedBallot, lockerPubKeys),
		MetadataBase:            *NewMetadataBase(SealedLv3DCBBallotMeta),
	}
}

type NormalDCBBallotFromSealerMetadata struct {
	Ballot             []byte
	LockerPubKey       [][]byte
	PointerToLv1Ballot common.Hash
	PointerToLv3Ballot common.Hash
	MetadataBase
}

func NewNormalDCBBallotFromSealerMetadata(ballot []byte, lockerPubKey [][]byte, pointerToLv1Ballot common.Hash, pointerToLv3Ballot common.Hash) *NormalDCBBallotFromSealerMetadata {
	return &NormalDCBBallotFromSealerMetadata{
		Ballot:             ballot,
		LockerPubKey:       lockerPubKey,
		PointerToLv1Ballot: pointerToLv1Ballot,
		PointerToLv3Ballot: pointerToLv3Ballot,
		MetadataBase:       *NewMetadataBase(NormalDCBBallotMetaFromSealerMeta),
	}
}

func (normalDCBBallotFromSealerMetadata *NormalDCBBallotFromSealerMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	for _, i := range normalDCBBallotFromSealerMetadata.LockerPubKey {
		if len(i) != common.PubKeyLength {
			return true, false, nil
		}
	}
	return true, true, nil
}

func (normalDCBBallotFromSealerMetadata *NormalDCBBallotFromSealerMetadata) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(normalDCBBallotFromSealerMetadata.LockerPubKey); index1++ {
		pub1 := normalDCBBallotFromSealerMetadata.LockerPubKey[index1]
		for index2 := index1 + 1; index2 < len(normalDCBBallotFromSealerMetadata.LockerPubKey); index2++ {
			pub2 := normalDCBBallotFromSealerMetadata.LockerPubKey[index2]
			if !common.ByteEqual(pub1, pub2) {
				return false
			}
		}
	}
	return true
}

func (normalDCBBallotFromSealerMetadata *NormalDCBBallotFromSealerMetadata) Hash() *common.Hash {
	record := string(normalDCBBallotFromSealerMetadata.Ballot)
	for _, i := range normalDCBBallotFromSealerMetadata.LockerPubKey {
		record += string(i)
	}
	record += string(common.ToBytes(normalDCBBallotFromSealerMetadata.PointerToLv1Ballot))
	record += string(common.ToBytes(normalDCBBallotFromSealerMetadata.PointerToLv3Ballot))
	record += string(normalDCBBallotFromSealerMetadata.MetadataBase.Hash()[:])
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (normalDCBBallotFromSealerMetadata *NormalDCBBallotFromSealerMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	//Validate these pubKeys are in board
	dcbBoardPubKeys := bcr.GetDCBBoardPubKeys()
	for _, j := range normalDCBBallotFromSealerMetadata.LockerPubKey {
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
	_, _, _, lv1Tx, _ := bcr.GetTransactionByHash(&normalDCBBallotFromSealerMetadata.PointerToLv1Ballot)
	if lv1Tx.GetMetadataType() != SealedLv1DCBBallotMeta {
		return false, nil
	}
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&normalDCBBallotFromSealerMetadata.PointerToLv3Ballot)
	if lv3Tx.GetMetadataType() != SealedLv3DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv1 := lv1Tx.GetMetadata().(*SealedLv1DCBBallotMetadata)
	for i := 0; i < len(normalDCBBallotFromSealerMetadata.LockerPubKey); i++ {
		if !common.ByteEqual(normalDCBBallotFromSealerMetadata.LockerPubKey[i], metaLv1.LockerPubKeys[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(normalDCBBallotFromSealerMetadata.Ballot, common.Encrypt(metaLv1.SealedBallot, metaLv1.LockerPubKeys[0]).([]byte)) {
		return false, nil
	}
	return true, nil
}

type NormalDCBBallotFromOwnerMetadata struct {
	Ballot             []byte
	LockerPubKey       [][]byte
	PointerToLv3Ballot common.Hash
	MetadataBase
}

func NewNormalDCBBallotFromOwnerMetadata(ballot []byte, lockerPubKey [][]byte, pointerToLv3Ballot common.Hash) *NormalDCBBallotFromOwnerMetadata {
	return &NormalDCBBallotFromOwnerMetadata{
		Ballot:             ballot,
		LockerPubKey:       lockerPubKey,
		PointerToLv3Ballot: pointerToLv3Ballot,
		MetadataBase:       *NewMetadataBase(NormalDCBBallotMetaFromOwnerMeta),
	}
}

func (normalDCBBallotFromOwnerMetadata *NormalDCBBallotFromOwnerMetadata) Hash() *common.Hash {
	record := string(normalDCBBallotFromOwnerMetadata.Ballot)
	for _, i := range normalDCBBallotFromOwnerMetadata.LockerPubKey {
		record += string(i)
	}
	record += string(common.ToBytes(normalDCBBallotFromOwnerMetadata.PointerToLv3Ballot))
	record += string(normalDCBBallotFromOwnerMetadata.MetadataBase.Hash()[:])
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

func (normalDCBBallotFromOwnerMetadata *NormalDCBBallotFromOwnerMetadata) ValidateTxWithBlockChain(tx Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	//Validate these pubKeys are in board
	dcbBoardPubKeys := bcr.GetDCBBoardPubKeys()
	for _, j := range normalDCBBallotFromOwnerMetadata.LockerPubKey {
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
	_, _, _, lv3Tx, _ := bcr.GetTransactionByHash(&normalDCBBallotFromOwnerMetadata.PointerToLv3Ballot)
	if lv3Tx.GetMetadataType() != SealedLv3DCBBallotMeta {
		return false, nil
	}

	// check 2 array equal
	metaLv3 := lv3Tx.GetMetadata().(*SealedLv3DCBBallotMetadata)
	for i := 0; i < len(normalDCBBallotFromOwnerMetadata.LockerPubKey); i++ {
		if !common.ByteEqual(normalDCBBallotFromOwnerMetadata.LockerPubKey[i], metaLv3.LockerPubKeys[i]) {
			return false, nil
		}
	}

	// Check encrypting
	if !common.ByteEqual(
		metaLv3.SealedBallot,
		common.Encrypt(
			common.Encrypt(
				common.Encrypt(
					normalDCBBallotFromOwnerMetadata.Ballot,
					metaLv3.LockerPubKeys[2],
				),
				metaLv3.LockerPubKeys[1],
			),
			metaLv3.LockerPubKeys[0],
		).([]byte)) {
		return false, nil
	}
	return true, nil
}

func (normalDCBBallotFromOwnerMetadata *NormalDCBBallotFromOwnerMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	for _, i := range normalDCBBallotFromOwnerMetadata.LockerPubKey {
		if len(i) != common.PubKeyLength {
			return true, false, nil
		}
	}
	return true, true, nil
}

func (normalDCBBallotFromOwnerMetadata *NormalDCBBallotFromOwnerMetadata) ValidateMetadataByItself() bool {
	for index1 := 0; index1 < len(normalDCBBallotFromOwnerMetadata.LockerPubKey); index1++ {
		pub1 := normalDCBBallotFromOwnerMetadata.LockerPubKey[index1]
		for index2 := index1 + 1; index2 < len(normalDCBBallotFromOwnerMetadata.LockerPubKey); index2++ {
			pub2 := normalDCBBallotFromOwnerMetadata.LockerPubKey[index2]
			if !common.ByteEqual(pub1, pub2) {
				return false
			}
		}
	}
	return true
}

type PunishDCBDecryptMetadata struct {
	pubKey []byte
	MetadataBase
}

func NewPunishDCBDecryptMetadata(pubKey []byte) *PunishDCBDecryptMetadata {
	return &PunishDCBDecryptMetadata{
		pubKey:       pubKey,
		MetadataBase: *NewMetadataBase(PunishDCBDecryptMeta),
	}
}

func (punishDCBDecryptMetadata *PunishDCBDecryptMetadata) Hash() *common.Hash {
	record := string(punishDCBDecryptMetadata.pubKey)
	record += string(punishDCBDecryptMetadata.MetadataBase.Hash()[:])
	hash := common.DoubleHashH([]byte(record))
	return &hash
}

//todo @0xjackalope validate within blockchain and current block

func (punishDCBDecryptMetadata *PunishDCBDecryptMetadata) ValidateTxWithBlockChain(Transaction, BlockchainRetriever, byte, database.DatabaseInterface) (bool, error) {
	return true, nil
}

func (punishDCBDecryptMetadata *PunishDCBDecryptMetadata) ValidateSanityData(BlockchainRetriever, Transaction) (bool, bool, error) {
	if len(punishDCBDecryptMetadata.pubKey) != common.PubKeyLength {
		return true, false, nil
	}
	return true, true, nil
}

func (punishDCBDecryptMetadata *PunishDCBDecryptMetadata) ValidateMetadataByItself() bool {
	return true
}
