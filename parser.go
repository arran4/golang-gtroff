package groff

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNoTokenMatch = errors.New("no token match")
)

type Token interface {
	String() string
}

type TokenText string

func (t TokenText) String() string {
	return string(t)
}

var _ Token = (TokenText)("")

func ParseLine(line string) (Token, error) {
	if !strings.HasPrefix(line, ".") {
		return TokenText(line), nil
	}
	line = line[1:]
	ss := strings.SplitN(line, " ", 2)
	line = ss[1]
	switch ss[0] {
	case `\"`:
		return ParseComment(ss[1])
	case "TH":
		return ParseTokenTH(ss[1])
	case "SS":
		return ParseTokenSS(ss[1])
	case "TP":
		return ParseTokenTP(ss[1])
	case "P":
		return ParseTokenP(ss[1])
	case "HP":
		return ParseTokenHP(ss[1])
	case "SB":
		return ParseTokenSB(ss[1])
	case "IB":
		return ParseTokenIB(ss[1])
	case "RI":
		return ParseTokenRI(ss[1])
	case "IR":
		return ParseTokenIR(ss[1])
	case "BR":
		return ParseTokenBR(ss[1])
	case "RB":
		return ParseTokenRB(ss[1])
	case "DT":
		return ParseTokenDT(ss[1])
	case "PD":
		return ParseTokenPD(ss[1])
	case "AT":
		return ParseTokenAT(ss[1])
	case "UC":
		return ParseTokenUC(ss[1])
	case "CT":
		return ParseTokenCT(ss[1])
	case "EX":
		return ParseTokenEX(ss[1])
	case "EE":
		return ParseTokenEE(ss[1])
	case "G":
		return ParseTokenG(ss[1])
	case "GL":
		return ParseTokenGL(ss[1])
	case "HB":
		return ParseTokenHB(ss[1])
	case "TB":
		return ParseTokenTB(ss[1])
	case "MS":
		return ParseTokenMS(ss[1])
	case "NT":
		return ParseTokenNT(ss[1])
	case "NE":
		return ParseTokenNE(ss[1])
	case "PN":
		return ParseTokenPN(ss[1])
	case "Pn":
		return ParseTokenPn(ss[1])
	case "RN":
		return ParseTokenRN(ss[1])
	case "VS":
		return ParseTokenVS(ss[1])
	case "VE":
		return ParseTokenVE(ss[1])
	case "RP":
		return ParseTokenRP(ss[1])
	case "P1":
		return ParseTokenP1(ss[1])
	case "DA":
		return ParseTokenDA(ss[1])
	case "ND":
		return ParseTokenND(ss[1])
	case "TL":
		return ParseTokenTL(ss[1])
	case "AU":
		return ParseTokenAU(ss[1])
	case "AI":
		return ParseTokenAI(ss[1])
	case "AB":
		return ParseTokenAB(ss[1])
	case "AE":
		return ParseTokenAE(ss[1])
	case "PP":
		return ParseTokenPP(ss[1])
	case "LP":
		return ParseTokenLP(ss[1])
	case "QP":
		return ParseTokenQP(ss[1])
	case "QS":
		return ParseTokenQS(ss[1])
	case "QE":
		return ParseTokenQE(ss[1])
	case "XP":
		return ParseTokenXP(ss[1])
	case "NH":
		return ParseTokenNH(ss[1])
	case "SH":
		return ParseTokenSH(ss[1])
	case "B":
		return ParseTokenB(ss[1])
	case "R":
		return ParseTokenR(ss[1])
	case "I":
		return ParseTokenI(ss[1])
	case "BI":
		return ParseTokenBI(ss[1])
	case "BX":
		return ParseTokenBX(ss[1])
	case "UL":
		return ParseTokenUL(ss[1])
	case "LG":
		return ParseTokenLG(ss[1])
	case "SM":
		return ParseTokenSM(ss[1])
	case "NL":
		return ParseTokenNL(ss[1])
	case "IP":
		return ParseTokenIP(ss[1])
	case "RS":
		return ParseTokenRS(ss[1])
	case "RE":
		return ParseTokenRE(ss[1])
	case "TA":
		return ParseTokenTA(ss[1])
	case "LD":
		return ParseTokenLD(ss[1])
	case "ID":
		return ParseTokenID(ss[1])
	case "BD":
		return ParseTokenBD(ss[1])
	case "CD":
		return ParseTokenCD(ss[1])
	case "RD":
		return ParseTokenRD(ss[1])
	case "DE":
		return ParseTokenDE(ss[1])
	case "Ds":
		return ParseTokenDs(ss[1])
	case "De":
		return ParseTokenDe(ss[1])
	case "KS":
		return ParseTokenKS(ss[1])
	case "KF":
		return ParseTokenKF(ss[1])
	case "KE":
		return ParseTokenKE(ss[1])
	case "B1":
		return ParseTokenB1(ss[1])
	case "B2":
		return ParseTokenB2(ss[1])
	case "TS":
		return ParseTokenTS(ss[1])
	case "TE":
		return ParseTokenTE(ss[1])
	case "PS":
		return ParseTokenPS(ss[1])
	case "PE":
		return ParseTokenPE(ss[1])
	case "EQ":
		return ParseTokenEQ(ss[1])
	case "EN":
		return ParseTokenEN(ss[1])
	case "[":
		return ParseTokenReferOpen(ss[1])
	case "]":
		return ParseTokenReferClose(ss[1])
	case "FS":
		return ParseTokenFS(ss[1])
	case "FE":
		return ParseTokenFE(ss[1])
	case "OH":
		return ParseTokenOH(ss[1])
	case "EH":
		return ParseTokenEH(ss[1])
	case "OF":
		return ParseTokenOF(ss[1])
	case "EF":
		return ParseTokenEF(ss[1])
	case "PT":
		return ParseTokenPT(ss[1])
	case "HD":
		return ParseTokenHD(ss[1])
	case "BT":
		return ParseTokenBT(ss[1])
	case "1C":
		return ParseToken1C(ss[1])
	case "2C":
		return ParseToken2C(ss[1])
	case "MC":
		return ParseTokenMC(ss[1])
	case "XS":
		return ParseTokenXS(ss[1])
	case "XA":
		return ParseTokenXA(ss[1])
	case "XE":
		return ParseTokenXE(ss[1])
	case "TC":
		return ParseTokenTC(ss[1])
	case "PX":
		return ParseTokenPX(ss[1])
	case "AM":
		return ParseTokenAM(ss[1])
	case "DS":
		return ParseTokenDS(ss[1])
	case "CW":
		return ParseTokenCW(ss[1])
	case "IX":
		return ParseTokenIX(ss[1])
	}
	return nil, fmt.Errorf("%w: %s", ErrNoTokenMatch, line)
}

type TokenComment string

func (t TokenComment) String() string {
	return `.\" ` + string(t)
}

var _ Token = (TokenComment)("")

func ParseComment(s string) (Token, error) {
	return TokenComment(s), nil
}
