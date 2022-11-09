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
type TokenTH struct {/* title */}

// TokenSS Set up an unnumbered (sub)section heading.  Prints out all the text
// following SS up to the end of the line (or the text in the next
// line if there is no argument to SS) in bold face (or the font
// specified by the string HF), at the same size as the base
// document size.  Additionally, the left margin and the indentation for
// the following text is reset to its default value.
type TokenSS struct {/* [heading] */}

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
type TokenTP struct {/* [nnn] */}

// TokenP These macros are mutual aliases.  Any of them causes a line break at the
// current position, followed by a vertical space downwards by the amount
// specified by the PD macro.  The font size and shape are reset to
// the default value (10pt roman if no -rS option is given
// on the command line).  Finally, the current left margin and the
// indentation is restored.
type TokenP struct {/*  */}

// TokenHP Set up a paragraph with hanging left indentation.  The indentation is
// set to nnn if that argument is supplied (default unit is
// ‘n’), otherwise it is set to the previous indentation value
// specified with TP, IP, or HP (or the default value
// if non of them have been used yet).  Font size and face are reset to
// their default values.
type TokenHP struct {/* [nnn] */}

// TokenSB Set the text on the same line or the text on the next line in bold face
// font, one point size smaller than the default font.
type TokenSB struct {/* [text] */}

// TokenIB Set its arguments alternately in italic and bold face, without a space
// between the arguments.
type TokenIB struct {/* text */}

// TokenRI Set its arguments alternately in roman and italic, without a space
// between the arguments.
type TokenRI struct {/* text */}

// TokenIR Set its arguments alternately in italic and roman, without a space
// between the arguments.
type TokenIR struct {/* text */}

// TokenBR Set its arguments alternately in bold face and roman, without a space
// between the arguments.
type TokenBR struct {/* text */}

// TokenRB Set its arguments alternately in roman and bold face, without a space
// between the arguments.
type TokenRB struct {/* text */}

// TokenDT Set tabs every 0.5 inches.  Since this macro is always executed
// during a call to the TH macro, it makes sense to call it only if
// the tab positions have been changed.
type TokenDT struct {/*  */}

// TokenPD Adjust the empty space before a new paragraph (or section).  The
// optional argument gives the amount of space (default unit is ‘v’);
// without parameter, the value is reset to its default value (1 line
// in nroff mode, 0.4v otherwise).
//
// This affects the macros SH, SS, TP, LP (as
// well as PP and P), IP, and HP.
type TokenPD struct {/* [nnn] */}

// TokenAT Alter the footer for use with AT&T manpages.  This command
// exists only for compatibility; don’t use it.  The first argument
// system can be:
//
//
// 3
// 7th Edition (the default)
//
//
// 4
// System III
//
//
// 5
// System V
//
//
//
// An optional second argument release to AT specifies the
// release number (such as “System V Release 3”).
type TokenAT struct {/* [system */}

// TokenUC Alters the footer for use with BSD manpages.  This command exists only
// for compatibility; don’t use it.  The argument can be:
//
//
// 3
// 3rd Berkeley Distribution (the default)
//
//
// 4
// 4th Berkeley Distribution
//
//
// 5
// 4.2 Berkeley Distribution
//
//
// 6
// 4.3 Berkeley Distribution
//
//
// 7
// 4.4 Berkeley Distribution
type TokenUC struct {/* [version] */}

// TokenCT Print ‘<CTRL/key>’.
type TokenCT struct {/* key */}

// TokenEX Begin a non-filled display using the constant width (Courier) typeface.
// Use the optional indent argument to indent the display.
type TokenEX struct {/* [indent] */}

// TokenEE End a non-filled display started with EX.
type TokenEE struct {/*  */}

// TokenG Set text in Helvetica.  If no text is present on the line where
// the macro is called, then the text of the next line appears in
// Helvetica.
type TokenG struct {/* [text] */}

// TokenGL Set text in Helvetica Oblique.  If no text is present on the line
// where the macro is called, then the text of the next line appears in
// Helvetica Oblique.
type TokenGL struct {/* [text] */}

// TokenHB Set text in Helvetica Bold.  If no text is present on the line
// where the macro is called, then all text up to the next HB
// appears in Helvetica Bold.
type TokenHB struct {/* [text] */}

// TokenTB Identical to HB.
type TokenTB struct {/* [text] */}

// TokenMS Set a manpage reference in Ultrix format.  The title is in Courier
// instead of italic.  Optional punctuation follows the section number
// without an intervening space.
type TokenMS struct {/* title */}

// TokenNT Begin a note.  Print the optional title, or the word “Note”,
// centered on the page.  Text following the macro makes up the body of the
// note, and is indented on both sides.  If the first argument is C,
// the body of the note is printed centered (the second argument replaces
// the word “Note” if specified).
type TokenNT struct {/* [C] */}

// TokenNE End a note begun with NT.
type TokenNE struct {/*  */}

// TokenPN Set the path name in constant width (Courier), followed by optional
// punctuation.
type TokenPN struct {/* path */}

// TokenPn If called with two arguments, identical to PN.  If called with
// three arguments, set the second argument in constant width (Courier),
// bracketed by the first and third arguments in the current font.
type TokenPn struct {/* [punct] */}

// TokenRN Print the string ‘<RETURN>’.
type TokenRN struct {/*  */}

// TokenVS Start printing a change bar in the margin if the number 4 is
// specified.  Otherwise, this macro does nothing.
type TokenVS struct {/* [4] */}

// TokenVE End printing the change bar begun by VS.
type TokenVE struct {/*  */}

// TokenRP Specifies the report format for your document.  The report format
// creates a separate cover page.  The default action (no RP macro)
// is to print a subset of the cover page on page 1 of your document.
//
// If you use the word no as an optional argument, groff
// prints a title page but does not repeat any of the title page
// information (title, author, abstract, etc.) on page 1 of the
// document.
type TokenRP struct {/* [no] */}

// TokenP1 (P-one) Prints the header on page 1.  The default is to suppress
// the header.
type TokenP1 struct {/*  */}

// TokenDA (optional) Prints the current date, or the arguments to the macro if
// any, on the title page (if specified) and in the footers.  This is the
// default for nroff.
type TokenDA struct {/* […] */}

// TokenND (optional) Prints the current date, or the arguments to the macro if
// any, on the title page (if specified) but not in the footers.  This is
// the default for troff.
type TokenND struct {/* […] */}

// TokenTL Specifies the document title.  groff collects text following the
// TL macro into the title, until reaching the author name or
// abstract.
type TokenTL struct {/*  */}

// TokenAU Specifies the author’s name, which appears on the line (or lines)
// immediately following.  You can specify multiple authors as follows:
//
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
type TokenAU struct {/*  */}

// TokenAI Specifies the author’s institution.  You can specify multiple
// institutions in the same way that you specify multiple authors.
type TokenAI struct {/*  */}

// TokenAB Begins the abstract.  The default is to print the word
// ABSTRACT, centered and in italics, above the text of the
// abstract.  The word no as an optional argument suppresses this
// heading.
type TokenAB struct {/* [no] */}

// TokenAE Ends the abstract.
type TokenAE struct {/*  */}

// TokenPP Sets a paragraph with an initial indentation.
//
type TokenPP struct {/*  */}

// TokenLP Sets a paragraph without an initial indentation.
//
type TokenLP struct {/*  */}

// TokenQP Sets a paragraph that is indented at both left and right margins
// by the amount of the register QI.
// The effect is identical to the HTML <BLOCKQUOTE> element.
// The next paragraph or heading returns margins to normal.
// QP inserts vertical space of amount set by register PD
// before the paragraph.
type TokenQP struct {/*  */}

// TokenQS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenQS struct {/*  */}

// TokenQE These macros begin and end a quoted section.
// The QI register
// controls the amount of indentation.
// Both QS and QE insert inter-paragraph vertical space
// set by register PD.
// The text between QS and QE can be structured further
// by use of the macros LP or PP.
type TokenQE struct {/*  */}

// TokenXP Sets a paragraph whose lines are indented, except for the first line.
// This is a Berkeley extension.
type TokenXP struct {/*  */}

// TokenNH Numbered heading.  The argument is either a numeric argument to indicate
// the level of the heading, or the letter S followed by
// numeric arguments to set the heading level explicitly.
//
// If you specify heading levels out of sequence, such as invoking
// ‘.NH 3’ after ‘.NH 1’, groff prints a warning on
// standard error.
//
type TokenNH struct {/* S | curr-level */}

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
type TokenSH struct {/* [match-level] | [heading] */}

// TokenB Sets its first argument in bold type.  If you specify a second
// argument, groff prints it in the previous font after the bold
// text, with no intervening space (this allows you to set punctuation
// after the highlighted text without highlighting the punctuation).
// Similarly, it prints the third argument (if any) in the previous font
// before the first argument.  For example,
//
//
// .B foo ) (
//
//
// prints (foo).
//
// If you give this macro no arguments, groff prints all text
// following in bold until the next highlighting, paragraph, or heading
// macro.
// Set text in bold face.  If no text is present on the line where
// the macro is called, then the text of the next line appears in bold
// face.
type TokenB struct {/* [txt | [text] */}

// TokenR Sets its first argument in roman (or regular) type.  It operates
// similarly to the B macro otherwise.
// Switch to roman font and turn off any underlining in effect.
type TokenR struct {/* [txt | undefined */}

// TokenI Sets its first argument in italic type.  It operates similarly
// to the B macro otherwise.
// Set text in italic.  If no text is present on the line where the
// macro is called, then the text of the next line appears in italic.
type TokenI struct {/* [txt | [text] */}

// TokenBI Sets its first argument in bold italic type.  It operates similarly to
// the B macro otherwise.
// Set its arguments alternately in bold face and italic, without a space
// between the arguments.  Thus,
//
//
// .BI this "word and" that
//
//
// produces “thisword andthat” with “this” and “that” in bold face,
// and “word and” in italics.
type TokenBI struct {/* [txt | text */}

// TokenBX Prints its argument and draws a box around it.  If you want to box a
// string that contains spaces, use a digit-width space (\0).
type TokenBX struct {/* [txt] */}

// TokenUL Prints its first argument with an underline.  If you specify a second
// argument, groff prints it in the previous font after the
// underlined text, with no intervening space.
type TokenUL struct {/* [txt */}

// TokenLG Prints all text following in larger type (two points larger than the
// current point size) until the next font size, highlighting, paragraph,
// or heading macro.  You can specify this macro multiple times to
// enlarge the point size as needed.
type TokenLG struct {/*  */}

// TokenSM Prints all text following in smaller type (two points smaller than the
// current point size) until the next type size, highlighting, paragraph,
// or heading macro.  You can specify this macro multiple times to reduce
// the point size as needed.
// Set the text on the same line or the text on the next line in a font
// that is one point size smaller than the default font.
type TokenSM struct {/* undefined | [text] */}

// TokenNL Prints all text following in the normal point size (that is, the value
// of the PS register).
type TokenNL struct {/*  */}

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
//
// .IP \(bu 4
type TokenIP struct {/* [marker | [designator */}

// TokenRS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
// Move the left margin to the right by the value nnn if specified
// (default unit is ‘n’); otherwise it is set to the previous
// indentation value specified with TP, IP, or HP (or
// to the default value if none of them have been used yet).  The
// indentation value is then set to the default.
//
// Calls to the RS macro can be nested.
type TokenRS struct {/* undefined | [nnn] */}

// TokenRE These macros begin and end an indented section.  The PI register
// controls the amount of indentation, allowing the indented text to line
// up under hanging and indented paragraphs.
// Move the left margin back to level nnn, restoring the previous
// left margin.  If no argument is given, it moves one level back.  The
// first level (i.e., no call to RS yet) has number 1, and each
// call to RS increases the level by 1.
type TokenRE struct {/* undefined | [nnn] */}

// TokenTA Use this macro to reset the tab stops to the default for ms
// (every 5n).  You can redefine the TA macro to create a different
// set of default tab stops.
type TokenTA struct {/*  */}

// TokenLD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenLD struct {/*  */}

// TokenID Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenID struct {/*  */}

// TokenBD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenBD struct {/*  */}

// TokenCD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenCD struct {/*  */}

// TokenRD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenRD struct {/*  */}

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
type TokenDE struct {/*  */}

// TokenDs Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
// Begin a non-filled display.
type TokenDs struct {/*  */}

// TokenDe These two macros were formerly provided as aliases for DS and
// DE, respectively.  They have been removed, and should no longer
// be used.  The original implementations of DS and DE are
// retained, and should be used instead.  X11 documents that actually use
// Ds and De always load a specific macro file from the X11
// distribution (macros.t) that provides proper definitions for the
// two macros.
// End a non-filled display started with Ds.
type TokenDe struct {/*  */}

// TokenKS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenKS struct {/*  */}

// TokenKF Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenKF struct {/*  */}

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
type TokenKE struct {/*  */}

// TokenB1 Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenB1 struct {/*  */}

// TokenB2 Marks the beginning and ending of text that is to have a box drawn
// around it.  The B1 macro begins the box; the B2 macro ends
// it.  Text in the box is automatically placed in a diversion (keep).
type TokenB2 struct {/*  */}

// TokenTS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenTS struct {/* [H] */}

// TokenTE Denotes a table, to be processed by the tbl preprocessor.  The
// optional argument H to TS instructs groff to
// create a running header with the information up to the TH macro.
// groff prints the header at the beginning of the table; if the
// table runs onto another page, groff prints the header on the next
// page as well.
type TokenTE struct {/*  */}

// TokenPS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenPS struct {/*  */}

// TokenPE Denotes a graphic, to be processed by the pic preprocessor.  You
// can create a pic file by hand, using the AT&T
// pic manual available on the Web as a reference, or by using a
// graphics program such as xfig.
type TokenPE struct {/*  */}

// TokenEQ Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenEQ struct {/* [align] */}

// TokenEN Denotes an equation, to be processed by the eqn preprocessor.
// The optional align argument can be C, L,
// or I to center (the default), left-justify, or indent the
// equation.
type TokenEN struct {/*  */}

// Token[ Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type Token[ struct {/*  */}

// Token] Denotes a reference, to be processed by the refer preprocessor.
// The GNU refer(1) man page provides a comprehensive reference to
// the preprocessor and the format of the bibliographic database.
type Token] struct {/*  */}

// TokenFS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenFS struct {/*  */}

// TokenFE Specifies the text of the footnote.  The default action is to create a
// numbered footnote; you can create a symbolic footnote by specifying a
// mark glyph (such as \[dg] for the dagger glyph) in the body
// text and as an argument to the FS macro, followed by the text of
// the footnote and the FE macro.
type TokenFE struct {/*  */}

// TokenOH Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenOH struct {/* 'left'center'right' */}

// TokenEH Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenEH struct {/* 'left'center'right' */}

// TokenOF Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenOF struct {/* 'left'center'right' */}

// TokenEF The OH and EH macros define headers for the odd and even
// pages; the OF and EF macros define footers for the odd and
// even pages.  This is more flexible than defining the individual strings.
//
// You can replace the quote (') marks with any character not
// appearing in the header or footer text.
type TokenEF struct {/* 'left'center'right' */}

// TokenPT Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
// Control the content of the headers.  Normally, the header prints the
// command name and section number on either side, and the optional fifth
// argument to TH in the center.
type TokenPT struct {/*  */}

// TokenHD Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenHD struct {/*  */}

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
type TokenBT struct {/*  */}

// Token1C Single-column mode.
type Token1C struct {/*  */}

// Token2C Two-column mode.
type Token2C struct {/*  */}

// TokenMC Multi-column mode.  If you specify no arguments, it is equivalent to the
// 2C macro.  Otherwise, width is the width of each column and
// gutter is the space between columns.  The MINGW number
// register controls the default gutter width.
type TokenMC struct {/* [width */}

// TokenXS Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenXS struct {/* [page] */}

// TokenXA Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenXA struct {/* [page] */}

// TokenXE These macros define a table of contents or an individual entry in the
// table of contents, depending on their use.  The macros are very simple;
// they cannot indent a heading based on its level.  The easiest way to
// work around this is to add tabs to the table of contents string.  The
// following is an example:
//
//
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
//
//
// You can manually create a table of contents by beginning with the
// XS macro for the first entry, specifying the page number for that
// entry as the argument to XS.  Add subsequent entries using the
// XA macro, specifying the page number for that entry as the
// argument to XA.  The following is an example:
//
//
//
// .XS 1
// Introduction
// .XA 2
// A Brief History of the Universe
// .XA 729
// Details of Galactic Formation
// ...
// .XE
type TokenXE struct {/*  */}

// TokenTC Prints the table of contents on a new page, setting the page number
// to i (Roman lowercase numeral one).  You should usually
// place this macro at the end of the file, since groff is a
// single-pass formatter and can only print what has been collected up to
// the point that the TC macro appears.
//
// The optional argument no suppresses printing the title specified
// by the string register TOC.
type TokenTC struct {/* [no] */}

// TokenPX Prints the table of contents on a new page, using the current page
// numbering sequence.  Use this macro to print a manually generated table
// of contents at the beginning of your document.
//
// The optional argument no suppresses printing the title specified
// by the string register TOC.
type TokenPX struct {/* [no] */}

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
type TokenAM struct {/*  */}

// TokenDS Indented display.  The default behavior of AT&T troff
// -ms was to indent; the groff default prints displays flush left
// with the body text.
//
//
//
//
//
type TokenDS struct {/* I | R | C | B | I | L */}

// TokenCW Print text in constant width (Courier) font.
// Sets its first argument in a constant width face.  It operates
// similarly to the B macro otherwise.
// Print subsequent text using the constant width (Courier) typeface.
type TokenCW struct {/* undefined | [txt | undefined */}

// TokenIX Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenIX struct {/*  */}

func ParseLine(line string) (Token, error) {
	if !strings.HasPrefix(line, ".") {
		return TokenText(line), nil
	}
	line = line[1:]
	ss := strings.SplitN(line, " ", 2)
	line = ss[1]
	switch ss[0] {
	case `\\"`:

	}
	return nil, fmt.Errorf("%w: %s", ErrNoTokenMatch, line)
}
