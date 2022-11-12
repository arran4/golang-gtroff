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

// TokenTH Set the title of the man page to title and the section to
// section, which must have a value between 1 and 8.  The value
// of section may also have a string appended, e.g. ‘.pm’, to
// indicate a specific subsection of the man pages.
//
// Both title and section are positioned at the left and right
// in the header line (with section in parentheses immediately
// appended to title.  extra1 is positioned in the middle of
// the footer line.  extra2 is positioned at the left in the footer
// line (or at the left on even pages and at the right on odd pages if
// double-sided printing is active).  extra3 is centered in the
// header line.
//
// For HTML and XHTML output, headers and footers are
// completely suppressed.
//
// Additionally, this macro starts a new page; the new line number
// is 1 again (except if the -rC1 option is given on the
// command line) – this feature is intended only for formatting multiple
// man pages; a single man page should contain exactly one TH macro
// at the beginning of the file.
type TokenTH struct { /* title */
}

func ParseTokenTH(args string) (Token, error) {
	return &TokenTH{ /* title */ }, nil
}

func (t *TokenTH) String() string {
	return ".TH" /* title */
}

var _ Token = (*TokenTH)(nil)

// TokenSS Set up an unnumbered (sub)section heading.  Prints out all the text
// following SS up to the end of the line (or the text in the next
// line if there is no argument to SS) in bold face (or the font
// specified by the string HF), at the same size as the base
// document size.  Additionally, the left margin and the indentation for
// the following text is reset to its default value.
type TokenSS struct { /* [heading] */
}

func ParseTokenSS(args string) (Token, error) {
	return &TokenSS{ /* [heading] */ }, nil
}

func (t *TokenSS) String() string {
	return ".SS" /* [heading] */
}

var _ Token = (*TokenSS)(nil)

// TokenTP Set up an indented paragraph with label.  The indentation is set to
// nnn if that argument is supplied (the default unit is ‘n’ if
// omitted), otherwise it is set to the previous indentation value
// specified with TP, IP, or HP (or to the default
// value if none of them have been used yet).
//
// The first line of text following this macro is interpreted as a string
// to be printed flush-left, as it is appropriate for a label.  It is not
// interpreted as part of a paragraph, so there is no attempt to fill the
// first line with text from the following input lines.  Nevertheless, if
// the label is not as wide as the indentation the paragraph starts at the
// same line (but indented), continuing on the following lines.  If the
// label is wider than the indentation the descriptive part of the
// paragraph begins on the line following the label, entirely indented.
// Note that neither font shape nor font size of the label is set to a
// default value; on the other hand, the rest of the text has default font
// settings.
type TokenTP struct { /* [nnn] */
}

func ParseTokenTP(args string) (Token, error) {
	return &TokenTP{ /* [nnn] */ }, nil
}

func (t *TokenTP) String() string {
	return ".TP" /* [nnn] */
}

var _ Token = (*TokenTP)(nil)

// TokenP These macros are mutual aliases.  Any of them causes a line break at the
// current position, followed by a vertical space downwards by the amount
// specified by the PD macro.  The font size and shape are reset to
// the default value (10pt roman if no -rS option is given
// on the command line).  Finally, the current left margin and the
// indentation is restored.
type TokenP struct { /*  */
}

func ParseTokenP(args string) (Token, error) {
	return &TokenP{ /*  */ }, nil
}

func (t *TokenP) String() string {
	return ".P" /*  */
}

var _ Token = (*TokenP)(nil)

// TokenHP Set up a paragraph with hanging left indentation.  The indentation is
// set to nnn if that argument is supplied (default unit is
// ‘n’), otherwise it is set to the previous indentation value
// specified with TP, IP, or HP (or the default value
// if non of them have been used yet).  Font size and face are reset to
// their default values.
type TokenHP struct { /* [nnn] */
}

func ParseTokenHP(args string) (Token, error) {
	return &TokenHP{ /* [nnn] */ }, nil
}

func (t *TokenHP) String() string {
	return ".HP" /* [nnn] */
}

var _ Token = (*TokenHP)(nil)

// TokenSB Set the text on the same line or the text on the next line in bold face
// font, one point size smaller than the default font.
type TokenSB struct { /* [text] */
}

func ParseTokenSB(args string) (Token, error) {
	return &TokenSB{ /* [text] */ }, nil
}

func (t *TokenSB) String() string {
	return ".SB" /* [text] */
}

var _ Token = (*TokenSB)(nil)

// TokenIB Set its arguments alternately in italic and bold face, without a space
// between the arguments.
type TokenIB struct { /* text */
}

func ParseTokenIB(args string) (Token, error) {
	return &TokenIB{ /* text */ }, nil
}

func (t *TokenIB) String() string {
	return ".IB" /* text */
}

var _ Token = (*TokenIB)(nil)

// TokenRI Set its arguments alternately in roman and italic, without a space
// between the arguments.
type TokenRI struct { /* text */
}

func ParseTokenRI(args string) (Token, error) {
	return &TokenRI{ /* text */ }, nil
}

func (t *TokenRI) String() string {
	return ".RI" /* text */
}

var _ Token = (*TokenRI)(nil)

// TokenIR Set its arguments alternately in italic and roman, without a space
// between the arguments.
type TokenIR struct { /* text */
}

func ParseTokenIR(args string) (Token, error) {
	return &TokenIR{ /* text */ }, nil
}

func (t *TokenIR) String() string {
	return ".IR" /* text */
}

var _ Token = (*TokenIR)(nil)

// TokenBR Set its arguments alternately in bold face and roman, without a space
// between the arguments.
type TokenBR struct { /* text */
}

func ParseTokenBR(args string) (Token, error) {
	return &TokenBR{ /* text */ }, nil
}

func (t *TokenBR) String() string {
	return ".BR" /* text */
}

var _ Token = (*TokenBR)(nil)

// TokenRB Set its arguments alternately in roman and bold face, without a space
// between the arguments.
type TokenRB struct { /* text */
}

func ParseTokenRB(args string) (Token, error) {
	return &TokenRB{ /* text */ }, nil
}

func (t *TokenRB) String() string {
	return ".RB" /* text */
}

var _ Token = (*TokenRB)(nil)

// TokenDT Set tabs every 0.5 inches.  Since this macro is always executed
// during a call to the TH macro, it makes sense to call it only if
// the tab positions have been changed.
type TokenDT struct { /*  */
}

func ParseTokenDT(args string) (Token, error) {
	return &TokenDT{ /*  */ }, nil
}

func (t *TokenDT) String() string {
	return ".DT" /*  */
}

var _ Token = (*TokenDT)(nil)

// TokenPD Adjust the empty space before a new paragraph (or section).  The
// optional argument gives the amount of space (default unit is ‘v’);
// without parameter, the value is reset to its default value (1 line
// in nroff mode, 0.4v otherwise).
//
// This affects the macros SH, SS, TP, LP (as
// well as PP and P), IP, and HP.
type TokenPD struct { /* [nnn] */
}

func ParseTokenPD(args string) (Token, error) {
	return &TokenPD{ /* [nnn] */ }, nil
}

func (t *TokenPD) String() string {
	return ".PD" /* [nnn] */
}

var _ Token = (*TokenPD)(nil)

// TokenAT Alter the footer for use with AT&T manpages.  This command
// exists only for compatibility; don’t use it.  The first argument
// system can be:
//
// 3
// 7th Edition (the default)
//
// 4
// System III
//
// 5
// System V
//
// An optional second argument release to AT specifies the
// release number (such as “System V Release 3”).
type TokenAT struct { /* [system */
}

func ParseTokenAT(args string) (Token, error) {
	return &TokenAT{ /* [system */ }, nil
}

func (t *TokenAT) String() string {
	return ".AT" /* [system */
}

var _ Token = (*TokenAT)(nil)

// TokenUC Alters the footer for use with BSD manpages.  This command exists only
// for compatibility; don’t use it.  The argument can be:
//
// 3
// 3rd Berkeley Distribution (the default)
//
// 4
// 4th Berkeley Distribution
//
// 5
// 4.2 Berkeley Distribution
//
// 6
// 4.3 Berkeley Distribution
//
// 7
// 4.4 Berkeley Distribution
type TokenUC struct { /* [version] */
}

func ParseTokenUC(args string) (Token, error) {
	return &TokenUC{ /* [version] */ }, nil
}

func (t *TokenUC) String() string {
	return ".UC" /* [version] */
}

var _ Token = (*TokenUC)(nil)

// TokenCT Print ‘<CTRL/key>’.
type TokenCT struct { /* key */
}

func ParseTokenCT(args string) (Token, error) {
	return &TokenCT{ /* key */ }, nil
}

func (t *TokenCT) String() string {
	return ".CT" /* key */
}

var _ Token = (*TokenCT)(nil)

// TokenEX Begin a non-filled display using the constant width (Courier) typeface.
// Use the optional indent argument to indent the display.
type TokenEX struct { /* [indent] */
}

func ParseTokenEX(args string) (Token, error) {
	return &TokenEX{ /* [indent] */ }, nil
}

func (t *TokenEX) String() string {
	return ".EX" /* [indent] */
}

var _ Token = (*TokenEX)(nil)

// TokenEE End a non-filled display started with EX.
type TokenEE struct { /*  */
}

func ParseTokenEE(args string) (Token, error) {
	return &TokenEE{ /*  */ }, nil
}

func (t *TokenEE) String() string {
	return ".EE" /*  */
}

var _ Token = (*TokenEE)(nil)

// TokenG Set text in Helvetica.  If no text is present on the line where
// the macro is called, then the text of the next line appears in
// Helvetica.
type TokenG struct { /* [text] */
}

func ParseTokenG(args string) (Token, error) {
	return &TokenG{ /* [text] */ }, nil
}

func (t *TokenG) String() string {
	return ".G" /* [text] */
}

var _ Token = (*TokenG)(nil)

// TokenGL Set text in Helvetica Oblique.  If no text is present on the line
// where the macro is called, then the text of the next line appears in
// Helvetica Oblique.
type TokenGL struct { /* [text] */
}

func ParseTokenGL(args string) (Token, error) {
	return &TokenGL{ /* [text] */ }, nil
}

func (t *TokenGL) String() string {
	return ".GL" /* [text] */
}

var _ Token = (*TokenGL)(nil)

// TokenHB Set text in Helvetica Bold.  If no text is present on the line
// where the macro is called, then all text up to the next HB
// appears in Helvetica Bold.
type TokenHB struct { /* [text] */
}

func ParseTokenHB(args string) (Token, error) {
	return &TokenHB{ /* [text] */ }, nil
}

func (t *TokenHB) String() string {
	return ".HB" /* [text] */
}

var _ Token = (*TokenHB)(nil)

// TokenTB Identical to HB.
type TokenTB struct { /* [text] */
}

func ParseTokenTB(args string) (Token, error) {
	return &TokenTB{ /* [text] */ }, nil
}

func (t *TokenTB) String() string {
	return ".TB" /* [text] */
}

var _ Token = (*TokenTB)(nil)

// TokenMS Set a manpage reference in Ultrix format.  The title is in Courier
// instead of italic.  Optional punctuation follows the section number
// without an intervening space.
type TokenMS struct { /* title */
}

func ParseTokenMS(args string) (Token, error) {
	return &TokenMS{ /* title */ }, nil
}

func (t *TokenMS) String() string {
	return ".MS" /* title */
}

var _ Token = (*TokenMS)(nil)

// TokenNT Begin a note.  Print the optional title, or the word “Note”,
// centered on the page.  Text following the macro makes up the body of the
// note, and is indented on both sides.  If the first argument is C,
// the body of the note is printed centered (the second argument replaces
// the word “Note” if specified).
type TokenNT struct { /* [C] */
}

func ParseTokenNT(args string) (Token, error) {
	return &TokenNT{ /* [C] */ }, nil
}

func (t *TokenNT) String() string {
	return ".NT" /* [C] */
}

var _ Token = (*TokenNT)(nil)

// TokenNE End a note begun with NT.
type TokenNE struct { /*  */
}

func ParseTokenNE(args string) (Token, error) {
	return &TokenNE{ /*  */ }, nil
}

func (t *TokenNE) String() string {
	return ".NE" /*  */
}

var _ Token = (*TokenNE)(nil)

// TokenPN Set the path name in constant width (Courier), followed by optional
// punctuation.
type TokenPN struct { /* path */
}

func ParseTokenPN(args string) (Token, error) {
	return &TokenPN{ /* path */ }, nil
}

func (t *TokenPN) String() string {
	return ".PN" /* path */
}

var _ Token = (*TokenPN)(nil)

// TokenPn If called with two arguments, identical to PN.  If called with
// three arguments, set the second argument in constant width (Courier),
// bracketed by the first and third arguments in the current font.
type TokenPn struct { /* [punct] */
}

func ParseTokenPn(args string) (Token, error) {
	return &TokenPn{ /* [punct] */ }, nil
}

func (t *TokenPn) String() string {
	return ".Pn" /* [punct] */
}

var _ Token = (*TokenPn)(nil)

// TokenRN Print the string ‘<RETURN>’.
type TokenRN struct { /*  */
}

func ParseTokenRN(args string) (Token, error) {
	return &TokenRN{ /*  */ }, nil
}

func (t *TokenRN) String() string {
	return ".RN" /*  */
}

var _ Token = (*TokenRN)(nil)

// TokenVS Start printing a change bar in the margin if the number 4 is
// specified.  Otherwise, this macro does nothing.
type TokenVS struct { /* [4] */
}

func ParseTokenVS(args string) (Token, error) {
	return &TokenVS{ /* [4] */ }, nil
}

func (t *TokenVS) String() string {
	return ".VS" /* [4] */
}

var _ Token = (*TokenVS)(nil)

// TokenVE End printing the change bar begun by VS.
type TokenVE struct { /*  */
}

func ParseTokenVE(args string) (Token, error) {
	return &TokenVE{ /*  */ }, nil
}

func (t *TokenVE) String() string {
	return ".VE" /*  */
}

var _ Token = (*TokenVE)(nil)

// TokenRP Specifies the report format for your document.  The report format
// creates a separate cover page.  The default action (no RP macro)
// is to print a subset of the cover page on page 1 of your document.
//
// If you use the word no as an optional argument, groff
// prints a title page but does not repeat any of the title page
// information (title, author, abstract, etc.) on page 1 of the
// document.
type TokenRP struct { /* [no] */
}

func ParseTokenRP(args string) (Token, error) {
	return &TokenRP{ /* [no] */ }, nil
}

func (t *TokenRP) String() string {
	return ".RP" /* [no] */
}

var _ Token = (*TokenRP)(nil)

// TokenP1 (P-one) Prints the header on page 1.  The default is to suppress
// the header.
type TokenP1 struct { /*  */
}

func ParseTokenP1(args string) (Token, error) {
	return &TokenP1{ /*  */ }, nil
}

func (t *TokenP1) String() string {
	return ".P1" /*  */
}

var _ Token = (*TokenP1)(nil)

// TokenDA (optional) Prints the current date, or the arguments to the macro if
// any, on the title page (if specified) and in the footers.  This is the
// default for nroff.
type TokenDA struct { /* […] */
}

func ParseTokenDA(args string) (Token, error) {
	return &TokenDA{ /* […] */ }, nil
}

func (t *TokenDA) String() string {
	return ".DA" /* […] */
}

var _ Token = (*TokenDA)(nil)

// TokenND (optional) Prints the current date, or the arguments to the macro if
// any, on the title page (if specified) but not in the footers.  This is
// the default for troff.
type TokenND struct { /* […] */
}

func ParseTokenND(args string) (Token, error) {
	return &TokenND{ /* […] */ }, nil
}

func (t *TokenND) String() string {
	return ".ND" /* […] */
}

var _ Token = (*TokenND)(nil)

// TokenTL Specifies the document title.  groff collects text following the
// TL macro into the title, until reaching the author name or
// abstract.
type TokenTL struct { /*  */
}

func ParseTokenTL(args string) (Token, error) {
	return &TokenTL{ /*  */ }, nil
}

func (t *TokenTL) String() string {
	return ".TL" /*  */
}

var _ Token = (*TokenTL)(nil)

// TokenAU Specifies the author’s name, which appears on the line (or lines)
// immediately following.  You can specify multiple authors as follows:
//
// .AU
// John Doe
// .AI
// University of West Bumblefuzz
// .AU
// Martha Buck
// .AI
// Monolithic Corporation
//
// ...
type TokenAU struct { /*  */
}

func ParseTokenAU(args string) (Token, error) {
	return &TokenAU{ /*  */ }, nil
}

func (t *TokenAU) String() string {
	return ".AU" /*  */
}

var _ Token = (*TokenAU)(nil)

// TokenAI Specifies the author’s institution.  You can specify multiple
// institutions in the same way that you specify multiple authors.
type TokenAI struct { /*  */
}

func ParseTokenAI(args string) (Token, error) {
	return &TokenAI{ /*  */ }, nil
}

func (t *TokenAI) String() string {
	return ".AI" /*  */
}

var _ Token = (*TokenAI)(nil)

// TokenAB Begins the abstract.  The default is to print the word
// ABSTRACT, centered and in italics, above the text of the
// abstract.  The word no as an optional argument suppresses this
// heading.
type TokenAB struct { /* [no] */
}

func ParseTokenAB(args string) (Token, error) {
	return &TokenAB{ /* [no] */ }, nil
}

func (t *TokenAB) String() string {
	return ".AB" /* [no] */
}

var _ Token = (*TokenAB)(nil)

// TokenAE Ends the abstract.
type TokenAE struct { /*  */
}

func ParseTokenAE(args string) (Token, error) {
	return &TokenAE{ /*  */ }, nil
}

func (t *TokenAE) String() string {
	return ".AE" /*  */
}

var _ Token = (*TokenAE)(nil)

// TokenPP Sets a paragraph with an initial indentation.
type TokenPP struct { /*  */
}

func ParseTokenPP(args string) (Token, error) {
	return &TokenPP{ /*  */ }, nil
}

func (t *TokenPP) String() string {
	return ".PP" /*  */
}

var _ Token = (*TokenPP)(nil)

// TokenLP Sets a paragraph without an initial indentation.
type TokenLP struct { /*  */
}

func ParseTokenLP(args string) (Token, error) {
	return &TokenLP{ /*  */ }, nil
}

func (t *TokenLP) String() string {
	return ".LP" /*  */
}

var _ Token = (*TokenLP)(nil)

// TokenQP Sets a paragraph that is indented at both left and right margins
// by the amount of the register QI.
// The effect is identical to the HTML <BLOCKQUOTE> element.
// The next paragraph or heading returns margins to normal.
// QP inserts vertical space of amount set by register PD
// before the paragraph.
type TokenQP struct { /*  */
}

func ParseTokenQP(args string) (Token, error) {
	return &TokenQP{ /*  */ }, nil
}

func (t *TokenQP) String() string {
	return ".QP" /*  */
}

var _ Token = (*TokenQP)(nil)

// TokenQS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenQS struct { /*  */
}

func ParseTokenQS(args string) (Token, error) {
	return &TokenQS{ /*  */ }, nil
}

func (t *TokenQS) String() string {
	return ".QS" /*  */
}

var _ Token = (*TokenQS)(nil)

// TokenQE These macros begin and end a quoted section.
// The QI register
// controls the amount of indentation.
// Both QS and QE insert inter-paragraph vertical space
// set by register PD.
// The text between QS and QE can be structured further
// by use of the macros LP or PP.
type TokenQE struct { /*  */
}

func ParseTokenQE(args string) (Token, error) {
	return &TokenQE{ /*  */ }, nil
}

func (t *TokenQE) String() string {
	return ".QE" /*  */
}

var _ Token = (*TokenQE)(nil)

// TokenXP Sets a paragraph whose lines are indented, except for the first line.
// This is a Berkeley extension.
type TokenXP struct { /*  */
}

func ParseTokenXP(args string) (Token, error) {
	return &TokenXP{ /*  */ }, nil
}

func (t *TokenXP) String() string {
	return ".XP" /*  */
}

var _ Token = (*TokenXP)(nil)

// TokenNH Numbered heading.  The argument is either a numeric argument to indicate
// the level of the heading, or the letter S followed by
// numeric arguments to set the heading level explicitly.
//
// If you specify heading levels out of sequence, such as invoking
// ‘.NH 3’ after ‘.NH 1’, groff prints a warning on
// standard error.
type TokenNH struct { /* S | curr-level */
}

func ParseTokenNH(args string) (Token, error) {
	return &TokenNH{ /* S | curr-level */ }, nil
}

func (t *TokenNH) String() string {
	return ".NH" /* S | curr-level */
}

var _ Token = (*TokenNH)(nil)

// TokenSH Unnumbered subheading.
//
// The optional match-level argument is a GNU extension.  It is a
// number indicating the level of the heading, in a manner analogous to the
// curr-level argument to .NH.  Its purpose is to match the
// point size, at which the heading is printed, to the size of a numbered
// heading at the same level, when the GROWPS and PSINCR
// heading size adjustment mechanism is in effect.  See ms Document Control Registers.
// Set up an unnumbered section heading sticking out to the left.  Prints
// out all the text following SH up to the end of the line (or the
// text in the next line if there is no argument to SH) in bold face
// (or the font specified by the string HF), one size larger than
// the base document size.  Additionally, the left margin and the
// indentation for the following text is reset to its default value.
type TokenSH struct { /* [match-level] | [heading] */
}

func ParseTokenSH(args string) (Token, error) {
	return &TokenSH{ /* [match-level] | [heading] */ }, nil
}

func (t *TokenSH) String() string {
	return ".SH" /* [match-level] | [heading] */
}

var _ Token = (*TokenSH)(nil)

// TokenB Sets its first argument in bold type.  If you specify a second
// argument, groff prints it in the previous font after the bold
// text, with no intervening space (this allows you to set punctuation
// after the highlighted text without highlighting the punctuation).
// Similarly, it prints the third argument (if any) in the previous font
// before the first argument.  For example,
//
// .B foo ) (
//
// prints (foo).
//
// If you give this macro no arguments, groff prints all text
// following in bold until the next highlighting, paragraph, or heading
// macro.
// Set text in bold face.  If no text is present on the line where
// the macro is called, then the text of the next line appears in bold
// face.
type TokenB struct { /* [txt | [text] */
}

func ParseTokenB(args string) (Token, error) {
	return &TokenB{ /* [txt | [text] */ }, nil
}

func (t *TokenB) String() string {
	return ".B" /* [txt | [text] */
}

var _ Token = (*TokenB)(nil)

// TokenR Sets its first argument in roman (or regular) type.  It operates
// similarly to the B macro otherwise.
// Switch to roman font and turn off any underlining in effect.
type TokenR struct { /* [txt | undefined */
}

func ParseTokenR(args string) (Token, error) {
	return &TokenR{ /* [txt | undefined */ }, nil
}

func (t *TokenR) String() string {
	return ".R" /* [txt | undefined */
}

var _ Token = (*TokenR)(nil)

// TokenI Sets its first argument in italic type.  It operates similarly
// to the B macro otherwise.
// Set text in italic.  If no text is present on the line where the
// macro is called, then the text of the next line appears in italic.
type TokenI struct { /* [txt | [text] */
}

func ParseTokenI(args string) (Token, error) {
	return &TokenI{ /* [txt | [text] */ }, nil
}

func (t *TokenI) String() string {
	return ".I" /* [txt | [text] */
}

var _ Token = (*TokenI)(nil)

// TokenBI Sets its first argument in bold italic type.  It operates similarly to
// the B macro otherwise.
// Set its arguments alternately in bold face and italic, without a space
// between the arguments.  Thus,
//
// .BI this "word and" that
//
// produces “thisword andthat” with “this” and “that” in bold face,
// and “word and” in italics.
type TokenBI struct { /* [txt | text */
}

func ParseTokenBI(args string) (Token, error) {
	return &TokenBI{ /* [txt | text */ }, nil
}

func (t *TokenBI) String() string {
	return ".BI" /* [txt | text */
}

var _ Token = (*TokenBI)(nil)

// TokenBX Prints its argument and draws a box around it.  If you want to box a
// string that contains spaces, use a digit-width space (\0).
type TokenBX struct { /* [txt] */
}

func ParseTokenBX(args string) (Token, error) {
	return &TokenBX{ /* [txt] */ }, nil
}

func (t *TokenBX) String() string {
	return ".BX" /* [txt] */
}

var _ Token = (*TokenBX)(nil)

// TokenUL Prints its first argument with an underline.  If you specify a second
// argument, groff prints it in the previous font after the
// underlined text, with no intervening space.
type TokenUL struct { /* [txt */
}

func ParseTokenUL(args string) (Token, error) {
	return &TokenUL{ /* [txt */ }, nil
}

func (t *TokenUL) String() string {
	return ".UL" /* [txt */
}

var _ Token = (*TokenUL)(nil)

// TokenLG Prints all text following in larger type (two points larger than the
// current point size) until the next font size, highlighting, paragraph,
// or heading macro.  You can specify this macro multiple times to
// enlarge the point size as needed.
type TokenLG struct { /*  */
}

func ParseTokenLG(args string) (Token, error) {
	return &TokenLG{ /*  */ }, nil
}

func (t *TokenLG) String() string {
	return ".LG" /*  */
}

var _ Token = (*TokenLG)(nil)

// TokenSM Prints all text following in smaller type (two points smaller than the
// current point size) until the next type size, highlighting, paragraph,
// or heading macro.  You can specify this macro multiple times to reduce
// the point size as needed.
// Set the text on the same line or the text on the next line in a font
// that is one point size smaller than the default font.
type TokenSM struct { /* undefined | [text] */
}

func ParseTokenSM(args string) (Token, error) {
	return &TokenSM{ /* undefined | [text] */ }, nil
}

func (t *TokenSM) String() string {
	return ".SM" /* undefined | [text] */
}

var _ Token = (*TokenSM)(nil)

// TokenNL Prints all text following in the normal point size (that is, the value
// of the PS register).
type TokenNL struct { /*  */
}

func ParseTokenNL(args string) (Token, error) {
	return &TokenNL{ /*  */ }, nil
}

func (t *TokenNL) String() string {
	return ".NL" /*  */
}

var _ Token = (*TokenNL)(nil)

// TokenIP The marker is usually a bullet glyph (\[bu]) for unordered
// lists, a number (or auto-incrementing number register) for numbered
// lists, or a word or phrase for indented (glossary-style) lists.
//
// The width specifies the indentation for the body of each list
// item; its default unit is ‘n’.  Once specified, the indentation
// remains the same for all list items in the document until specified
// again.
//
// The PORPHANS register (see ms Document Control Registers)
// operates in conjunction with the IP macro, to inhibit the
// printing of orphaned list markers at the bottom of any page.
// Set up an indented paragraph, using designator as a tag to mark
// its beginning.  The indentation is set to nnn if that argument is
// supplied (default unit is ‘n’), otherwise it is set to the previous
// indentation value specified with TP, IP, or HP (or
// the default value if none of them have been used yet).  Font size and
// face of the paragraph (but not the designator) are reset to their
// default values.
//
// To start an indented paragraph with a particular indentation but without
// a designator, use ‘""’ (two double quotes) as the first argument of
// IP.
//
// For example, to start a paragraph with bullets as the designator and
// 4 en indentation, write
//
// .IP \(bu 4
type TokenIP struct { /* [marker | [designator */
}

func ParseTokenIP(args string) (Token, error) {
	return &TokenIP{ /* [marker | [designator */ }, nil
}

func (t *TokenIP) String() string {
	return ".IP" /* [marker | [designator */
}

var _ Token = (*TokenIP)(nil)

// TokenRS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
// Move the left margin to the right by the value nnn if specified
// (default unit is ‘n’); otherwise it is set to the previous
// indentation value specified with TP, IP, or HP (or
// to the default value if none of them have been used yet).  The
// indentation value is then set to the default.
//
// Calls to the RS macro can be nested.
type TokenRS struct { /* undefined | [nnn] */
}

func ParseTokenRS(args string) (Token, error) {
	return &TokenRS{ /* undefined | [nnn] */ }, nil
}

func (t *TokenRS) String() string {
	return ".RS" /* undefined | [nnn] */
}

var _ Token = (*TokenRS)(nil)

// TokenRE These macros begin and end an indented section.  The PI register
// controls the amount of indentation, allowing the indented text to line
// up under hanging and indented paragraphs.
// Move the left margin back to level nnn, restoring the previous
// left margin.  If no argument is given, it moves one level back.  The
// first level (i.e., no call to RS yet) has number 1, and each
// call to RS increases the level by 1.
type TokenRE struct { /* undefined | [nnn] */
}

func ParseTokenRE(args string) (Token, error) {
	return &TokenRE{ /* undefined | [nnn] */ }, nil
}

func (t *TokenRE) String() string {
	return ".RE" /* undefined | [nnn] */
}

var _ Token = (*TokenRE)(nil)

// TokenTA Use this macro to reset the tab stops to the default for ms
// (every 5n).  You can redefine the TA macro to create a different
// set of default tab stops.
type TokenTA struct { /*  */
}

func ParseTokenTA(args string) (Token, error) {
	return &TokenTA{ /*  */ }, nil
}

func (t *TokenTA) String() string {
	return ".TA" /*  */
}

var _ Token = (*TokenTA)(nil)

// TokenLD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenLD struct { /*  */
}

func ParseTokenLD(args string) (Token, error) {
	return &TokenLD{ /*  */ }, nil
}

func (t *TokenLD) String() string {
	return ".LD" /*  */
}

var _ Token = (*TokenLD)(nil)

// TokenID Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenID struct { /*  */
}

func ParseTokenID(args string) (Token, error) {
	return &TokenID{ /*  */ }, nil
}

func (t *TokenID) String() string {
	return ".ID" /*  */
}

var _ Token = (*TokenID)(nil)

// TokenBD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenBD struct { /*  */
}

func ParseTokenBD(args string) (Token, error) {
	return &TokenBD{ /*  */ }, nil
}

func (t *TokenBD) String() string {
	return ".BD" /*  */
}

var _ Token = (*TokenBD)(nil)

// TokenCD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenCD struct { /*  */
}

func ParseTokenCD(args string) (Token, error) {
	return &TokenCD{ /*  */ }, nil
}

func (t *TokenCD) String() string {
	return ".CD" /*  */
}

var _ Token = (*TokenCD)(nil)

// TokenRD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenRD struct { /*  */
}

func ParseTokenRD(args string) (Token, error) {
	return &TokenRD{ /*  */ }, nil
}

func (t *TokenRD) String() string {
	return ".RD" /*  */
}

var _ Token = (*TokenRD)(nil)

// TokenDE Right-justifies each line in the display.  The ‘.DS R’ call
// generates a page break, if necessary, to keep the entire display on one
// page.  The RD macro allows the display to break across pages.
// The DE macro ends the display.
// Sets a centered display: each line in the display is centered.  The
// ‘.DS C’ call generates a page break, if necessary, to keep the
// entire display on one page.  The CD macro allows the display to
// break across pages.  The DE macro ends the display.
// Sets a block-centered display: the entire display is left-justified, but
// indented so that the longest line in the display is centered on the
// page.  The ‘.DS B’ call generates a page break, if necessary, to
// keep the entire display on one page.  The BD macro allows the
// display to break across pages.  The DE macro ends the display.
// Indents the display as defined by the DI register.  The ‘.DS
// I’ call generates a page break, if necessary, to keep the entire display
// on one page.  The ID macro allows the display to break across
// pages.  The DE macro ends the display.
// Left-justified display.  The ‘.DS L’ call generates a page break,
// if necessary, to keep the entire display on one page.  The LD
// macro allows the display to break across pages.  The DE macro
// ends the display.
type TokenDE struct { /*  */
}

func ParseTokenDE(args string) (Token, error) {
	return &TokenDE{ /*  */ }, nil
}

func (t *TokenDE) String() string {
	return ".DE" /*  */
}

var _ Token = (*TokenDE)(nil)

// TokenDs Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
// Begin a non-filled display.
type TokenDs struct { /*  */
}

func ParseTokenDs(args string) (Token, error) {
	return &TokenDs{ /*  */ }, nil
}

func (t *TokenDs) String() string {
	return ".Ds" /*  */
}

var _ Token = (*TokenDs)(nil)

// TokenDe These two macros were formerly provided as aliases for DS and
// DE, respectively.  They have been removed, and should no longer
// be used.  The original implementations of DS and DE are
// retained, and should be used instead.  X11 documents that actually use
// Ds and De always load a specific macro file from the X11
// distribution (macros.t) that provides proper definitions for the
// two macros.
// End a non-filled display started with Ds.
type TokenDe struct { /*  */
}

func ParseTokenDe(args string) (Token, error) {
	return &TokenDe{ /*  */ }, nil
}

func (t *TokenDe) String() string {
	return ".De" /*  */
}

var _ Token = (*TokenDe)(nil)

// TokenKS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenKS struct { /*  */
}

func ParseTokenKS(args string) (Token, error) {
	return &TokenKS{ /*  */ }, nil
}

func (t *TokenKS) String() string {
	return ".KS" /*  */
}

var _ Token = (*TokenKS)(nil)

// TokenKF Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenKF struct { /*  */
}

func ParseTokenKF(args string) (Token, error) {
	return &TokenKF{ /*  */ }, nil
}

func (t *TokenKF) String() string {
	return ".KF" /*  */
}

var _ Token = (*TokenKF)(nil)

// TokenKE Specifies a floating keep; if the keep cannot fit on the current
// page, groff holds the contents of the keep and allows text
// following the keep (in the source file) to fill in the remainder of the
// current page.  When the page breaks, whether by an explicit bp
// request or by reaching the end of the page, groff prints the
// floating keep at the top of the new page.  This is useful for printing
// large graphics or tables that do not need to appear exactly where
// specified.
// The KS macro begins a block of text to be kept on a single page,
// and the KE macro ends the block.
type TokenKE struct { /*  */
}

func ParseTokenKE(args string) (Token, error) {
	return &TokenKE{ /*  */ }, nil
}

func (t *TokenKE) String() string {
	return ".KE" /*  */
}

var _ Token = (*TokenKE)(nil)

// TokenB1 Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenB1 struct { /*  */
}

func ParseTokenB1(args string) (Token, error) {
	return &TokenB1{ /*  */ }, nil
}

func (t *TokenB1) String() string {
	return ".B1" /*  */
}

var _ Token = (*TokenB1)(nil)

// TokenB2 Marks the beginning and ending of text that is to have a box drawn
// around it.  The B1 macro begins the box; the B2 macro ends
// it.  Text in the box is automatically placed in a diversion (keep).
type TokenB2 struct { /*  */
}

func ParseTokenB2(args string) (Token, error) {
	return &TokenB2{ /*  */ }, nil
}

func (t *TokenB2) String() string {
	return ".B2" /*  */
}

var _ Token = (*TokenB2)(nil)

// TokenTS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenTS struct { /* [H] */
}

func ParseTokenTS(args string) (Token, error) {
	return &TokenTS{ /* [H] */ }, nil
}

func (t *TokenTS) String() string {
	return ".TS" /* [H] */
}

var _ Token = (*TokenTS)(nil)

// TokenTE Denotes a table, to be processed by the tbl preprocessor.  The
// optional argument H to TS instructs groff to
// create a running header with the information up to the TH macro.
// groff prints the header at the beginning of the table; if the
// table runs onto another page, groff prints the header on the next
// page as well.
type TokenTE struct { /*  */
}

func ParseTokenTE(args string) (Token, error) {
	return &TokenTE{ /*  */ }, nil
}

func (t *TokenTE) String() string {
	return ".TE" /*  */
}

var _ Token = (*TokenTE)(nil)

// TokenPS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenPS struct { /*  */
}

func ParseTokenPS(args string) (Token, error) {
	return &TokenPS{ /*  */ }, nil
}

func (t *TokenPS) String() string {
	return ".PS" /*  */
}

var _ Token = (*TokenPS)(nil)

// TokenPE Denotes a graphic, to be processed by the pic preprocessor.  You
// can create a pic file by hand, using the AT&T
// pic manual available on the Web as a reference, or by using a
// graphics program such as xfig.
type TokenPE struct { /*  */
}

func ParseTokenPE(args string) (Token, error) {
	return &TokenPE{ /*  */ }, nil
}

func (t *TokenPE) String() string {
	return ".PE" /*  */
}

var _ Token = (*TokenPE)(nil)

// TokenEQ Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenEQ struct { /* [align] */
}

func ParseTokenEQ(args string) (Token, error) {
	return &TokenEQ{ /* [align] */ }, nil
}

func (t *TokenEQ) String() string {
	return ".EQ" /* [align] */
}

var _ Token = (*TokenEQ)(nil)

// TokenEN Denotes an equation, to be processed by the eqn preprocessor.
// The optional align argument can be C, L,
// or I to center (the default), left-justify, or indent the
// equation.
type TokenEN struct { /*  */
}

func ParseTokenEN(args string) (Token, error) {
	return &TokenEN{ /*  */ }, nil
}

func (t *TokenEN) String() string {
	return ".EN" /*  */
}

var _ Token = (*TokenEN)(nil)

// Token[ Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenReferOpen struct { /*  */
}

func ParseTokenReferOpen(args string) (Token, error) {
	return &TokenReferOpen{ /*  */ }, nil
}

func (t *TokenReferOpen) String() string {
	return ".[" /*  */
}

var _ Token = (*TokenReferOpen)(nil)

// Token] Denotes a reference, to be processed by the refer preprocessor.
// The GNU refer(1) man page provides a comprehensive reference to
// the preprocessor and the format of the bibliographic database.
type TokenReferClose struct { /*  */
}

func ParseTokenReferClose(args string) (Token, error) {
	return &TokenReferClose{ /*  */ }, nil
}

func (t *TokenReferClose) String() string {
	return ".]" /*  */
}

var _ Token = (*TokenReferClose)(nil)

// TokenFS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenFS struct { /*  */
}

func ParseTokenFS(args string) (Token, error) {
	return &TokenFS{ /*  */ }, nil
}

func (t *TokenFS) String() string {
	return ".FS" /*  */
}

var _ Token = (*TokenFS)(nil)

// TokenFE Specifies the text of the footnote.  The default action is to create a
// numbered footnote; you can create a symbolic footnote by specifying a
// mark glyph (such as \[dg] for the dagger glyph) in the body
// text and as an argument to the FS macro, followed by the text of
// the footnote and the FE macro.
type TokenFE struct { /*  */
}

func ParseTokenFE(args string) (Token, error) {
	return &TokenFE{ /*  */ }, nil
}

func (t *TokenFE) String() string {
	return ".FE" /*  */
}

var _ Token = (*TokenFE)(nil)

// TokenOH Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenOH struct { /* 'left'center'right' */
}

func ParseTokenOH(args string) (Token, error) {
	return &TokenOH{ /* 'left'center'right' */ }, nil
}

func (t *TokenOH) String() string {
	return ".OH" /* 'left'center'right' */
}

var _ Token = (*TokenOH)(nil)

// TokenEH Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenEH struct { /* 'left'center'right' */
}

func ParseTokenEH(args string) (Token, error) {
	return &TokenEH{ /* 'left'center'right' */ }, nil
}

func (t *TokenEH) String() string {
	return ".EH" /* 'left'center'right' */
}

var _ Token = (*TokenEH)(nil)

// TokenOF Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenOF struct { /* 'left'center'right' */
}

func ParseTokenOF(args string) (Token, error) {
	return &TokenOF{ /* 'left'center'right' */ }, nil
}

func (t *TokenOF) String() string {
	return ".OF" /* 'left'center'right' */
}

var _ Token = (*TokenOF)(nil)

// TokenEF The OH and EH macros define headers for the odd and even
// pages; the OF and EF macros define footers for the odd and
// even pages.  This is more flexible than defining the individual strings.
//
// You can replace the quote (') marks with any character not
// appearing in the header or footer text.
type TokenEF struct { /* 'left'center'right' */
}

func ParseTokenEF(args string) (Token, error) {
	return &TokenEF{ /* 'left'center'right' */ }, nil
}

func (t *TokenEF) String() string {
	return ".EF" /* 'left'center'right' */
}

var _ Token = (*TokenEF)(nil)

// TokenPT Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
// Control the content of the headers.  Normally, the header prints the
// command name and section number on either side, and the optional fifth
// argument to TH in the center.
type TokenPT struct { /*  */
}

func ParseTokenPT(args string) (Token, error) {
	return &TokenPT{ /*  */ }, nil
}

func (t *TokenPT) String() string {
	return ".PT" /*  */
}

var _ Token = (*TokenPT)(nil)

// TokenHD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenHD struct { /*  */
}

func ParseTokenHD(args string) (Token, error) {
	return &TokenHD{ /*  */ }, nil
}

func (t *TokenHD) String() string {
	return ".HD" /*  */
}

var _ Token = (*TokenHD)(nil)

// TokenBT The PT macro defines a custom header; the BT macro defines
// a custom footer.  These macros must handle odd/even/first page
// differences if necessary.
//
// The HD macro defines additional header processing to take place
// after executing the PT macro.
// Control the content of the footers.  Normally, the footer prints the
// page number and the third and fourth arguments to TH.
//
// Use the FT number register to specify the footer position.  The
// default is -0.5i.
type TokenBT struct { /*  */
}

func ParseTokenBT(args string) (Token, error) {
	return &TokenBT{ /*  */ }, nil
}

func (t *TokenBT) String() string {
	return ".BT" /*  */
}

var _ Token = (*TokenBT)(nil)

// Token1C Single-column mode.
type Token1C struct { /*  */
}

func ParseToken1C(args string) (Token, error) {
	return &Token1C{ /*  */ }, nil
}

func (t *Token1C) String() string {
	return ".1C" /*  */
}

var _ Token = (*Token1C)(nil)

// Token2C Two-column mode.
type Token2C struct { /*  */
}

func ParseToken2C(args string) (Token, error) {
	return &Token2C{ /*  */ }, nil
}

func (t *Token2C) String() string {
	return ".2C" /*  */
}

var _ Token = (*Token2C)(nil)

// TokenMC Multi-column mode.  If you specify no arguments, it is equivalent to the
// 2C macro.  Otherwise, width is the width of each column and
// gutter is the space between columns.  The MINGW number
// register controls the default gutter width.
type TokenMC struct { /* [width */
}

func ParseTokenMC(args string) (Token, error) {
	return &TokenMC{ /* [width */ }, nil
}

func (t *TokenMC) String() string {
	return ".MC" /* [width */
}

var _ Token = (*TokenMC)(nil)

// TokenXS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenXS struct { /* [page] */
}

func ParseTokenXS(args string) (Token, error) {
	return &TokenXS{ /* [page] */ }, nil
}

func (t *TokenXS) String() string {
	return ".XS" /* [page] */
}

var _ Token = (*TokenXS)(nil)

// TokenXA Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenXA struct { /* [page] */
}

func ParseTokenXA(args string) (Token, error) {
	return &TokenXA{ /* [page] */ }, nil
}

func (t *TokenXA) String() string {
	return ".XA" /* [page] */
}

var _ Token = (*TokenXA)(nil)

// TokenXE These macros define a table of contents or an individual entry in the
// table of contents, depending on their use.  The macros are very simple;
// they cannot indent a heading based on its level.  The easiest way to
// work around this is to add tabs to the table of contents string.  The
// following is an example:
//
// .NH 1
// Introduction
// .XS
// Introduction
// .XE
// .LP
// ...
// .CW
// .NH 2
// Methodology
// .XS
// Methodology
// .XE
// .LP
// ...
//
// You can manually create a table of contents by beginning with the
// XS macro for the first entry, specifying the page number for that
// entry as the argument to XS.  Add subsequent entries using the
// XA macro, specifying the page number for that entry as the
// argument to XA.  The following is an example:
//
// .XS 1
// Introduction
// .XA 2
// A Brief History of the Universe
// .XA 729
// Details of Galactic Formation
// ...
// .XE
type TokenXE struct { /*  */
}

func ParseTokenXE(args string) (Token, error) {
	return &TokenXE{ /*  */ }, nil
}

func (t *TokenXE) String() string {
	return ".XE" /*  */
}

var _ Token = (*TokenXE)(nil)

// TokenTC Prints the table of contents on a new page, setting the page number
// to i (Roman lowercase numeral one).  You should usually
// place this macro at the end of the file, since groff is a
// single-pass formatter and can only print what has been collected up to
// the point that the TC macro appears.
//
// The optional argument no suppresses printing the title specified
// by the string register TOC.
type TokenTC struct { /* [no] */
}

func ParseTokenTC(args string) (Token, error) {
	return &TokenTC{ /* [no] */ }, nil
}

func (t *TokenTC) String() string {
	return ".TC" /* [no] */
}

var _ Token = (*TokenTC)(nil)

// TokenPX Prints the table of contents on a new page, using the current page
// numbering sequence.  Use this macro to print a manually generated table
// of contents at the beginning of your document.
//
// The optional argument no suppresses printing the title specified
// by the string register TOC.
type TokenPX struct { /* [no] */
}

func ParseTokenPX(args string) (Token, error) {
	return &TokenPX{ /* [no] */ }, nil
}

func (t *TokenPX) String() string {
	return ".PX" /* [no] */
}

var _ Token = (*TokenPX)(nil)

// TokenAM Improved accent marks.  See ms Strings and Special Characters, for
// details.
// Specify this macro at the beginning of your document to enable extended
// accent marks and special characters.  This is a Berkeley extension.
//
// To use the accent marks, place them after the character being
// accented.
//
// Note that groff’s native support for accents is superior to the
// following definitions.
type TokenAM struct { /*  */
}

func ParseTokenAM(args string) (Token, error) {
	return &TokenAM{ /*  */ }, nil
}

func (t *TokenAM) String() string {
	return ".AM" /*  */
}

var _ Token = (*TokenAM)(nil)

// TokenDS Indented display.  The default behavior of AT&T troff
// -ms was to indent; the groff default prints displays flush left
// with the body text.
type TokenDS struct { /* I | R | C | B | I | L */
}

func ParseTokenDS(args string) (Token, error) {
	return &TokenDS{ /* I | R | C | B | I | L */ }, nil
}

func (t *TokenDS) String() string {
	return ".DS" /* I | R | C | B | I | L */
}

var _ Token = (*TokenDS)(nil)

// TokenCW Print text in constant width (Courier) font.
// Sets its first argument in a constant width face.  It operates
// similarly to the B macro otherwise.
// Print subsequent text using the constant width (Courier) typeface.
type TokenCW struct { /* undefined | [txt | undefined */
}

func ParseTokenCW(args string) (Token, error) {
	return &TokenCW{ /* undefined | [txt | undefined */ }, nil
}

func (t *TokenCW) String() string {
	return ".CW" /* undefined | [txt | undefined */
}

var _ Token = (*TokenCW)(nil)

// TokenIX Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenIX struct { /*  */
}

func ParseTokenIX(args string) (Token, error) {
	return &TokenIX{ /*  */ }, nil
}

func (t *TokenIX) String() string {
	return ".IX" /*  */
}

var _ Token = (*TokenIX)(nil)

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
	case `\\"`:
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
