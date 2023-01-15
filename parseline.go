package groff

import (
	"fmt"
	"strings"
)

func ParseLine(line string) (Token, error) {
	if !strings.HasPrefix(line, ".") {
		return TokenText(line), nil
	}
	line = line[1:]
	ss := wsRe.Split(line, 2)
	if len(ss) > 1 {
		line = ss[1]
	} else {
		line = ""
	}
	switch ss[0] {

	case `.TH`:
		return ParseTokenMacroTH(line)

	case `.SH`:
		return ParseTokenMacroSH(line)

	case `.SS`:
		return ParseTokenMacroSS(line)

	case `.TP`:
		return ParseTokenMacroTP(line)

	case `.P`:
		return ParseTokenMacroP(line)

	case `.IP`:
		return ParseTokenMacroIP(line)

	case `.HP`:
		return ParseTokenMacroHP(line)

	case `.RS`:
		return ParseTokenMacroRS(line)

	case `.RE`:
		return ParseTokenMacroRE(line)

	case `.SM`:
		return ParseTokenMacroSM(line)

	case `.SB`:
		return ParseTokenMacroSB(line)

	case `.BI`:
		return ParseTokenMacroBI(line)

	case `.IB`:
		return ParseTokenMacroIB(line)

	case `.RI`:
		return ParseTokenMacroRI(line)

	case `.IR`:
		return ParseTokenMacroIR(line)

	case `.BR`:
		return ParseTokenMacroBR(line)

	case `.RB`:
		return ParseTokenMacroRB(line)

	case `.B`:
		return ParseTokenMacroB(line)

	case `.I`:
		return ParseTokenMacroI(line)

	case `.DT`:
		return ParseTokenMacroDT(line)

	case `.PD`:
		return ParseTokenMacroPD(line)

	case `.AT`:
		return ParseTokenMacroAT(line)

	case `.UC`:
		return ParseTokenMacroUC(line)

	case `.PT`:
		return ParseTokenMacroPT(line)

	case `.BT`:
		return ParseTokenMacroBT(line)

	case `.CT`:
		return ParseTokenMacroCT(line)

	case `.CW`:
		return ParseTokenMacroCW(line)

	case `.Ds`:
		return ParseTokenMacroDs(line)

	case `.De`:
		return ParseTokenMacroDe(line)

	case `.EX`:
		return ParseTokenMacroEX(line)

	case `.EE`:
		return ParseTokenMacroEE(line)

	case `.G`:
		return ParseTokenMacroG(line)

	case `.GL`:
		return ParseTokenMacroGL(line)

	case `.HB`:
		return ParseTokenMacroHB(line)

	case `.TB`:
		return ParseTokenMacroTB(line)

	case `.MS`:
		return ParseTokenMacroMS(line)

	case `.NT`:
		return ParseTokenMacroNT(line)

	case `.NE`:
		return ParseTokenMacroNE(line)

	case `.PN`:
		return ParseTokenMacroPN(line)

	case `.Pn`:
		return ParseTokenMacroPn(line)

	case `.R`:
		return ParseTokenMacroR(line)

	case `.RN`:
		return ParseTokenMacroRN(line)

	case `.VS`:
		return ParseTokenMacroVS(line)

	case `.VE`:
		return ParseTokenMacroVE(line)

	case `.RP`:
		return ParseTokenMacroRP(line)

	case `.P1`:
		return ParseTokenMacroP1(line)

	case `.DA`:
		return ParseTokenMacroDA(line)

	case `.ND`:
		return ParseTokenMacroND(line)

	case `.TL`:
		return ParseTokenMacroTL(line)

	case `.AU`:
		return ParseTokenMacroAU(line)

	case `.AI`:
		return ParseTokenMacroAI(line)

	case `.AB`:
		return ParseTokenMacroAB(line)

	case `.AE`:
		return ParseTokenMacroAE(line)

	case `.PP`:
		return ParseTokenMacroPP(line)

	case `.LP`:
		return ParseTokenMacroLP(line)

	case `.QP`:
		return ParseTokenMacroQP(line)

	case `.QE`:
		return ParseTokenMacroQE(line)

	case `.XP`:
		return ParseTokenMacroXP(line)

	case `.NH`:
		return ParseTokenMacroNH(line)

	case `.BX`:
		return ParseTokenMacroBX(line)

	case `.UL`:
		return ParseTokenMacroUL(line)

	case `.LG`:
		return ParseTokenMacroLG(line)

	case `.NL`:
		return ParseTokenMacroNL(line)

	case `.TA`:
		return ParseTokenMacroTA(line)

	case `.DE`:
		return ParseTokenMacroDE(line)

	case `.KE`:
		return ParseTokenMacroKE(line)

	case `.B2`:
		return ParseTokenMacroB2(line)

	case `.TE`:
		return ParseTokenMacroTE(line)

	case `.PE`:
		return ParseTokenMacroPE(line)

	case `.EN`:
		return ParseTokenMacroEN(line)

	case `.]`:
		return ParseTokenMacroSquareBracketClose(line)

	case `.FE`:
		return ParseTokenMacroFE(line)

	case `.EF`:
		return ParseTokenMacroEF(line)

	case `.1C`:
		return ParseTokenMacro1C(line)

	case `.2C`:
		return ParseTokenMacro2C(line)

	case `.MC`:
		return ParseTokenMacroMC(line)

	case `.XE`:
		return ParseTokenMacroXE(line)

	case `.TC`:
		return ParseTokenMacroTC(line)

	case `.PX`:
		return ParseTokenMacroPX(line)

	case `.AM`:
		return ParseTokenMacroAM(line)

	case `.DS`:
		return ParseTokenMacroDS(line)

	case `.IX`:
		return ParseTokenMacroIX(line)

	case `.XA`:
		return ParseTokenMacroXA(line)

	case `.XS`:
		return ParseTokenMacroXS(line)

	case `.HD`:
		return ParseTokenMacroHD(line)

	case `.OF`:
		return ParseTokenMacroOF(line)

	case `.EH`:
		return ParseTokenMacroEH(line)

	case `.OH`:
		return ParseTokenMacroOH(line)

	case `.FS`:
		return ParseTokenMacroFS(line)

	case `.[`:
		return ParseTokenMacroSquareBracketOpen(line)

	case `.EQ`:
		return ParseTokenMacroEQ(line)

	case `.PS`:
		return ParseTokenMacroPS(line)

	case `.TS`:
		return ParseTokenMacroTS(line)

	case `.B1`:
		return ParseTokenMacroB1(line)

	case `.KF`:
		return ParseTokenMacroKF(line)

	case `.KS`:
		return ParseTokenMacroKS(line)

	case `.RD`:
		return ParseTokenMacroRD(line)

	case `.CD`:
		return ParseTokenMacroCD(line)

	case `.BD`:
		return ParseTokenMacroBD(line)

	case `.ID`:
		return ParseTokenMacroID(line)

	case `.LD`:
		return ParseTokenMacroLD(line)

	case `.QS`:
		return ParseTokenMacroQS(line)

	}
	return nil, fmt.Errorf("%w: %#v: %s", ErrNoTokenMatch, ss[0], line)
}
