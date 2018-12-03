package zkp

import (
	"fmt"
	"math/big"

	"github.com/ninjadotorg/constant/privacy-protocol"
)

// PKComZeroOneProtocol is a protocol for Zero-knowledge Proof of Knowledge of committed zero or one
// include Witness: CommitedValue, r []byte
type PKComZeroOneProtocol struct {
	Witness PKComZeroOneWitness
	Proof   PKComZeroOneProof
}

// PKComZeroOneProof contains Proof's value
type PKComZeroOneProof struct {
	ca, cb     []byte // 34 bytes
	f, za, zb  []byte //32 bytes
	commitment []byte // 34 bytes
	index      byte
}

// PKComZeroOneProof contains Proof's value
type PKComZeroOneWitness struct {
	Commitment    []byte
	CommitedValue []byte
	Rand          []byte
	Index         byte
}

// SetWitness sets Witness
func (pro *PKComZeroOneProtocol) SetWitness(witness PKComZeroOneWitness) {
	pro.Witness = witness
}

// SetWitness sets Witness
func (pro *PKComZeroOneProtocol) SetProof(proof PKComZeroOneProof) {
	pro.Proof = proof
}

func getindex(bigint *big.Int) int {
	return 32 - len(bigint.Bytes())
}

// Prove creates a Proof for Commitment to zero or one
func (pro *PKComZeroOneProtocol) Prove() (*PKComZeroOneProof, error) {
	// Check Index
	if pro.Witness.Index < 0 || pro.Witness.Index > 2 {
		return nil, fmt.Errorf("Index must be between 0 and 2")
	}

	// Check whether commited value is zero or one or not
	commitedValue := big.NewInt(0)
	commitedValue.SetBytes(pro.Witness.CommitedValue)

	if commitedValue.Cmp(big.NewInt(0)) != 0 && commitedValue.Cmp(big.NewInt(0)) != 1 {
		return nil, fmt.Errorf("commited value must be zero or one")
	}

	proof := new(PKComZeroOneProof)

	// Generate random numbers
	a := privacy.RandBytes(32)
	aInt := new(big.Int).SetBytes(a)
	aInt.Mod(aInt, privacy.Curve.Params().N)

	s := privacy.RandBytes(32)
	sInt := new(big.Int).SetBytes(s)
	sInt.Mod(sInt, privacy.Curve.Params().N)

	t := privacy.RandBytes(32)
	tInt := new(big.Int).SetBytes(t)
	tInt.Mod(tInt, privacy.Curve.Params().N)

	// Calculate ca, cb
	proof.ca = make([]byte, 34)
	proof.ca = privacy.Elcm.CommitSpecValue(aInt.Bytes(), sInt.Bytes(), pro.Witness.Index)

	am := big.NewInt(0)
	am.Mul(aInt, commitedValue)
	proof.cb = make([]byte, 34)
	proof.cb = privacy.Elcm.CommitSpecValue(am.Bytes(), tInt.Bytes(), pro.Witness.Index)

	// Calculate x = hash (G0||G1||G2||G3||ca||cb||cm)
	x := big.NewInt(0)
	x.SetBytes(privacy.Elcm.GetHashOfValues([][]byte{proof.ca, proof.cb, pro.Witness.Commitment}))
	x.Mod(x, privacy.Curve.Params().N)

	// Calculate f = mx + a
	f := big.NewInt(0)
	f.Mul(commitedValue, x)
	f.Add(f, aInt)
	f.Mod(f, privacy.Curve.Params().N)
	proof.f = f.Bytes()

	// Calculate za = rx + s
	za := big.NewInt(1)
	za.Mul(new(big.Int).SetBytes(pro.Witness.Rand), x)
	za.Add(za, sInt)
	za.Mod(za, privacy.Curve.Params().N)
	proof.za = za.Bytes()

	// Calculate zb = r(x-f) + t
	zb := big.NewInt(1)
	xSubF := new(big.Int).Sub(x, f)
	xSubF.Mod(xSubF, privacy.Curve.Params().N)
	zb.Mul(new(big.Int).SetBytes(pro.Witness.Rand), xSubF)
	zb.Add(zb, tInt)
	zb.Mod(zb, privacy.Curve.Params().N)
	proof.zb = zb.Bytes()

	xm := big.NewInt(1)
	xm.Mul(x, new(big.Int).SetBytes(pro.Witness.CommitedValue))
	point := privacy.EllipticPoint{big.NewInt(0), big.NewInt(0)}
	point.X, point.Y = privacy.Elcm.G[pro.Witness.Index].X, privacy.Elcm.G[pro.Witness.Index].Y
	point.X, point.Y = privacy.Curve.ScalarMult(point.X, point.Y, xm.Bytes())

	xr := big.NewInt(1)
	xr.Mul(x, new(big.Int).SetBytes(pro.Witness.Rand))
	point2 := privacy.EllipticPoint{big.NewInt(0), big.NewInt(0)}
	point2.X, point2.Y = privacy.Elcm.G[privacy.CM_CAPACITY-1].X, privacy.Elcm.G[privacy.CM_CAPACITY-1].Y
	point2.X, point2.Y = privacy.Curve.ScalarMult(point2.X, point2.Y, xr.Bytes())

	point.X, point.Y = privacy.Curve.Add(point.X, point.Y, point2.X, point2.Y)

	aG := privacy.Elcm.G[pro.Witness.Index]
	aG.X, aG.Y = privacy.Curve.ScalarMult(aG.X, aG.Y, aInt.Bytes())
	sH := privacy.Elcm.G[privacy.CM_CAPACITY-1]
	sH.X, sH.Y = privacy.Curve.ScalarMult(sH.X, sH.Y, sInt.Bytes())
	aG.X, aG.Y = privacy.Curve.Add(aG.X, aG.Y, sH.X, sH.Y)
	aG.X, aG.Y = privacy.Curve.Add(aG.X, aG.Y, point.X, point.Y)

	proof.commitment = pro.Witness.Commitment
	proof.index = pro.Witness.Index

	return proof, nil
}

// Verify verifies the Proof for Commitment to zero or one
func (pro *PKComZeroOneProtocol) Verify() bool {
	//Decompress Commitment  value
	comPoint, err := privacy.DecompressCommitment(pro.Proof.commitment)
	if err != nil {
		fmt.Printf("Can not decompress Commitment value to ECC point")
		return false
	}

	// Calculate x = hash (G0||G1||G2||G3||ca||cb||cm)
	x := big.NewInt(0)
	x.SetBytes(privacy.Elcm.GetHashOfValues([][]byte{pro.Proof.ca, pro.Proof.cb, pro.Proof.commitment}))
	x.Mod(x, privacy.Curve.Params().N)

	// Decompress ca, cb of Proof
	caPoint, err := privacy.DecompressCommitment(pro.Proof.ca)
	if err != nil {
		fmt.Printf("Can not decompress Proof ca to ECC point")
		return false
	}
	cbPoint, err := privacy.DecompressCommitment(pro.Proof.cb)
	fmt.Printf("cb Point verify: %+v\n", cbPoint)
	if err != nil {
		fmt.Printf("Can not decompress Proof cb to ECC point")
		return false
	}

	// Calculate leftPoint1 = c^x * ca
	leftPoint1 := privacy.EllipticPoint{big.NewInt(0), big.NewInt(0)}
	leftPoint1.X, leftPoint1.Y = privacy.Curve.ScalarMult(comPoint.X, comPoint.Y, x.Bytes())
	leftPoint1.X, leftPoint1.Y = privacy.Curve.Add(leftPoint1.X, leftPoint1.Y, caPoint.X, caPoint.Y)

	// Calculate rightPoint1 = Com(f, za)
	rightValue1 := privacy.Elcm.CommitSpecValue(pro.Proof.f, pro.Proof.za, pro.Proof.index)

	rightPoint1, err := privacy.DecompressCommitment(rightValue1)
	fmt.Printf("Method 1\n")
	fmt.Printf("left point 1 X: %v\n", rightPoint1.X)
	fmt.Printf("right point 1 X: %v\n", rightPoint1.Y)
	if err != nil {
		fmt.Printf("Can not decompress comitment for f")
		return false
	}

	// Calculate leftPoint2 = c^(x-f) * cb
	leftPoint2 := privacy.EllipticPoint{big.NewInt(0), big.NewInt(0)}
	xSubF := new(big.Int)
	xSubF.Sub(x, new(big.Int).SetBytes(pro.Proof.f))
	xSubF.Mod(xSubF, privacy.Curve.Params().N)
	leftPoint2.X, leftPoint2.Y = privacy.Curve.ScalarMult(comPoint.X, comPoint.Y, xSubF.Bytes())
	leftPoint2.X, leftPoint2.Y = privacy.Curve.Add(leftPoint2.X, leftPoint2.Y, cbPoint.X, cbPoint.Y)

	// Calculate rightPoint1 = Com(0, zb)
	rightValue2 := privacy.Elcm.CommitSpecValue(big.NewInt(0).Bytes(), pro.Proof.zb, pro.Proof.index)
	rightPoint2, err := privacy.DecompressCommitment(rightValue2)
	if err != nil {
		fmt.Printf("Can not decompress comitment for zero")
		return false
	}

	if leftPoint1.X.Cmp(rightPoint1.X) == 0 && leftPoint1.Y.Cmp(rightPoint1.Y) == 0 && leftPoint2.X.Cmp(rightPoint2.X) == 0 && leftPoint2.Y.Cmp(rightPoint2.Y) == 0 {
		return true
	}

	return false
}

// TestPKComZeroOne tests prove and verify function for PK for Commitment to zero or one
func TestPKComZeroOne() {

	privacy.Elcm.InitCommitment()
	//spendingKey := GenerateSpendingKey(new(big.Int).SetInt64(123).Bytes())
	//fmt.Printf("\nSpending key: %v\n", spendingKey)
	//
	//pubKey := GeneratePublicKey(spendingKey)
	//serialNumber := privacy.RandBytes(32)
	//
	//value := make([]byte, 32, 32)
	//valueInt := big.NewInt(1)
	//value = valueInt.Bytes()
	//
	//r := privacy.RandBytes(32)
	//coin := Coin{
	//	PublicKey:      pubKey,
	//	SerialNumber:   serialNumber,
	//	CoinCommitment: nil,
	//	R:              r,
	//	VoteAmount:          value,
	//}
	//coin.CommitAll()
	//fmt.Println(coin.CoinCommitment)
	res := true
	for res {
		valueRand := privacy.RandBytes(32)
		vInt := new(big.Int).SetBytes(valueRand)
		vInt.Mod(vInt, big.NewInt(2))
		rand := privacy.RandBytes(32)

		partialCommitment := privacy.Elcm.CommitSpecValue(vInt.Bytes(), rand, privacy.VALUE)

		//witness := [][]byte{
		//	vInt.Bytes(),
		//	Rand,
		//}

		var zk PKComZeroOneProtocol
		var witness PKComZeroOneWitness
		witness.CommitedValue = vInt.Bytes()
		witness.Rand = rand
		witness.Commitment = partialCommitment
		witness.Index = privacy.VALUE

		zk.SetWitness(witness)
		proof, _ := zk.Prove()

		zk.SetProof(*proof)

		fmt.Printf("Proof: %+v\n", proof)

		res = zk.Verify()
		fmt.Println(res)
	}
}
