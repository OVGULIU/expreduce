//go:generate go tool yacc -p Calc -o interp.go interp.y
//go:generate golex -o tokenizer.go tokenizer.l

package expreduce

import (
	"bytes"
	"github.com/op/go-logging"
	"os"
	"log"
	"runtime/debug"
	"sort"
	"strings"
)

var format = logging.MustStringFormatter(
	//`%{color}%{time:15:04:05.000} %{callpath} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	`%{color}%{time:15:04:05.000} %{callpath} ▶ %{id:03x}%{color:reset} %{message}`,
)

type ToStringFnType (func(*Expression, string) (bool, string))

var toStringFns = make(map[string]ToStringFnType)

type CASLogger struct {
	_log       *logging.Logger
	leveled    logging.LeveledBackend
	debugState bool
}

type PDManager struct {
	patternDefined map[string]Ex
}

func EmptyPD() *PDManager {
	return &PDManager{make(map[string]Ex)}
}

func CopyPD(orig *PDManager) (dest *PDManager) {
	dest = EmptyPD()
	// We do not care that this iterates in a random order.
	for k, v := range (*orig).patternDefined {
		(*dest).patternDefined[k] = v.DeepCopy()
	}
	return
}

func (this *PDManager) Update(toAdd *PDManager) {
	// We do not care that this iterates in a random order.
	for k, v := range (*toAdd).patternDefined {
		(*this).patternDefined[k] = v
	}
}

type Def struct {
	downvalues []Expression
	attributes Attributes

	// A function defined here will override downvalues.
	legacyEvalFn (func(*Expression, *EvalState) Ex)
}

type EvalState struct {
	// Embedded type for logging
	CASLogger
	// So that we can give some functions read-only access to the EvalState
	//ReadOnlyState

	defined map[string]Def
	NoInit  bool
}

type Rule struct {
	Lhs string
	Rhs string
}

type Definition struct {
	// The symbol name, like "Mean", and "Total"
	Name  string
	Usage string
	// Currently used for SetDelayed, since other definitions depend on
	// SetDelayed, we define it first.
	Bootstrap bool

	// Regular rules to define. This should never include a map, as maps have
	// indeterminate iteration.
	Rules []Rule
	// Map symbol to Eval() function
	legacyEvalFn    (func(*Expression, *EvalState) Ex)
	SimpleExamples  []TestInstruction
	FurtherExamples []TestInstruction
	Tests           []TestInstruction

	toString ToStringFnType

	Attributes []string
}

func (this *EvalState) Load(def Definition) {
	// TODO: do we really need SetDelayed here, or should we just write to
	// downvalues directly? If we did this, we could potentially remove the
	// "bootstrap" attribute that SetDelayed has.
	for _, rule := range def.Rules {
		(&Expression{[]Ex{
			&Symbol{"SetDelayed"},
			Interp(rule.Lhs),
			Interp(rule.Rhs),
		}}).Eval(this)
	}

	if len(def.Usage) > 0 {
		(&Expression{[]Ex{
			&Symbol{"SetDelayed"},
			&Expression{[]Ex{
				&Symbol{"MessageName"},
				&Symbol{def.Name},
				&String{"usage"},
			}},
			&String{def.Usage},
		}}).Eval(this)
	}

	newDef, foundDef := this.defined[def.Name]
	if !foundDef {
		newDef = Def{}
	}

	if def.legacyEvalFn != nil {
		newDef.legacyEvalFn = def.legacyEvalFn
	}
	protectedAttrs := append(def.Attributes, "Protected")
	newDef.attributes = stringsToAttributes(protectedAttrs)
	if def.toString != nil {
		// Global so that standard String() interface can access these
		toStringFns[def.Name] = def.toString
	}
	this.defined[def.Name] = newDef
}

type NamedDefSet struct {
	Name string
	Defs []Definition
}

func GetAllDefinitions() (defs []NamedDefSet) {
	defs = append(defs, NamedDefSet{"combinatorics", GetCombinatoricsDefinitions()})
	defs = append(defs, NamedDefSet{"calculus", GetCalculusDefinitions()})
	defs = append(defs, NamedDefSet{"comparison", GetComparisonDefinitions()})
	defs = append(defs, NamedDefSet{"constants", GetConstantsDefinitions()})
	defs = append(defs, NamedDefSet{"expression", GetExpressionDefinitions()})
	defs = append(defs, NamedDefSet{"flowcontrol", GetFlowControlDefinitions()})
	defs = append(defs, NamedDefSet{"list", GetListDefinitions()})
	defs = append(defs, NamedDefSet{"order", GetOrderDefinitions()})
	defs = append(defs, NamedDefSet{"plus", GetPlusDefinitions()})
	defs = append(defs, NamedDefSet{"power", GetPowerDefinitions()})
	defs = append(defs, NamedDefSet{"random", GetRandomDefinitions()})
	defs = append(defs, NamedDefSet{"replacement", GetReplacementDefinitions()})
	defs = append(defs, NamedDefSet{"sort", GetSortDefinitions()})
	defs = append(defs, NamedDefSet{"symbol", GetSymbolDefinitions()})
	defs = append(defs, NamedDefSet{"system", GetSystemDefinitions()})
	defs = append(defs, NamedDefSet{"string", GetStringDefinitions()})
	defs = append(defs, NamedDefSet{"time", GetTimeDefinitions()})
	defs = append(defs, NamedDefSet{"times", GetTimesDefinitions()})
	defs = append(defs, NamedDefSet{"pattern", GetPatternDefinitions()})

	// Check for duplicate definitions
	definedNames := make(map[string]bool)
	for _, defSet := range defs {
		for _, def := range defSet.Defs {
			_, alreadyDefined := definedNames[def.Name]
			if alreadyDefined {
				log.Fatalf("Found duplicate definition: %v\n", def.Name)
			}
			definedNames[def.Name] = true
		}
	}
	return
}

func InitCAS(es *EvalState) {
	// System initialization
	EvalInterp("SeedRandom[UnixTime[]]", es)
}

func (es *EvalState) Init(loadAllDefs bool) {
	es.defined = make(map[string]Def)

	es.NoInit = !loadAllDefs
	if !es.NoInit {
		// Init modules
		for _, defSet := range GetAllDefinitions() {
			for _, def := range defSet.Defs {
				if def.Bootstrap {
					es.Load(def)
				}
			}
		}
		for _, defSet := range GetAllDefinitions() {
			for _, def := range defSet.Defs {
				if !def.Bootstrap {
					es.Load(def)
				}
			}
		}
		InitCAS(es)
	}
}

func NewEvalState() *EvalState {
	var es EvalState
	es.Init(true)

	// Set up logging
	es.CASLogger._log = logging.MustGetLogger("example")
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)
	es.CASLogger.leveled = logging.AddModuleLevel(formatter)
	logging.SetBackend(es.CASLogger.leveled)
	es.DebugOff()

	return &es
}

func NewEvalStateNoLog(loadAllDefs bool) *EvalState {
	var es EvalState
	es.Init(loadAllDefs)
	es.CASLogger.debugState = false
	return &es
}

func (this *CASLogger) Debugf(fmt string, args ...interface{}) {
	if this.debugState {
		//this._log.Debugf(this.Pre() + fmt, args...)
		this._log.Debugf(fmt, args...)
	}
}

func (this *CASLogger) Infof(fmt string, args ...interface{}) {
	if this.debugState {
		//this._log.Infof(this.Pre() + fmt, args...)
		this._log.Infof(fmt, args...)
	}
}

func (this *CASLogger) DebugOn() {
	this.leveled.SetLevel(logging.DEBUG, "")
	this.debugState = true
}

func (this *CASLogger) DebugOff() {
	this.leveled.SetLevel(logging.ERROR, "")
	this.debugState = false
}

func (this *CASLogger) DebugState() bool {
	return this.debugState
}

func (this *CASLogger) Pre() string {
	toReturn := ""
	if this.leveled.GetLevel("") != logging.ERROR {
		depth := (bytes.Count(debug.Stack(), []byte{'\n'}) - 15) / 2
		for i := 0; i < depth; i++ {
			toReturn += " "
		}
	}
	return toReturn
}

func (this *EvalState) GetDef(name string, lhs Ex) (Ex, bool) {
	_, isd := this.defined[name]
	if !isd {
		return nil, false
	}
	this.Debugf("Inside GetDef(\"%s\",%s)", name, lhs)
	for i := range this.defined[name].downvalues {
		ismatchq, _ := IsMatchQ(lhs, this.defined[name].downvalues[i].Parts[1], EmptyPD(), &this.CASLogger)
		if ismatchq {
			res := ReplaceAll(lhs, &this.defined[name].downvalues[i], &this.CASLogger, EmptyPD())
			return res, true
		}
	}
	return nil, false
}

func (this *EvalState) Define(name string, lhs Ex, rhs Ex) {
	this.Debugf("Inside es.Define(\"%s\",%s,%s)", name, lhs, rhs)
	_, isd := this.defined[name]
	if !isd {
		newDef := Def{
			downvalues: []Expression{{[]Ex{&Symbol{"Rule"}, lhs, rhs}}},
		}
		this.defined[name] = newDef
		return
	}

	for i := range this.defined[name].downvalues {
		if IsSameQ(this.defined[name].downvalues[i].Parts[1], lhs, &this.CASLogger) {
			this.defined[name].downvalues[i].Parts[2] = rhs
			return
		}
	}

	// Insert into definitions for name. Maintain order of decreasing
	// complexity. I define complexity as the length of the Lhs.String()
	// because it is simple, and it works for most of the common cases. We wish
	// to attempt f[x_Integer] before we attempt f[x_]. If LHSs map to the same
	// "complexity" score, order then matters. TODO: Create better measure of
	// complexity (or specificity)
	var tmp = this.defined[name]
	newLhsLen := len(lhs.StringForm("InputForm"))
	for i := range this.defined[name].downvalues {
		thisLhsLen := len(this.defined[name].downvalues[i].Parts[1].String())
		if thisLhsLen < newLhsLen {
			tmp.downvalues = append(tmp.downvalues[:i], append([]Expression{{[]Ex{&Symbol{"Rule"}, lhs, rhs}}}, this.defined[name].downvalues[i:]...)...)
			this.defined[name] = tmp
			return
		}
	}
	tmp.downvalues = append(tmp.downvalues, Expression{[]Ex{&Symbol{"Rule"}, lhs, rhs}})
	this.defined[name] = tmp
}

func (this *EvalState) ClearAll() {
	this.Init(!this.NoInit)
}

func (this *EvalState) Clear(name string) {
	_, ok := this.defined[name]
	if ok {
		delete(this.defined, name)
	}
}

func CopyDefs(in map[string]Def) map[string]Def {
	out := make(map[string]Def)
	for k, v := range in {
		newDef := Def{}
		for _, rule := range v.downvalues {
			newDef.downvalues = append(newDef.downvalues, *rule.DeepCopy().(*Expression))
		}
		out[k] = newDef
	}
	return out
}

func (this *EvalState) GetDefinedSnapshot() map[string]Def {
	return CopyDefs(this.defined)
}

func (this *EvalState) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	// We sort the keys here such that converting identical EvalStates always
	// produces the same string.
	keys := []string{}
	for k := range this.defined {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := this.defined[k]
		buffer.WriteString(k)
		buffer.WriteString(": ")
		buffer.WriteString(v.String())
		buffer.WriteString(", ")
	}
	if strings.HasSuffix(buffer.String(), ", ") {
		buffer.Truncate(buffer.Len() - 2)
	}
	buffer.WriteString("}")
	return buffer.String()
}

func (this *PDManager) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	// We sort the keys here such that converting identical PDManagers always
	// produces the same string.
	keys := []string{}
	for k := range this.patternDefined {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := this.patternDefined[k]
		buffer.WriteString(k)
		buffer.WriteString("_: ")
		buffer.WriteString(v.String())
		buffer.WriteString(", ")
	}
	if strings.HasSuffix(buffer.String(), ", ") {
		buffer.Truncate(buffer.Len() - 2)
	}
	buffer.WriteString("}")
	return buffer.String()
}

// Ex stands for Expression. Most structs should implement this
type Ex interface {
	Eval(es *EvalState) Ex
	String() string
	StringForm(form string) string
	IsEqual(b Ex, cl *CASLogger) string
	DeepCopy() Ex
}

// Some utility functions that span multiple files

func ExArrayToString(exArray []Ex) string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	for i, e := range exArray {
		buffer.WriteString(e.String())
		if i != len(exArray)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}

func (this *Def) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	for i, e := range this.downvalues {
		buffer.WriteString(e.String())
		if i != len(this.downvalues)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}

func ExArrayContainsFloat(a []Ex) bool {
	res := false
	for _, e := range a {
		_, isfloat := e.(*Flt)
		res = res || isfloat
	}
	return res
}

func ExtractBlankSequences(components []Ex) (nonBS []Ex, bs []Ex) {
	for _, c := range components {
		pat, isPat := HeadAssertion(c, "Pattern")
		_, isBns := HeadAssertion(c, "BlankNullSequence")
		_, isBs := HeadAssertion(c, "BlankSequence")
		if isPat {
			_, isBns = HeadAssertion(pat.Parts[2], "BlankNullSequence")
			_, isBs = HeadAssertion(pat.Parts[2], "BlankSequence")
		}
		if isBs || isBns {
			bs = append(bs, c)
		} else {
			nonBS = append(nonBS, c)
		}
	}
	return
}

// Should a MatchQ call do:
// 1. Modify pm directly <- bad idea. If we attempt a match and it partially
//    matches, we'll have to restore pm from a snapshot
// 2. Return a modified pm <- probably simplest
// 3. Return a pm with fields to add <- would be most efficient, but complicated
//    and could easily be incorrectly used.
// See IsBlankCapturing for a good example of good use.
func OrderlessIsMatchQ(components []Ex, lhs_components []Ex, pm *PDManager, cl *CASLogger) (bool, *PDManager) {
	pm = CopyPD(pm)
	if cl.debugState {
		cl.Infof("Entering OrderlessIsMatchQ(components: %s, lhs_components: %s, pm: %s)", ExArrayToString(components), ExArrayToString(lhs_components), pm)
	}
	nonBS, bs := ExtractBlankSequences(lhs_components)
	// This is because MatchQ[a + b + c, b + c] == False. We should be careful
	// though because MatchQ[a + b + c, c + __] == True.
	if len(bs) == 0 && len(components) != len(lhs_components) {
		cl.Debugf("len(components) != len(lhs_components). OrderlessMatchQ failed")
		return false, pm
	} else if len(nonBS) > len(components) {
		cl.Debugf("len(nonBS) > len(components). OrderlessMatchQ failed")
		return false, pm
	}

	// After determining that there is a blanksequence, I should go through
	// Each element of the pattern to be matched to see if it even exists within
	// components. I should use MemberQ for this. This can avoid the time-
	// consuming algorithms below

	// These lines are causing MatchQ[a + b, a + b + x___Plus] == True to fail
	for _, mustContain := range lhs_components {
		if !MemberQ(components, mustContain, cl) {
			return false, pm
		}
	}

	kConstant := len(components)
	if len(bs) == 1 {
		// This is probably the most common case. It would be rare for us to
		// have multiple BlankSequences in the same LHS. It saves us a lot of
		// time by doing this
		kConstant = len(nonBS)
	}

	// Start iterating through each permutation of LHS expressions
	perm, cont := make([]int, len(components)), 1
	for i := range perm {
		perm[i] = i
	}
	// Order lhs_components because if we have len(bs) == 1, we will depend on
	// the last n-k items to be orderless. This means that the BlankSequence
	// must be at the end. Eventually this may not be needed once automatic
	// sorting is implemented
	ordered_lhs_components := append(nonBS, bs...)
	for cont == 1 {
		cl.Debugf("Using perm: %v\n", perm)

		// Build a version of components with the correct order. Can I do this
		// more efficiently with a slice notation? Let's copy for now.
		orderedComponents := make([]Ex, len(components))
		for oci, ci := range perm {
			orderedComponents[oci] = components[ci].DeepCopy()
		}
		if cl.debugState {
			cl.Infof("%s", ExArrayToString(orderedComponents))
		}
		ncIsMatchQ, newPm := NonOrderlessIsMatchQ(orderedComponents, ordered_lhs_components, pm, cl)
		if ncIsMatchQ {
			cl.Debugf("OrderlessIsMatchQ succeeded. Context: %s", pm)
			return true, newPm
		}

		// Generate next permutation, if any
		cont = nextKPermutation(perm, len(components), kConstant)
	}
	cl.Debugf("OrderlessIsMatchQ failed. Context: %s", pm)
	return false, pm
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func NonOrderlessIsMatchQ(components []Ex, lhs_components []Ex, pm *PDManager, cl *CASLogger) (bool, *PDManager) {
	pm = CopyPD(pm)
	// This function is now recursive because of the existence of BlankSequence.
	if cl.debugState {
		cl.Infof("Entering NonOrderlessIsMatchQ(components: %s, lhs_components: %s, pm: %s)", ExArrayToString(components), ExArrayToString(lhs_components), pm)
	}
	// A base case for the recursion
	if len(components) == 0 && len(lhs_components) == 0 {
		return true, pm
	}
	if len(components) != 0 && len(lhs_components) == 0 {
		return false, pm
	}

	progressI := 0
	for i := 0; i < Max(len(components), len(lhs_components)); i++ {
		progressI = i
		if i >= len(lhs_components) {
			return false, pm
		}
		if i >= len(components) {
			cl.Debugf("Checking if IsMatchQ(INDEX_ERROR, %s). i=%d, Current context: %v\n", lhs_components[i], i, pm)
		} else {
			cl.Debugf("Checking if IsMatchQ(%s, %s). i=%d, Current context: %v\n", components[i], lhs_components[i], i, pm)
		}
		pat, isPat := HeadAssertion(lhs_components[i], "Pattern")
		bns, isBns := HeadAssertion(lhs_components[i], "BlankNullSequence")
		bs, isBs := HeadAssertion(lhs_components[i], "BlankSequence")
		if isPat {
			bns, isBns = HeadAssertion(pat.Parts[2], "BlankNullSequence")
			bs, isBs = HeadAssertion(pat.Parts[2], "BlankSequence")
		}
		if isBns || isBs {
			cl.Debugf("Encountered BS or BNS!")
			remainingLhs := make([]Ex, len(lhs_components)-i-1)
			for k := i + 1; k < len(lhs_components); k++ {
				remainingLhs[k-i-1] = lhs_components[k].DeepCopy()
			}
			startI := 0
			if isBns {
				startI = i - 1
			} else {
				startI = i
			}
			for j := startI; j < len(components); j++ {
				// This process involves a lot of extraneous copying. I should
				// test to see how much of these arrays need to be copied from
				// scratch on every iteration.
				seqToTry := make([]Ex, j-i+1)
				for k := i; k <= j; k++ {
					seqToTry[k-i] = components[k].DeepCopy()
				}
				seqMatches := false
				if isBns {
					seqMatches = ExArrayTestRepeatingMatch(seqToTry, BlankNullSequenceToBlank(bns), cl)
				} else {
					seqMatches = ExArrayTestRepeatingMatch(seqToTry, BlankSequenceToBlank(bs), cl)
				}
				cl.Debugf("%v", seqMatches)
				remainingComps := make([]Ex, len(components)-j-1)
				for k := j + 1; k < len(components); k++ {
					remainingComps[k-j-1] = components[k].DeepCopy()
				}
				if cl.debugState {
					cl.Debugf("%d %s %s %s", j, ExArrayToString(seqToTry), ExArrayToString(remainingComps), ExArrayToString(remainingLhs))
				}
				matchq, newPDs := NonOrderlessIsMatchQ(remainingComps, remainingLhs, pm, cl)
				if seqMatches && matchq {
					pm.Update(newPDs)
					if isPat {
						sAsSymbol, sAsSymbolOk := pat.Parts[1].(*Symbol)
						if sAsSymbolOk {
							toTryParts := []Ex{&Symbol{"Sequence"}}
							toTryParts = append(toTryParts, seqToTry...)
							target := &Expression{toTryParts}
							_, ispd := pm.patternDefined[sAsSymbol.Name]
							if !ispd {
								pm.patternDefined[sAsSymbol.Name] = target
							}
							if !IsSameQ(pm.patternDefined[sAsSymbol.Name], target, cl) {
								return false, pm
							}
						}
					}
					return true, pm
				}
			}
			return false, pm
		}
		if i >= len(components) {
			return false, pm
		}
		ismatchq, toAdd := IsMatchQ(components[i].DeepCopy(), lhs_components[i], pm, cl)
		if ismatchq {
			cl.Debugf("Returned True!\n")
			pm.Update(toAdd)
		} else {
			cl.Debugf("NonOrderlessIsMatchQ failed. Context: %s", pm)
			return false, pm
		}
	}
	if progressI == len(lhs_components)-1 {
		return true, pm
	} else {
		return false, pm
	}
}

func FunctionIsSameQ(components []Ex, other_components []Ex, cl *CASLogger) bool {
	if len(components) != len(other_components) {
		return false
	}
	for i := range components {
		res := IsSameQ(components[i], other_components[i], cl)
		if !res {
			return false
		}
	}
	return true
}

func IterableReplace(components *[]Ex, r *Expression, pm *PDManager, cl *CASLogger) {
	pm = CopyPD(pm)
	for i := range *components {
		cl.Debugf("Attempting IsMatchQ(%s, %s, %s)", (*components)[i], r.Parts[1], pm)
		if res, _ := IsMatchQ((*components)[i], r.Parts[1], pm, cl); res {
			(*components)[i] = r.Parts[2].DeepCopy()
			cl.Debugf("IsMatchQ succeeded, new components: %s", ExArrayToString(*components))
		}
	}
}

func permutations(iterable []int, r int) [][]int {
	res := make([][]int, 0)
	pool := iterable
	n := len(pool)

	if r > n {
		return res
	}

	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	cycles := make([]int, r)
	for i := range cycles {
		cycles[i] = n - i
	}

	result := make([]int, r)
	for i, el := range indices[:r] {
		result[i] = pool[el]
	}

	c := make([]int, len(result))
	copy(c, result)
	res = append(res, c)

	for n > 0 {
		i := r - 1
		for ; i >= 0; i -= 1 {
			cycles[i] -= 1
			if cycles[i] == 0 {
				index := indices[i]
				for j := i; j < n-1; j += 1 {
					indices[j] = indices[j+1]
				}
				indices[n-1] = index
				cycles[i] = n - i
			} else {
				j := cycles[i]
				indices[i], indices[n-j] = indices[n-j], indices[i]

				for k := i; k < r; k += 1 {
					result[k] = pool[indices[k]]
				}

				c := make([]int, len(result))
				copy(c, result)
				res = append(res, c)

				break
			}
		}

		if i < 0 {
			return res
		}

	}
	return res

}

func OrderlessReplace(components *[]Ex, lhs_components []Ex, rhs Ex, cl *CASLogger) {
	// TODO: Doesn't take a PDManager as an input right now. Will add this later.
	cl.Infof("Entering OrderlessReplace(components: *%s, lhs_components: %s)", ExArrayToString(*components), ExArrayToString(lhs_components))
	// Each permutation is a potential order of the Rule's LHS in which matches
	// may occur in components.
	toPermute := make([]int, len(*components))
	for i := range toPermute {
		toPermute[i] = i
	}
	perms := permutations(toPermute, len(lhs_components))
	cl.Debugf("Permutations to try: %v\n", perms)

	for _, perm := range perms {
		used := make([]int, len(perm))
		pi := 0
		pm := EmptyPD()
		//cl.Debugf("Before snapshot. Context: %v\n", es)
		for i := range perm {
			//cl.Debugf("%s %s\n", (*components)[perm[i]], lhs_components[i])
			mq, matches := IsMatchQ((*components)[perm[i]].DeepCopy(), lhs_components[i], pm, cl)
			if mq {
				pm.Update(matches)
				used[pi] = perm[i]
				pi = pi + 1

				if pi == len(perm) {
					sort.Ints(used)
					cl.Debugf("About to delete components matching lhs.")
					cl.Debugf("components before: %s", ExArrayToString(*components))
					for tdi, todelete := range used {
						*components = append((*components)[:todelete-tdi], (*components)[todelete-tdi+1:]...)
					}
					cl.Debugf("components after: %s", ExArrayToString(*components))
					cl.Debugf("Appending %s\n", rhs)
					//cl.Debugf("Context: %v\n", es)
					*components = append(*components, []Ex{ReplacePD(rhs.DeepCopy(), cl, matches)}...)
					cl.Debugf("components after append: %s", ExArrayToString(*components))
					//cl.Debugf("After clear. Context: %v\n", es)
					return
				}
			}
			//cl.Debugf("Done checking. Context: %v\n", es)
		}
		//cl.Debugf("After clear. Context: %v\n", es)
	}
}