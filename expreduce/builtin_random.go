package expreduce

import (
	"math/big"
	"math/rand"

	"github.com/corywalker/expreduce/pkg/expreduceapi"
)

func GetRandomDefinitions() (defs []Definition) {
	defs = append(defs, Definition{
		Name: "RandomReal",
		Details: "`SeedRandom[UnixTime[]]` is called automatically upon " +
			"initialization of Expreduce, so random number sequences will not " +
			"repeat over subsequent sessions.",
		legacyEvalFn: func(this *expreduceapi.ExpressionInterface, es *expreduceapi.EvalStateInterface) expreduceapi.Ex {
			if len(this.Parts) != 1 {
				return this
			}

			return NewReal(big.NewFloat(rand.Float64()))
		},
	})
	defs = append(defs, Definition{
		Name: "SeedRandom",
		Details: "`SeedRandom[UnixTime[]]` is called automatically upon " +
			"initialization of Expreduce, so random number sequences will not " +
			"repeat over subsequent sessions.",
		legacyEvalFn: func(this *expreduceapi.ExpressionInterface, es *expreduceapi.EvalStateInterface) expreduceapi.Ex {
			if len(this.Parts) != 2 {
				return this
			}

			asInt, isInt := this.Parts[1].(*Integer)
			if isInt {
				rand.Seed(asInt.Val.Int64())
				return NewSymbol("System`Null")
			}

			return this
		},
	})
	return
}
