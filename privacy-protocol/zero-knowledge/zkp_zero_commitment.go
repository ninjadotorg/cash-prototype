package zkp

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ninjadotorg/constant/privacy-protocol"
)

/*Protocol for opening a Commitment to 0
Prove:
	commitmentValue is Commitment value of Zero, that is statement needed to prove
	commitmentValue is calculated by Comm_ck(VoteAmount,PRDNumber)
	commitmentRnd is PRDNumber, which is used to calculate commitmentValue
	s <- Zp; P is privacy.Curve base point's order, is N
	B <- Comm_ck(0,s);  Comm_ck is PedersenCommit function using public params - privacy.Curve.Params() (G0,G1...)
						but is just commit special value (in this case, special value is 0),
						which is stick with G[Index] (in this case, Index is the Index stick with commitmentValue)
						B is a.k.a commitmentZeroS
	x <- Hash(G0||G1||G2||G3||commitmentvalue) x is pseudorandom number, which could be computed easily by Verifier
	z <- rx + s; z in Zp, r is commitmentRnd
	return commitmentZeroS, z

Verify:
	commitmentValue is Commitment value of Zero, that is statement needed to prove
	commitmentValue is calculated by Comm_ck(VoteAmount,PRDNumber), a.k.a A
	commitmentZeroS, z are output of Prove function, commitmentZeroS is a.k.a B
	x <- Hash(G0||G1||G2||G3||commitmentvalue)
	boolValue <- (Comm_ck(0,z) == A.x + B); in this case, A and B needed to convert to privacy.EllipticPoint
	return boolValue
)
*/

//ProveIsZero generate a Proof prove that the Commitment is zero
func ProveIsZero(commitmentValue, commitmentRnd []byte, index byte) ([]byte, *big.Int) {
	//var x big.Int
	//s is a random number in Zp, with p is N, which is order of base point of privacy.Curve
	sRnd, err := rand.Int(rand.Reader, privacy.Curve.Params().N)
	if err != nil {
		panic(err)
	}

	//Generate zero number to commit
	zeroInt := big.NewInt(0)

	//Calculate B = commitmentZeroS = comm_ck(0,s,Index)
	commitmentZeroS := privacy.Elcm.CommitSpecValue(zeroInt.Bytes(), sRnd.Bytes(), index)

	//Generate random x in Zp
	xRnd := big.NewInt(0)
	xRnd.SetBytes(privacy.Elcm.GetHashOfValues([][]byte{commitmentValue}))
	xRnd.Mod(xRnd, privacy.Curve.Params().N)

	//Calculate z=r*x + s (mod N)
	z := big.NewInt(0)
	z.SetBytes(commitmentRnd)
	z.Mul(z, xRnd)
	z.Mod(z, privacy.Curve.Params().N)
	z.Add(z, sRnd)
	z.Mod(z, privacy.Curve.Params().N)

	//return B, z
	return commitmentZeroS, z
}

//VerifyIsZero verify that under Commitment is zero
func VerifyIsZero(commitmentValue, commitmentZeroS []byte, index byte, z *big.Int) bool {
	//Calculate x
	xRnd := big.NewInt(0)
	xRnd.SetBytes(privacy.Elcm.GetHashOfValues([][]byte{commitmentValue}))
	xRnd.Mod(xRnd, privacy.Curve.Params().N)

	//convert commitmentValue []byte to Point in ECC
	commitmentValuePoint, err := privacy.DecompressKey(commitmentValue)
	if err != nil {
		return false
	}
	if (!privacy.Curve.IsOnCurve(commitmentValuePoint.X, commitmentValuePoint.Y)) || (z.Cmp(privacy.Curve.Params().N) > -1) {
		return false
	}

	//convert commitmentZeroS (a.k.a B) to Point in ECC
	commitmentZeroSPoint, err := privacy.DecompressCommitment(commitmentZeroS)
	if err != nil {
		return false
	}
	if (!privacy.Curve.IsOnCurve(commitmentZeroSPoint.X, commitmentZeroSPoint.Y)) || (z.Cmp(privacy.Curve.Params().N) > -1) {
		return false
	}

	//verifyPoint is result of A.x + B (in ECC)
	verifyPoint := new(privacy.EllipticPoint)
	verifyPoint.X = big.NewInt(0)
	verifyPoint.Y = big.NewInt(0)
	//Set verifyPoint = A
	verifyPoint.X.SetBytes(commitmentValuePoint.X.Bytes())
	verifyPoint.Y.SetBytes(commitmentValuePoint.Y.Bytes())
	//verifyPoint = verifyPoint.x
	verifyPoint.X, verifyPoint.Y = privacy.Curve.ScalarMult(verifyPoint.X, verifyPoint.Y, xRnd.Bytes())
	//verifyPoint = verifyPoint + B
	verifyPoint.X, verifyPoint.Y = privacy.Curve.Add(verifyPoint.X, verifyPoint.Y, commitmentZeroSPoint.X, commitmentZeroSPoint.Y)

	//Generate Zero number
	zeroInt := big.NewInt(0)

	//Calculate comm_ck(0,z, Index)
	commitmentZeroZ := privacy.Elcm.CommitSpecValue(zeroInt.Bytes(), z.Bytes(), index)

	//convert result to point
	commitmentZeroZPoint, err := privacy.DecompressCommitment(commitmentZeroZ)
	if err != nil {
		return false
	}
	if (!privacy.Curve.IsOnCurve(commitmentZeroZPoint.X, commitmentZeroZPoint.Y)) || (z.Cmp(privacy.Curve.Params().N) > -1) {
		return false
	}

	if commitmentZeroZPoint.X.CmpAbs(verifyPoint.X) != 0 {
		return false
	}
	if commitmentZeroZPoint.Y.CmpAbs(verifyPoint.Y) != 0 {
		return false
	}

	return true
}

//TestProofIsZero test prove and verify function
func TestProofIsZero() bool {
	//Generate a random Commitment

	//First, generate random value to commit and calculate two Commitment with different PRDNumber
	//Random value
	serialNumber := privacy.RandBytes(32)

	//Random two PRDNumber in Zp
	r1Int := big.NewInt(0)
	r2Int := big.NewInt(0)
	r1 := privacy.RandBytes(32)
	r2 := privacy.RandBytes(32)
	r1Int.SetBytes(r1)
	r2Int.SetBytes(r2)
	r1Int.Mod(r1Int, privacy.Curve.Params().N)
	r2Int.Mod(r2Int, privacy.Curve.Params().N)
	r1 = r1Int.Bytes()
	r2 = r2Int.Bytes()

	//Calculate two Pedersen Commitment
	committemp1 := privacy.Elcm.CommitSpecValue(serialNumber, r1, 0)
	committemp2 := privacy.Elcm.CommitSpecValue(serialNumber, r2, 0)

	//Converting them to ECC Point
	committemp1Point, err := privacy.DecompressCommitment(committemp1)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	committemp2Point, err := privacy.DecompressCommitment(committemp2)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	//Compute inverse of commitment2 cuz we wanna calculate A1 + A2^-1 in ECC
	//Inverse of A(x,y) in ECC is A'(x,P-y) with P is order of field
	inverse_committemp2Point := new(privacy.EllipticPoint)
	inverse_committemp2Point.X = big.NewInt(0)
	inverse_committemp2Point.Y = big.NewInt(0)
	inverse_committemp2Point.X.SetBytes(committemp2Point.X.Bytes())
	inverse_committemp2Point.Y.SetBytes(committemp2Point.Y.Bytes())
	inverse_committemp2Point.Y.Sub(privacy.Curve.Params().P, committemp2Point.Y)

	//So, when we have A1+A2^-1, we need compute r = r1 - r2 (mod N), which is r of zero Commitment
	rInt := big.NewInt(0)
	rInt.Sub(r1Int, r2Int)
	rInt.Mod(rInt, privacy.Curve.Params().N)

	//Convert result of A1 + A2^-1 to ECC Point
	resPoint := privacy.EllipticPoint{big.NewInt(0), big.NewInt(0)}
	resPoint.X, resPoint.Y = privacy.Curve.Add(committemp1Point.X, committemp1Point.Y, inverse_committemp2Point.X, inverse_committemp2Point.Y)

	//Convert it to byte array
	commitZero := resPoint.CompressPoint()

	//Compute Proof
	proofZero, z := ProveIsZero(commitZero, rInt.Bytes(), 0)

	//verify Proof
	boolValue := VerifyIsZero(commitZero, proofZero, 0, z)
	fmt.Println("Test ProofIsZero resulit: ", boolValue)
	return boolValue
}
