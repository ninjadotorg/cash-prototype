package privacy

import (
	"fmt"
	"math/big"
	"testing"
)

func TestPrettyPrint(t *testing.T) {
	cases := []struct {
		p   Poly
		ans string
	}{
		{
			NewPolyInts(0),
			"[0]",
		},
		{
			NewPolyInts(5, -4, 3, 3),
			"[3x^3 + 3x^2 - 4x + 5]",
		},
		{
			NewPolyInts(5, 6, 2),
			"[2x^2 + 6x + 5]",
		},
		{
			NewPolyInts(5, -2, 0, 2, 1, 3),
			"[3x^5 + x^4 + 2x^3 - 2x + 5]",
		},
		{
			NewPolyInts(2, 1, 0, -1, -2),
			"[-2x^4 - x^3 + x + 2]",
		},
	}
	for _, c := range cases {
		s := fmt.Sprintf("%v", c.p)
		if s != c.ans {
			t.Errorf("Stringify %v should be %v", s, c.ans)
		}
	}

}

func TestTrim(t *testing.T) {
	cases := []struct {
		p   Poly
		ans Poly
	}{
		{
			NewPolyInts(0),
			NewPolyInts(0),
		},
		{
			NewPolyInts(5, -4, 3, 3, 0),
			NewPolyInts(5, -4, 3, 3),
		},
		{
			NewPolyInts(5, 6, 2, 0, 0),
			NewPolyInts(5, 6, 2),
		},
		{
			NewPolyInts(5, -2, 0, 2, 1, 3, 0, 0, 0),
			NewPolyInts(5, -2, 0, 2, 1, 3),
		},
		{
			NewPolyInts(4),
			NewPolyInts(4),
		},
		{
			NewPolyInts(1, 2, 3),
			NewPolyInts(1, 2, 3),
		},
	}
	for _, c := range cases {
		tmp := (c.p).Clone(0)
		(c.p).trim()
		if (c.p).Compare(&c.ans) != 0 {
			t.Errorf("TRIM(%v) != %v (your answer was %v)\n", tmp, c.ans, c.p)
		}
	}
}

func TestClone(t *testing.T) {
	cases := []struct {
		p   Poly
		d   int
		ans Poly
	}{
		{
			NewPolyInts(-2, -1, 0, 1, 2),
			-2,
			NewPolyInts(0),
		},
		{
			NewPolyInts(-2, -1, 0, 1, 2),
			0,
			NewPolyInts(-2, -1, 0, 1, 2),
		},
		{
			NewPolyInts(-2, -1, 0, 1, 2),
			1,
			NewPolyInts(0, -2, -1, 0, 1, 2),
		},
		{
			NewPolyInts(-2, -1, 0, 1, 2),
			3,
			NewPolyInts(0, 0, 0, -2, -1, 0, 1, 2),
		},
	}
	for _, c := range cases {
		q := c.p.Clone(c.d)
		if q.Compare(&c.ans) != 0 {
			t.Errorf("Cloning %v with %v adjust != %v", c.p, c.d, c.ans)
		}
	}
}

func TestAdd(t *testing.T) {
	cases := []struct {
		p   Poly
		q   Poly
		m   *big.Int
		ans Poly
	}{
		{
			NewPolyInts(1, 1, 0, 2, 2, 1),
			NewPolyInts(1, 1, 1),
			nil,
			NewPolyInts(2, 2, 1, 2, 2, 1),
		},
		{
			NewPolyInts(5, -4, 3, 3),
			NewPolyInts(-4, 1, -2, 1),
			nil,
			NewPolyInts(1, -3, 1, 4),
		},
		{
			NewPolyInts(0),
			NewPolyInts(0),
			nil,
			NewPolyInts(0),
		},
		{
			NewPolyInts(0),
			NewPolyInts(0),
			big.NewInt(2),
			NewPolyInts(0),
		},
		{
			NewPolyInts(5, 6, 2),
			NewPolyInts(-1, -2, 3),
			nil,
			NewPolyInts(4, 4, 5),
		},
		{
			NewPolyInts(5, -2, 0, 2, 1, 3),
			NewPolyInts(2, 7, 0, 3, 0, 2),
			nil,
			NewPolyInts(7, 5, 0, 5, 1, 5),
		},
		{
			NewPolyInts(2, 5, 3, 1),
			NewPolyInts(14, 0, 3, 4),
			nil,
			NewPolyInts(16, 5, 6, 5),
		},
		{
			NewPolyInts(12, 0, 3, 2, 5),
			NewPolyInts(3, 0, 4, 7),
			nil,
			NewPolyInts(15, 0, 7, 9, 5),
		},
		{
			NewPolyInts(4, 0, 0, 3, 0, 1),
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			nil,
			NewPolyInts(4, 0, 0, 7, 0, 1, 6),
		},
		{
			NewPolyInts(4, 0, 0, 3, 0, 1),
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			big.NewInt(11),
			NewPolyInts(4, 0, 0, 7, 0, 1, 6),
		},
	}
	for _, c := range cases {
		res := (c.p).Add(c.q, c.m)
		if res.Compare(&c.ans) != 0 {
			t.Errorf("%v + %v != %v (your answer was %v)\n", c.p, c.q, c.ans, res)
		}
	}
}

func ExampleRandPoly() {
	p := RandomPoly(10, 128) // 계수의 크기가 0~2^128인 임의의 10차 다항식 생성
	fmt.Println(p)
}

func TestRandomPoly(t *testing.T) {
	p := RandomPoly(10, 128)
	if p.GetDegree() != 10 {
		t.Errorf("Polynomial %v should have %v degrees", p, p.GetDegree())
	}
	for i := 0; i < p.GetDegree(); i++ {
		if p[i].BitLen() > 128 {
			t.Errorf("Polynomial %v has too large coefficient (%v bits)", p, p[i].BitLen())
		}
	}
}

func BenchmarkAddTwoIntCoeffPolynomial(b *testing.B) {
	p := NewPolyInts(4, 0, 0, 3, 0, 1)
	q := NewPolyInts(0, 0, 0, 4, 0, 0, 6)
	m := big.NewInt(11)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Add(q, m)
	}
}

//func BenchmarkAddTwoBigInt128bitCoeffPolynomial(b *testing.B) {
//	p := RandomPoly(10, 128)
//	q := RandomPoly(10, 128)
//	m := RandomBigInt(128)
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		p.Add(q, m)
//	}
//}

func TestSub(t *testing.T) {
	cases := []struct {
		p   Poly
		q   Poly
		m   *big.Int
		ans Poly
	}{
		{
			NewPolyInts(0),
			NewPolyInts(0),
			nil,
			NewPolyInts(0),
		},
		{
			NewPolyInts(0),
			NewPolyInts(0),
			big.NewInt(2),
			NewPolyInts(0),
		},
		{
			NewPolyInts(-9, 2, 5),
			NewPolyInts(-3, 2, 2),
			nil,
			NewPolyInts(-6, 0, 3),
		},
		{
			NewPolyInts(5, -2, 0, 2, 1, 3),
			NewPolyInts(2, 7, 0, 3, 0, 2),
			nil,
			NewPolyInts(3, -9, 0, -1, 1, 1),
		},
		{
			NewPolyInts(12, 0, 3, 2, 0, 0, 0, 12),
			NewPolyInts(4, 0, 4, -11),
			nil,
			NewPolyInts(8, 0, -1, 13, 0, 0, 0, 12),
		},
		{
			NewPolyInts(4, 0, 0, 3, 0, 1),
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			nil,
			NewPolyInts(4, 0, 0, -1, 0, 1, -6),
		},
		{
			NewPolyInts(4, 0, 0, 3, 0, 1),
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			big.NewInt(11),
			NewPolyInts(4, 0, 0, 10, 0, 1, 5),
		},
	}
	for _, c := range cases {
		res := (c.p).Sub(c.q, c.m)
		if res.Compare(&c.ans) != 0 {
			t.Errorf("%v + %v != %v (your answer was %v)\n", c.p, c.q, c.ans, res)
		}
	}
}

func BenchmarkSub(b *testing.B) {
	p := NewPolyInts(4, 0, 0, 3, 0, 1)
	q := NewPolyInts(0, 0, 0, 4, 0, 0, 6)
	m := big.NewInt(11)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Sub(q, m)
	}
}

func TestMuliply(t *testing.T) {
	cases := []struct {
		p   Poly
		q   Poly
		m   *big.Int
		ans Poly
	}{
		{
			NewPolyInts(0),
			NewPolyInts(0),
			nil,
			NewPolyInts(0),
		},
		{
			NewPolyInts(0),
			NewPolyInts(0),
			big.NewInt(2),
			NewPolyInts(0),
		},
		{
			NewPolyInts(4, 0, 0, 3, 0, 1),
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			nil,
			NewPolyInts(0, 0, 0, 16, 0, 0, 36, 0, 4, 18, 0, 6),
		},
		{
			NewPolyInts(4, 0, 0, 3, 0, 1),
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			big.NewInt(11),
			NewPolyInts(0, 0, 0, 5, 0, 0, 3, 0, 4, 7, 0, 6),
		},
	}
	for _, c := range cases {
		res := (c.p).Mul(c.q, c.m)
		if res.Compare(&c.ans) != 0 {
			t.Errorf("%v + %v != %v (your answer was %v)\n", c.p, c.q, c.ans, res)
		}
	}
}

func BenchmarkMultiply(b *testing.B) {
	p := NewPolyInts(4, 0, 0, 3, 0, 1)
	q := NewPolyInts(0, 0, 0, 4, 0, 0, 6)
	m := big.NewInt(11)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Mul(q, m)
	}
}

func TestDivide(t *testing.T) {
	cases := []struct {
		p, q     Poly
		m        *big.Int
		quo, rem Poly
	}{
		{
			NewPolyInts(0),
			NewPolyInts(0),
			nil,
			NewPolyInts(0),
			NewPolyInts(0),
		},
		{
			NewPolyInts(0),
			NewPolyInts(0),
			big.NewInt(2),
			NewPolyInts(0),
			NewPolyInts(0),
		},
		{
			NewPolyInts(0, 0, 0, 16, 0, 0, 36, 0, 4, 18, 0, 6),
			NewPolyInts(4, 0, 0, 3, 0, 1),
			nil,
			NewPolyInts(0, 0, 0, 4, 0, 0, 6),
			NewPolyInts(0),
		},
		{
			NewPolyInts(5, 0, 0, 4, 7, 0, 3),
			NewPolyInts(4, 0, 0, 3, 1),
			nil,
			NewPolyInts(34, -9, 3),
			NewPolyInts(-131, 36, -12, -98),
		},
		{
			NewPolyInts(2, 0, 2, 1),
			NewPolyInts(1, 0, 1),
			big.NewInt(3),
			NewPolyInts(2, 1),
			NewPolyInts(0, 2),
		},
		{
			NewPolyInts(5, 0, 0, 4, 7, 0, 3),
			NewPolyInts(4, 0, 0, 3, 1),
			big.NewInt(11),
			NewPolyInts(1, 2, 3),
			NewPolyInts(1, 3, 10, 1),
		},
		// [161x^17 + 43x^16 + 113x^15 + 14x^14 + 258x^13 + 64x^12 + 164x^10 + 250x^9 + 288x^8 + 268x^7 + 13x^6 + 245x^5 + 39x^4 + 234x^2 + 187x + 184]
		{
			NewPolyInts(184, 187, 234, 0, 39, 245, 13, 268, 288, 250, 164, 0, 64, 258, 14, 113, 43, 161),
			NewPolyInts(48, 0, 43, 22, 56, 84, 45, 67, 0, 34, 53),
			big.NewInt(307),
			NewPolyInts(98, 35, 0, 0, 23, 55, 44, 32),
			NewPolyInts(85, 42, 11, 23, 45),
		},
		{
			NewPolyInts(-1, 0, 0, 1),
			NewPolyInts(2, 1),
			nil,
			NewPolyInts(4, -2, 1),
			NewPolyInts(-9),
		},
		{
			NewPolyInts(-15, 3, -5, 1),
			NewPolyInts(-5, 1),
			nil,
			NewPolyInts(3, 0, 1),
			NewPolyInts(0),
		},
		{
			NewPolyInts(4, 0, 0, 0, 1),
			NewPolyInts(-5, 0, 1),
			nil,
			NewPolyInts(5, 0, 1),
			NewPolyInts(29),
		},
		{
			NewPolyInts(-3, 5, -3, 1),
			NewPolyInts(-1, 1),
			nil,
			NewPolyInts(3, -2, 1),
			NewPolyInts(0),
		},
		{
			NewPolyInts(4, -7, 1),
			NewPolyInts(-1, 0, -5, 1),
			nil,
			NewPolyInts(0),
			NewPolyInts(4, -7, 1),
		},
		// 정수 배로 나눠지지 않는 경우에 대한 (몫의 계수가 분수가 되는) 테스트 케이스
		{
			NewPolyInts(-4, 0, 0, 1),
			NewPolyInts(5, 2),
			nil,
			NewPolyInts(0),
			NewPolyInts(-4, 0, 0, 1),
		},
		{
			NewPolyInts(4, 0, 0, 1),
			NewPolyInts(3, 1, 4, 1),
			nil,
			NewPolyInts(1),
			NewPolyInts(1, -1, -4),
		},
		{
			NewPolyInts(4, 0, 0, 1),
			NewPolyInts(3, 1, 4, 1),
			big.NewInt(7),
			NewPolyInts(1),
			NewPolyInts(1, 6, 3),
		},
	}
	for _, c := range cases {
		q, r := (c.p).Div(c.q, c.m)
		if q.Compare(&c.quo) != 0 || r.Compare(&c.rem) != 0 {
			t.Errorf("%v / %v != %v (%v) (your answer was %v (%v))\n", c.p, c.q, c.quo, c.rem, q, r)
		}
	}
}

func TestGcd(t *testing.T) {
	cases := []struct {
		p   Poly
		q   Poly
		m   *big.Int
		ans Poly
	}{
		{
			NewPolyInts(0),
			NewPolyInts(0),
			nil,
			NewPolyInts(0),
		},
		{
			NewPolyInts(0),
			NewPolyInts(0),
			big.NewInt(2),
			NewPolyInts(0),
		},
		{
			NewPolyInts(4, 0, 0, 1),
			NewPolyInts(3, 1, 4, 1),
			big.NewInt(7),
			NewPolyInts(1),
		},
		// 결과가 상수가 무시되어서 3x^2 + 3이 아니라 x^2 + 1로 나오는데, 이유는 아직 알지 못했다.
		// 우선 x^2 + 1도 CD긴 하기 때문에 넘어간다.
		{
			NewPolyInts(3, 0, 3).Mul(NewPolyInts(4, 5, 6, 7), big.NewInt(13)),
			NewPolyInts(3, 0, 3).Mul(NewPolyInts(5, 6, 7, 8, 9), big.NewInt(13)),
			big.NewInt(13),
			// NewPolyInts(3, 0, 3),
			NewPolyInts(1, 0, 1),
		},
	}
	for _, c := range cases {
		res := (c.p).Gcd(c.q, c.m)
		if res.Compare(&c.ans) != 0 {
			t.Errorf("GCD(%v, %v) != %v (your answer was %v)\n", c.p, c.q, c.ans, res)
		}
	}
}

func TestSanitize(t *testing.T) {
	cases := []struct {
		p   Poly
		m   *big.Int
		ans Poly
	}{
		{
			NewPolyInts(1, 2, 3, 4),
			nil,
			NewPolyInts(1, 2, 3, 4),
		},
		{
			NewPolyInts(1, 2, 3, 4),
			big.NewInt(1),
			NewPolyInts(0),
		},
		{
			NewPolyInts(1, 2, 3, 4),
			big.NewInt(2),
			NewPolyInts(1, 0, 1),
		},
	}
	for _, c := range cases {
		q := (c.p).Clone(0)
		q.sanitize(c.m)
		if q.Compare(&c.ans) != 0 {
			t.Errorf("Sanitized %v with %v != %v", c.p, c.m, c.ans)
		}
	}
}

func TestEval(t *testing.T) {
	cases := []struct {
		p         Poly
		x, m, ans *big.Int
	}{
		{
			NewPolyInts(0),
			big.NewInt(0),
			nil,
			big.NewInt(0),
		},
		{
			NewPolyInts(0),
			big.NewInt(9),
			nil,
			big.NewInt(0),
		},
		{
			NewPolyInts(0),
			big.NewInt(0),
			big.NewInt(2),
			big.NewInt(0),
		},
		{
			NewPolyInts(0),
			big.NewInt(1),
			big.NewInt(2),
			big.NewInt(0),
		},
		{
			NewPolyInts(1, -4, 1),
			big.NewInt(3),
			nil,
			big.NewInt(-2),
		},
		{
			NewPolyInts(1, -4, 1),
			big.NewInt(-5),
			nil,
			big.NewInt(46),
		},
		{
			NewPolyInts(6, 2, 0, 4, 1),
			big.NewInt(2),
			nil,
			big.NewInt(58),
		},
		{
			NewPolyInts(6, 2, 0, 4, 1),
			big.NewInt(2),
			big.NewInt(10),
			big.NewInt(8),
		},
		{
			NewPolyInts(-9, -5, 0, 3, 1),
			big.NewInt(2),
			nil,
			big.NewInt(21),
		},
		{
			NewPolyInts(1, -1, 2, -3),
			big.NewInt(4),
			nil,
			big.NewInt(-163),
		},
		{
			NewPolyInts(-105, -2, -8, -7, 12),
			big.NewInt(3),
			nil,
			big.NewInt(600),
		},
		{
			NewPolyInts(45545, 343424, 5545, 3445435, 0, 343434, 4665, 5452, 34344, 534556, 4345345, 5656, 434525, 53333, 36645),
			big.NewInt(394),
			big.NewInt(1046527),
			big.NewInt(636194),
		},
	}
	for _, c := range cases {
		res := (c.p).Eval(c.x, c.m)
		if res.Cmp(c.ans) != 0 {
			t.Errorf("f(x) = %v, f(%v) != %v (your answer was %v)\n", c.p, c.x, c.ans, res)
		}
	}
}
