%{
package cas

import (
    "log"
    "strconv"
    "math/big"
)

type Calclexer struct{
    s       string
    pos     int
    buf     []rune
    empty   bool
    current rune
}

func newLexer(s string) (y *Calclexer) {
    y = &Calclexer{s: s}
    if y.pos != len(y.s) {
        y.current = rune(y.s[y.pos])
    }
    /*fmt.Printf("y.current: %d, y.pos: %d, '%s'\n", y.current, y.pos, y.buf)*/
    y.pos += 1
    return
}

func (y *Calclexer) getc() rune {
    if y.current != 0 {
        y.buf = append(y.buf, y.current)
    }
    y.current = 0
    if y.pos != len(y.s) {
        y.current = rune(y.s[y.pos])
    }
    /*fmt.Printf("y.current: %d, y.pos: %d, '%s'\n", y.current, y.pos, y.buf)*/
    y.pos += 1
    return y.current
}

func (y Calclexer) Error(e string) {
    log.Fatal(e)
}

func (y *Calclexer) Lex(lval *CalcSymType) int {
    var err error
    c := y.current
    if y.empty {
        c, y.empty = y.getc(), false
    }
%}

%yyc c
%yyn c = y.getc()

D  [0-9]+
/*E  [eE][-+]?{D}*/
/*F  {D}"."{D}?{E}?|{D}{E}?|"."{D}{E}?*/
F  {D}"."{D}?|"."{D}|{D}
L [A-Za-z]
ident {L}({L}|{D})*

%%
    y.buf = y.buf[:0]

[ \t\r]+

{D}
    var tmpi int64
    var base int = 10
    tmpi, err = strconv.ParseInt(string(y.buf), base, 64)
    if err != nil {
        log.Fatal(err)
    }
    lval.val = &Integer{big.NewInt(tmpi)}

    return INTEGER

{F}
    var tmpf float64
    tmpf, err = strconv.ParseFloat(string(y.buf), 64)
    if err != nil {
        log.Fatal(err)
    }
    lval.val = &Flt{big.NewFloat(tmpf)}

    return FLOAT

\( return LPARSYM
\) return RPARSYM
, return COMMASYM
\[ return LBRACKETSYM
\] return RBRACKETSYM
\+ return PLUSSYM
\- return MINUSSYM
\* return MULTSYM
\^  return EXPSYM
= return SETSYM
:= return SETDELAYEDSYM
== return EQUALSYM

{ident}
    lval.val = &Symbol{string(y.buf)}
    return NAME

%%
    y.empty = true
    return int(c)
}