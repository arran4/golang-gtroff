package groff

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrNoTokenMatch  = errors.New("no token match")
	ErrUnclosedQuote = errors.New("quote wasn't closed")
)

type Token interface {
	String() string
}

type TokenText string

func (t TokenText) String() string {
	return string(t)
}

var _ Token = (TokenText)("")

var wsRe = regexp.MustCompile("\\s+")

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
	case `\"`:
		return ParseComment(line)
	case "TH":
		return ParseTokenTH(line)
	case "SS":
		return ParseTokenSS(line)
	case "TP":
		return ParseTokenTP(line)
	case "P":
		return ParseTokenP(line)
	case "HP":
		return ParseTokenHP(line)
	case "SB":
		return ParseTokenSB(line)
	case "IB":
		return ParseTokenIB(line)
	case "RI":
		return ParseTokenRI(line)
	case "IR":
		return ParseTokenIR(line)
	case "BR":
		return ParseTokenBR(line)
	case "RB":
		return ParseTokenRB(line)
	case "DT":
		return ParseTokenDT(line)
	case "PD":
		return ParseTokenPD(line)
	case "AT":
		return ParseTokenAT(line)
	case "UC":
		return ParseTokenUC(line)
	case "CT":
		return ParseTokenCT(line)
	case "EX":
		return ParseTokenEX(line)
	case "EE":
		return ParseTokenEE(line)
	case "G":
		return ParseTokenG(line)
	case "GL":
		return ParseTokenGL(line)
	case "HB":
		return ParseTokenHB(line)
	case "TB":
		return ParseTokenTB(line)
	case "MS":
		return ParseTokenMS(line)
	case "NT":
		return ParseTokenNT(line)
	case "NE":
		return ParseTokenNE(line)
	case "PN":
		return ParseTokenPN(line)
	case "Pn":
		return ParseTokenPn(line)
	case "RN":
		return ParseTokenRN(line)
	case "VS":
		return ParseTokenVS(line)
	case "VE":
		return ParseTokenVE(line)
	case "RP":
		return ParseTokenRP(line)
	case "P1":
		return ParseTokenP1(line)
	case "DA":
		return ParseTokenDA(line)
	case "ND":
		return ParseTokenND(line)
	case "TL":
		return ParseTokenTL(line)
	case "AU":
		return ParseTokenAU(line)
	case "AI":
		return ParseTokenAI(line)
	case "AB":
		return ParseTokenAB(line)
	case "AE":
		return ParseTokenAE(line)
	case "PP":
		return ParseTokenPP(line)
	case "LP":
		return ParseTokenLP(line)
	case "QP":
		return ParseTokenQP(line)
	case "QS":
		return ParseTokenQS(line)
	case "QE":
		return ParseTokenQE(line)
	case "XP":
		return ParseTokenXP(line)
	case "NH":
		return ParseTokenNH(line)
	case "SH":
		return ParseTokenSH(line)
	case "B":
		return ParseTokenB(line)
	case "R":
		return ParseTokenR(line)
	case "I":
		return ParseTokenI(line)
	case "BI":
		return ParseTokenBI(line)
	case "BX":
		return ParseTokenBX(line)
	case "UL":
		return ParseTokenUL(line)
	case "LG":
		return ParseTokenLG(line)
	case "SM":
		return ParseTokenSM(line)
	case "NL":
		return ParseTokenNL(line)
	case "IP":
		return ParseTokenIP(line)
	case "RS":
		return ParseTokenRS(line)
	case "RE":
		return ParseTokenRE(line)
	case "TA":
		return ParseTokenTA(line)
	case "LD":
		return ParseTokenLD(line)
	case "ID":
		return ParseTokenID(line)
	case "BD":
		return ParseTokenBD(line)
	case "CD":
		return ParseTokenCD(line)
	case "RD":
		return ParseTokenRD(line)
	case "DE":
		return ParseTokenDE(line)
	case "Ds":
		return ParseTokenDs(line)
	case "De":
		return ParseTokenDe(line)
	case "KS":
		return ParseTokenKS(line)
	case "KF":
		return ParseTokenKF(line)
	case "KE":
		return ParseTokenKE(line)
	case "B1":
		return ParseTokenB1(line)
	case "B2":
		return ParseTokenB2(line)
	case "TS":
		return ParseTokenTS(line)
	case "TE":
		return ParseTokenTE(line)
	case "PS":
		return ParseTokenPS(line)
	case "PE":
		return ParseTokenPE(line)
	case "EQ":
		return ParseTokenEQ(line)
	case "EN":
		return ParseTokenEN(line)
	case "[":
		return ParseTokenReferOpen(line)
	case "]":
		return ParseTokenReferClose(line)
	case "FS":
		return ParseTokenFS(line)
	case "FE":
		return ParseTokenFE(line)
	case "OH":
		return ParseTokenOH(line)
	case "EH":
		return ParseTokenEH(line)
	case "OF":
		return ParseTokenOF(line)
	case "EF":
		return ParseTokenEF(line)
	case "PT":
		return ParseTokenPT(line)
	case "HD":
		return ParseTokenHD(line)
	case "BT":
		return ParseTokenBT(line)
	case "1C":
		return ParseToken1C(line)
	case "2C":
		return ParseToken2C(line)
	case "MC":
		return ParseTokenMC(line)
	case "XS":
		return ParseTokenXS(line)
	case "XA":
		return ParseTokenXA(line)
	case "XE":
		return ParseTokenXE(line)
	case "TC":
		return ParseTokenTC(line)
	case "PX":
		return ParseTokenPX(line)
	case "AM":
		return ParseTokenAM(line)
	case "DS":
		return ParseTokenDS(line)
	case "CW":
		return ParseTokenCW(line)
	case "IX":
		return ParseTokenIX(line)
	case "ig":
		return ParseTokenig(line)
	case "rr":
		return ParseTokenrr(line)
	case "rnn":
		return ParseTokenrnn(line)
	case "aln":
		return ParseTokenaln(line)
	case "nr":
		return ParseTokennr(line)
	case "af":
		return ParseTokenaf(line)
	case "br":
		return ParseTokenbr(line)
	case "fi":
		return ParseTokenfi(line)
	case "nf":
		return ParseTokennf(line)
	case "ad":
		return ParseTokenad(line)
	case "na":
		return ParseTokenna(line)
	case "brp":
		return ParseTokenbrp(line)
	case "ss":
		return ParseTokenss(line)
	case "ce":
		return ParseTokence(line)
	case "rj":
		return ParseTokenrj(line)
	case "hy":
		return ParseTokenhy(line)
	case "nh":
		return ParseTokennh(line)
	case "hlm":
		return ParseTokenhlm(line)
	case "hw":
		return ParseTokenhw(line)
	case "hc":
		return ParseTokenhc(line)
	case "hpf":
		return ParseTokenhpf(line)
	case "hpfa":
		return ParseTokenhpfa(line)
	case "hpfcode":
		return ParseTokenhpfcode(line)
	case "hcode":
		return ParseTokenhcode(line)
	case "hym":
		return ParseTokenhym(line)
	case "hys":
		return ParseTokenhys(line)
	case "shc":
		return ParseTokenshc(line)
	case "hla":
		return ParseTokenhla(line)
	case "sp":
		return ParseTokensp(line)
	case "ls":
		return ParseTokenls(line)
	case "ns":
		return ParseTokenns(line)
	case "rs":
		return ParseTokenrs(line)
	case "ta":
		return ParseTokenta(line)
	case "tc":
		return ParseTokentc(line)
	case "linetabs":
		return ParseTokenlinetabs(line)
	case "lc":
		return ParseTokenlc(line)
	case "fc":
		return ParseTokenfc(line)
	case "cc":
		return ParseTokencc(line)
	case "c2":
		return ParseTokenc2(line)
	case "eo":
		return ParseTokeneo(line)
	case "ec":
		return ParseTokenec(line)
	case "ecs":
		return ParseTokenecs(line)
	case "ecr":
		return ParseTokenecr(line)
	case "tr":
		return ParseTokentr(line)
	case "trin":
		return ParseTokentrin(line)
	case "trnt":
		return ParseTokentrnt(line)
	case "troff":
		return ParseTokentroff(line)
	case "nroff":
		return ParseTokennroff(line)
	case "po":
		return ParseTokenpo(line)
	case "in":
		return ParseTokenin(line)
	case "ti":
		return ParseTokenti(line)
	case "ll":
		return ParseTokenll(line)
	case "pl":
		return ParseTokenpl(line)
	case "tl":
		return ParseTokentl(line)
	case "lt":
		return ParseTokenlt(line)
	case "pn":
		return ParseTokenpn(line)
	case "pc":
		return ParseTokenpc(line)
	case "bp":
		return ParseTokenbp(line)
	case "ne":
		return ParseTokenne(line)
	case "sv":
		return ParseTokensv(line)
	case "os":
		return ParseTokenos(line)
	case "ftr":
		return ParseTokenftr(line)
	case "fzoom":
		return ParseTokenfzoom(line)
	case "fam":
		return ParseTokenfam(line)
	case "sty":
		return ParseTokensty(line)
	case "fp":
		return ParseTokenfp(line)
	case "ft":
		return ParseTokenft(line)
	case "composite":
		return ParseTokencomposite(line)
	case "cflags":
		return ParseTokencflags(line)
	case "char":
		return ParseTokenchar(line)
	case "fchar":
		return ParseTokenfchar(line)
	case "fschar":
		return ParseTokenfschar(line)
	case "schar":
		return ParseTokenschar(line)
	case "rchar":
		return ParseTokenrchar(line)
	case "rfschar":
		return ParseTokenrfschar(line)
	case "class":
		return ParseTokenclass(line)
	case "special":
		return ParseTokenspecial(line)
	case "fspecial":
		return ParseTokenfspecial(line)
	case "ul":
		return ParseTokenul(line)
	case "cu":
		return ParseTokencu(line)
	case "uf":
		return ParseTokenuf(line)
	case "bd":
		return ParseTokenbd(line)
	case "cs":
		return ParseTokencs(line)
	case "lg":
		return ParseTokenlg(line)
	case "kern":
		return ParseTokenkern(line)
	case "tkf":
		return ParseTokentkf(line)
	case "ps":
		return ParseTokenps(line)
	case "sizes":
		return ParseTokensizes(line)
	case "vs":
		return ParseTokenvs(line)
	case "pvs":
		return ParseTokenpvs(line)
	case "ds":
		return ParseTokends(line)
	case "ds1":
		return ParseTokends1(line)
	case "as":
		return ParseTokenas(line)
	case "as1":
		return ParseTokenas1(line)
	case "substring":
		return ParseTokensubstring(line)
	case "length":
		return ParseTokenlength(line)
	case "rn":
		return ParseTokenrn(line)
	case "rm":
		return ParseTokenrm(line)
	case "als":
		return ParseTokenals(line)
	case "chop":
		return ParseTokenchop(line)
	case "if":
		return ParseTokenif(line)
	case "nop":
		return ParseTokennop(line)
	case "ie":
		return ParseTokenie(line)
	case "el":
		return ParseTokenel(line)
	case "while":
		return ParseTokenwhile(line)
	case "break":
		return ParseTokenbreak(line)
	case "continue":
		return ParseTokencontinue(line)
	case "de":
		return ParseTokende(line)
	case "de1":
		return ParseTokende1(line)
	case "dei":
		return ParseTokendei(line)
	case "dei1":
		return ParseTokendei1(line)
	case "am":
		return ParseTokenam(line)
	case "am1":
		return ParseTokenam1(line)
	case "ami":
		return ParseTokenami(line)
	case "ami1":
		return ParseTokenami1(line)
	case "return":
		return ParseTokenreturn(line)
	case "shift":
		return ParseTokenshift(line)
	case "mk":
		return ParseTokenmk(line)
	case "rt":
		return ParseTokenrt(line)
	case "vpt":
		return ParseTokenvpt(line)
	case "wh":
		return ParseTokenwh(line)
	case "ch":
		return ParseTokench(line)
	case "dt":
		return ParseTokendt(line)
	case "it":
		return ParseTokenit(line)
	case "itc":
		return ParseTokenitc(line)
	case "blm":
		return ParseTokenblm(line)
	case "lsm":
		return ParseTokenlsm(line)
	case "em":
		return ParseTokenem(line)
	case "di":
		return ParseTokendi(line)
	case "da":
		return ParseTokenda(line)
	case "box":
		return ParseTokenbox(line)
	case "boxa":
		return ParseTokenboxa(line)
	case "output":
		return ParseTokenoutput(line)
	case "asciify":
		return ParseTokenasciify(line)
	case "unformat":
		return ParseTokenunformat(line)
	case "ev":
		return ParseTokenev(line)
	case "evc":
		return ParseTokenevc(line)
	case "color":
		return ParseTokencolor(line)
	case "defcolor":
		return ParseTokendefcolor(line)
	case "gcolor":
		return ParseTokengcolor(line)
	case "fcolor":
		return ParseTokenfcolor(line)
	case "so":
		return ParseTokenso(line)
	case "pso":
		return ParseTokenpso(line)
	case "mso":
		return ParseTokenmso(line)
	case "trf":
		return ParseTokentrf(line)
	case "cf":
		return ParseTokencf(line)
	case "nx":
		return ParseTokennx(line)
	case "rd":
		return ParseTokenrd(line)
	case "pi":
		return ParseTokenpi(line)
	case "sy":
		return ParseTokensy(line)
	case "open":
		return ParseTokenopen(line)
	case "opena":
		return ParseTokenopena(line)
	case "write":
		return ParseTokenwrite(line)
	case "writec":
		return ParseTokenwritec(line)
	case "writem":
		return ParseTokenwritem(line)
	case "close":
		return ParseTokenclose(line)
	case "device":
		return ParseTokendevice(line)
	case "devicem":
		return ParseTokendevicem(line)
	case "nm":
		return ParseTokennm(line)
	case "nn":
		return ParseTokennn(line)
	case "mc":
		return ParseTokenmc(line)
	case "psbb":
		return ParseTokenpsbb(line)
	case "lf":
		return ParseTokenlf(line)
	case "tm":
		return ParseTokentm(line)
	case "tm1":
		return ParseTokentm1(line)
	case "tmc":
		return ParseTokentmc(line)
	case "ab":
		return ParseTokenab(line)
	case "ex":
		return ParseTokenex(line)
	case "pev":
		return ParseTokenpev(line)
	case "pm":
		return ParseTokenpm(line)
	case "pnr":
		return ParseTokenpnr(line)
	case "ptr":
		return ParseTokenptr(line)
	case "fl":
		return ParseTokenfl(line)
	case "backtrace":
		return ParseTokenbacktrace(line)
	case "warnscale":
		return ParseTokenwarnscale(line)
	case "spreadwarn":
		return ParseTokenspreadwarn(line)
	case "warn":
		return ParseTokenwarn(line)
	case "cp":
		return ParseTokencp(line)
	case "do":
		return ParseTokendo(line)
	}
	return nil, fmt.Errorf("%w: %#v: %s", ErrNoTokenMatch, ss[0], line)
}

type TokenComment string

func (t TokenComment) String() string {
	return `.\" ` + string(t)
}

var _ Token = (TokenComment)("")

func ParseComment(s string) (Token, error) {
	return TokenComment(s), nil
}

func ArgumentParser(l string) ([]string, int, error) {
	stack := []rune{}
	result := []string{}
	a := ""
	acomment := false
	i := 0
	rs := []rune(l)
	for ; i < len(rs) && !acomment; i++ {
		switch rs[i] {
		case ' ':
			if len(stack) == 0 {
				if len(a) != 0 {
					result = append(result, a)
				}
				a = ""
				continue
			}
		case '\\':
			i++
			if len(stack) == 0 && rs[i] == '"' {
				acomment = true
				i -= 2
				continue
			}
		case '"':
			if len(stack) > 0 && stack[len(stack)-1] == '"' {
				stack = stack[:len(stack)-1]
			} else {
				stack = append(stack, '"')
			}
			continue
		}
		a += string([]byte{byte(rs[i])})
	}
	if len(stack) > 0 {
		return nil, i, fmt.Errorf("%w: stack %s", ErrUnclosedQuote, string(stack))
	}
	if len(a) > 0 {
		result = append(result, a)
	}
	return result, i, nil
}
