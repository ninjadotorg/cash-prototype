package common

const (
	EmptyString         = ""
	MiliConstant        = 3 // 1 constant = 10^3 mili constant, we will use 1 miliconstant as minimum unit constant in tx
	IncMerkleTreeHeight = 29
	RefundPeriod        = 1000 // after 1000 blocks since a tx (small & no-privacy) happens, the network will refund an amount of constants to tx initiator automatically
)

const (
	TxSubmitDCBProposal = "pd"  // submit DCB proposal tx
	TxSubmitGOVProposal = "pg"  // submit GOV proposal tx
	TxVoteDCBProposal   = "vd"  // submit DCB proposal voted tx
	TxVoteGOVProposal   = "vg"  // submit GOV proposal voted tx
	TxVoteDCBBoard      = "vbd" // vote DCB board tx
	TxVoteGOVBoard      = "vbg" // vote DCB board tx
	TxAcceptDCBBoard    = "adb" //accept new DCB board
	TxAcceptGOVBoard    = "agb" //accept new GOV board
	TxAcceptDCBProposal = "ad"  // accept DCB proposal
	TxAcceptGOVProposal = "ag"  // accept GOV proposal

	TxNormalType         = "n" // normal tx(send and receive coin)
	TxSalaryType         = "s" // salary tx(gov pay salary for block producer)
	TxCustomTokenType    = "t" // token  tx
	TxLoanRequest        = "lr"
	TxLoanResponse       = "ls"
	TxLoanPayment        = "lp"
	TxLoanWithdraw       = "lw"
	TxDividendPayout     = "td"
	TxCrowdsale          = "cs"
	TxBuyFromGOVRequest  = "bgr"
	TxBuySellDCBRequest  = "bsdr"
	TxBuySellDCBResponse = "bsdrs"
	TxBuyFromGOVResponse = "bgrs"
	TxBuyBackRequest     = "bbr"
	TxBuyBackResponse    = "bbrs"
)

// for mining consensus
const (
	DurationOfTermDCB     = 1000    //number of block one DCB board in charge
	DurationOfTermGOV     = 1000    //number of block one GOV board in charge
	MaxBlockSize          = 5000000 //byte 5MB
	MaxTxsInBlock         = 1000
	MinTxsInBlock         = 10                    // minium txs for block to get immediate process (meaning no wait time)
	MinBlockWaitTime      = 3                     // second
	MaxBlockWaitTime      = 20 - MinBlockWaitTime // second
	MaxSyncChainTime      = 5                     // second
	MaxBlockSigWaitTime   = 5                     // second
	MaxBlockPerTurn       = 100                   // maximum blocks that a validator can create per turn
	TotalValidators       = 20                    // = TOTAL CHAINS
	MinBlockSigs          = (TotalValidators / 2) + 1
	GetChainStateInterval = 10 //second
	MaxBlockTime          = 10 //second Maximum for a chain to grow

	// voting
	SumOfVoteDCBToken = 100000000
	SumOfVoteGOVToken = 100000000
)

// board types
const (
	DCB = 1
	GOV = 2
)
