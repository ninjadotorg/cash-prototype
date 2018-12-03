package transaction

type TxAcceptDCBBoard struct {
	*Tx
	DCBBoardPubKeys     []string
	StartAmountDCBToken uint64
}

type TxAcceptGOVBoard struct {
	*Tx
	GOVBoardPubKeys     []string
	StartAmountGOVToken uint64
}
