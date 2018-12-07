package metadata

const (
	InvalidMeta = iota
	LoanRequestMeta
	LoanResponseMeta
	BuySellRequestMeta

	//Voting
	SubmitDCBProposalMeta
	VoteDCBProposalMeta
	VoteDCBBoardMeta
	AcceptDCBProposalMeta
	AcceptDCBBoardMeta

	SubmitGOVProposalMeta
	VoteGOVProposalMeta
	VoteGOVBoardMeta
	AcceptGOVProposalMeta
	AcceptGOVBoardMeta
)
