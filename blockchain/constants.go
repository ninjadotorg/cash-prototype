package blockchain

import "github.com/ninjadotorg/constant/common"

// constant for network
const (
	//Network fixed params
	ThresholdRatioOfDCBCrisis = 90
	ThresholdRatioOfGovCrisis = 90

	// Mainnet
	Mainnet               = 0x01
	MainetName            = "mainnet"
	MainnetDefaultPort    = "9333"
	MainnetInitFundSalary = 0
	MainnetInitDCBToken   = 0
	MainnetInitGovToken   = 0
	MainnetInitCmBToken   = 0
	MainnetInitBondToken  = 0
	MainnetVote
	MainnetGenesisblockPaymentAddress = "1UuyYcHgVFLMd8Qy7T1ZWRmfFvaEgogF7cEsqY98ubQjoQUy4VozTqyfSNjkjhjR85C6GKBmw1JKekgMwCeHtHex25XSKwzb9QPQ2g6a3"

	// Testnet
	Testnet                           = 0x02
	TestnetName                       = "testnet"
	TestnetDefaultPort                = "9444"
	TestnetInitFundSalary             = 1000000000000000
	TestnetInitDCBToken               = 10000
	TestnetInitGovToken               = 10000
	TestnetInitCmBToken               = 0
	TestnetInitBondToken              = 0
	TestnetGenesisBlockPaymentAddress = "1Uv12YEcd5w5Qm79sTGHSHYnCfVKM2ui8mbapD1dgziUf9211b5cnCSdxVb1DoXyDD19V1THMSnaAWZ18sJtmaVnh56wVhwb1HuYpkTa4"

	//board and proposal parameters
	NumberOfDCBGovernors = 50
	NumberOfGOVGovernors = 50
)

// board addresses
var (
	DCBAddress = []byte{}
	GOVAddress = []byte{}
)

// special token ids (aka. PropertyID in custom token)
var (
	DCBTokenID     = [common.HashSize]byte{1}
	GOVTokenID     = [common.HashSize]byte{2}
	CMBTokenID     = [common.HashSize]byte{3}
	BondTokenID    = [common.HashSize]byte{4}
	VoteDCBTokenID = [common.HashSize]byte{5}
	VoteGOVTokenID = [common.HashSize]byte{6}
)

const (
	// BlockVersion is the current latest supported block version.
	BlockVersion = 1
)

// global variables for genesis blok
var (
/*GENESIS_BLOCK_ANCHORS           = [][32]byte{[32]byte{}, [32]byte{}}
  GENESIS_BLOCK_NULLIFIERS        = []string{"88d35350b1846ecc34d6d04a10355ad9a8e1252e9d7f3af130186b4faf1a9832", "286b563fc45b7d5b9f929fb2c2766382a9126483d8d64c9b0197d049d4e89bf7"}
  GENESIS_BLOCK_COMMITMENTS       = []string{"d26356e6f726dfb4c0a395f3af134851139ce1c64cfed3becc3530c8c8ad5660", "5aaf71f995db014006d630dedf7ffcbfa8854055e6a8cc9ef153629e3045b7e1"}
  GENESIS_BLOCK_OUTPUT_R1         = [32]byte{1}
  GENESIS_BLOCK_OUTPUT_R2         = [32]byte{2}
  GENESIS_BLOCK_OUTPUT_R          = [][]byte{GENESIS_BLOCK_OUTPUT_R1[:], GENESIS_BLOCK_OUTPUT_R2[:]}
  GENESIS_BLOCK_SEED              = [32]byte{1}
  GENESIS_BLOCK_PHI               = [32]byte{1}
  GENESIS_BLOCK_JSPUBKEY          = "8a8ae7ff31597a4d87be0780a5c887c990c2965f454740dfc5b4177e900104c2"
  GENESIS_BLOCK_EPHEMERAL_PRIVKEY = [32]byte{1}
  GENESIS_BLOCK_EPHEMERAL_PUBKEY  = "2fe57da347cd62431528daac5fbb290730fff684afc4cfc2ed90995f58cb3b74"
  GENESIS_BLOCK_ENCRYPTED_DATA    = []string{"6a666d50427a6575436c546b50444b464b3442323567565436495962655442655374484777594d653157765f76686f56354b524d445f4b4975436b6a343764426a4f50784c304e642d666e756f6370775f42426248617946566f546955724a3730776554576851714235537039494a75714e6330654445656d614e653075634e57495a674d4b7667536c54474346446953625053563769687765655933536f4c62696754494e6253706444455f5231644c52336f49433655354a464d65656c7a543131485a33426573654b574c61416653504f786267", "416d354c584b4d7262745161722d5343536151424d4c625738334450665a526873455357666e646e3668696974474573663833473251497173614a316338664669714a6e6432566b59784b2d4d526d335a79353939356175396f704e6b6f4230683342756b795f527143634c57457465636f35366d4838796c396346526e5653656a2d4b6a496d327669593673396d44354f75687853586171536334584843774f4150544c5a432d34484a5f78384356326f386e7235543545493378746e6642"}
  GENESIS_BLOCK_G_A               = "001ffdd8f7bb3be19bc8c9f019d2aa247a5bd98328f3d074e2103f45c9cc99c085"
  GENESIS_BLOCK_G_APrime          = "0101fe18b5df1fa94d015f3d7d87062a20d3fd4f59698ded00085fa569c73ccf72"
  GENESIS_BLOCK_G_B               = "000430e3b82a36063b135b263fc9126f9de509dd70d46331ef36e029b109dbc73ac18118b8afbc410e668bd08c6babad5f59a040f65b63a150ddc418c9425397ca"
  GENESIS_BLOCK_G_BPrime          = "001a60d0574f8986568fce1664e00094a895964510d900b08b916e6d690df506e1"
  GENESIS_BLOCK_G_C               = "01099d85dac251c40be768687500df8e0c9530fd582715c6fc56fe4bf105a6b882"
  GENESIS_BLOCK_G_CPrime          = "0122e491fe746d601b2cb456c5b1fc49c0d26e4c408de37e0d59a99f3000a12380"
  GENESIS_BLOCK_G_K               = "0013d6c59db3015fd4bc2e10f4e3a8b686e69b66b3f836ad8ec3d529358d73f0f2"
  GENESIS_BLOCK_G_H               = "01213b662cd5e1bd881825534fff619a5e0486c30bfb1ebf8933d2228422cd1808"
  GENESIS_BLOCK_VMACS             = []string{"5264e44bd87cc4d555d57069f53990e9237c10d91f32e2b0c3e5ea54a9d4c7cb", "9f1187c5cf2e999904e43595ebcce0cfde7f022747b823e7134fd389c1e5a5ad"}*/
)
