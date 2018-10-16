package expreduce

import (
	"math/big"

	"github.com/corywalker/expreduce/expreduce/logging"
)

type singleParamQType (func(Ex) bool)
type singleParamQLogType (func(Ex, *logging.CASLogger) bool)
type doubleParamQLogType (func(Ex, Ex, *logging.CASLogger) bool)
type evalFnType (func(*Expression, *EvalState) Ex)

func singleParamQEval(fn singleParamQType) evalFnType {
	return (func(this *Expression, es *EvalState) Ex {
		if len(this.Parts) != 2 {
			return this
		}
		if fn(this.Parts[1]) {
			return NewSymbol("System`True")
		}
		return NewSymbol("System`False")
	})
}

func singleParamQLogEval(fn singleParamQLogType) evalFnType {
	return (func(this *Expression, es *EvalState) Ex {
		if len(this.Parts) != 2 {
			return this
		}
		if fn(this.Parts[1], &es.CASLogger) {
			return NewSymbol("System`True")
		}
		return NewSymbol("System`False")
	})
}

func doubleParamQLogEval(fn doubleParamQLogType) evalFnType {
	return (func(this *Expression, es *EvalState) Ex {
		if len(this.Parts) != 3 {
			return this
		}
		if fn(this.Parts[1], this.Parts[2], &es.CASLogger) {
			return NewSymbol("System`True")
		}
		return NewSymbol("System`False")
	})
}

func numberQ(e Ex) bool {
	_, ok := e.(*Integer)
	if ok {
		return true
	}
	_, ok = e.(*Flt)
	if ok {
		return true
	}
	_, ok = e.(*Rational)
	if ok {
		return true
	}
	_, ok = e.(*Complex)
	if ok {
		return true
	}
	return false
}

func vectorQ(e Ex) bool {
	l, isL := HeadAssertion(e, "System`List")
	if isL {
		for i := 1; i < len(l.Parts); i++ {
			_, subIsL := HeadAssertion(l.Parts[i], "System`List")
			if subIsL {
				return false
			}
		}
		return true
	}
	return false
}

func matrixQ(e Ex, cl *logging.CASLogger) bool {
	l, isL := HeadAssertion(e, "System`List")
	if isL {
		return len(dimensions(l, 0, cl)) == 2
	}
	return false
}

func symbolNameQ(e Ex, name string, cl *logging.CASLogger) bool {
	sym, isSym := e.(*Symbol)
	if isSym {
		return sym.Name == name
	}
	return false
}

func trueQ(e Ex, cl *logging.CASLogger) bool {
	return symbolNameQ(e, "System`True", cl)
}

func falseQ(e Ex, cl *logging.CASLogger) bool {
	return symbolNameQ(e, "System`False", cl)
}

func booleanQ(e Ex, cl *logging.CASLogger) bool {
	sym, isSym := e.(*Symbol)
	if isSym {
		return sym.Name == "System`False" || sym.Name == "System`True"
	}
	return false
}

func primeQ(e Ex) bool {
	asInt, ok := e.(*Integer)
	if !ok {
		return false
	}
	abs := big.NewInt(0)
	abs.Abs(asInt.Val)
	return abs.ProbablyPrime(16)
}
