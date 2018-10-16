package expreduce

import (
	"fmt"
	"hash/fnv"
	"math/big"

	"github.com/corywalker/expreduce/expreduce/logging"
)

type Rational struct {
	Num       *big.Int
	Den       *big.Int
	needsEval bool
}

func (this *Rational) Eval(es *EvalState) Ex {
	if this.Num.Cmp(big.NewInt(0)) == 0 && this.Den.Cmp(big.NewInt(0)) == 0 {
		return NewSymbol("System`Indeterminate")
	}
	if this.Den.Cmp(big.NewInt(0)) == 0 {
		return NewSymbol("System`ComplexInfinity")
	}
	if this.Num.Cmp(big.NewInt(0)) == 0 {
		return NewInteger(big.NewInt(0))
	}
	negNum := this.Num.Cmp(big.NewInt(0)) == -1
	negDen := this.Den.Cmp(big.NewInt(0)) == -1
	negateRes := negNum != negDen
	absNum := big.NewInt(0)
	absNum.Abs(this.Num)
	absDen := big.NewInt(0)
	absDen.Abs(this.Den)

	gcd := big.NewInt(0)
	gcd.GCD(nil, nil, absNum, absDen)
	absNum.Div(absNum, gcd)
	absDen.Div(absDen, gcd)

	if absDen.Cmp(big.NewInt(1)) == 0 {
		if !negateRes {
			return NewInteger(absNum)
		} else {
			return NewInteger(absNum.Neg(absNum))
		}
	}

	if !negateRes {
		this.Num.Set(absNum)
		this.Den.Set(absDen)
		this.needsEval = false
		return this
	}
	this.Num.Set(absNum.Neg(absNum))
	this.Den.Set(absDen)
	this.needsEval = false
	return this
}

func (this *Rational) StringForm(params ToStringParams) string {
	if params.form == "FullForm" {
		return fmt.Sprintf("Rational[%d, %d]", this.Num, this.Den)
	}
	if params.form == "TeXForm" {
		return fmt.Sprintf("\\frac{%d}{%d}", this.Num, this.Den)
	}
	if needsParens("System`Times", params.previousHead) {
		return fmt.Sprintf("(%d/%d)", this.Num, this.Den)
	}
	return fmt.Sprintf("%d/%d", this.Num, this.Den)
}

func (this *Rational) String(es *EvalState) string {
	context, contextPath := DefaultStringFormArgs()
	return this.StringForm(ToStringParams{form: "InputForm", context: context, contextPath: contextPath, es: es})
}

func (this *Rational) IsEqual(other Ex, cl *logging.CASLogger) string {
	otherConv, otherIsRational := other.(*Rational)
	if !otherIsRational {
		return "EQUAL_FALSE"
	}
	// Assume rational already simplified
	if (this.Num.Cmp(otherConv.Num) != 0) || (this.Den.Cmp(otherConv.Den) != 0) {
		return "EQUAL_FALSE"
	}
	return "EQUAL_TRUE"
}

func (this *Rational) DeepCopy() Ex {
	tmpn := big.NewInt(0)
	tmpn.Set(this.Num)
	tmpd := big.NewInt(0)
	tmpd.Set(this.Den)
	return &Rational{tmpn, tmpd, this.needsEval}
}

func (this *Rational) Copy() Ex {
	return this.DeepCopy()
}

func (this *Rational) AsBigRat() *big.Rat {
	res := big.NewRat(1, 1)
	return res.SetFrac(this.Num, this.Den)
}

func (this *Rational) NeedsEval() bool {
	return this.needsEval
}

func NewRational(n *big.Int, d *big.Int) *Rational {
	return &Rational{n, d, true}
}

func (this *Rational) Hash() uint64 {
	h := fnv.New64a()
	h.Write([]byte{90, 82, 214, 51, 52, 7, 7, 33})
	nBytes, _ := this.Num.MarshalText()
	h.Write(nBytes)
	dBytes, _ := this.Den.MarshalText()
	h.Write(dBytes)
	return h.Sum64()
}

func (this *Rational) AsBigFloat() *big.Float {
	num := big.NewFloat(0)
	den := big.NewFloat(0)
	newquo := big.NewFloat(0)
	num.SetInt(this.Num)
	den.SetInt(this.Den)
	newquo.Quo(num, den)
	return newquo
}

func (this *Rational) AddI(i *Integer) {
	tmp := big.NewInt(0)
	tmp.Mul(i.Val, this.Den)
	this.Num.Add(this.Num, tmp)
}

func (this *Rational) AddR(r *Rational) {
	tmp := big.NewInt(0)
	// lastrNum/lastrDen + theratNum/theratDen // Together
	tmp.Mul(this.Den, r.Num)
	this.Den.Mul(this.Den, r.Den)
	this.Num.Mul(this.Num, r.Den)
	this.Num.Add(this.Num, tmp)
}

func (this *Rational) MulI(i *Integer) {
	this.Num.Mul(this.Num, i.Val)
}

func (this *Rational) MulBigI(i *big.Int) {
	this.Num.Mul(this.Num, i)
}

func (this *Rational) MulR(r *Rational) {
	this.Num.Mul(this.Num, r.Num)
	this.Den.Mul(this.Den, r.Den)
}
