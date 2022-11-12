package groff

// Tokenig Ignore all input until gtroff encounters the macro named
// .end on a line by itself (or .. if end is not
// specified).  This is useful for commenting out large blocks of text:
//
// text text text...
// .ig
// This is part of a large block
// of text that has been
// temporarily(?) commented out.
//
// We can restore it simply by removing
// the .ig request and the ".." at the
// end of the block.
// ..
// More text text text...
//
// produces
//
// text text text…  More text text text…
//
// Note that the commented-out block of text does not cause a break.
//
// The input is read in copy-mode; auto-incremented registers are
// affected (see Auto-increment).
type Tokenig struct { /* [end] */
}

func ParseTokenig(args string) (Token, error) {
	return &Tokenig{ /* [end] */ }, nil
}

func (t *Tokenig) String() string {
	return ".ig" /* [end] */
}

var _ Token = (*Tokenig)(nil)

// Tokenrr Remove number register ident.  If ident doesn’t exist, the
// request is ignored.
type Tokenrr struct { /* ident */
}

func ParseTokenrr(args string) (Token, error) {
	return &Tokenrr{ /* ident */ }, nil
}

func (t *Tokenrr) String() string {
	return ".rr" /* ident */
}

var _ Token = (*Tokenrr)(nil)

// Tokenrnn Rename number register ident1 to ident2.  If either
// ident1 or ident2 doesn’t exist, the request is ignored.
type Tokenrnn struct { /* ident1 */
}

func ParseTokenrnn(args string) (Token, error) {
	return &Tokenrnn{ /* ident1 */ }, nil
}

func (t *Tokenrnn) String() string {
	return ".rnn" /* ident1 */
}

var _ Token = (*Tokenrnn)(nil)

// Tokenaln Create an alias ident1 for a number register ident2.  The
// new name and the old name are exactly equivalent.  If ident1 is
// undefined, a warning of type ‘reg’ is generated, and the request is
// ignored.  See Debugging, for information about warnings.
type Tokenaln struct { /* ident1 */
}

func ParseTokenaln(args string) (Token, error) {
	return &Tokenaln{ /* ident1 */ }, nil
}

func (t *Tokenaln) String() string {
	return ".aln" /* ident1 */
}

var _ Token = (*Tokenaln)(nil)

// Tokennr Set number register ident to value; the increment for
// auto-incrementing is set to incr.  Note that the \R escape
// doesn’t support this notation.
// Escape: \R'ident +value'
type Tokennr struct { /* ident */
}

func ParseTokennr(args string) (Token, error) {
	return &Tokennr{ /* ident */ }, nil
}

func (t *Tokennr) String() string {
	return ".nr" /* ident */
}

var _ Token = (*Tokennr)(nil)

// Tokenaf Change the output format of a number register.  The first argument
// ident is the name of the number register to be changed, and the
// second argument format is the output format.  The following output
// formats are available:
//
// 1
// Decimal arabic numbers.  This is the default format: 0, 1, 2,
// 3, ...
//
// 0…0
// Decimal numbers with as many digits as specified.  So, ‘00’ would
// result in printing numbers as 01, 02, 03, ...
//
// In fact, any digit instead of zero does work; gtroff only counts
// how many digits are specified.  As a consequence, af’s default
// format ‘1’ could be specified as ‘0’ also (and exactly this is
// returned by the \g escape, see below).
//
// # I
//
// Upper-case Roman numerals: 0, I, II, III, IV, ...
//
// i
// Lower-case Roman numerals: 0, i, ii, iii, iv, ...
//
// A
// Upper-case letters: 0, A, B, C, …, Z, AA, AB, ...
//
// a
// Lower-case letters: 0, a, b, c, …, z, aa, ab, ...
//
// Omitting the number register format causes a warning of type
// ‘missing’.  See Debugging, for more details.  Specifying a
// nonexistent format causes an error.
//
// The following example produces ‘10, X, j, 010’:
//
// .nr a 10
// .af a 1           \" the default format
// \na,
// .af a I
// \na,
// .af a a
// \na,
// .af a 001
// \na
//
// The largest number representable for the ‘i’ and ‘I’ formats
// is 39999 (or -39999); Unix troff uses ‘z’ and
// ‘w’ to represent 10000 and 5000 in Roman numerals, and so does
// gtroff.  Currently, the correct glyphs of Roman numeral five
// thousand and Roman numeral ten thousand (Unicode code points
// U+2182 and U+2181, respectively) are not available.
//
// If ident doesn’t exist, it is created.
//
// Changing the output format of a read-only register causes an error.  It
// is necessary to first copy the register’s value to a writeable register,
// then apply the af request to this other register.
type Tokenaf struct { /* ident */
}

func ParseTokenaf(args string) (Token, error) {
	return &Tokenaf{ /* ident */ }, nil
}

func (t *Tokenaf) String() string {
	return ".af" /* ident */
}

var _ Token = (*Tokenaf)(nil)

// Tokenbr Break the current line, i.e., the input collected so far is emitted
// without adjustment.
//
// If the no-break control character is used, gtroff suppresses the
// break:
//
// a
// 'br
// b
//
//	⇒ a b
type Tokenbr struct { /*  */
}

func ParseTokenbr(args string) (Token, error) {
	return &Tokenbr{ /*  */ }, nil
}

func (t *Tokenbr) String() string {
	return ".br" /*  */
}

var _ Token = (*Tokenbr)(nil)

type Tokenfi struct { /*  */
}

func ParseTokenfi(args string) (Token, error) {
	return &Tokenfi{ /*  */ }, nil
}

func (t *Tokenfi) String() string {
	return ".fi" /*  */
}

var _ Token = (*Tokenfi)(nil)

// Tokennf Activate no-fill mode.  Input lines are output as-is, retaining line
// breaks and ignoring the current line length.  This command implicitly
// disables adjusting; it also causes a break.  The number register
// .u is set to 0.
//
// The fill mode status is associated with the current environment
// (see Environments).
//
// See Line Control, for interaction with the \c escape.
type Tokennf struct { /*  */
}

func ParseTokennf(args string) (Token, error) {
	return &Tokennf{ /*  */ }, nil
}

func (t *Tokennf) String() string {
	return ".nf" /*  */
}

var _ Token = (*Tokennf)(nil)

type Tokenad struct { /* [mode] */
}

func ParseTokenad(args string) (Token, error) {
	return &Tokenad{ /* [mode] */ }, nil
}

func (t *Tokenad) String() string {
	return ".ad" /* [mode] */
}

var _ Token = (*Tokenad)(nil)

// Tokenna Disable adjusting.  This request won’t change the current adjustment
// mode: A subsequent call to ad uses the previous adjustment
// setting.
//
// The adjustment mode status is associated with the current environment
// (see Environments).
type Tokenna struct { /*  */
}

func ParseTokenna(args string) (Token, error) {
	return &Tokenna{ /*  */ }, nil
}

func (t *Tokenna) String() string {
	return ".na" /*  */
}

var _ Token = (*Tokenna)(nil)

type Tokenbrp struct { /*  */
}

func ParseTokenbrp(args string) (Token, error) {
	return &Tokenbrp{ /*  */ }, nil
}

func (t *Tokenbrp) String() string {
	return ".brp" /*  */
}

var _ Token = (*Tokenbrp)(nil)

type Tokenss struct { /* word_space_size */
}

func ParseTokenss(args string) (Token, error) {
	return &Tokenss{ /* word_space_size */ }, nil
}

func (t *Tokenss) String() string {
	return ".ss" /* word_space_size */
}

var _ Token = (*Tokenss)(nil)

type Tokence struct { /* [nnn] */
}

func ParseTokence(args string) (Token, error) {
	return &Tokence{ /* [nnn] */ }, nil
}

func (t *Tokence) String() string {
	return ".ce" /* [nnn] */
}

var _ Token = (*Tokence)(nil)

type Tokenrj struct { /* [nnn] */
}

func ParseTokenrj(args string) (Token, error) {
	return &Tokenrj{ /* [nnn] */ }, nil
}

func (t *Tokenrj) String() string {
	return ".rj" /* [nnn] */
}

var _ Token = (*Tokenrj)(nil)

type Tokenhy struct { /* [mode] */
}

func ParseTokenhy(args string) (Token, error) {
	return &Tokenhy{ /* [mode] */ }, nil
}

func (t *Tokenhy) String() string {
	return ".hy" /* [mode] */
}

var _ Token = (*Tokenhy)(nil)

// Tokennh Disable hyphenation (i.e., set the hyphenation mode to zero).  Note that
// the hyphenation mode of the last call to hy is not remembered.
//
// The hyphenation mode is associated with the current environment
// (see Environments).
type Tokennh struct { /*  */
}

func ParseTokennh(args string) (Token, error) {
	return &Tokennh{ /*  */ }, nil
}

func (t *Tokennh) String() string {
	return ".nh" /*  */
}

var _ Token = (*Tokennh)(nil)

type Tokenhlm struct { /* [nnn] */
}

func ParseTokenhlm(args string) (Token, error) {
	return &Tokenhlm{ /* [nnn] */ }, nil
}

func (t *Tokenhlm) String() string {
	return ".hlm" /* [nnn] */
}

var _ Token = (*Tokenhlm)(nil)

// Tokenhw Define how word1, word2, etc. are to be hyphenated.  The
// words must be given with hyphens at the hyphenation points.  For
// example:
//
// .hw in-sa-lub-rious
//
// Besides the space character, any character whose hyphenation code value
// is zero can be used to separate the arguments of hw (see the
// documentation for the hcode request below for more information).
// In addition, this request can be used more than once.
//
// Hyphenation points specified with hw are not subject to the
// restrictions given by the hy request.
//
// Hyphenation exceptions specified with the hw request are
// associated with the current hyphenation language; it causes an error if
// there is no current hyphenation language.
//
// This request is ignored if there is no parameter.
//
// In old versions of troff there was a limited amount of space to
// store such information; fortunately, with gtroff, this is no
// longer a restriction.
type Tokenhw struct { /* word1 */
}

func ParseTokenhw(args string) (Token, error) {
	return &Tokenhw{ /* word1 */ }, nil
}

func (t *Tokenhw) String() string {
	return ".hw" /* word1 */
}

var _ Token = (*Tokenhw)(nil)

// Tokenhc Change the hyphenation character to char.  This character then
// works the same as the \% escape, and thus, no longer appears in
// the output.  Without an argument, hc resets the hyphenation
// character to be \% (the default) only.
//
// The hyphenation character is associated with the current environment
// (see Environments).
type Tokenhc struct { /* [char] */
}

func ParseTokenhc(args string) (Token, error) {
	return &Tokenhc{ /* [char] */ }, nil
}

func (t *Tokenhc) String() string {
	return ".hc" /* [char] */
}

var _ Token = (*Tokenhc)(nil)

type Tokenhpf struct { /* pattern_file */
}

func ParseTokenhpf(args string) (Token, error) {
	return &Tokenhpf{ /* pattern_file */ }, nil
}

func (t *Tokenhpf) String() string {
	return ".hpf" /* pattern_file */
}

var _ Token = (*Tokenhpf)(nil)

type Tokenhpfa struct { /* pattern_file */
}

func ParseTokenhpfa(args string) (Token, error) {
	return &Tokenhpfa{ /* pattern_file */ }, nil
}

func (t *Tokenhpfa) String() string {
	return ".hpfa" /* pattern_file */
}

var _ Token = (*Tokenhpfa)(nil)

// Tokenhpfcode Read in a file of hyphenation patterns.  This file is searched for in
// the same way as name.tmac (or tmac.name) is
// searched for if the -mname option is specified.
//
// It should have the same format as (simple) TeX patterns files.  More
// specifically, the following scanning rules are implemented.
//
//	A percent sign starts a comment (up to the end of the line) even if
//
// preceded by a backslash.
//
//	No support for ‘digraphs’ like \$.
//
//	^^xx (x is 0–9 or a–f) and ^^x
//
// (character code of x in the range 0–127) are recognized; other use
// of ^ causes an error.
//
//	No macro expansion.
//
//	hpf checks for the expression \patterns{…}
//
// (possibly with whitespace before and after the braces).  Everything
// between the braces is taken as hyphenation patterns.  Consequently,
// { and } are not allowed in patterns.
//
//	Similarly, \hyphenation{…} gives a list of hyphenation
//
// exceptions.
//
//	\endinput is recognized also.
//
//	For backwards compatibility, if \patterns is missing, the whole
//
// file is treated as a list of hyphenation patterns (only recognizing the
// % character as the start of a comment).
//
// If no hpf request is specified (either in the document or in a
// macro package), gtroff won’t hyphenate at all.
//
// The hpfa request appends a file of patterns to the current list.
//
// The hpfcode request defines mapping values for character codes in
// hyphenation patterns.  hpf or hpfa then apply the mapping
// (after reading the patterns) before replacing or appending them to the
// current list of patterns.  Its arguments are pairs of character codes –
// integers from 0 to 255.  The request maps character
// code a to code b, code c to
// code d, and so on.  You can use character codes that would
// be invalid otherwise.  By default, everything maps to itself except
// letters ‘A’ to ‘Z’, which map to ‘a’ to ‘z’.
//
// The set of hyphenation patterns is associated with the current language
// set by the hla request.  The hpf request is usually
// invoked by the troffrc or troffrc-end file; by default,
// troffrc loads hyphenation patterns and exceptions for American
// English (in files hyphen.us and hyphenex.us).
//
// A second call to hpf (for the same language) replaces the
// hyphenation patterns with the new ones.
//
// Invoking hpf causes an error if there is no current hyphenation
// language.
type Tokenhpfcode struct { /* a */
}

func ParseTokenhpfcode(args string) (Token, error) {
	return &Tokenhpfcode{ /* a */ }, nil
}

func (t *Tokenhpfcode) String() string {
	return ".hpfcode" /* a */
}

var _ Token = (*Tokenhpfcode)(nil)

// Tokenhcode Set the hyphenation code of character c1 to code1, that of
// c2 to code2, etc.  A hyphenation code must be a single input
// character (not a special character) other than a digit or a space.
//
// To make hyphenation work, hyphenation codes must be set up.  At
// start-up, groff only assigns hyphenation codes to the letters
// ‘a’–‘z’ (mapped to themselves) and to the letters
// ‘A’–‘Z’ (mapped to ‘a’–‘z’); all other hyphenation
// codes are set to zero.  Normally, hyphenation patterns contain only
// lowercase letters, which should be applied regardless of case.  In
// other words, the words ‘FOO’ and ‘Foo’ should be hyphenated exactly the
// same way as the word ‘foo’ is hyphenated, and this is what hcode
// is good for.  Words that contain other letters won’t be hyphenated
// properly if the corresponding hyphenation patterns actually do contain
// them.  For example, the following hcode requests are necessary to
// assign hyphenation codes to the letters ‘ÄäÖöÜüß’ (this is needed
// for German):
//
// .hcode ä ä  Ä ä
// .hcode ö ö  Ö ö
// .hcode ü ü  Ü ü
// .hcode ß ß
//
// Without those assignments, groff treats German words like
// ‘Kindergärten’ (the plural form of ‘kindergarten’) as two substrings
// ‘kinderg’ and ‘rten’ because the hyphenation code of the
// umlaut a is zero by default.  There is a German hyphenation pattern
// that covers ‘kinder’, so groff finds the hyphenation ‘kin-der’.
// The other two hyphenation points (‘kin-der-gär-ten’) are missed.
//
// This request is ignored if it has no parameter.
type Tokenhcode struct { /* c1 */
}

func ParseTokenhcode(args string) (Token, error) {
	return &Tokenhcode{ /* c1 */ }, nil
}

func (t *Tokenhcode) String() string {
	return ".hcode" /* c1 */
}

var _ Token = (*Tokenhcode)(nil)

type Tokenhym struct { /* [length] */
}

func ParseTokenhym(args string) (Token, error) {
	return &Tokenhym{ /* [length] */ }, nil
}

func (t *Tokenhym) String() string {
	return ".hym" /* [length] */
}

var _ Token = (*Tokenhym)(nil)

type Tokenhys struct { /* [hyphenation_space] */
}

func ParseTokenhys(args string) (Token, error) {
	return &Tokenhys{ /* [hyphenation_space] */ }, nil
}

func (t *Tokenhys) String() string {
	return ".hys" /* [hyphenation_space] */
}

var _ Token = (*Tokenhys)(nil)

// Tokenshc Set the soft hyphen character to glyph.14  If the
// argument is omitted, the soft hyphen character is set to the default
// glyph \(hy (this is the start-up value of gtroff also).
// The soft hyphen character is the glyph that is inserted when a word is
// hyphenated at a line break.  If the soft hyphen character does not exist
// in the font of the character immediately preceding a potential break
// point, then the line is not broken at that point.  Neither definitions
// (specified with the char request) nor translations (specified
// with the tr request) are considered when finding the soft hyphen
// character.
type Tokenshc struct { /* [glyph] */
}

func ParseTokenshc(args string) (Token, error) {
	return &Tokenshc{ /* [glyph] */ }, nil
}

func (t *Tokenshc) String() string {
	return ".shc" /* [glyph] */
}

var _ Token = (*Tokenshc)(nil)

type Tokenhla struct { /* language */
}

func ParseTokenhla(args string) (Token, error) {
	return &Tokenhla{ /* language */ }, nil
}

func (t *Tokenhla) String() string {
	return ".hla" /* language */
}

var _ Token = (*Tokenhla)(nil)

// Tokensp Space downwards distance.  With no argument it advances
// 1 line.  A negative argument causes gtroff to move up the
// page the specified distance.  If the argument is preceded by a ‘|’
// then gtroff moves that distance from the top of the page.  This
// request causes a line break, and that adds the current line spacing to
// the space you have just specified.  The default scaling indicator is
// ‘v’.
//
// For convenience you may wish to use the following macros to set the
// height of the next line at a given distance from the top or the bottom
// of the page:
//
// .de y-from-top-down
// .  sp |\\$1-\\n[.v]u
// ..
// .
// .de y-from-bot-up
// .  sp |\\n[.p]u-\\$1-\\n[.v]u
// ..
//
// A call to ‘.y-from-bot-up 10c’ means that the bottom of the next
// line will be at 10 cm from the paper edge at the bottom.
//
// If a vertical trap is sprung during execution of sp, the amount
// of vertical space after the trap is discarded.  For example, this
//
// .de xxx
// ..
// .
// .wh 0 xxx
// .
// .pl 5v
// foo
// .sp 2
// bar
// .sp 50
// baz
//
// results in
//
// foo
//
// bar
//
// baz
//
// The amount of discarded space is available in the number register
// .trunc.
//
// To protect sp against vertical traps, use the vpt request:
//
// .vpt 0
// .sp -3
// .vpt 1
type Tokensp struct { /* [distance] */
}

func ParseTokensp(args string) (Token, error) {
	return &Tokensp{ /* [distance] */ }, nil
}

func (t *Tokensp) String() string {
	return ".sp" /* [distance] */
}

var _ Token = (*Tokensp)(nil)

type Tokenls struct { /* [nnn] */
}

func ParseTokenls(args string) (Token, error) {
	return &Tokenls{ /* [nnn] */ }, nil
}

func (t *Tokenls) String() string {
	return ".ls" /* [nnn] */
}

var _ Token = (*Tokenls)(nil)

type Tokenns struct { /*  */
}

func ParseTokenns(args string) (Token, error) {
	return &Tokenns{ /*  */ }, nil
}

func (t *Tokenns) String() string {
	return ".ns" /*  */
}

var _ Token = (*Tokenns)(nil)

type Tokenrs struct { /*  */
}

func ParseTokenrs(args string) (Token, error) {
	return &Tokenrs{ /*  */ }, nil
}

func (t *Tokenrs) String() string {
	return ".rs" /*  */
}

var _ Token = (*Tokenrs)(nil)

type Tokenta struct { /* [n1 */
}

func ParseTokenta(args string) (Token, error) {
	return &Tokenta{ /* [n1 */ }, nil
}

func (t *Tokenta) String() string {
	return ".ta" /* [n1 */
}

var _ Token = (*Tokenta)(nil)

// Tokentc Normally gtroff fills the space to the next tab stop with
// whitespace.  This can be changed with the tc request.  With no
// argument gtroff reverts to using whitespace, which is the
// default.  The value of this tab repetition character is associated
// with the current environment (see Environments).15
type Tokentc struct { /* [fill-glyph] */
}

func ParseTokentc(args string) (Token, error) {
	return &Tokentc{ /* [fill-glyph] */ }, nil
}

func (t *Tokentc) String() string {
	return ".tc" /* [fill-glyph] */
}

var _ Token = (*Tokentc)(nil)

type Tokenlinetabs struct { /* n */
}

func ParseTokenlinetabs(args string) (Token, error) {
	return &Tokenlinetabs{ /* n */ }, nil
}

func (t *Tokenlinetabs) String() string {
	return ".linetabs" /* n */
}

var _ Token = (*Tokenlinetabs)(nil)

// Tokenlc Declare the leader repetition character.16
// Without an argument, leaders act the same as tabs (i.e., using
// whitespace for filling).  gtroff’s start-up value is a dot
// (‘.’).  The value of the leader repetition character is associated
// with the current environment (see Environments).
type Tokenlc struct { /* [fill-glyph] */
}

func ParseTokenlc(args string) (Token, error) {
	return &Tokenlc{ /* [fill-glyph] */ }, nil
}

func (t *Tokenlc) String() string {
	return ".lc" /* [fill-glyph] */
}

var _ Token = (*Tokenlc)(nil)

// Tokenfc Define a delimiting and a padding character for fields.  If the latter
// is missing, the padding character defaults to a space character.  If
// there is no argument at all, the field mechanism is disabled (which is
// the default).  Note that contrary to e.g. the tab repetition
// character, delimiting and padding characters are not associated
// to the current environment (see Environments).
//
// Example:
//
// .fc # ^
// .ta T 3i
// #foo^bar^smurf#
// .br
// #foo^^bar^smurf#
//
// and here the result:
//
// foo         bar          smurf
// foo            bar       smurf
type Tokenfc struct { /* [delim-char */
}

func ParseTokenfc(args string) (Token, error) {
	return &Tokenfc{ /* [delim-char */ }, nil
}

func (t *Tokenfc) String() string {
	return ".fc" /* [delim-char */
}

var _ Token = (*Tokenfc)(nil)

// Tokencc Set the control character to c.  With no argument the default
// control character ‘.’ is restored.  The value of the control
// character is associated with the current environment
// (see Environments).
type Tokencc struct { /* [c] */
}

func ParseTokencc(args string) (Token, error) {
	return &Tokencc{ /* [c] */ }, nil
}

func (t *Tokencc) String() string {
	return ".cc" /* [c] */
}

var _ Token = (*Tokencc)(nil)

// Tokenc2 Set the no-break control character to c.  With no argument
// the default control character ‘'’ is restored.  The value of the
// no-break control character is associated with the current environment
// (see Environments).
type Tokenc2 struct { /* [c] */
}

func ParseTokenc2(args string) (Token, error) {
	return &Tokenc2{ /* [c] */ }, nil
}

func (t *Tokenc2) String() string {
	return ".c2" /* [c] */
}

var _ Token = (*Tokenc2)(nil)

// Tokeneo Disable the escape mechanism completely.  After executing this request,
// the backslash character ‘\’ no longer starts an escape sequence.
//
// This request can be very helpful in writing macros since it is not
// necessary then to double the escape character.  Here an example:
//
// .\" This is a simplified version of the
// .\" .BR request from the man macro package
// .eo
// .de BR
// .  ds result \&
// .  while (\n[.$] >= 2) \{\
// .    as result \fB\$1\fR\$2
// .    shift 2
// .  \}
// .  if \n[.$] .as result \fB\$1
// \*[result]
// .  ft R
// ..
// .ec
type Tokeneo struct { /*  */
}

func ParseTokeneo(args string) (Token, error) {
	return &Tokeneo{ /*  */ }, nil
}

func (t *Tokeneo) String() string {
	return ".eo" /*  */
}

var _ Token = (*Tokeneo)(nil)

// Tokenec Set the escape character to c.  With no argument the default
// escape character ‘\’ is restored.  It can be also used to re-enable
// the escape mechanism after an eo request.
//
// Note that changing the escape character globally likely breaks macro
// packages since gtroff has no mechanism to ‘intern’ macros, i.e.,
// to convert a macro definition into an internal form that is independent
// of its representation (TeX has this mechanism).  If a macro is
// called, it is executed literally.
type Tokenec struct { /* [c] */
}

func ParseTokenec(args string) (Token, error) {
	return &Tokenec{ /* [c] */ }, nil
}

func (t *Tokenec) String() string {
	return ".ec" /* [c] */
}

var _ Token = (*Tokenec)(nil)

type Tokenecs struct { /*  */
}

func ParseTokenecs(args string) (Token, error) {
	return &Tokenecs{ /*  */ }, nil
}

func (t *Tokenecs) String() string {
	return ".ecs" /*  */
}

var _ Token = (*Tokenecs)(nil)

// Tokenecr The ecs request saves the current escape character in an internal
// register.  Use this request in combination with the ec request to
// temporarily change the escape character.
//
// The ecr request restores the escape character saved with
// ecs.  Without a previous call to ecs, this request sets
// the escape character to \.
type Tokenecr struct { /*  */
}

func ParseTokenecr(args string) (Token, error) {
	return &Tokenecr{ /*  */ }, nil
}

func (t *Tokenecr) String() string {
	return ".ecr" /*  */
}

var _ Token = (*Tokenecr)(nil)

type Tokentr struct { /* abcd… */
}

func ParseTokentr(args string) (Token, error) {
	return &Tokentr{ /* abcd… */ }, nil
}

func (t *Tokentr) String() string {
	return ".tr" /* abcd… */
}

var _ Token = (*Tokentr)(nil)

// Tokentrin Translate character a to glyph b, character c to
// glyph d, etc.  If there is an odd number of arguments, the
// last one is translated to an unstretchable space (‘\ ’).
//
// The trin request is identical to tr, but when you unformat
// a diversion with asciify it ignores the translation.
// See Diversions, for details about the asciify request.
//
// Some notes:
//
// Special characters (\(xx, \[xxx],
// \C'xxx', \', \`, \-, \_),
// glyphs defined with the char request, and numbered glyphs
// (\N'xxx') can be translated also.
//
// The \e escape can be translated also.
//
// Characters can be mapped onto the \% and \~ escapes (but
// \% and \~ can’t be mapped onto another glyph).
//
// The following characters can’t be translated: space (with one exception,
// see below), backspace, newline, leader (and \a), tab (and
// \t).
//
// Translations are not considered for finding the soft hyphen character
// set with the shc request.
//
// The pair ‘c\&’ (this is an arbitrary character c
// followed by the zero width space character) maps this character to
// nothing.
//
// .tr a\&
// foo bar
//
//	⇒ foo br
//
// It is even possible to map the space character to nothing:
//
// .tr aa \&
// foo bar
//
//	⇒ foobar
//
// As shown in the example, the space character can’t be the first
// character/glyph pair as an argument of tr.  Additionally, it is
// not possible to map the space character to any other glyph; requests
// like ‘.tr aa x’ undo ‘.tr aa \&’ instead.
//
// If justification is active, lines are justified in spite of the ‘empty’
// space character (but there is no minimal distance, i.e. the space
// character, between words).
//
//	After an output glyph has been constructed (this happens at the moment
//
// immediately before the glyph is appended to an output glyph list, either
// by direct output, in a macro, diversion, or string), it is no longer
// affected by tr.
//
//	Translating character to glyphs where one of them or both are undefined
//
// is possible also; tr does not check whether the entities in its
// argument do exist.
//
// See Gtroff Internals.
//
//	troff no longer has a hard-coded dependency on Latin-1; all
//
// charXXX entities have been removed from the font
// description files.  This has a notable consequence that shows up in
// warnings like can't find character with input code XXX if
// the tr request isn’t handled properly.
//
// Consider the following translation:
//
// .tr éÉ
//
// This maps input character é onto glyph É, which is
// identical to glyph char201.  But this glyph intentionally doesn’t
// exist!  Instead, \[char201] is treated as an input character
// entity and is by default mapped onto \['E], and gtroff
// doesn’t handle translations of translations.
//
// # The right way to write the above translation is
//
// .tr é\['E]
//
// In other words, the first argument of tr should be an input
// character or entity, and the second one a glyph entity.
//
//	Without an argument, the tr request is ignored.
type Tokentrin struct { /* abcd… */
}

func ParseTokentrin(args string) (Token, error) {
	return &Tokentrin{ /* abcd… */ }, nil
}

func (t *Tokentrin) String() string {
	return ".trin" /* abcd… */
}

var _ Token = (*Tokentrin)(nil)

// Tokentrnt trnt is the same as the tr request except that the
// translations do not apply to text that is transparently throughput into
// a diversion with \!.  See Diversions, for more information.
//
// For example,
//
// .tr ab
// .di x
// \!.tm a
// .di
// .x
//
// prints ‘b’ to the standard error stream; if trnt is used
// instead of tr it prints ‘a’.
type Tokentrnt struct { /* abcd… */
}

func ParseTokentrnt(args string) (Token, error) {
	return &Tokentrnt{ /* abcd… */ }, nil
}

func (t *Tokentrnt) String() string {
	return ".trnt" /* abcd… */
}

var _ Token = (*Tokentrnt)(nil)

// Tokentroff Make the ‘t’ built-in condition true (and the ‘n’ built-in
// condition false) for if, ie, and while conditional
// requests.  This is the default if gtroff (not
// groff) is started with the -R switch to avoid loading of
// the start-up files troffrc and troffrc-end.  Without
// -R, gtroff stays in troff mode if the output device is
// not a TTY (e.g. ‘ps’).
type Tokentroff struct { /*  */
}

func ParseTokentroff(args string) (Token, error) {
	return &Tokentroff{ /*  */ }, nil
}

func (t *Tokentroff) String() string {
	return ".troff" /*  */
}

var _ Token = (*Tokentroff)(nil)

// Tokennroff Make the ‘n’ built-in condition true (and the ‘t’ built-in
// condition false) for if, ie, and while conditional
// requests.  This is the default if gtroff uses a TTY output
// device; the code for switching to nroff mode is in the file
// tty.tmac, which is loaded by the start-up file troffrc.
type Tokennroff struct { /*  */
}

func ParseTokennroff(args string) (Token, error) {
	return &Tokennroff{ /*  */ }, nil
}

func (t *Tokennroff) String() string {
	return ".nroff" /*  */
}

var _ Token = (*Tokennroff)(nil)

// Tokenpo Register: \n[.o]
// Request: .po -offset
type Tokenpo struct { /* -offset | +offset | [offset] */
}

func ParseTokenpo(args string) (Token, error) {
	return &Tokenpo{ /* -offset | +offset | [offset] */ }, nil
}

func (t *Tokenpo) String() string {
	return ".po" /* -offset | +offset | [offset] */
}

var _ Token = (*Tokenpo)(nil)

// Tokenin Register: \n[.i]
// Request: .in -indent
type Tokenin struct { /* -indent | +indent | [indent] */
}

func ParseTokenin(args string) (Token, error) {
	return &Tokenin{ /* -indent | +indent | [indent] */ }, nil
}

func (t *Tokenin) String() string {
	return ".in" /* -indent | +indent | [indent] */
}

var _ Token = (*Tokenin)(nil)

// Tokenti Register: \n[.in]
// Request: .ti -offset
type Tokenti struct { /* -offset | +offset | offset */
}

func ParseTokenti(args string) (Token, error) {
	return &Tokenti{ /* -offset | +offset | offset */ }, nil
}

func (t *Tokenti) String() string {
	return ".ti" /* -offset | +offset | offset */
}

var _ Token = (*Tokenti)(nil)

// Tokenll Register: \n[.l]
// Request: .ll -length
type Tokenll struct { /* -length | +length | [length] */
}

func ParseTokenll(args string) (Token, error) {
	return &Tokenll{ /* -length | +length | [length] */ }, nil
}

func (t *Tokenll) String() string {
	return ".ll" /* -length | +length | [length] */
}

var _ Token = (*Tokenll)(nil)

// Tokenpl Register: \n[.p]
// Request: .pl -length
type Tokenpl struct { /* -length | +length | [length] */
}

func ParseTokenpl(args string) (Token, error) {
	return &Tokenpl{ /* -length | +length | [length] */ }, nil
}

func (t *Tokenpl) String() string {
	return ".pl" /* -length | +length | [length] */
}

var _ Token = (*Tokenpl)(nil)

// Tokentl Print a title line.  It consists of three parts: a left justified
// portion, a centered portion, and a right justified portion.  The
// argument separator ‘'’ can be replaced with any character not
// occurring in the title line.  The ‘%’ character is replaced with
// the current page number.  This character can be changed with the
// pc request (see below).
//
// Without argument, tl is ignored.
//
// Some notes:
//
//	The line length set by the ll request is not honoured by
//
// tl; use the lt request (described below) instead, to
// control line length for text set by tl.
//
//	A title line is not restricted to the top or bottom of a page.
//
//	tl prints the title line immediately, ignoring a partially filled
//
// line (which stays untouched).
//
//	It is not an error to omit closing delimiters.  For example,
//
// ‘.tl /foo’ is equivalent to ‘.tl /foo///’: It prints a
// title line with the left justified word ‘foo’; the centered and
// right justified parts are empty.
//
//	tl accepts the same parameter delimiting characters as the
//
// \A escape; see Escapes.
type Tokentl struct { /* 'left'center'right' */
}

func ParseTokentl(args string) (Token, error) {
	return &Tokentl{ /* 'left'center'right' */ }, nil
}

func (t *Tokentl) String() string {
	return ".tl" /* 'left'center'right' */
}

var _ Token = (*Tokentl)(nil)

// Tokenlt Register: \n[.lt]
// Request: .lt -length
type Tokenlt struct { /* -length | +length | [length] */
}

func ParseTokenlt(args string) (Token, error) {
	return &Tokenlt{ /* -length | +length | [length] */ }, nil
}

func (t *Tokenlt) String() string {
	return ".lt" /* -length | +length | [length] */
}

var _ Token = (*Tokenlt)(nil)

// Tokenpn Register: \n[.pn]
// Request: .pn -page
type Tokenpn struct { /* -page | +page | page */
}

func ParseTokenpn(args string) (Token, error) {
	return &Tokenpn{ /* -page | +page | page */ }, nil
}

func (t *Tokenpn) String() string {
	return ".pn" /* -page | +page | page */
}

var _ Token = (*Tokenpn)(nil)

// Tokenpc Change the page number character (used by the tl request) to a
// different character.  With no argument, this mechanism is disabled.
// Note that this doesn’t affect the number register %.
type Tokenpc struct { /* [char] */
}

func ParseTokenpc(args string) (Token, error) {
	return &Tokenpc{ /* [char] */ }, nil
}

func (t *Tokenpc) String() string {
	return ".pc" /* [char] */
}

var _ Token = (*Tokenpc)(nil)

// Tokenbp Register: \n[%]
// Request: .bp -page
type Tokenbp struct { /* -page | +page | [page] */
}

func ParseTokenbp(args string) (Token, error) {
	return &Tokenbp{ /* -page | +page | [page] */ }, nil
}

func (t *Tokenbp) String() string {
	return ".bp" /* -page | +page | [page] */
}

var _ Token = (*Tokenbp)(nil)

// Tokenne It is often necessary to force a certain amount of space before a new
// page occurs.  This is most useful to make sure that there is not a
// single orphan line left at the bottom of a page.  The ne
// request ensures that there is a certain distance, specified by the first
// argument, before the next page is triggered (see Traps, for
// further information).  The default scaling indicator for ne is
// ‘v’; the default value of space is 1v if no
// argument is given.
//
// For example, to make sure that no fewer than 2 lines get orphaned,
// do the following before each paragraph:
//
// .ne 2
// text text text
//
// ne then automatically causes a page break if there is space for
// one line only.
type Tokenne struct { /* [space] */
}

func ParseTokenne(args string) (Token, error) {
	return &Tokenne{ /* [space] */ }, nil
}

func (t *Tokenne) String() string {
	return ".ne" /* [space] */
}

var _ Token = (*Tokenne)(nil)

type Tokensv struct { /* [space] */
}

func ParseTokensv(args string) (Token, error) {
	return &Tokensv{ /* [space] */ }, nil
}

func (t *Tokensv) String() string {
	return ".sv" /* [space] */
}

var _ Token = (*Tokensv)(nil)

// Tokenos sv is similar to the ne request; it reserves the specified
// amount of vertical space.  If the desired amount of space exists before
// the next trap (or the bottom page boundary if no trap is set), the space
// is output immediately (ignoring a partially filled line, which stays
// untouched).  If there is not enough space, it is stored for later output
// via the os request.  The default value is 1v if no
// argument is given; the default scaling indicator is ‘v’.
//
// Both sv and os ignore no-space mode.  While the sv
// request allows negative values for space, os ignores them.
type Tokenos struct { /*  */
}

func ParseTokenos(args string) (Token, error) {
	return &Tokenos{ /*  */ }, nil
}

func (t *Tokenos) String() string {
	return ".os" /*  */
}

var _ Token = (*Tokenos)(nil)

// Tokenftr Translate font f to font g.  Whenever a font
// named f is referred to in a \f escape sequence, in the
// F and S conditional operators, or in the ft,
// ul, bd, cs, tkf, special,
// fspecial, fp, or sty requests, font g is
// used.  If g is missing or equal to f the translation is
// undone.
//
// Note that it is not possible to chain font translations.  Example:
//
// .ftr XXX TR
// .ftr XXX YYY
// .ft XXX
//
//	⇒ warning: can't find font `XXX'
type Tokenftr struct { /* f */
}

func ParseTokenftr(args string) (Token, error) {
	return &Tokenftr{ /* f */ }, nil
}

func (t *Tokenftr) String() string {
	return ".ftr" /* f */
}

var _ Token = (*Tokenftr)(nil)

type Tokenfzoom struct { /* f */
}

func ParseTokenfzoom(args string) (Token, error) {
	return &Tokenfzoom{ /* f */ }, nil
}

func (t *Tokenfzoom) String() string {
	return ".fzoom" /* f */
}

var _ Token = (*Tokenfzoom)(nil)

type Tokenfam struct { /* [family] */
}

func ParseTokenfam(args string) (Token, error) {
	return &Tokenfam{ /* [family] */ }, nil
}

func (t *Tokenfam) String() string {
	return ".fam" /* [family] */
}

var _ Token = (*Tokenfam)(nil)

// Tokensty Associate style with font position n.  A font position
// can be associated either with a font or with a style.  The current font
// is the index of a font position and so is also either a font or a style.
// If it is a style, the font that is actually used is the font which name
// is the concatenation of the name of the current family and the name of
// the current style.  For example, if the current font is 1 and font
// position 1 is associated with style ‘R’ and the current font
// family is ‘T’, then font ‘TR’ is used.  If the current font is
// not a style, then the current family is ignored.  If the requests
// cs, bd, tkf, uf, or fspecial are
// applied to a style, they are instead applied to the member of the
// current family corresponding to that style.
//
// n must be a non-negative integer value.
//
// The default family can be set with the -f option (see Groff Options).  The styles command in the DESC file controls
// which font positions (if any) are initially associated with styles
// rather than fonts.  For example, the default setting for POSTSCRIPT
// fonts
//
// styles R I B BI
//
// is equivalent to
//
// .sty 1 R
// .sty 2 I
// .sty 3 B
// .sty 4 BI
//
// fam and \F always check whether the current font position
// is valid; this can give surprising results if the current font position
// is associated with a style.
//
// In the following example, we want to access the POSTSCRIPT font
// FooBar from the font family Foo:
//
// .sty \n[.fp] Bar
// .fam Foo
//
//	⇒ warning: can't find font `FooR'
//
// The default font position at start-up is 1; for the POSTSCRIPT
// device, this is associated with style ‘R’, so gtroff tries
// to open FooR.
//
// A solution to this problem is to use a dummy font like the following:
//
// .fp 0 dummy TR    \" set up dummy font at position 0
// .sty \n[.fp] Bar  \" register style `Bar'
// .ft 0             \" switch to font at position 0
// .fam Foo          \" activate family `Foo'
// .ft Bar           \" switch to font `FooBar'
//
// See Font Positions.
type Tokensty struct { /* n */
}

func ParseTokensty(args string) (Token, error) {
	return &Tokensty{ /* n */ }, nil
}

func (t *Tokensty) String() string {
	return ".sty" /* n */
}

var _ Token = (*Tokensty)(nil)

type Tokenfp struct { /* pos */
}

func ParseTokenfp(args string) (Token, error) {
	return &Tokenfp{ /* pos */ }, nil
}

func (t *Tokenfp) String() string {
	return ".fp" /* pos */
}

var _ Token = (*Tokenfp)(nil)

// Tokenft
type Tokenft struct { /* nnn | [font] */
}

func ParseTokenft(args string) (Token, error) {
	return &Tokenft{ /* nnn | [font] */ }, nil
}

func (t *Tokenft) String() string {
	return ".ft" /* nnn | [font] */
}

var _ Token = (*Tokenft)(nil)

// Tokencomposite Map glyph name from to glyph name to if it is used in
// \[...] with more than one component.  See above for examples.
//
// This mapping is based on glyph names only; no check for the existence of
// either glyph is done.
//
// A set of default mappings for many accents can be found in the file
// composite.tmac, which is loaded at start-up.
type Tokencomposite struct { /* from */
}

func ParseTokencomposite(args string) (Token, error) {
	return &Tokencomposite{ /* from */ }, nil
}

func (t *Tokencomposite) String() string {
	return ".composite" /* from */
}

var _ Token = (*Tokencomposite)(nil)

// Tokencflags Input characters and symbols have certain properties associated with
// it.19  These properties can be
// modified with the cflags request.  The first argument is the sum
// of the desired flags and the remaining arguments are the characters or
// symbols to have those properties.  It is possible to omit the spaces
// between the characters or symbols.  Instead of single characters or
// symbols you can also use character classes (see Character Classes
// for more details).
//
// 1
//
// The character ends sentences (initially characters ‘.?!’ have this
// property).
//
// 2
//
// Lines can be broken before the character (initially no characters have
// this property).  This only works if both the characters before and after
// have non-zero hyphenation codes (as set with the hcode request).
// Use value 64 to override this behaviour.
//
// 4
//
// Lines can be broken after the character (initially the character
// ‘-’ and the symbols ‘\[hy]’ and ‘\[em]’ have this
// property).  This only works if both the characters before and after have
// non-zero hyphenation codes (as set with the hcode request).  Use
// value 64 to override this behaviour.
//
// 8
//
// The character overlaps horizontally if used as a horizontal line
// building element.  Initially the symbols ‘\[ul]’, ‘\[rn]’,
// ‘\[ru]’, ‘\[radicalex]’, and ‘\[sqrtex]’ have this
// property.
//
// 16
//
// The character overlaps vertically if used as vertical line building
// element.  Initially symbol ‘\[br]’ has this property.
//
// 32
//
// An end-of-sentence character followed by any number of characters with
// this property is treated as the end of a sentence if followed by a
// newline or two spaces; in other words the character is transparent
// for the purposes of end-of-sentence recognition – this is the same as
// having a zero space factor in TeX (initially characters ‘"')]*’
// and the symbols ‘\[dg]’, ‘\[rq]’, and ‘\[cq]’ have this
// property).
//
// 64
// Ignore hyphenation code values of the surrounding characters.  Use this
// in combination with values 2 and 4 (initially no characters have
// this property).  For example, if you need an automatic break point after
// the en-dash in number ranges like ‘3000–5000’, insert
//
// .cflags 68 \(en
//
// into your document.  Note, however, that this can lead to bad layout if
// done without thinking; in most situations, a better solution instead of
// changing the cflags value is to insert \: right after the
// hyphen at the places that really need a break point.
//
// 128
// Prohibit a line break before the character, but allow a line break after
// the character.  This works only in combination with flags 256 and 512
// (see below) and has no effect otherwise.
//
// 256
// Prohibit a line break after the character, but allow a line break before
// the character.  This works only in combination with flags 128 and 512
// (see below) and has no effect otherwise.
//
// 512
// Allow line break before or after the character.  This works only in
// combination with flags 128 and 256 and has no effect otherwise.
//
// Contrary to flag values 2 and 4, the flags 128, 256, and 512 work
// pairwise.  If, for example, the left character has value 512, and the
// right character 128, no line break gets inserted.  If we use
// value 6 instead for the left character, a line break after the
// character can’t be suppressed since the right neighbour character
// doesn’t get examined.
type Tokencflags struct { /* n */
}

func ParseTokencflags(args string) (Token, error) {
	return &Tokencflags{ /* n */ }, nil
}

func (t *Tokencflags) String() string {
	return ".cflags" /* n */
}

var _ Token = (*Tokencflags)(nil)

type Tokenchar struct { /* g */
}

func ParseTokenchar(args string) (Token, error) {
	return &Tokenchar{ /* g */ }, nil
}

func (t *Tokenchar) String() string {
	return ".char" /* g */
}

var _ Token = (*Tokenchar)(nil)

type Tokenfchar struct { /* g */
}

func ParseTokenfchar(args string) (Token, error) {
	return &Tokenfchar{ /* g */ }, nil
}

func (t *Tokenfchar) String() string {
	return ".fchar" /* g */
}

var _ Token = (*Tokenfchar)(nil)

type Tokenfschar struct { /* f */
}

func ParseTokenfschar(args string) (Token, error) {
	return &Tokenfschar{ /* f */ }, nil
}

func (t *Tokenfschar) String() string {
	return ".fschar" /* f */
}

var _ Token = (*Tokenfschar)(nil)

// Tokenschar Define a new glyph g to be string (which can be
// empty).20  Every time glyph g needs to be printed,
// string is processed in a temporary environment and the result is
// wrapped up into a single object.  Compatibility mode is turned off and
// the escape character is set to ‘\’ while string is being
// processed.  Any emboldening, constant spacing or track kerning is
// applied to this object rather than to individual characters in
// string.
//
// A glyph defined by these requests can be used just like a normal glyph
// provided by the output device.  In particular, other characters can be
// translated to it with the tr or trin requests; it can be
// made the leader character by the lc request; repeated patterns
// can be drawn with the glyph using the \l and \L escape
// sequences; words containing the glyph can be hyphenated correctly if the
// hcode request is used to give the glyph’s symbol a hyphenation
// code.
//
// There is a special anti-recursion feature: Use of g within the
// glyph’s definition is handled like normal characters and symbols not
// defined with char.
//
// Note that the tr and trin requests take precedence if
// char accesses the same symbol.
//
// .tr XY
// X
//
//	⇒ Y
//
// .char X Z
// X
//
//	⇒ Y
//
// .tr XX
// X
//
//	⇒ Z
//
// The fchar request defines a fallback glyph: gtroff only
// checks for glyphs defined with fchar if it cannot find the glyph
// in the current font.  gtroff carries out this test before
// checking special fonts.
//
// fschar defines a fallback glyph for font f:
// gtroff checks for glyphs defined with fschar after the
// list of fonts declared as font-specific special fonts with the
// fspecial request, but before the list of fonts declared as global
// special fonts with the special request.
//
// Finally, the schar request defines a global fallback glyph:
// gtroff checks for glyphs defined with schar after the list
// of fonts declared as global special fonts with the special
// request, but before the already mounted special fonts.
//
// See Using Symbols, for a detailed description of the glyph searching
// mechanism in gtroff.
type Tokenschar struct { /* g */
}

func ParseTokenschar(args string) (Token, error) {
	return &Tokenschar{ /* g */ }, nil
}

func (t *Tokenschar) String() string {
	return ".schar" /* g */
}

var _ Token = (*Tokenschar)(nil)

type Tokenrchar struct { /* c1 */
}

func ParseTokenrchar(args string) (Token, error) {
	return &Tokenrchar{ /* c1 */ }, nil
}

func (t *Tokenrchar) String() string {
	return ".rchar" /* c1 */
}

var _ Token = (*Tokenrchar)(nil)

// Tokenrfschar Remove the definitions of glyphs c1, c2, ...
// This undoes the effect of a char, fchar, or schar
// request.
//
// It is possible to omit the whitespace between arguments.
//
// The request rfschar removes glyph definitions defined with
// fschar for glyph f.
type Tokenrfschar struct { /* f */
}

func ParseTokenrfschar(args string) (Token, error) {
	return &Tokenrfschar{ /* f */ }, nil
}

func (t *Tokenrfschar) String() string {
	return ".rfschar" /* f */
}

var _ Token = (*Tokenrfschar)(nil)

// Tokenclass In groff, a character class (or simply “class”) is a set
// of characters, grouped by some user aspect.  The class request
// defines such classes so that other requests can refer to all characters
// belonging to this set with a single class name.  Currently, only the
// cflags request can handle character classes.
//
// A class request takes a class name followed by a list of
// entities.  In its simplest form, the entities are characters or symbols:
//
// .class [prepunct] , : ; > }
//
// Since class and glyph names share the same namespace, it is recommended
// to start and end the class name with [ and ],
// respectively, to avoid collisions with normal groff symbols (and
// symbols defined by the user).  In particular, the presence of ]
// in the symbol name intentionally prevents the usage of \[...],
// thus you must use the \C escape to access a class with such a
// name.
//
// You can also use a special character range notation, consisting of a
// start character or symbol, followed by ‘-’, and an end character or
// symbol.  Internally, gtroff converts these two symbol names to
// Unicode values (according to the groff glyph gist), which then give the
// start and end value of the range.  If that fails, the class definition
// is skipped.
//
// Finally, classes can be nested, too.
//
// Here is a more complex example:
//
// .class [prepunctx] \C'[prepunct]' \[u2013]-\[u2016]
//
// The class ‘prepunctx’ now contains the contents of the class
// prepunct as defined above (the set ‘, : ; > }’), and
// characters in the range between U+2013 and U+2016.
//
// If you want to add ‘-’ to a class, it must be the first character
// value in the argument list, otherwise it gets misinterpreted as a range.
//
// Note that it is not possible to use class names within range
// definitions.
//
// Typical use of the class request is to control line-breaking and
// hyphenation rules as defined by the cflags request.  For example,
// to inhibit line breaks before the characters belonging to the
// prepunctx class, you can write:
//
// .cflags 2 \C'[prepunctx]'
//
// See the cflags request in Using Symbols, for more details.
type Tokenclass struct { /* n */
}

func ParseTokenclass(args string) (Token, error) {
	return &Tokenclass{ /* n */ }, nil
}

func (t *Tokenclass) String() string {
	return ".class" /* n */
}

var _ Token = (*Tokenclass)(nil)

type Tokenspecial struct { /* [s1 */
}

func ParseTokenspecial(args string) (Token, error) {
	return &Tokenspecial{ /* [s1 */ }, nil
}

func (t *Tokenspecial) String() string {
	return ".special" /* [s1 */
}

var _ Token = (*Tokenspecial)(nil)

// Tokenfspecial Use the special request to define special fonts.  Initially, this
// list is empty.
//
// Use the fspecial request to designate special fonts only when
// font f is active.  Initially, this list is empty.
//
// Previous calls to special or fspecial are overwritten;
// without arguments, the particular list of special fonts is set to empty.
// Special fonts are searched in the order they appear as arguments.
//
// All fonts that appear in a call to special or fspecial
// are loaded.
//
// See Using Symbols, for the exact search order of glyphs.
type Tokenfspecial struct { /* f */
}

func ParseTokenfspecial(args string) (Token, error) {
	return &Tokenfspecial{ /* f */ }, nil
}

func (t *Tokenfspecial) String() string {
	return ".fspecial" /* f */
}

var _ Token = (*Tokenfspecial)(nil)

// Tokenul The ul request normally underlines subsequent lines if a TTY
// output device is used.  Otherwise, the lines are printed in italics
// (only the term ‘underlined’ is used in the following).  The single
// argument is the number of input lines to be underlined; with no
// argument, the next line is underlined.  If lines is zero or
// negative, stop the effects of ul (if it was active).  Requests
// and empty lines do not count for computing the number of underlined
// input lines, even if they produce some output like tl.  Lines
// inserted by macros (e.g. invoked by a trap) do count.
//
// At the beginning of ul, the current font is stored and the
// underline font is activated.  Within the span of a ul request, it
// is possible to change fonts, but after the last line affected by
// ul the saved font is restored.
//
// This number of lines still to be underlined is associated with the
// current environment (see Environments).  The underline font can be
// changed with the uf request.
//
// The ul request does not underline spaces.
type Tokenul struct { /* [lines] */
}

func ParseTokenul(args string) (Token, error) {
	return &Tokenul{ /* [lines] */ }, nil
}

func (t *Tokenul) String() string {
	return ".ul" /* [lines] */
}

var _ Token = (*Tokenul)(nil)

// Tokencu The cu request is similar to ul but underlines spaces as
// well (if a TTY output device is used).
type Tokencu struct { /* [lines] */
}

func ParseTokencu(args string) (Token, error) {
	return &Tokencu{ /* [lines] */ }, nil
}

func (t *Tokencu) String() string {
	return ".cu" /* [lines] */
}

var _ Token = (*Tokencu)(nil)

// Tokenuf Set the underline font (globally) used by ul and cu.  By
// default, this is the font at position 2.  font can be either
// a non-negative font position or the name of a font.
type Tokenuf struct { /* font */
}

func ParseTokenuf(args string) (Token, error) {
	return &Tokenuf{ /* font */ }, nil
}

func (t *Tokenuf) String() string {
	return ".uf" /* font */
}

var _ Token = (*Tokenuf)(nil)

// Tokenbd Register: \n[.b]
type Tokenbd struct { /* font1 | font */
}

func ParseTokenbd(args string) (Token, error) {
	return &Tokenbd{ /* font1 | font */ }, nil
}

func (t *Tokenbd) String() string {
	return ".bd" /* font1 | font */
}

var _ Token = (*Tokenbd)(nil)

// Tokencs Switch to and from constant glyph space mode.  If activated, the
// width of every glyph is width/36 ems.  The em size is given
// absolutely by em-size; if this argument is missing, the em value
// is taken from the current font size (as set with the ps request)
// when the font is effectively in use.  Without second and third argument,
// constant glyph space mode is deactivated.
//
// Default scaling indicator for em-size is ‘z’; width is
// an integer.
type Tokencs struct { /* font */
}

func ParseTokencs(args string) (Token, error) {
	return &Tokencs{ /* font */ }, nil
}

func (t *Tokencs) String() string {
	return ".cs" /* font */
}

var _ Token = (*Tokencs)(nil)

type Tokenlg struct { /* [flag] */
}

func ParseTokenlg(args string) (Token, error) {
	return &Tokenlg{ /* [flag] */ }, nil
}

func (t *Tokenlg) String() string {
	return ".lg" /* [flag] */
}

var _ Token = (*Tokenlg)(nil)

type Tokenkern struct { /* [flag] */
}

func ParseTokenkern(args string) (Token, error) {
	return &Tokenkern{ /* [flag] */ }, nil
}

func (t *Tokenkern) String() string {
	return ".kern" /* [flag] */
}

var _ Token = (*Tokenkern)(nil)

// Tokentkf Enable track kerning for font f.  If the current font
// is f the width of every glyph is increased by an amount
// between n1 and n2 (n1, n2 can be negative); if
// the current point size is less than or equal to s1 the width is
// increased by n1; if it is greater than or equal to s2 the
// width is increased by n2; if the point size is greater than or
// equal to s1 and less than or equal to s2 the increase in
// width is a linear function of the point size.
//
// The default scaling indicator is ‘z’ for s1 and s2,
// ‘p’ for n1 and n2.
//
// Note that the track kerning amount is added even to the rightmost glyph
// in a line; for large values it is thus recommended to increase the line
// length by the same amount to compensate it.
type Tokentkf struct { /* f */
}

func ParseTokentkf(args string) (Token, error) {
	return &Tokentkf{ /* f */ }, nil
}

func (t *Tokentkf) String() string {
	return ".tkf" /* f */
}

var _ Token = (*Tokentkf)(nil)

// Tokenps Escape: \ssize
// Request: .ps -size
type Tokenps struct { /* -size | +size | [size] */
}

func ParseTokenps(args string) (Token, error) {
	return &Tokenps{ /* -size | +size | [size] */ }, nil
}

func (t *Tokenps) String() string {
	return ".ps" /* -size | +size | [size] */
}

var _ Token = (*Tokenps)(nil)

// Tokensizes Some devices may only have certain permissible sizes, in which case
// gtroff rounds to the nearest permissible size.  The DESC
// file specifies which sizes are permissible for the device.
//
// Use the sizes request to change the permissible sizes for the
// current output device.  Arguments are in scaled points; the
// sizescale line in the DESC file for the output device
// provides the scaling factor.  For example, if the scaling factor is
// 1000, then the value 12000 is 12 points.
//
// Each argument can be a single point size (such as ‘12000’), or a
// range of sizes (such as ‘4000-72000’).  You can optionally end the
// list with a zero.
type Tokensizes struct { /* s1 */
}

func ParseTokensizes(args string) (Token, error) {
	return &Tokensizes{ /* s1 */ }, nil
}

func (t *Tokensizes) String() string {
	return ".sizes" /* s1 */
}

var _ Token = (*Tokensizes)(nil)

// Tokenvs Register: \n[.v]
// Request: .vs -space
type Tokenvs struct { /* -space | +space | [space] */
}

func ParseTokenvs(args string) (Token, error) {
	return &Tokenvs{ /* -space | +space | [space] */ }, nil
}

func (t *Tokenvs) String() string {
	return ".vs" /* -space | +space | [space] */
}

var _ Token = (*Tokenvs)(nil)

// Tokenpvs Register: \n[.pvs]
// Request: .pvs -space
type Tokenpvs struct { /* -space | +space | [space] */
}

func ParseTokenpvs(args string) (Token, error) {
	return &Tokenpvs{ /* -space | +space | [space] */ }, nil
}

func (t *Tokenpvs) String() string {
	return ".pvs" /* -space | +space | [space] */
}

var _ Token = (*Tokenpvs)(nil)

type Tokends struct { /* name */
}

func ParseTokends(args string) (Token, error) {
	return &Tokends{ /* name */ }, nil
}

func (t *Tokends) String() string {
	return ".ds" /* name */
}

var _ Token = (*Tokends)(nil)

type Tokends1 struct { /* name */
}

func ParseTokends1(args string) (Token, error) {
	return &Tokends1{ /* name */ }, nil
}

func (t *Tokends1) String() string {
	return ".ds1" /* name */
}

var _ Token = (*Tokends1)(nil)

type Tokenas struct { /* name */
}

func ParseTokenas(args string) (Token, error) {
	return &Tokenas{ /* name */ }, nil
}

func (t *Tokenas) String() string {
	return ".as" /* name */
}

var _ Token = (*Tokenas)(nil)

// Tokenas1 The as request is similar to ds but appends string
// to the string stored as name instead of redefining it.  If
// name doesn’t exist yet, it is created.
//
// .as sign " with shallots, onions and garlic,
//
// The as1 request is similar to as, but compatibility mode
// is switched off while the appended string is interpreted.  To be more
// precise, a compatibility save input token is inserted at the
// beginning of the appended string, and a compatibility restore
// input token at the end.
type Tokenas1 struct { /* name */
}

func ParseTokenas1(args string) (Token, error) {
	return &Tokenas1{ /* name */ }, nil
}

func (t *Tokenas1) String() string {
	return ".as1" /* name */
}

var _ Token = (*Tokenas1)(nil)

// Tokensubstring Replace the string named str with the substring defined by the
// indices n1 and n2.  The first character in the string has
// index 0.  If n2 is omitted, it is implicitly set to the
// largest valid value (the string length minus one).  If the index value
// n1 or n2 is negative, it is counted from the end of the
// string, going backwards: The last character has index -1,
// the character before the last character has index -2, etc.
//
// .ds xxx abcdefgh
// .substring xxx 1 -4
// \*[xxx]
//
//	⇒ bcde
//
// .substring xxx 2
// \*[xxx]
//
//	⇒ de
type Tokensubstring struct { /* str */
}

func ParseTokensubstring(args string) (Token, error) {
	return &Tokensubstring{ /* str */ }, nil
}

func (t *Tokensubstring) String() string {
	return ".substring" /* str */
}

var _ Token = (*Tokensubstring)(nil)

// Tokenlength Compute the number of characters of str and return it in the
// number register reg.  If reg doesn’t exist, it is created.
// str is read in copy mode.
//
// .ds xxx abcd\h'3i'efgh
// .length yyy \*[xxx]
// \n[yyy]
//
//	⇒ 14
type Tokenlength struct { /* reg */
}

func ParseTokenlength(args string) (Token, error) {
	return &Tokenlength{ /* reg */ }, nil
}

func (t *Tokenlength) String() string {
	return ".length" /* reg */
}

var _ Token = (*Tokenlength)(nil)

// Tokenrn Rename the request, macro, diversion, or string xx to yy.
type Tokenrn struct { /* xx */
}

func ParseTokenrn(args string) (Token, error) {
	return &Tokenrn{ /* xx */ }, nil
}

func (t *Tokenrn) String() string {
	return ".rn" /* xx */
}

var _ Token = (*Tokenrn)(nil)

// Tokenrm Remove the request, macro, diversion, or string xx.  gtroff
// treats subsequent invocations as if the object had never been defined.
type Tokenrm struct { /* xx */
}

func ParseTokenrm(args string) (Token, error) {
	return &Tokenrm{ /* xx */ }, nil
}

func (t *Tokenrm) String() string {
	return ".rm" /* xx */
}

var _ Token = (*Tokenrm)(nil)

// Tokenals Create an alias named new for the request, string, macro, or
// diversion object named old.  The new name and the old name are
// exactly equivalent (it is similar to a hard rather than a soft link).  If
// old is undefined, gtroff generates a warning of type
// ‘mac’ and ignores the request.
//
// To understand how the als request works it is probably best to
// think of two different pools: one pool for objects (macros, strings,
// etc.), and another one for names.  As soon as an object is defined,
// gtroff adds it to the object pool, adds its name to the name
// pool, and creates a link between them.  When als creates an
// alias, it adds a new name to the name pool that gets linked to the same
// object as the old name.
//
// Now consider this example.
//
// .de foo
// ..
// .
// .als bar foo
// .
// .de bar
// .  foo
// ..
// .
// .bar
//
//	⇒ input stack limit exceeded
//
// The definition of macro bar replaces the old object this name is
// linked to.  However, the alias to foo is still active!  In
// other words, foo is still linked to the same object as
// bar, and the result of calling bar is an infinite,
// recursive loop that finally leads to an error.
//
// To undo an alias, simply call rm on the aliased name.  The object
// itself is not destroyed until there are no more aliases.
type Tokenals struct { /* new */
}

func ParseTokenals(args string) (Token, error) {
	return &Tokenals{ /* new */ }, nil
}

func (t *Tokenals) String() string {
	return ".als" /* new */
}

var _ Token = (*Tokenals)(nil)

// Tokenchop Remove (chop) the last character from the macro, string, or diversion
// named xx.  This is useful for removing the newline from the end of
// diversions that are to be interpolated as strings.  This command can be
// used repeatedly; see Gtroff Internals, for details on nodes
// inserted additionally by gtroff.
type Tokenchop struct { /* xx */
}

func ParseTokenchop(args string) (Token, error) {
	return &Tokenchop{ /* xx */ }, nil
}

func (t *Tokenchop) String() string {
	return ".chop" /* xx */
}

var _ Token = (*Tokenchop)(nil)

// Tokenif Evaluate the expression expr, and executes anything (the
// remainder of the line) if expr evaluates to a value greater than
// zero (true).  anything is interpreted as though it was on a line
// by itself (except that leading spaces are swallowed).
// See Operators in Conditionals, for more info.
//
// .nr xxx 1
// .nr yyy 2
// .if ((\n[xxx] == 1) & (\n[yyy] == 2)) true
//
//	⇒ true
type Tokenif struct { /* expr */
}

func ParseTokenif(args string) (Token, error) {
	return &Tokenif{ /* expr */ }, nil
}

func (t *Tokenif) String() string {
	return ".if" /* expr */
}

var _ Token = (*Tokenif)(nil)

// Tokennop Executes anything.  This is similar to .if 1.
type Tokennop struct { /* anything */
}

func ParseTokennop(args string) (Token, error) {
	return &Tokennop{ /* anything */ }, nil
}

func (t *Tokennop) String() string {
	return ".nop" /* anything */
}

var _ Token = (*Tokennop)(nil)

type Tokenie struct { /* expr */
}

func ParseTokenie(args string) (Token, error) {
	return &Tokenie{ /* expr */ }, nil
}

func (t *Tokenie) String() string {
	return ".ie" /* expr */
}

var _ Token = (*Tokenie)(nil)

// Tokenel Use the ie and el requests to write an if-then-else.  The
// first request is the ‘if’ part and the latter is the ‘else’ part.
//
// .ie n .ls 2 \" double-spacing in nroff
// .el   .ls 1 \" single-spacing in troff
type Tokenel struct { /* anything */
}

func ParseTokenel(args string) (Token, error) {
	return &Tokenel{ /* anything */ }, nil
}

func (t *Tokenel) String() string {
	return ".el" /* anything */
}

var _ Token = (*Tokenel)(nil)

// Tokenwhile Evaluate the expression expr, and repeatedly execute
// anything (the remainder of the line) until expr evaluates
// to 0.
//
// .nr a 0 1
// .while (\na < 9) \{\
// \n+a,
// .\}
// \n+a
//
//	⇒ 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
//
// Some remarks.
//
//	The body of a while request is treated like the body of a
//
// de request: gtroff temporarily stores it in a macro that
// is deleted after the loop has been exited.  It can considerably slow
// down a macro if the body of the while request (within the macro)
// is large.  Each time the macro is executed, the while body is
// parsed and stored again as a temporary macro.
//
// .de xxx
// .  nr num 10
// .  while (\\n[num] > 0) \{\
// .    \" many lines of code
// .    nr num -1
// .  \}
// ..
//
// The traditional and often better solution (Unix troff doesn’t
// have the while request) is to use a recursive macro instead that
// is parsed only once during its definition.
//
// .de yyy
// .  if (\\n[num] > 0) \{\
// .    \" many lines of code
// .    nr num -1
// .    yyy
// .  \}
// ..
// .
// .de xxx
// .  nr num 10
// .  yyy
// ..
//
// Note that the number of available recursion levels is set to 1000
// (this is a compile-time constant value of gtroff).
//
//	The closing brace of a while body must end a line.
//
// .if 1 \{\
// .  nr a 0 1
// .  while (\n[a] < 10) \{\
// .    nop \n+[a]
// .\}\}
//
//	⇒ unbalanced \{ \}
type Tokenwhile struct { /* expr */
}

func ParseTokenwhile(args string) (Token, error) {
	return &Tokenwhile{ /* expr */ }, nil
}

func (t *Tokenwhile) String() string {
	return ".while" /* expr */
}

var _ Token = (*Tokenwhile)(nil)

// Tokenbreak Break out of a while loop.  Be sure not to confuse this with the
// br request (causing a line break).
type Tokenbreak struct { /*  */
}

func ParseTokenbreak(args string) (Token, error) {
	return &Tokenbreak{ /*  */ }, nil
}

func (t *Tokenbreak) String() string {
	return ".break" /*  */
}

var _ Token = (*Tokenbreak)(nil)

// Tokencontinue Finish the current iteration of a while loop, immediately
// restarting the next iteration.
type Tokencontinue struct { /*  */
}

func ParseTokencontinue(args string) (Token, error) {
	return &Tokencontinue{ /*  */ }, nil
}

func (t *Tokencontinue) String() string {
	return ".continue" /*  */
}

var _ Token = (*Tokencontinue)(nil)

type Tokende struct { /* name */
}

func ParseTokende(args string) (Token, error) {
	return &Tokende{ /* name */ }, nil
}

func (t *Tokende) String() string {
	return ".de" /* name */
}

var _ Token = (*Tokende)(nil)

type Tokende1 struct { /* name */
}

func ParseTokende1(args string) (Token, error) {
	return &Tokende1{ /* name */ }, nil
}

func (t *Tokende1) String() string {
	return ".de1" /* name */
}

var _ Token = (*Tokende1)(nil)

type Tokendei struct { /* name */
}

func ParseTokendei(args string) (Token, error) {
	return &Tokendei{ /* name */ }, nil
}

func (t *Tokendei) String() string {
	return ".dei" /* name */
}

var _ Token = (*Tokendei)(nil)

// Tokendei1 Define a new macro named name.  gtroff copies subsequent
// lines (starting with the next one) into an internal buffer until it
// encounters the line ‘..’ (two dots).  If the optional second
// argument to de is present it is used as the macro closure
// request instead of ‘..’.
//
// There can be whitespace after the first dot in the line containing the
// ending token (either ‘.’ or macro ‘end’).  Don’t insert
// a tab character immediately after the ‘..’, otherwise it isn’t
// recognized as the end-of-macro symbol.24
//
// Here a small example macro called ‘P’ that causes a break and
// inserts some vertical space.  It could be used to separate paragraphs.
//
// .de P
// .  br
// .  sp .8v
// ..
//
// The following example defines a macro within another.  Remember that
// expansion must be protected twice; once for reading the macro and once
// for executing.
//
// \# a dummy macro to avoid a warning
// .de end
// ..
// .
// .de foo
// .  de bar end
// .    nop \f[B]Hello \\\\$1!\f[]
// .  end
// ..
// .
// .foo
// .bar Joe
//
//	⇒ Hello Joe!
//
// Since \f has no expansion, it isn’t necessary to protect its
// backslash.  Had we defined another macro within bar that takes a
// parameter, eight backslashes would be necessary before ‘$1’.
//
// The de1 request turns off compatibility mode while executing the
// macro.  On entry, the current compatibility mode is saved and restored
// at exit.
//
// .nr xxx 12345
// .
// .de aa
// The value of xxx is \\n[xxx].
// ..
// .de1 bb
// The value of xxx is \\n[xxx].
// ..
// .
// .cp 1
// .
// .aa
//
//	⇒ warning: number register `[' not defined
//	⇒ The value of xxx is 0xxx].
//
// .bb
//
//	⇒ The value of xxx is 12345.
//
// The dei request defines a macro indirectly.  That is, it expands
// strings whose names are name or end before performing the
// append.
//
// This:
//
// .ds xx aa
// .ds yy bb
// .dei xx yy
//
// is equivalent to:
//
// .de aa bb
//
// The dei1 request is similar to dei but with compatibility
// mode switched off during execution of the defined macro.
//
// If compatibility mode is on, de (and dei) behave similar
// to de1 (and dei1): A ‘compatibility save’ token is
// inserted at the beginning, and a ‘compatibility restore’ token at the
// end, with compatibility mode switched on during execution.  See Gtroff Internals, for more information on switching compatibility mode on and
// off in a single document.
//
// Using trace.tmac, you can trace calls to de and
// de1.
//
// Note that macro identifiers are shared with identifiers for strings and
// diversions.
//
// See the description of the als request, for possible
// pitfalls if redefining a macro that has been aliased.
type Tokendei1 struct { /* name */
}

func ParseTokendei1(args string) (Token, error) {
	return &Tokendei1{ /* name */ }, nil
}

func (t *Tokendei1) String() string {
	return ".dei1" /* name */
}

var _ Token = (*Tokendei1)(nil)

type Tokenam struct { /* name */
}

func ParseTokenam(args string) (Token, error) {
	return &Tokenam{ /* name */ }, nil
}

func (t *Tokenam) String() string {
	return ".am" /* name */
}

var _ Token = (*Tokenam)(nil)

type Tokenam1 struct { /* name */
}

func ParseTokenam1(args string) (Token, error) {
	return &Tokenam1{ /* name */ }, nil
}

func (t *Tokenam1) String() string {
	return ".am1" /* name */
}

var _ Token = (*Tokenam1)(nil)

type Tokenami struct { /* name */
}

func ParseTokenami(args string) (Token, error) {
	return &Tokenami{ /* name */ }, nil
}

func (t *Tokenami) String() string {
	return ".ami" /* name */
}

var _ Token = (*Tokenami)(nil)

// Tokenami1 Works similarly to de except it appends onto the macro named
// name.  So, to make the previously defined ‘P’ macro actually
// do indented instead of block paragraphs, add the necessary code to the
// existing macro like this:
//
// .am P
// .ti +5n
// ..
//
// The am1 request turns off compatibility mode while executing the
// appended macro piece.  To be more precise, a compatibility save
// input token is inserted at the beginning of the appended code, and a
// compatibility restore input token at the end.
//
// The ami request appends indirectly, meaning that gtroff
// expands strings whose names are name or end before
// performing the append.
//
// The ami1 request is similar to ami but compatibility mode
// is switched off during execution of the defined macro.
//
// Using trace.tmac, you can trace calls to am and
// am1.
type Tokenami1 struct { /* name */
}

func ParseTokenami1(args string) (Token, error) {
	return &Tokenami1{ /* name */ }, nil
}

func (t *Tokenami1) String() string {
	return ".ami1" /* name */
}

var _ Token = (*Tokenami1)(nil)

// Tokenreturn Exit a macro, immediately returning to the caller.
//
// If called with an argument, exit twice, namely the current macro and the
// macro one level higher.  This is used to define a wrapper macro for
// return in trace.tmac.
type Tokenreturn struct { /* [anything] */
}

func ParseTokenreturn(args string) (Token, error) {
	return &Tokenreturn{ /* [anything] */ }, nil
}

func (t *Tokenreturn) String() string {
	return ".return" /* [anything] */
}

var _ Token = (*Tokenreturn)(nil)

// Tokenshift Shift the arguments 1 position, or as many positions as specified
// by its argument.  After executing this request, argument i
// becomes argument i-n; arguments 1 to n are
// no longer available.  Shifting by negative amounts is currently
// undefined.
//
// The register .$ is adjusted accordingly.
type Tokenshift struct { /* [n] */
}

func ParseTokenshift(args string) (Token, error) {
	return &Tokenshift{ /* [n] */ }, nil
}

func (t *Tokenshift) String() string {
	return ".shift" /* [n] */
}

var _ Token = (*Tokenshift)(nil)

type Tokenmk struct { /* [reg] */
}

func ParseTokenmk(args string) (Token, error) {
	return &Tokenmk{ /* [reg] */ }, nil
}

func (t *Tokenmk) String() string {
	return ".mk" /* [reg] */
}

var _ Token = (*Tokenmk)(nil)

// Tokenrt The request mk can be used to mark a location on a page, for
// movement to later.  This request takes a register name as an argument in
// which to store the current page location.  With no argument it stores
// the location in an internal register.  The results of this can be used
// later by the rt or the sp request (or the \v
// escape).
//
// The rt request returns upwards to the location marked with
// the last mk request.  If used with an argument, return to a
// position which distance from the top of the page is dist (no
// previous call to mk is necessary in this case).  Default scaling
// indicator is ‘v’.
//
// If a page break occurs between a mk request and its
// matching rt request, the rt is silently ignored.
//
// Here a primitive solution for a two-column macro.
//
// .nr column-length 1.5i
// .nr column-gap 4m
// .nr bottom-margin 1m
// .
//
// .de 2c
// .  br
// .  mk
// .  ll \\n[column-length]u
// .  wh -\\n[bottom-margin]u 2c-trap
// .  nr right-side 0
// ..
// .
//
// .de 2c-trap
// .  ie \\n[right-side] \{\
// .    nr right-side 0
// .    po -(\\n[column-length]u + \\n[column-gap]u)
// .    \" remove trap
// .    wh -\\n[bottom-margin]u
// .  \}
// .  el \{\
// .    \" switch to right side
// .    nr right-side 1
// .    po +(\\n[column-length]u + \\n[column-gap]u)
// .    rt
// .  \}
// ..
// .
//
// .pl 1.5i
// .ll 4i
// This is a small test that shows how the
// rt request works in combination with mk.
//
// .2c
// Starting here, text is typeset in two columns.
// Note that this implementation isn't robust
// and thus not suited for a real two-column
// macro.
//
// Result:
//
// This is a small test that shows how the
// rt request works in combination with mk.
//
// Starting  here,    isn't    robust
// text is typeset    and   thus  not
// in two columns.    suited  for   a
// Note that  this    real two-column
// implementation     macro.
type Tokenrt struct { /* [dist] */
}

func ParseTokenrt(args string) (Token, error) {
	return &Tokenrt{ /* [dist] */ }, nil
}

func (t *Tokenrt) String() string {
	return ".rt" /* [dist] */
}

var _ Token = (*Tokenrt)(nil)

type Tokenvpt struct { /* flag */
}

func ParseTokenvpt(args string) (Token, error) {
	return &Tokenvpt{ /* flag */ }, nil
}

func (t *Tokenvpt) String() string {
	return ".vpt" /* flag */
}

var _ Token = (*Tokenvpt)(nil)

// Tokenwh Set a page location trap.  Non-negative values for dist set the
// trap relative to the top of the page; negative values set the trap
// relative to the bottom of the page.  Default scaling indicator is
// ‘v’; values of dist are always rounded to be multiples of the
// vertical resolution (as given in register .V).
//
// macro is the name of the macro to execute when the trap is sprung.
// If macro is missing, remove the first trap (if any) at dist.
//
// The following is a simple example of how many macro packages set headers
// and footers.
//
// .de hd                \" Page header
// '  sp .5i
// .  tl 'Title”date'
// '  sp .3i
// ..
// .
// .de fo                \" Page footer
// '  sp 1v
// .  tl ”%”
// '  bp
// ..
// .
// .wh 0   hd            \" trap at top of the page
// .wh -1i fo            \" trap one inch from bottom
//
// A trap at or below the bottom of the page is ignored; it can be made
// active by either moving it up or increasing the page length so that the
// trap is on the page.
//
// Negative trap values always use the current page length; they are
// not converted to an absolute vertical position:
//
// .pl 5i
// .wh -1i xx
// .ptr
//
//	⇒ xx      -240
//
// .pl 100i
// .ptr
//
//	⇒ xx      -240
//
// It is possible to have more than one trap at the same location; to do
// so, the traps must be defined at different locations, then moved
// together with the ch request; otherwise the second trap would
// replace the first one.  Earlier defined traps hide later defined traps
// if moved to the same position (the many empty lines caused by the
// bp request are omitted in the following example):
//
// .de a
// .  nop a
// ..
// .de b
// .  nop b
// ..
// .de c
// .  nop c
// ..
// .
// .wh 1i a
// .wh 2i b
// .wh 3i c
// .bp
//
//	⇒ a b c
//
// .ch b 1i
// .ch c 1i
// .bp
//
//	⇒ a
//
// .ch a 0.5i
// .bp
//
//	⇒ a b
type Tokenwh struct { /* dist */
}

func ParseTokenwh(args string) (Token, error) {
	return &Tokenwh{ /* dist */ }, nil
}

func (t *Tokenwh) String() string {
	return ".wh" /* dist */
}

var _ Token = (*Tokenwh)(nil)

// Tokench Change the location of a trap.  The first argument is the name of the
// macro to be invoked at the trap, and the second argument is the new
// location for the trap (note that the parameters are specified in
// opposite order as in the wh request).  This is useful for
// building up footnotes in a diversion to allow more space at the bottom
// of the page for them.
//
// Default scaling indicator for dist is ‘v’.  If dist is
// missing, the trap is removed.
type Tokench struct { /* macro */
}

func ParseTokench(args string) (Token, error) {
	return &Tokench{ /* macro */ }, nil
}

func (t *Tokench) String() string {
	return ".ch" /* macro */
}

var _ Token = (*Tokench)(nil)

// Tokendt Set a trap within a diversion.  dist is the location of the
// trap (identical to the wh request; default scaling indicator is
// ‘v’) and macro is the name of the macro to be invoked.  If
// called without arguments, the diversion trap is removed.
//
// Note that there exists only a single diversion trap.
//
// The number register .t still works within diversions.
// See Diversions, for more information.
type Tokendt struct { /* [dist */
}

func ParseTokendt(args string) (Token, error) {
	return &Tokendt{ /* [dist */ }, nil
}

func (t *Tokendt) String() string {
	return ".dt" /* [dist */
}

var _ Token = (*Tokendt)(nil)

type Tokenit struct { /* n */
}

func ParseTokenit(args string) (Token, error) {
	return &Tokenit{ /* n */ }, nil
}

func (t *Tokenit) String() string {
	return ".it" /* n */
}

var _ Token = (*Tokenit)(nil)

// Tokenitc Set an input line trap.  n is the number of lines of input
// that may be read before springing the trap, macro is the macro to
// be invoked.  Request lines are not counted as input lines.
//
// For example, one possible use is to have a macro that prints the next
// n lines in a bold font.
//
// .de B
// .  it \\$1 B-end
// .  ft B
// ..
// .
// .de B-end
// .  ft R
// ..
//
// The itc request is identical except that an interrupted text line
// (ending with \c) is not counted as a separate line.
//
// Both requests are associated with the current environment
// (see Environments); switching to another environment disables the
// current input trap, and going back reactivates it, restoring the number
// of already processed lines.
type Tokenitc struct { /* n */
}

func ParseTokenitc(args string) (Token, error) {
	return &Tokenitc{ /* n */ }, nil
}

func (t *Tokenitc) String() string {
	return ".itc" /* n */
}

var _ Token = (*Tokenitc)(nil)

// Tokenblm Set a blank line trap.  gtroff executes macro when it
// encounters a blank line in the input file.
type Tokenblm struct { /* macro */
}

func ParseTokenblm(args string) (Token, error) {
	return &Tokenblm{ /* macro */ }, nil
}

func (t *Tokenblm) String() string {
	return ".blm" /* macro */
}

var _ Token = (*Tokenblm)(nil)

type Tokenlsm struct { /* macro */
}

func ParseTokenlsm(args string) (Token, error) {
	return &Tokenlsm{ /* macro */ }, nil
}

func (t *Tokenlsm) String() string {
	return ".lsm" /* macro */
}

var _ Token = (*Tokenlsm)(nil)

// Tokenem Set a trap at the end of input.  macro is executed after the last
// line of the input file has been processed.
//
// For example, if the document had to have a section at the bottom of the
// last page for someone to approve it, the em request could be
// used.
//
// .de approval
// \c
// .  ne 3v
// .  sp (\\n[.t]u - 3v)
// .  in +4i
// .  lc _
// .  br
// Approved:\t\a
// .  sp
// Date:\t\t\a
// ..
// .
// .em approval
//
// The \c in the above example needs explanation.  For historical
// reasons (and for compatibility with AT&T troff), the
// end macro exits as soon as it causes a page break and no remaining
// data is in the partially collected line.
//
// Let us assume that there is no \c in the above approval
// macro, and that the page is full and has been ended with, say, a
// br request.  The ne request now causes the start of a new
// page, which in turn makes troff exit immediately for the reasons
// just described.  In most situations this is not intended.
//
// To always force processing the whole end macro independently of this
// behaviour it is thus advisable to insert something that starts an
// empty partially filled line (\c) whenever there is a chance that
// a page break can happen.  In the above example, the call of the
// ne request assures that the remaining code stays on the same
// page, so we have to insert \c only once.
//
// The next example shows how to append three lines, then starting a new
// page unconditionally.  Since ‘.ne 1’ doesn’t give the desired
// effect – there is always one line available or we are already at the
// beginning of the next page – we temporarily increase the page length
// by one line so that we can use ‘.ne 2’.
//
// .de EM
// .pl +1v
// \c
// .ne 2
// line one
// .br
// \c
// .ne 2
// line two
// .br
// \c
// .ne 2
// line three
// .br
// .pl -1v
// \c
// 'bp
// ..
// .em EM
//
// Note that this specific feature affects only the first potential page
// break caused by the end macro; further page breaks emitted by the end
// macro are handled normally.
//
// Another possible use of the em request is to make gtroff
// emit a single large page instead of multiple pages.  For example, one
// may want to produce a long plain-text file for reading on-screen.  The
// idea is to set the page length at the beginning of the document to a
// very large value to hold all the text, and automatically adjust it to
// the exact height of the document after the text has been output.
//
// .de adjust-page-length
// .  br
// .  pl \\n[nl]u \" \n[nl] holds the current vert. position
// ..
// .
// .de single-page-mode
// .  pl 99999
// .  em adjust-page-length
// ..
// .
// .\" activate the above code
// .single-page-mode
//
// Since only one end-of-input trap does exist and other macro packages
// may already use it, care must be taken not to break the mechanism.  A
// simple solution would be to append the above macro to the macro
// package’s end-of-input macro using the .am request.
type Tokenem struct { /* macro */
}

func ParseTokenem(args string) (Token, error) {
	return &Tokenem{ /* macro */ }, nil
}

func (t *Tokenem) String() string {
	return ".em" /* macro */
}

var _ Token = (*Tokenem)(nil)

type Tokendi struct { /* macro */
}

func ParseTokendi(args string) (Token, error) {
	return &Tokendi{ /* macro */ }, nil
}

func (t *Tokendi) String() string {
	return ".di" /* macro */
}

var _ Token = (*Tokendi)(nil)

// Tokenda Begin a diversion.  Like the de request, it takes an argument of
// a macro name to divert subsequent text into.  The da macro
// appends to an existing diversion.
//
// di or da without an argument ends the diversion.
//
// The current partially filled line is included into the diversion.  See
// the box request below for an example.  Note that switching to
// another (empty) environment (with the ev request) avoids the
// inclusion of the current partially filled line.
type Tokenda struct { /* macro */
}

func ParseTokenda(args string) (Token, error) {
	return &Tokenda{ /* macro */ }, nil
}

func (t *Tokenda) String() string {
	return ".da" /* macro */
}

var _ Token = (*Tokenda)(nil)

type Tokenbox struct { /* macro */
}

func ParseTokenbox(args string) (Token, error) {
	return &Tokenbox{ /* macro */ }, nil
}

func (t *Tokenbox) String() string {
	return ".box" /* macro */
}

var _ Token = (*Tokenbox)(nil)

// Tokenboxa Begin (or append to) a diversion like the di and da
// requests.  The difference is that box and boxa do not
// include a partially filled line in the diversion.
//
// Compare this:
//
// Before the box.
// .box xxx
// In the box.
// .br
// .box
// After the box.
// .br
//
//	⇒ Before the box.  After the box.
//
// .xxx
//
//	⇒ In the box.
//
// with this:
//
// Before the diversion.
// .di yyy
// In the diversion.
// .br
// .di
// After the diversion.
// .br
//
//	⇒ After the diversion.
//
// .yyy
//
//	⇒ Before the diversion.  In the diversion.
//
// box or boxa without an argument ends the diversion.
type Tokenboxa struct { /* macro */
}

func ParseTokenboxa(args string) (Token, error) {
	return &Tokenboxa{ /* macro */ }, nil
}

func (t *Tokenboxa) String() string {
	return ".boxa" /* macro */
}

var _ Token = (*Tokenboxa)(nil)

// Tokenoutput Emit string directly to the gtroff intermediate output
// (subject to copy mode interpretation); this is similar to \! used
// at the top level.  An initial double quote in string is stripped
// off to allow initial blanks.
//
// This request can’t be used before the first page has started – if you
// get an error, simply insert .br before the output request.
//
// Without argument, output is ignored.
//
// Use with caution!  It is normally only needed for mark-up used by a
// postprocessor that does something with the output before sending it to
// the output device, filtering out string again.
type Tokenoutput struct { /* string */
}

func ParseTokenoutput(args string) (Token, error) {
	return &Tokenoutput{ /* string */ }, nil
}

func (t *Tokenoutput) String() string {
	return ".output" /* string */
}

var _ Token = (*Tokenoutput)(nil)

// Tokenasciify Unformat the diversion specified by div in such a way that
// ASCII characters, characters translated with the trin
// request, space characters, and some escape sequences that were formatted
// and diverted are treated like ordinary input characters when the
// diversion is reread.  It can be also used for gross hacks; for example,
// the following sets register n to 1.
//
// .tr @.
// .di x
// @nr n 1
// .br
// .di
// .tr @@
// .asciify x
// .x
//
// Note that asciify cannot return all items in a diversion back
// to their source equivalent, nodes such as \N[...] will still
// remain as nodes, so the result cannot be guaranteed to be a pure string.
//
// See Copy-in Mode.
type Tokenasciify struct { /* div */
}

func ParseTokenasciify(args string) (Token, error) {
	return &Tokenasciify{ /* div */ }, nil
}

func (t *Tokenasciify) String() string {
	return ".asciify" /* div */
}

var _ Token = (*Tokenasciify)(nil)

// Tokenunformat Like asciify, unformat the specified diversion.  However,
// unformat only unformats spaces and tabs between words.
// Unformatted tabs are treated as input tokens, and spaces are stretchable
// again.
//
// The vertical size of lines is not preserved; glyph information (font,
// font size, space width, etc.) is retained.
type Tokenunformat struct { /* div */
}

func ParseTokenunformat(args string) (Token, error) {
	return &Tokenunformat{ /* div */ }, nil
}

func (t *Tokenunformat) String() string {
	return ".unformat" /* div */
}

var _ Token = (*Tokenunformat)(nil)

type Tokenev struct { /* [env] */
}

func ParseTokenev(args string) (Token, error) {
	return &Tokenev{ /* [env] */ }, nil
}

func (t *Tokenev) String() string {
	return ".ev" /* [env] */
}

var _ Token = (*Tokenev)(nil)

// Tokenevc Copy the environment env into the current environment.
//
// The following environment data is not copied:
//
//	Partially filled lines.
//
//	The status whether the previous line was interrupted.
//
//	The number of lines still to center, or to right-justify, or to
//
// underline (with or without underlined spaces); they are set to zero.
//
//	The status whether a temporary indentation is active.
//
//	Input traps and its associated data.
//
//	Line numbering mode is disabled; it can be reactivated with ‘.nm +0’.
//
//	The number of consecutive hyphenated lines (set to zero).
type Tokenevc struct { /* env */
}

func ParseTokenevc(args string) (Token, error) {
	return &Tokenevc{ /* env */ }, nil
}

func (t *Tokenevc) String() string {
	return ".evc" /* env */
}

var _ Token = (*Tokenevc)(nil)

type Tokencolor struct { /* [n] */
}

func ParseTokencolor(args string) (Token, error) {
	return &Tokencolor{ /* [n] */ }, nil
}

func (t *Tokencolor) String() string {
	return ".color" /* [n] */
}

var _ Token = (*Tokencolor)(nil)

// Tokendefcolor Define color with name ident.  scheme can be one of the
// following values: rgb (three components), cmy (three
// components), cmyk (four components), and gray or
// grey (one component).
//
// Color components can be given either as a hexadecimal string or as
// positive decimal integers in the range 0–65535.  A hexadecimal string
// contains all color components concatenated.  It must start with either
// # or ##; the former specifies hex values in the range
// 0–255 (which are internally multiplied by 257), the latter in the
// range 0–65535.  Examples: #FFC0CB (pink), ##ffff0000ffff
// (magenta).  The default color name value is device-specific (usually black).  It is possible that the
// default color for \m and \M is not identical.
//
// A new scaling indicator f has been introduced, which
// multiplies its value by 65536; this makes it convenient to specify color
// components as fractions in the range 0 to 1 (1f equals 65536u).
// Example:
//
// .defcolor darkgreen rgb 0.1f 0.5f 0.2f
//
// Note that f is the default scaling indicator for the
// defcolor request, thus the above statement is equivalent to
//
// .defcolor darkgreen rgb 0.1 0.5 0.2
type Tokendefcolor struct { /* ident */
}

func ParseTokendefcolor(args string) (Token, error) {
	return &Tokendefcolor{ /* ident */ }, nil
}

func (t *Tokendefcolor) String() string {
	return ".defcolor" /* ident */
}

var _ Token = (*Tokendefcolor)(nil)

type Tokengcolor struct { /* [color] */
}

func ParseTokengcolor(args string) (Token, error) {
	return &Tokengcolor{ /* [color] */ }, nil
}

func (t *Tokengcolor) String() string {
	return ".gcolor" /* [color] */
}

var _ Token = (*Tokengcolor)(nil)

type Tokenfcolor struct { /* [color] */
}

func ParseTokenfcolor(args string) (Token, error) {
	return &Tokenfcolor{ /* [color] */ }, nil
}

func (t *Tokenfcolor) String() string {
	return ".fcolor" /* [color] */
}

var _ Token = (*Tokenfcolor)(nil)

// Tokenso Read in the specified file and includes it in place of the
// so request.  This is quite useful for large documents, e.g.
// keeping each chapter in a separate file.  See gsoelim, for more
// information.
//
// Since gtroff replaces the so request with the contents of
// file, it makes a difference whether the data is terminated with a
// newline or not: Assuming that file xxx contains the word
// ‘foo’ without a final newline, this
//
// This is
// .so xxx
// bar
//
// yields ‘This is foobar’.
//
// The search path for file can be controlled with the -I
// command-line option.
type Tokenso struct { /* file */
}

func ParseTokenso(args string) (Token, error) {
	return &Tokenso{ /* file */ }, nil
}

func (t *Tokenso) String() string {
	return ".so" /* file */
}

var _ Token = (*Tokenso)(nil)

// Tokenpso Read the standard output from the specified command and includes
// it in place of the pso request.
//
// This request causes an error if used in safer mode (which is the
// default).  Use groff’s or troff’s -U option to
// activate unsafe mode.
//
// The comment regarding a final newline for the so request is valid
// for pso also.
type Tokenpso struct { /* command */
}

func ParseTokenpso(args string) (Token, error) {
	return &Tokenpso{ /* command */ }, nil
}

func (t *Tokenpso) String() string {
	return ".pso" /* command */
}

var _ Token = (*Tokenpso)(nil)

// Tokenmso Identical to the so request except that gtroff searches
// for the specified file in the same directories as macro files for
// the -m command-line option.
// If the file name to be included has the form name.tmac and it
// isn’t found, mso tries to include tmac.name and vice
// versa.
// If the file does not exist, a warning of type ‘file’ is emitted.
// See Debugging, for information about warnings.
type Tokenmso struct { /* file */
}

func ParseTokenmso(args string) (Token, error) {
	return &Tokenmso{ /* file */ }, nil
}

func (t *Tokenmso) String() string {
	return ".mso" /* file */
}

var _ Token = (*Tokenmso)(nil)

type Tokentrf struct { /* file */
}

func ParseTokentrf(args string) (Token, error) {
	return &Tokentrf{ /* file */ }, nil
}

func (t *Tokentrf) String() string {
	return ".trf" /* file */
}

var _ Token = (*Tokentrf)(nil)

// Tokencf Transparently output the contents of file.  Each line is output as
// if it were preceded by \!; however, the lines are not
// subject to copy mode interpretation.  If the file does not end with a
// newline, then a newline is added (trf only).  For example, to
// define a macro x containing the contents of
// file f, use
//
// .ev 1
// .di x
// .trf f
// .di
// .ev
//
// The calls to ev prevent that the current partial input line
// becomes part of the diversion.
//
// Both trf and cf, when used in a diversion, embeds an
// object in the diversion which, when reread, causes the contents of
// file to be transparently copied through to the output.  In
// Unix troff, the contents of file is immediately copied
// through to the output regardless of whether there is a current
// diversion; this behaviour is so anomalous that it must be considered a
// bug.
//
// While cf copies the contents of file completely
// unprocessed, trf disallows characters such as NUL that are not
// valid gtroff input characters (see Identifiers).
//
// For cf, within a diversion, ‘completely unprocessed’ means that
// each line of a file to be inserted is handled as if it were preceded by
// \!\\!.
//
// Both requests cause a line break.
type Tokencf struct { /* file */
}

func ParseTokencf(args string) (Token, error) {
	return &Tokencf{ /* file */ }, nil
}

func (t *Tokencf) String() string {
	return ".cf" /* file */
}

var _ Token = (*Tokencf)(nil)

// Tokennx Force gtroff to continue processing of the file specified as an
// argument.  If no argument is given, immediately jump to the end of file.
type Tokennx struct { /* [file] */
}

func ParseTokennx(args string) (Token, error) {
	return &Tokennx{ /* [file] */ }, nil
}

func (t *Tokennx) String() string {
	return ".nx" /* [file] */
}

var _ Token = (*Tokennx)(nil)

// Tokenrd Read from standard input, and include what is read as though it were
// part of the input file.  Text is read until a blank line is encountered.
//
// If standard input is a TTY input device (keyboard), write prompt
// to standard error, followed by a colon (or send BEL for a beep if no
// argument is given).
//
// Arguments after prompt are available for the input.  For example,
// the line
//
// .rd data foo bar
//
// with the input ‘This is \$2.’ prints
//
// This is bar.
type Tokenrd struct { /* [prompt */
}

func ParseTokenrd(args string) (Token, error) {
	return &Tokenrd{ /* [prompt */ }, nil
}

func (t *Tokenrd) String() string {
	return ".rd" /* [prompt */
}

var _ Token = (*Tokenrd)(nil)

// Tokenpi Pipe the output of gtroff to the shell command(s) specified by
// pipe.  This request must occur before gtroff has a chance
// to print anything.
//
// pi causes an error if used in safer mode (which is the default).
// Use groff’s or troff’s -U option to activate
// unsafe mode.
//
// Multiple calls to pi are allowed, acting as a chain.  For
// example,
//
// .pi foo
// .pi bar
// ...
//
// is the same as ‘.pi foo | bar’.
//
// Note that the intermediate output format of gtroff is piped to
// the specified commands.  Consequently, calling groff without the
// -Z option normally causes a fatal error.
type Tokenpi struct { /* pipe */
}

func ParseTokenpi(args string) (Token, error) {
	return &Tokenpi{ /* pipe */ }, nil
}

func (t *Tokenpi) String() string {
	return ".pi" /* pipe */
}

var _ Token = (*Tokenpi)(nil)

type Tokensy struct { /* cmds */
}

func ParseTokensy(args string) (Token, error) {
	return &Tokensy{ /* cmds */ }, nil
}

func (t *Tokensy) String() string {
	return ".sy" /* cmds */
}

var _ Token = (*Tokensy)(nil)

type Tokenopen struct { /* stream */
}

func ParseTokenopen(args string) (Token, error) {
	return &Tokenopen{ /* stream */ }, nil
}

func (t *Tokenopen) String() string {
	return ".open" /* stream */
}

var _ Token = (*Tokenopen)(nil)

// Tokenopena Open the specified file for writing and associates the specified
// stream with it.
//
// The opena request is like open, but if the file exists,
// append to it instead of truncating it.
//
// Both open and opena cause an error if used in safer mode
// (which is the default).  Use groff’s or troff’s
// -U option to activate unsafe mode.
type Tokenopena struct { /* stream */
}

func ParseTokenopena(args string) (Token, error) {
	return &Tokenopena{ /* stream */ }, nil
}

func (t *Tokenopena) String() string {
	return ".opena" /* stream */
}

var _ Token = (*Tokenopena)(nil)

type Tokenwrite struct { /* stream */
}

func ParseTokenwrite(args string) (Token, error) {
	return &Tokenwrite{ /* stream */ }, nil
}

func (t *Tokenwrite) String() string {
	return ".write" /* stream */
}

var _ Token = (*Tokenwrite)(nil)

// Tokenwritec Write to the file associated with the specified stream.  The
// stream must previously have been the subject of an open request.  The
// remainder of the line is interpreted as the ds request reads its
// second argument: A leading ‘"’ is stripped, and it is read in
// copy-in mode.
//
// The writec request is like write, but only write
// appends a newline to the data.
type Tokenwritec struct { /* stream */
}

func ParseTokenwritec(args string) (Token, error) {
	return &Tokenwritec{ /* stream */ }, nil
}

func (t *Tokenwritec) String() string {
	return ".writec" /* stream */
}

var _ Token = (*Tokenwritec)(nil)

// Tokenwritem Write the contents of the macro or string xx to the file
// associated with the specified stream.
//
// xx is read in copy mode, i.e., already formatted elements are
// ignored.  Consequently, diversions must be unformatted with the
// asciify request before calling writem.  Usually, this
// means a loss of information.
type Tokenwritem struct { /* stream */
}

func ParseTokenwritem(args string) (Token, error) {
	return &Tokenwritem{ /* stream */ }, nil
}

func (t *Tokenwritem) String() string {
	return ".writem" /* stream */
}

var _ Token = (*Tokenwritem)(nil)

// Tokenclose Close the specified stream; the stream is no longer an acceptable
// argument to the write request.
//
// Here a simple macro to write an index entry.
//
// .open idx test.idx
// .
// .de IX
// .  write idx \\n[%] \\$*
// ..
// .
// .IX test entry
// .
// .close idx
type Tokenclose struct { /* stream */
}

func ParseTokenclose(args string) (Token, error) {
	return &Tokenclose{ /* stream */ }, nil
}

func (t *Tokenclose) String() string {
	return ".close" /* stream */
}

var _ Token = (*Tokenclose)(nil)

type Tokendevice struct { /* xxx */
}

func ParseTokendevice(args string) (Token, error) {
	return &Tokendevice{ /* xxx */ }, nil
}

func (t *Tokendevice) String() string {
	return ".device" /* xxx */
}

var _ Token = (*Tokendevice)(nil)

type Tokendevicem struct { /* xx */
}

func ParseTokendevicem(args string) (Token, error) {
	return &Tokendevicem{ /* xx */ }, nil
}

func (t *Tokendevicem) String() string {
	return ".devicem" /* xx */
}

var _ Token = (*Tokendevicem)(nil)

// Tokennm Print line numbers.  start is the line number of the next
// output line.  inc indicates which line numbers are printed.  For
// example, the value 5 means to emit only line numbers that are
// multiples of 5; this defaults to 1.  space is the space
// to be left between the number and the text; this defaults to one digit
// space.  The fourth argument is the indentation of the line numbers,
// defaulting to zero.  Both space and indent are given as
// multiples of digit spaces; they can be negative also.  Without any
// arguments, line numbers are turned off.
//
// gtroff reserves three digit spaces for the line number (which is
// printed right-justified) plus the amount given by indent; the
// output lines are concatenated to the line numbers, separated by
// space, and without reducing the line length.  Depending on
// the value of the horizontal page offset (as set with the po
// request), line numbers that are longer than the reserved space stick
// out to the left, or the whole line is moved to the right.
//
// Parameters corresponding to missing arguments are not changed; any
// non-digit argument (to be more precise, any argument starting with a
// character valid as a delimiter for identifiers) is also treated as
// missing.
//
// If line numbering has been disabled with a call to nm without an
// argument, it can be reactivated with ‘.nm +0’, using the previously
// active line numbering parameters.
//
// The parameters of nm are associated with the current environment
// (see Environments).  The current output line number is available in
// the number register ln.
//
// .po 1m
// .ll 2i
// This test shows how line numbering works with groff.
// .nm 999
// This test shows how line numbering works with groff.
// .br
// .nm xxx 3 2
// .ll -\w'0'u
// This test shows how line numbering works with groff.
// .nn 2
// This test shows how line numbering works with groff.
//
// And here the result:
//
//	This  test shows how
//	line numbering works
//	999 with   groff.   This
//
// 1000 test shows how  line
// 1001 numbering works with
// 1002 groff.
//
//	     This test shows how
//	     line      numbering
//	works  with  groff.
//	This test shows how
//
// 1005  line      numbering
//
//	works with groff.
type Tokennm struct { /* [start */
}

func ParseTokennm(args string) (Token, error) {
	return &Tokennm{ /* [start */ }, nil
}

func (t *Tokennm) String() string {
	return ".nm" /* [start */
}

var _ Token = (*Tokennm)(nil)

// Tokennn Temporarily turn off line numbering.  The argument is the number of
// lines not to be numbered; this defaults to 1.
type Tokennn struct { /* [skip] */
}

func ParseTokennn(args string) (Token, error) {
	return &Tokennn{ /* [skip] */ }, nil
}

func (t *Tokennn) String() string {
	return ".nn" /* [skip] */
}

var _ Token = (*Tokennn)(nil)

// Tokenmc Print a margin character to the right of the
// text.25  The first argument is the glyph to be printed.  The
// second argument is the distance away from the right margin.  If missing,
// the previously set value is used; default is 10pt).  For text
// lines that are too long (that is, longer than the text length plus
// dist), the margin character is directly appended to the lines.
//
// With no arguments the margin character is turned off.  If this occurs
// before a break, no margin character is printed.
//
// For compatibility with AT&T troff, a call to mc
// to set the margin character can’t be undone immediately; at least one
// line gets a margin character.  Thus
//
// .ll 1i
// .mc \[br]
// .mc
// xxx
// .br
// xxx
//
// produces
//
// xxx        |
// xxx
//
// For empty lines and lines produced by the tl request no margin
// character is emitted.
//
// The margin character is associated with the current environment
// (see Environments).
//
// This is quite useful for indicating text that has changed, and, in fact,
// there are programs available for doing this (they are called
// nrchbar and changebar and can be found in any
// ‘comp.sources.unix’ archive).
//
// .ll 3i
// .mc |
// This paragraph is highlighted with a margin
// character.
// .sp
// Note that vertical space isn't marked.
// .br
// \&
// .br
// But we can fake it with `\&'.
//
// Result:
//
// This  paragraph is highlighted |
// with a margin character.       |
//
// Note that vertical space isn't |
// marked.                        |
//
//	|
//
// But we can fake it with `\&'.  |
type Tokenmc struct { /* glyph */
}

func ParseTokenmc(args string) (Token, error) {
	return &Tokenmc{ /* glyph */ }, nil
}

func (t *Tokenmc) String() string {
	return ".mc" /* glyph */
}

var _ Token = (*Tokenmc)(nil)

type Tokenpsbb struct { /* filename */
}

func ParseTokenpsbb(args string) (Token, error) {
	return &Tokenpsbb{ /* filename */ }, nil
}

func (t *Tokenpsbb) String() string {
	return ".psbb" /* filename */
}

var _ Token = (*Tokenpsbb)(nil)

// Tokenlf Change the line number and optionally the file name gtroff shall
// use for error and warning messages.  line is the input line number
// of the next line.
//
// Without argument, the request is ignored.
//
// This is a debugging aid for documents that are split into many files,
// then put together with soelim and other preprocessors.  Usually,
// it isn’t invoked manually.
//
// Note that other troff implementations (including the original
// AT&T version) handle lf differently.  For them,
// line changes the line number of the current line.
type Tokenlf struct { /* line */
}

func ParseTokenlf(args string) (Token, error) {
	return &Tokenlf{ /* line */ }, nil
}

func (t *Tokenlf) String() string {
	return ".lf" /* line */
}

var _ Token = (*Tokenlf)(nil)

type Tokentm struct { /* string */
}

func ParseTokentm(args string) (Token, error) {
	return &Tokentm{ /* string */ }, nil
}

func (t *Tokentm) String() string {
	return ".tm" /* string */
}

var _ Token = (*Tokentm)(nil)

type Tokentm1 struct { /* string */
}

func ParseTokentm1(args string) (Token, error) {
	return &Tokentm1{ /* string */ }, nil
}

func (t *Tokentm1) String() string {
	return ".tm1" /* string */
}

var _ Token = (*Tokentm1)(nil)

// Tokentmc Send string to the standard error output; this is very useful for
// printing debugging messages among other things.
//
// string is read in copy mode.
//
// The tm request ignores leading spaces of string; tm1
// handles its argument similar to the ds request: a leading double
// quote in string is stripped to allow initial blanks.
//
// The tmc request is similar to tm1 but does not append a
// newline (as is done in tm and tm1).
type Tokentmc struct { /* string */
}

func ParseTokentmc(args string) (Token, error) {
	return &Tokentmc{ /* string */ }, nil
}

func (t *Tokentmc) String() string {
	return ".tmc" /* string */
}

var _ Token = (*Tokentmc)(nil)

// Tokenab Similar to the tm request, except that it causes gtroff to
// stop processing.  With no argument it prints ‘User Abort.’ to
// standard error.
type Tokenab struct { /* [string] */
}

func ParseTokenab(args string) (Token, error) {
	return &Tokenab{ /* [string] */ }, nil
}

func (t *Tokenab) String() string {
	return ".ab" /* [string] */
}

var _ Token = (*Tokenab)(nil)

// Tokenex The ex request also causes gtroff to stop processing; see
// also I/O.
type Tokenex struct { /*  */
}

func ParseTokenex(args string) (Token, error) {
	return &Tokenex{ /*  */ }, nil
}

func (t *Tokenex) String() string {
	return ".ex" /*  */
}

var _ Token = (*Tokenex)(nil)

// Tokenpev Print the contents of the current environment and all the currently
// defined environments (both named and numbered) on stderr.
type Tokenpev struct { /*  */
}

func ParseTokenpev(args string) (Token, error) {
	return &Tokenpev{ /*  */ }, nil
}

func (t *Tokenpev) String() string {
	return ".pev" /*  */
}

var _ Token = (*Tokenpev)(nil)

// Tokenpm Print the entire symbol table on stderr.  Names of all defined
// macros, strings, and diversions are print together with their size in
// bytes.  Since gtroff sometimes adds nodes by itself, the returned
// size can be larger than expected.
//
// This request differs from Unix troff: gtroff reports the
// sizes of diversions, ignores an additional argument to print only the
// total of the sizes, and the size isn’t returned in blocks of 128
// characters.
type Tokenpm struct { /*  */
}

func ParseTokenpm(args string) (Token, error) {
	return &Tokenpm{ /*  */ }, nil
}

func (t *Tokenpm) String() string {
	return ".pm" /*  */
}

var _ Token = (*Tokenpm)(nil)

// Tokenpnr Print the names and contents of all currently defined number registers
// on stderr.
type Tokenpnr struct { /*  */
}

func ParseTokenpnr(args string) (Token, error) {
	return &Tokenpnr{ /*  */ }, nil
}

func (t *Tokenpnr) String() string {
	return ".pnr" /*  */
}

var _ Token = (*Tokenpnr)(nil)

// Tokenptr Print the names and positions of all traps (not including input line
// traps and diversion traps) on stderr.  Empty slots in the page
// trap list are printed as well, because they can affect the priority of
// subsequently planted traps.
type Tokenptr struct { /*  */
}

func ParseTokenptr(args string) (Token, error) {
	return &Tokenptr{ /*  */ }, nil
}

func (t *Tokenptr) String() string {
	return ".ptr" /*  */
}

var _ Token = (*Tokenptr)(nil)

// Tokenfl Instruct gtroff to flush its output immediately.  The intent is
// for interactive use, but this behaviour is currently not implemented in
// gtroff.  Contrary to Unix troff, TTY output is sent to a
// device driver also (grotty), making it non-trivial to communicate
// interactively.
//
// This request causes a line break.
type Tokenfl struct { /*  */
}

func ParseTokenfl(args string) (Token, error) {
	return &Tokenfl{ /*  */ }, nil
}

func (t *Tokenfl) String() string {
	return ".fl" /*  */
}

var _ Token = (*Tokenfl)(nil)

// Tokenbacktrace Print a backtrace of the input stack to the standard error stream.
//
// Consider the following in file test:
//
// .de xxx
// .  backtrace
// ..
// .de yyy
// .  xxx
// ..
// .
// .yyy
//
// On execution, gtroff prints the following:
//
// test:2: backtrace: macro `xxx'
// test:5: backtrace: macro `yyy'
// test:8: backtrace: file `test'
//
// The option -b of gtroff internally calls a variant of
// this request on each error and warning.
type Tokenbacktrace struct { /*  */
}

func ParseTokenbacktrace(args string) (Token, error) {
	return &Tokenbacktrace{ /*  */ }, nil
}

func (t *Tokenbacktrace) String() string {
	return ".backtrace" /*  */
}

var _ Token = (*Tokenbacktrace)(nil)

// Tokenwarnscale Set the scaling indicator used in warnings to si.  Valid values
// for si are ‘u’, ‘i’, ‘c’, ‘p’, and ‘P’.
// At startup, it is set to ‘i’.
type Tokenwarnscale struct { /* si */
}

func ParseTokenwarnscale(args string) (Token, error) {
	return &Tokenwarnscale{ /* si */ }, nil
}

func (t *Tokenwarnscale) String() string {
	return ".warnscale" /* si */
}

var _ Token = (*Tokenwarnscale)(nil)

// Tokenspreadwarn Make gtroff emit a warning if the additional space inserted for
// each space between words in an output line is larger or equal to
// limit.  A negative value is changed to zero; no argument toggles
// the warning on and off without changing limit.  The default
// scaling indicator is ‘m’.  At startup, spreadwarn is
// deactivated, and limit is set to 3m.
//
// For example,
//
// .spreadwarn 0.2m
//
// causes a warning if gtroff must add 0.2m or more for each
// interword space in a line.
//
// This request is active only if text is justified to both margins (using
// ‘.ad b’).
type Tokenspreadwarn struct { /* [limit] */
}

func ParseTokenspreadwarn(args string) (Token, error) {
	return &Tokenspreadwarn{ /* [limit] */ }, nil
}

func (t *Tokenspreadwarn) String() string {
	return ".spreadwarn" /* [limit] */
}

var _ Token = (*Tokenspreadwarn)(nil)

type Tokenwarn struct { /* [flags] */
}

func ParseTokenwarn(args string) (Token, error) {
	return &Tokenwarn{ /* [flags] */ }, nil
}

func (t *Tokenwarn) String() string {
	return ".warn" /* [flags] */
}

var _ Token = (*Tokenwarn)(nil)

type Tokencp struct { /* [n] */
}

func ParseTokencp(args string) (Token, error) {
	return &Tokencp{ /* [n] */ }, nil
}

func (t *Tokencp) String() string {
	return ".cp" /* [n] */
}

var _ Token = (*Tokencp)(nil)

type Tokendo struct { /* cmd */
}

func ParseTokendo(args string) (Token, error) {
	return &Tokendo{ /* cmd */ }, nil
}

func (t *Tokendo) String() string {
	return ".do" /* cmd */
}

var _ Token = (*Tokendo)(nil)
