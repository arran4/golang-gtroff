package groff

import (
    "fmt"
    "strings"
)


// TokenMacroTH
// Set the title of the man page to title and the section to
// section, which must have a value between 1 and 8.  The value
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
// is 1 again (except if the -rC1 option is given on the
// command line) – this feature is intended only for formatting multiple
// man pages; a single man page should contain exactly one TH macro
// at the beginning of the file.
type TokenMacroTH struct {


    title string


    section string


    extra1 string


    extra2 string


    extra3 string

}

func ParseTokenMacroTH(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTH arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            false,
            
            true,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTH arguments: %s, %w", argText, err)
    }
	
        argtitle := matched[0]
    
        argsection := matched[1]
    
        argextra1 := matched[2]
    
        argextra2 := matched[3]
    
        argextra3 := matched[4]
    
	return &TokenMacroTH{
	
        title: argtitle,
    
        section: argsection,
    
        extra1: argextra1,
    
        extra2: argextra2,
    
        extra3: argextra3,
    
	}, nil
}

func (t *TokenMacroTH) String() string {
	var args []string
	
	args = append(args, t.title)
	
	args = append(args, t.section)
	
	args = append(args, t.extra1)
	
	args = append(args, t.extra2)
	
	args = append(args, t.extra3)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TH"
	}
	return ".TH " + strings.Join(args, " ")
}


// TokenMacroSH
// Set up an unnumbered section heading sticking out to the left.  Prints
// out all the text following SH up to the end of the line (or the
// text in the next line if there is no argument to SH) in bold face
// (or the font specified by the string HF), one size larger than
// the base document size.  Additionally, the left margin and the
// indentation for the following text is reset to its default value.
type TokenMacroSH struct {


    heading string

}

func ParseTokenMacroSH(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSH arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSH arguments: %s, %w", argText, err)
    }
	
        argheading := matched[0]
    
	return &TokenMacroSH{
	
        heading: argheading,
    
	}, nil
}

func (t *TokenMacroSH) String() string {
	var args []string
	
	args = append(args, t.heading)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".SH"
	}
	return ".SH " + strings.Join(args, " ")
}


// TokenMacroSS
// Set up an unnumbered (sub)section heading.  Prints out all the text
// following SS up to the end of the line (or the text in the next
// line if there is no argument to SS) in bold face (or the font
// specified by the string HF), at the same size as the base
// document size.  Additionally, the left margin and the indentation for
// the following text is reset to its default value.
type TokenMacroSS struct {


    heading string

}

func ParseTokenMacroSS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSS arguments: %s, %w", argText, err)
    }
	
        argheading := matched[0]
    
	return &TokenMacroSS{
	
        heading: argheading,
    
	}, nil
}

func (t *TokenMacroSS) String() string {
	var args []string
	
	args = append(args, t.heading)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".SS"
	}
	return ".SS " + strings.Join(args, " ")
}


// TokenMacroTP
// Set up an indented paragraph with label.  The indentation is set to
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
type TokenMacroTP struct {


    nnn string

}

func ParseTokenMacroTP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTP arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTP arguments: %s, %w", argText, err)
    }
	
        argnnn := matched[0]
    
	return &TokenMacroTP{
	
        nnn: argnnn,
    
	}, nil
}

func (t *TokenMacroTP) String() string {
	var args []string
	
	args = append(args, t.nnn)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TP"
	}
	return ".TP " + strings.Join(args, " ")
}


// TokenMacroP
// These macros are mutual aliases.  Any of them causes a line break at the
// current position, followed by a vertical space downwards by the amount
// specified by the PD macro.  The font size and shape are reset to
// the default value (10pt roman if no -rS option is given
// on the command line).  Finally, the current left margin and the
// indentation is restored.
type TokenMacroP struct {

}

func ParseTokenMacroP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroP arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroP arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroP{
	
	}, nil
}

func (t *TokenMacroP) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".P"
	}
	return ".P " + strings.Join(args, " ")
}


// TokenMacroIP
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
// 4 en indentation, write
// 
// 
// .IP \(bu 4
type TokenMacroIP struct {


    designator string


    nnn string

}

func ParseTokenMacroIP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIP arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIP arguments: %s, %w", argText, err)
    }
	
        argdesignator := matched[0]
    
        argnnn := matched[1]
    
	return &TokenMacroIP{
	
        designator: argdesignator,
    
        nnn: argnnn,
    
	}, nil
}

func (t *TokenMacroIP) String() string {
	var args []string
	
	args = append(args, t.designator)
	
	args = append(args, t.nnn)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".IP"
	}
	return ".IP " + strings.Join(args, " ")
}


// TokenMacroHP
// Set up a paragraph with hanging left indentation.  The indentation is
// set to nnn if that argument is supplied (default unit is
// ‘n’), otherwise it is set to the previous indentation value
// specified with TP, IP, or HP (or the default value
// if non of them have been used yet).  Font size and face are reset to
// their default values.
type TokenMacroHP struct {


    nnn string

}

func ParseTokenMacroHP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroHP arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroHP arguments: %s, %w", argText, err)
    }
	
        argnnn := matched[0]
    
	return &TokenMacroHP{
	
        nnn: argnnn,
    
	}, nil
}

func (t *TokenMacroHP) String() string {
	var args []string
	
	args = append(args, t.nnn)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".HP"
	}
	return ".HP " + strings.Join(args, " ")
}


// TokenMacroRS
// Move the left margin to the right by the value nnn if specified
// (default unit is ‘n’); otherwise it is set to the previous
// indentation value specified with TP, IP, or HP (or
// to the default value if none of them have been used yet).  The
// indentation value is then set to the default.
// 
// Calls to the RS macro can be nested.
type TokenMacroRS struct {


    nnn string

}

func ParseTokenMacroRS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRS arguments: %s, %w", argText, err)
    }
	
        argnnn := matched[0]
    
	return &TokenMacroRS{
	
        nnn: argnnn,
    
	}, nil
}

func (t *TokenMacroRS) String() string {
	var args []string
	
	args = append(args, t.nnn)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RS"
	}
	return ".RS " + strings.Join(args, " ")
}


// TokenMacroRE
// Move the left margin back to level nnn, restoring the previous
// left margin.  If no argument is given, it moves one level back.  The
// first level (i.e., no call to RS yet) has number 1, and each
// call to RS increases the level by 1.
type TokenMacroRE struct {


    nnn string

}

func ParseTokenMacroRE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRE arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRE arguments: %s, %w", argText, err)
    }
	
        argnnn := matched[0]
    
	return &TokenMacroRE{
	
        nnn: argnnn,
    
	}, nil
}

func (t *TokenMacroRE) String() string {
	var args []string
	
	args = append(args, t.nnn)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RE"
	}
	return ".RE " + strings.Join(args, " ")
}


// TokenMacroSM
// Set the text on the same line or the text on the next line in a font
// that is one point size smaller than the default font.
type TokenMacroSM struct {


    text string

}

func ParseTokenMacroSM(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSM arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSM arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroSM{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroSM) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".SM"
	}
	return ".SM " + strings.Join(args, " ")
}


// TokenMacroSB
// Set the text on the same line or the text on the next line in bold face
// font, one point size smaller than the default font.
type TokenMacroSB struct {


    text string

}

func ParseTokenMacroSB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSB arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroSB{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroSB) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".SB"
	}
	return ".SB " + strings.Join(args, " ")
}


// TokenMacroBI
// Set its arguments alternately in bold face and italic, without a space
// between the arguments.  Thus,
// 
// 
// .BI this "word and" that
// 
// 
// produces “thisword andthat” with “this” and “that” in bold face,
// and “word and” in italics.
type TokenMacroBI struct {


    text string

}

func ParseTokenMacroBI(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBI arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBI arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroBI{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroBI) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".BI"
	}
	return ".BI " + strings.Join(args, " ")
}


// TokenMacroIB
// Set its arguments alternately in italic and bold face, without a space
// between the arguments.
type TokenMacroIB struct {


    text string

}

func ParseTokenMacroIB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIB arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroIB{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroIB) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".IB"
	}
	return ".IB " + strings.Join(args, " ")
}


// TokenMacroRI
// Set its arguments alternately in roman and italic, without a space
// between the arguments.
type TokenMacroRI struct {


    text string

}

func ParseTokenMacroRI(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRI arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRI arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroRI{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroRI) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RI"
	}
	return ".RI " + strings.Join(args, " ")
}


// TokenMacroIR
// Set its arguments alternately in italic and roman, without a space
// between the arguments.
type TokenMacroIR struct {


    text string

}

func ParseTokenMacroIR(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIR arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIR arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroIR{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroIR) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".IR"
	}
	return ".IR " + strings.Join(args, " ")
}


// TokenMacroBR
// Set its arguments alternately in bold face and roman, without a space
// between the arguments.
type TokenMacroBR struct {


    text string

}

func ParseTokenMacroBR(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBR arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBR arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroBR{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroBR) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".BR"
	}
	return ".BR " + strings.Join(args, " ")
}


// TokenMacroRB
// Set its arguments alternately in roman and bold face, without a space
// between the arguments.
type TokenMacroRB struct {


    text string

}

func ParseTokenMacroRB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRB arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroRB{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroRB) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RB"
	}
	return ".RB " + strings.Join(args, " ")
}


// TokenMacroB
// Set text in bold face.  If no text is present on the line where
// the macro is called, then the text of the next line appears in bold
// face.
type TokenMacroB struct {


    text string

}

func ParseTokenMacroB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroB arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroB{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroB) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".B"
	}
	return ".B " + strings.Join(args, " ")
}


// TokenMacroI
// Set text in italic.  If no text is present on the line where the
// macro is called, then the text of the next line appears in italic.
type TokenMacroI struct {


    text string

}

func ParseTokenMacroI(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroI arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroI arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroI{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroI) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".I"
	}
	return ".I " + strings.Join(args, " ")
}


// TokenMacroDT
// Set tabs every 0.5 inches.  Since this macro is always executed
// during a call to the TH macro, it makes sense to call it only if
// the tab positions have been changed.
type TokenMacroDT struct {

}

func ParseTokenMacroDT(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDT arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDT arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroDT{
	
	}, nil
}

func (t *TokenMacroDT) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".DT"
	}
	return ".DT " + strings.Join(args, " ")
}


// TokenMacroPD
// Adjust the empty space before a new paragraph (or section).  The
// optional argument gives the amount of space (default unit is ‘v’);
// without parameter, the value is reset to its default value (1 line
// in nroff mode, 0.4v otherwise).
// 
// This affects the macros SH, SS, TP, LP (as
// well as PP and P), IP, and HP.
type TokenMacroPD struct {


    nnn string

}

func ParseTokenMacroPD(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPD arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPD arguments: %s, %w", argText, err)
    }
	
        argnnn := matched[0]
    
	return &TokenMacroPD{
	
        nnn: argnnn,
    
	}, nil
}

func (t *TokenMacroPD) String() string {
	var args []string
	
	args = append(args, t.nnn)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PD"
	}
	return ".PD " + strings.Join(args, " ")
}


// TokenMacroAT
// Alter the footer for use with AT&T manpages.  This command
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
type TokenMacroAT struct {


    system string


    release string

}

func ParseTokenMacroAT(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAT arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAT arguments: %s, %w", argText, err)
    }
	
        argsystem := matched[0]
    
        argrelease := matched[1]
    
	return &TokenMacroAT{
	
        system: argsystem,
    
        release: argrelease,
    
	}, nil
}

func (t *TokenMacroAT) String() string {
	var args []string
	
	args = append(args, t.system)
	
	args = append(args, t.release)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".AT"
	}
	return ".AT " + strings.Join(args, " ")
}


// TokenMacroUC
// Alters the footer for use with BSD manpages.  This command exists only
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
type TokenMacroUC struct {


    version string

}

func ParseTokenMacroUC(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroUC arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroUC arguments: %s, %w", argText, err)
    }
	
        argversion := matched[0]
    
	return &TokenMacroUC{
	
        version: argversion,
    
	}, nil
}

func (t *TokenMacroUC) String() string {
	var args []string
	
	args = append(args, t.version)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".UC"
	}
	return ".UC " + strings.Join(args, " ")
}


// TokenMacroPT
// Control the content of the headers.  Normally, the header prints the
// command name and section number on either side, and the optional fifth
// argument to TH in the center.
type TokenMacroPT struct {

}

func ParseTokenMacroPT(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPT arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPT arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroPT{
	
	}, nil
}

func (t *TokenMacroPT) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PT"
	}
	return ".PT " + strings.Join(args, " ")
}


// TokenMacroBT
// Control the content of the footers.  Normally, the footer prints the
// page number and the third and fourth arguments to TH.
// 
// Use the FT number register to specify the footer position.  The
// default is -0.5i.
type TokenMacroBT struct {

}

func ParseTokenMacroBT(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBT arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBT arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroBT{
	
	}, nil
}

func (t *TokenMacroBT) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".BT"
	}
	return ".BT " + strings.Join(args, " ")
}


// TokenMacroCT
// Print ‘<CTRL/key>’.
type TokenMacroCT struct {


    key string

}

func ParseTokenMacroCT(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroCT arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroCT arguments: %s, %w", argText, err)
    }
	
        argkey := matched[0]
    
	return &TokenMacroCT{
	
        key: argkey,
    
	}, nil
}

func (t *TokenMacroCT) String() string {
	var args []string
	
	args = append(args, t.key)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".CT"
	}
	return ".CT " + strings.Join(args, " ")
}


// TokenMacroCW
// Print subsequent text using the constant width (Courier) typeface.
type TokenMacroCW struct {

}

func ParseTokenMacroCW(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroCW arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroCW arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroCW{
	
	}, nil
}

func (t *TokenMacroCW) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".CW"
	}
	return ".CW " + strings.Join(args, " ")
}


// TokenMacroDs
// Begin a non-filled display.
type TokenMacroDs struct {

}

func ParseTokenMacroDs(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDs arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDs arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroDs{
	
	}, nil
}

func (t *TokenMacroDs) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".Ds"
	}
	return ".Ds " + strings.Join(args, " ")
}


// TokenMacroDe
// End a non-filled display started with Ds.
type TokenMacroDe struct {

}

func ParseTokenMacroDe(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDe arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDe arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroDe{
	
	}, nil
}

func (t *TokenMacroDe) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".De"
	}
	return ".De " + strings.Join(args, " ")
}


// TokenMacroEX
// Begin a non-filled display using the constant width (Courier) typeface.
// Use the optional indent argument to indent the display.
type TokenMacroEX struct {


    indent string

}

func ParseTokenMacroEX(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEX arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEX arguments: %s, %w", argText, err)
    }
	
        argindent := matched[0]
    
	return &TokenMacroEX{
	
        indent: argindent,
    
	}, nil
}

func (t *TokenMacroEX) String() string {
	var args []string
	
	args = append(args, t.indent)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".EX"
	}
	return ".EX " + strings.Join(args, " ")
}


// TokenMacroEE
// End a non-filled display started with EX.
type TokenMacroEE struct {

}

func ParseTokenMacroEE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroEE{
	
	}, nil
}

func (t *TokenMacroEE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".EE"
	}
	return ".EE " + strings.Join(args, " ")
}


// TokenMacroG
// Set text in Helvetica.  If no text is present on the line where
// the macro is called, then the text of the next line appears in
// Helvetica.
type TokenMacroG struct {


    text string

}

func ParseTokenMacroG(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroG arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroG arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroG{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroG) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".G"
	}
	return ".G " + strings.Join(args, " ")
}


// TokenMacroGL
// Set text in Helvetica Oblique.  If no text is present on the line
// where the macro is called, then the text of the next line appears in
// Helvetica Oblique.
type TokenMacroGL struct {


    text string

}

func ParseTokenMacroGL(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroGL arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroGL arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroGL{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroGL) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".GL"
	}
	return ".GL " + strings.Join(args, " ")
}


// TokenMacroHB
// Set text in Helvetica Bold.  If no text is present on the line
// where the macro is called, then all text up to the next HB
// appears in Helvetica Bold.
type TokenMacroHB struct {


    text string

}

func ParseTokenMacroHB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroHB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroHB arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroHB{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroHB) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".HB"
	}
	return ".HB " + strings.Join(args, " ")
}


// TokenMacroTB
// Identical to HB.
type TokenMacroTB struct {


    text string

}

func ParseTokenMacroTB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTB arguments: %s, %w", argText, err)
    }
	
        argtext := matched[0]
    
	return &TokenMacroTB{
	
        text: argtext,
    
	}, nil
}

func (t *TokenMacroTB) String() string {
	var args []string
	
	args = append(args, t.text)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TB"
	}
	return ".TB " + strings.Join(args, " ")
}


// TokenMacroMS
// Set a manpage reference in Ultrix format.  The title is in Courier
// instead of italic.  Optional punctuation follows the section number
// without an intervening space.
type TokenMacroMS struct {


    title string


    sect string


    punct string

}

func ParseTokenMacroMS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroMS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroMS arguments: %s, %w", argText, err)
    }
	
        argtitle := matched[0]
    
        argsect := matched[1]
    
        argpunct := matched[2]
    
	return &TokenMacroMS{
	
        title: argtitle,
    
        sect: argsect,
    
        punct: argpunct,
    
	}, nil
}

func (t *TokenMacroMS) String() string {
	var args []string
	
	args = append(args, t.title)
	
	args = append(args, t.sect)
	
	args = append(args, t.punct)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".MS"
	}
	return ".MS " + strings.Join(args, " ")
}


// TokenMacroNT
// Begin a note.  Print the optional title, or the word “Note”,
// centered on the page.  Text following the macro makes up the body of the
// note, and is indented on both sides.  If the first argument is C,
// the body of the note is printed centered (the second argument replaces
// the word “Note” if specified).
type TokenMacroNT struct {


    C string


    title string

}

func ParseTokenMacroNT(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNT arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNT arguments: %s, %w", argText, err)
    }
	
        argC := matched[0]
    
        argtitle := matched[1]
    
	return &TokenMacroNT{
	
        C: argC,
    
        title: argtitle,
    
	}, nil
}

func (t *TokenMacroNT) String() string {
	var args []string
	
	args = append(args, t.C)
	
	args = append(args, t.title)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".NT"
	}
	return ".NT " + strings.Join(args, " ")
}


// TokenMacroNE
// End a note begun with NT.
type TokenMacroNE struct {

}

func ParseTokenMacroNE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroNE{
	
	}, nil
}

func (t *TokenMacroNE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".NE"
	}
	return ".NE " + strings.Join(args, " ")
}


// TokenMacroPN
// Set the path name in constant width (Courier), followed by optional
// punctuation.
type TokenMacroPN struct {


    path string


    punct string

}

func ParseTokenMacroPN(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPN arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPN arguments: %s, %w", argText, err)
    }
	
        argpath := matched[0]
    
        argpunct := matched[1]
    
	return &TokenMacroPN{
	
        path: argpath,
    
        punct: argpunct,
    
	}, nil
}

func (t *TokenMacroPN) String() string {
	var args []string
	
	args = append(args, t.path)
	
	args = append(args, t.punct)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PN"
	}
	return ".PN " + strings.Join(args, " ")
}


// TokenMacroPn
// If called with two arguments, identical to PN.  If called with
// three arguments, set the second argument in constant width (Courier),
// bracketed by the first and third arguments in the current font.
type TokenMacroPn struct {


    punct0 string


    path string


    punct1 string

}

func ParseTokenMacroPn(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPn arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            true,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPn arguments: %s, %w", argText, err)
    }
	
        argpunct0 := matched[0]
    
        argpath := matched[1]
    
        argpunct1 := matched[2]
    
	return &TokenMacroPn{
	
        punct0: argpunct0,
    
        path: argpath,
    
        punct1: argpunct1,
    
	}, nil
}

func (t *TokenMacroPn) String() string {
	var args []string
	
	args = append(args, t.punct0)
	
	args = append(args, t.path)
	
	args = append(args, t.punct1)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".Pn"
	}
	return ".Pn " + strings.Join(args, " ")
}


// TokenMacroR
// Switch to roman font and turn off any underlining in effect.
type TokenMacroR struct {

}

func ParseTokenMacroR(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroR arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroR arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroR{
	
	}, nil
}

func (t *TokenMacroR) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".R"
	}
	return ".R " + strings.Join(args, " ")
}


// TokenMacroRN
// Print the string ‘<RETURN>’.
type TokenMacroRN struct {

}

func ParseTokenMacroRN(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRN arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRN arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroRN{
	
	}, nil
}

func (t *TokenMacroRN) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RN"
	}
	return ".RN " + strings.Join(args, " ")
}


// TokenMacroVS
// Start printing a change bar in the margin if the number 4 is
// specified.  Otherwise, this macro does nothing.
type TokenMacroVS struct {


    Four string

}

func ParseTokenMacroVS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroVS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroVS arguments: %s, %w", argText, err)
    }
	
        argFour := matched[0]
    
	return &TokenMacroVS{
	
        Four: argFour,
    
	}, nil
}

func (t *TokenMacroVS) String() string {
	var args []string
	
	args = append(args, t.Four)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".VS"
	}
	return ".VS " + strings.Join(args, " ")
}


// TokenMacroVE
// End printing the change bar begun by VS.
type TokenMacroVE struct {

}

func ParseTokenMacroVE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroVE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroVE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroVE{
	
	}, nil
}

func (t *TokenMacroVE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".VE"
	}
	return ".VE " + strings.Join(args, " ")
}


// TokenMacroRP
// Specifies the report format for your document.  The report format
// creates a separate cover page.  The default action (no RP macro)
// is to print a subset of the cover page on page 1 of your document.
// 
// If you use the word no as an optional argument, groff
// prints a title page but does not repeat any of the title page
// information (title, author, abstract, etc.) on page 1 of the
// document.
type TokenMacroRP struct {


    no string

}

func ParseTokenMacroRP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRP arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRP arguments: %s, %w", argText, err)
    }
	
        argno := matched[0]
    
	return &TokenMacroRP{
	
        no: argno,
    
	}, nil
}

func (t *TokenMacroRP) String() string {
	var args []string
	
	args = append(args, t.no)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RP"
	}
	return ".RP " + strings.Join(args, " ")
}


// TokenMacroP1
// (P-one) Prints the header on page 1.  The default is to suppress
// the header.
type TokenMacroP1 struct {

}

func ParseTokenMacroP1(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroP1 arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroP1 arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroP1{
	
	}, nil
}

func (t *TokenMacroP1) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".P1"
	}
	return ".P1 " + strings.Join(args, " ")
}


// TokenMacroDA
// (optional) Prints the current date, or the arguments to the macro if
// any, on the title page (if specified) and in the footers.  This is the
// default for nroff.
type TokenMacroDA struct {


    Ellipse string

}

func ParseTokenMacroDA(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDA arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDA arguments: %s, %w", argText, err)
    }
	
        argEllipse := matched[0]
    
	return &TokenMacroDA{
	
        Ellipse: argEllipse,
    
	}, nil
}

func (t *TokenMacroDA) String() string {
	var args []string
	
	args = append(args, t.Ellipse)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".DA"
	}
	return ".DA " + strings.Join(args, " ")
}


// TokenMacroND
// (optional) Prints the current date, or the arguments to the macro if
// any, on the title page (if specified) but not in the footers.  This is
// the default for troff.
type TokenMacroND struct {


    Ellipse string

}

func ParseTokenMacroND(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroND arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroND arguments: %s, %w", argText, err)
    }
	
        argEllipse := matched[0]
    
	return &TokenMacroND{
	
        Ellipse: argEllipse,
    
	}, nil
}

func (t *TokenMacroND) String() string {
	var args []string
	
	args = append(args, t.Ellipse)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".ND"
	}
	return ".ND " + strings.Join(args, " ")
}


// TokenMacroTL
// Specifies the document title.  groff collects text following the
// TL macro into the title, until reaching the author name or
// abstract.
type TokenMacroTL struct {

}

func ParseTokenMacroTL(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTL arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTL arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroTL{
	
	}, nil
}

func (t *TokenMacroTL) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TL"
	}
	return ".TL " + strings.Join(args, " ")
}


// TokenMacroAU
// Specifies the author’s name, which appears on the line (or lines)
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
type TokenMacroAU struct {

}

func ParseTokenMacroAU(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAU arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAU arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroAU{
	
	}, nil
}

func (t *TokenMacroAU) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".AU"
	}
	return ".AU " + strings.Join(args, " ")
}


// TokenMacroAI
// Specifies the author’s institution.  You can specify multiple
// institutions in the same way that you specify multiple authors.
type TokenMacroAI struct {

}

func ParseTokenMacroAI(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAI arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAI arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroAI{
	
	}, nil
}

func (t *TokenMacroAI) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".AI"
	}
	return ".AI " + strings.Join(args, " ")
}


// TokenMacroAB
// Begins the abstract.  The default is to print the word
// ABSTRACT, centered and in italics, above the text of the
// abstract.  The word no as an optional argument suppresses this
// heading.
type TokenMacroAB struct {


    no string

}

func ParseTokenMacroAB(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAB arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAB arguments: %s, %w", argText, err)
    }
	
        argno := matched[0]
    
	return &TokenMacroAB{
	
        no: argno,
    
	}, nil
}

func (t *TokenMacroAB) String() string {
	var args []string
	
	args = append(args, t.no)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".AB"
	}
	return ".AB " + strings.Join(args, " ")
}


// TokenMacroAE
// Ends the abstract.
type TokenMacroAE struct {

}

func ParseTokenMacroAE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroAE{
	
	}, nil
}

func (t *TokenMacroAE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".AE"
	}
	return ".AE " + strings.Join(args, " ")
}


// TokenMacroPP
// Sets a paragraph with an initial indentation.
type TokenMacroPP struct {

}

func ParseTokenMacroPP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPP arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPP arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroPP{
	
	}, nil
}

func (t *TokenMacroPP) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PP"
	}
	return ".PP " + strings.Join(args, " ")
}


// TokenMacroLP
// Sets a paragraph without an initial indentation.
type TokenMacroLP struct {

}

func ParseTokenMacroLP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroLP arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroLP arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroLP{
	
	}, nil
}

func (t *TokenMacroLP) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".LP"
	}
	return ".LP " + strings.Join(args, " ")
}


// TokenMacroQP
// Sets a paragraph that is indented at both left and right margins
// by the amount of the register QI.
// The effect is identical to the HTML <BLOCKQUOTE> element.
// The next paragraph or heading returns margins to normal.
// QP inserts vertical space of amount set by register PD
// before the paragraph.
type TokenMacroQP struct {

}

func ParseTokenMacroQP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroQP arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroQP arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroQP{
	
	}, nil
}

func (t *TokenMacroQP) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".QP"
	}
	return ".QP " + strings.Join(args, " ")
}


// TokenMacroQE
// These macros begin and end a quoted section.
// The QI register
// controls the amount of indentation.
// Both QS and QE insert inter-paragraph vertical space
// set by register PD.
// The text between QS and QE can be structured further
// by use of the macros LP or PP.
type TokenMacroQE struct {

}

func ParseTokenMacroQE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroQE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroQE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroQE{
	
	}, nil
}

func (t *TokenMacroQE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".QE"
	}
	return ".QE " + strings.Join(args, " ")
}


// TokenMacroXP
// Sets a paragraph whose lines are indented, except for the first line.
// This is a Berkeley extension.
type TokenMacroXP struct {

}

func ParseTokenMacroXP(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXP arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXP arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroXP{
	
	}, nil
}

func (t *TokenMacroXP) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".XP"
	}
	return ".XP " + strings.Join(args, " ")
}


// TokenMacroNH
// Numbered heading.  The argument is either a numeric argument to indicate
// the level of the heading, or the letter S followed by
// numeric arguments to set the heading level explicitly.
// 
// If you specify heading levels out of sequence, such as invoking
// ‘.NH 3’ after ‘.NH 1’, groff prints a warning on
// standard error.
type TokenMacroNH struct {


    S string


    level0 string


    Ellipse string

}

func ParseTokenMacroNH(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNH arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            
            false,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNH arguments: %s, %w", argText, err)
    }
	
        argS := matched[0]
    
        arglevel0 := matched[1]
    
        argEllipse := matched[2]
    
	return &TokenMacroNH{
	
        S: argS,
    
        level0: arglevel0,
    
        Ellipse: argEllipse,
    
	}, nil
}

func (t *TokenMacroNH) String() string {
	var args []string
	
	args = append(args, t.S)
	
	args = append(args, t.level0)
	
	args = append(args, t.Ellipse)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".NH"
	}
	return ".NH " + strings.Join(args, " ")
}


// TokenMacroBX
// Prints its argument and draws a box around it.  If you want to box a
// string that contains spaces, use a digit-width space (\0).
type TokenMacroBX struct {


    txt string

}

func ParseTokenMacroBX(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBX arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBX arguments: %s, %w", argText, err)
    }
	
        argtxt := matched[0]
    
	return &TokenMacroBX{
	
        txt: argtxt,
    
	}, nil
}

func (t *TokenMacroBX) String() string {
	var args []string
	
	args = append(args, t.txt)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".BX"
	}
	return ".BX " + strings.Join(args, " ")
}


// TokenMacroUL
// Prints its first argument with an underline.  If you specify a second
// argument, groff prints it in the previous font after the
// underlined text, with no intervening space.
type TokenMacroUL struct {


    txt string


    post string

}

func ParseTokenMacroUL(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroUL arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroUL arguments: %s, %w", argText, err)
    }
	
        argtxt := matched[0]
    
        argpost := matched[1]
    
	return &TokenMacroUL{
	
        txt: argtxt,
    
        post: argpost,
    
	}, nil
}

func (t *TokenMacroUL) String() string {
	var args []string
	
	args = append(args, t.txt)
	
	args = append(args, t.post)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".UL"
	}
	return ".UL " + strings.Join(args, " ")
}


// TokenMacroLG
// Prints all text following in larger type (two points larger than the
// current point size) until the next font size, highlighting, paragraph,
// or heading macro.  You can specify this macro multiple times to
// enlarge the point size as needed.
type TokenMacroLG struct {

}

func ParseTokenMacroLG(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroLG arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroLG arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroLG{
	
	}, nil
}

func (t *TokenMacroLG) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".LG"
	}
	return ".LG " + strings.Join(args, " ")
}


// TokenMacroNL
// Prints all text following in the normal point size (that is, the value
// of the PS register).
type TokenMacroNL struct {

}

func ParseTokenMacroNL(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNL arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNL arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroNL{
	
	}, nil
}

func (t *TokenMacroNL) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".NL"
	}
	return ".NL " + strings.Join(args, " ")
}


// TokenMacroTA
// Use this macro to reset the tab stops to the default for ms
// (every 5n).  You can redefine the TA macro to create a different
// set of default tab stops.
type TokenMacroTA struct {

}

func ParseTokenMacroTA(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTA arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTA arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroTA{
	
	}, nil
}

func (t *TokenMacroTA) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TA"
	}
	return ".TA " + strings.Join(args, " ")
}


// TokenMacroDE
// Left-justified display.  The ‘.DS L’ call generates a page break,
// if necessary, to keep the entire display on one page.  The LD
// macro allows the display to break across pages.  The DE macro
// ends the display.
type TokenMacroDE struct {

}

func ParseTokenMacroDE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroDE{
	
	}, nil
}

func (t *TokenMacroDE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".DE"
	}
	return ".DE " + strings.Join(args, " ")
}


// TokenMacroKE
// The KS macro begins a block of text to be kept on a single page,
// and the KE macro ends the block.
type TokenMacroKE struct {

}

func ParseTokenMacroKE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroKE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroKE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroKE{
	
	}, nil
}

func (t *TokenMacroKE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".KE"
	}
	return ".KE " + strings.Join(args, " ")
}


// TokenMacroB2
// Marks the beginning and ending of text that is to have a box drawn
// around it.  The B1 macro begins the box; the B2 macro ends
// it.  Text in the box is automatically placed in a diversion (keep).
type TokenMacroB2 struct {

}

func ParseTokenMacroB2(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroB2 arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroB2 arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroB2{
	
	}, nil
}

func (t *TokenMacroB2) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".B2"
	}
	return ".B2 " + strings.Join(args, " ")
}


// TokenMacroTE
// Denotes a table, to be processed by the tbl preprocessor.  The
// optional argument H to TS instructs groff to
// create a running header with the information up to the TH macro.
// groff prints the header at the beginning of the table; if the
// table runs onto another page, groff prints the header on the next
// page as well.
type TokenMacroTE struct {

}

func ParseTokenMacroTE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroTE{
	
	}, nil
}

func (t *TokenMacroTE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TE"
	}
	return ".TE " + strings.Join(args, " ")
}


// TokenMacroPE
// Denotes a graphic, to be processed by the pic preprocessor.  You
// can create a pic file by hand, using the AT&T
// pic manual available on the Web as a reference, or by using a
// graphics program such as xfig.
type TokenMacroPE struct {

}

func ParseTokenMacroPE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroPE{
	
	}, nil
}

func (t *TokenMacroPE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PE"
	}
	return ".PE " + strings.Join(args, " ")
}


// TokenMacroEN
// Denotes an equation, to be processed by the eqn preprocessor.
// The optional align argument can be C, L,
// or I to center (the default), left-justify, or indent the
// equation.
type TokenMacroEN struct {

}

func ParseTokenMacroEN(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEN arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEN arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroEN{
	
	}, nil
}

func (t *TokenMacroEN) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".EN"
	}
	return ".EN " + strings.Join(args, " ")
}


// TokenMacroSquareBracketClose
// Denotes a reference, to be processed by the refer preprocessor.
// The GNU refer(1) man page provides a comprehensive reference to
// the preprocessor and the format of the bibliographic database.
type TokenMacroSquareBracketClose struct {

}

func ParseTokenMacroSquareBracketClose(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSquareBracketClose arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSquareBracketClose arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroSquareBracketClose{
	
	}, nil
}

func (t *TokenMacroSquareBracketClose) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".]"
	}
	return ".] " + strings.Join(args, " ")
}


// TokenMacroFE
// Specifies the text of the footnote.  The default action is to create a
// numbered footnote; you can create a symbolic footnote by specifying a
// mark glyph (such as \[dg] for the dagger glyph) in the body
// text and as an argument to the FS macro, followed by the text of
// the footnote and the FE macro.
type TokenMacroFE struct {

}

func ParseTokenMacroFE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroFE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroFE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroFE{
	
	}, nil
}

func (t *TokenMacroFE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".FE"
	}
	return ".FE " + strings.Join(args, " ")
}


// TokenMacroEF
// The OH and EH macros define headers for the odd and even
// pages; the OF and EF macros define footers for the odd and
// even pages.  This is more flexible than defining the individual strings.
// 
// You can replace the quote (') marks with any character not
// appearing in the header or footer text.
type TokenMacroEF struct {


    QuoteleftQuotecenterQuoterightQuote string

}

func ParseTokenMacroEF(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEF arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEF arguments: %s, %w", argText, err)
    }
	
        argQuoteleftQuotecenterQuoterightQuote := matched[0]
    
	return &TokenMacroEF{
	
        QuoteleftQuotecenterQuoterightQuote: argQuoteleftQuotecenterQuoterightQuote,
    
	}, nil
}

func (t *TokenMacroEF) String() string {
	var args []string
	
	args = append(args, t.QuoteleftQuotecenterQuoterightQuote)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".EF"
	}
	return ".EF " + strings.Join(args, " ")
}


// TokenMacro1C
// Single-column mode.
type TokenMacro1C struct {

}

func ParseTokenMacro1C(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacro1C arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacro1C arguments: %s, %w", argText, err)
    }
	
	return &TokenMacro1C{
	
	}, nil
}

func (t *TokenMacro1C) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".1C"
	}
	return ".1C " + strings.Join(args, " ")
}


// TokenMacro2C
// Two-column mode.
type TokenMacro2C struct {

}

func ParseTokenMacro2C(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacro2C arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacro2C arguments: %s, %w", argText, err)
    }
	
	return &TokenMacro2C{
	
	}, nil
}

func (t *TokenMacro2C) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".2C"
	}
	return ".2C " + strings.Join(args, " ")
}


// TokenMacroMC
// Multi-column mode.  If you specify no arguments, it is equivalent to the
// 2C macro.  Otherwise, width is the width of each column and
// gutter is the space between columns.  The MINGW number
// register controls the default gutter width.
type TokenMacroMC struct {


    width string


    gutter string

}

func ParseTokenMacroMC(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroMC arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroMC arguments: %s, %w", argText, err)
    }
	
        argwidth := matched[0]
    
        arggutter := matched[1]
    
	return &TokenMacroMC{
	
        width: argwidth,
    
        gutter: arggutter,
    
	}, nil
}

func (t *TokenMacroMC) String() string {
	var args []string
	
	args = append(args, t.width)
	
	args = append(args, t.gutter)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".MC"
	}
	return ".MC " + strings.Join(args, " ")
}


// TokenMacroXE
// These macros define a table of contents or an individual entry in the
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
type TokenMacroXE struct {

}

func ParseTokenMacroXE(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXE arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXE arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroXE{
	
	}, nil
}

func (t *TokenMacroXE) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".XE"
	}
	return ".XE " + strings.Join(args, " ")
}


// TokenMacroTC
// Prints the table of contents on a new page, setting the page number
// to i (Roman lowercase numeral one).  You should usually
// place this macro at the end of the file, since groff is a
// single-pass formatter and can only print what has been collected up to
// the point that the TC macro appears.
// 
// The optional argument no suppresses printing the title specified
// by the string register TOC.
type TokenMacroTC struct {


    no string

}

func ParseTokenMacroTC(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTC arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTC arguments: %s, %w", argText, err)
    }
	
        argno := matched[0]
    
	return &TokenMacroTC{
	
        no: argno,
    
	}, nil
}

func (t *TokenMacroTC) String() string {
	var args []string
	
	args = append(args, t.no)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TC"
	}
	return ".TC " + strings.Join(args, " ")
}


// TokenMacroPX
// Prints the table of contents on a new page, using the current page
// numbering sequence.  Use this macro to print a manually generated table
// of contents at the beginning of your document.
// 
// The optional argument no suppresses printing the title specified
// by the string register TOC.
type TokenMacroPX struct {


    no string

}

func ParseTokenMacroPX(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPX arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPX arguments: %s, %w", argText, err)
    }
	
        argno := matched[0]
    
	return &TokenMacroPX{
	
        no: argno,
    
	}, nil
}

func (t *TokenMacroPX) String() string {
	var args []string
	
	args = append(args, t.no)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PX"
	}
	return ".PX " + strings.Join(args, " ")
}


// TokenMacroAM
// Specify this macro at the beginning of your document to enable extended
// accent marks and special characters.  This is a Berkeley extension.
// 
// To use the accent marks, place them after the character being
// accented.
// 
// Note that groff’s native support for accents is superior to the
// following definitions.
type TokenMacroAM struct {

}

func ParseTokenMacroAM(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAM arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAM arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroAM{
	
	}, nil
}

func (t *TokenMacroAM) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".AM"
	}
	return ".AM " + strings.Join(args, " ")
}


// TokenMacroDS
// Indented display.  The default behavior of AT&T troff
// -ms was to indent; the groff default prints displays flush left
// with the body text.
type TokenMacroDS struct {


    I string

}

func ParseTokenMacroDS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDS arguments: %s, %w", argText, err)
    }
	
        argI := matched[0]
    
	return &TokenMacroDS{
	
        I: argI,
    
	}, nil
}

func (t *TokenMacroDS) String() string {
	var args []string
	
	args = append(args, t.I)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".DS"
	}
	return ".DS " + strings.Join(args, " ")
}


// TokenMacroIX
// Indexing term (printed on standard error).  You can write a script to
// capture and process an index generated in this manner.
type TokenMacroIX struct {

}

func ParseTokenMacroIX(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIX arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIX arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroIX{
	
	}, nil
}

func (t *TokenMacroIX) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".IX"
	}
	return ".IX " + strings.Join(args, " ")
}


// TokenMacroXA
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroXA struct {


    page string

}

func ParseTokenMacroXA(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXA arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXA arguments: %s, %w", argText, err)
    }
	
        argpage := matched[0]
    
	return &TokenMacroXA{
	
        page: argpage,
    
	}, nil
}

func (t *TokenMacroXA) String() string {
	var args []string
	
	args = append(args, t.page)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".XA"
	}
	return ".XA " + strings.Join(args, " ")
}


// TokenMacroXS
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroXS struct {


    page string

}

func ParseTokenMacroXS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXS arguments: %s, %w", argText, err)
    }
	
        argpage := matched[0]
    
	return &TokenMacroXS{
	
        page: argpage,
    
	}, nil
}

func (t *TokenMacroXS) String() string {
	var args []string
	
	args = append(args, t.page)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".XS"
	}
	return ".XS " + strings.Join(args, " ")
}


// TokenMacroHD
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroHD struct {

}

func ParseTokenMacroHD(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroHD arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroHD arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroHD{
	
	}, nil
}

func (t *TokenMacroHD) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".HD"
	}
	return ".HD " + strings.Join(args, " ")
}


// TokenMacroOF
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroOF struct {


    QuoteleftQuotecenterQuoterightQuote string

}

func ParseTokenMacroOF(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroOF arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroOF arguments: %s, %w", argText, err)
    }
	
        argQuoteleftQuotecenterQuoterightQuote := matched[0]
    
	return &TokenMacroOF{
	
        QuoteleftQuotecenterQuoterightQuote: argQuoteleftQuotecenterQuoterightQuote,
    
	}, nil
}

func (t *TokenMacroOF) String() string {
	var args []string
	
	args = append(args, t.QuoteleftQuotecenterQuoterightQuote)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".OF"
	}
	return ".OF " + strings.Join(args, " ")
}


// TokenMacroEH
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroEH struct {


    QuoteleftQuotecenterQuoterightQuote string

}

func ParseTokenMacroEH(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEH arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEH arguments: %s, %w", argText, err)
    }
	
        argQuoteleftQuotecenterQuoterightQuote := matched[0]
    
	return &TokenMacroEH{
	
        QuoteleftQuotecenterQuoterightQuote: argQuoteleftQuotecenterQuoterightQuote,
    
	}, nil
}

func (t *TokenMacroEH) String() string {
	var args []string
	
	args = append(args, t.QuoteleftQuotecenterQuoterightQuote)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".EH"
	}
	return ".EH " + strings.Join(args, " ")
}


// TokenMacroOH
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroOH struct {


    QuoteleftQuotecenterQuoterightQuote string

}

func ParseTokenMacroOH(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroOH arguments: %s, %w", argText, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroOH arguments: %s, %w", argText, err)
    }
	
        argQuoteleftQuotecenterQuoterightQuote := matched[0]
    
	return &TokenMacroOH{
	
        QuoteleftQuotecenterQuoterightQuote: argQuoteleftQuotecenterQuoterightQuote,
    
	}, nil
}

func (t *TokenMacroOH) String() string {
	var args []string
	
	args = append(args, t.QuoteleftQuotecenterQuoterightQuote)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".OH"
	}
	return ".OH " + strings.Join(args, " ")
}


// TokenMacroFS
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroFS struct {

}

func ParseTokenMacroFS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroFS arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroFS arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroFS{
	
	}, nil
}

func (t *TokenMacroFS) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".FS"
	}
	return ".FS " + strings.Join(args, " ")
}


// TokenMacroSquareBracketOpen
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroSquareBracketOpen struct {

}

func ParseTokenMacroSquareBracketOpen(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSquareBracketOpen arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSquareBracketOpen arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroSquareBracketOpen{
	
	}, nil
}

func (t *TokenMacroSquareBracketOpen) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".["
	}
	return ".[ " + strings.Join(args, " ")
}


// TokenMacroEQ
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroEQ struct {


    align string

}

func ParseTokenMacroEQ(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEQ arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEQ arguments: %s, %w", argText, err)
    }
	
        argalign := matched[0]
    
	return &TokenMacroEQ{
	
        align: argalign,
    
	}, nil
}

func (t *TokenMacroEQ) String() string {
	var args []string
	
	args = append(args, t.align)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".EQ"
	}
	return ".EQ " + strings.Join(args, " ")
}


// TokenMacroPS
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroPS struct {

}

func ParseTokenMacroPS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPS arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPS arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroPS{
	
	}, nil
}

func (t *TokenMacroPS) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".PS"
	}
	return ".PS " + strings.Join(args, " ")
}


// TokenMacroTS
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroTS struct {


    H string

}

func ParseTokenMacroTS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTS arguments: %s, %w", argText, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTS arguments: %s, %w", argText, err)
    }
	
        argH := matched[0]
    
	return &TokenMacroTS{
	
        H: argH,
    
	}, nil
}

func (t *TokenMacroTS) String() string {
	var args []string
	
	args = append(args, t.H)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TS"
	}
	return ".TS " + strings.Join(args, " ")
}


// TokenMacroB1
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroB1 struct {

}

func ParseTokenMacroB1(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroB1 arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroB1 arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroB1{
	
	}, nil
}

func (t *TokenMacroB1) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".B1"
	}
	return ".B1 " + strings.Join(args, " ")
}


// TokenMacroKF
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroKF struct {

}

func ParseTokenMacroKF(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroKF arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroKF arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroKF{
	
	}, nil
}

func (t *TokenMacroKF) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".KF"
	}
	return ".KF " + strings.Join(args, " ")
}


// TokenMacroKS
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroKS struct {

}

func ParseTokenMacroKS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroKS arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroKS arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroKS{
	
	}, nil
}

func (t *TokenMacroKS) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".KS"
	}
	return ".KS " + strings.Join(args, " ")
}


// TokenMacroRD
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroRD struct {

}

func ParseTokenMacroRD(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRD arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRD arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroRD{
	
	}, nil
}

func (t *TokenMacroRD) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".RD"
	}
	return ".RD " + strings.Join(args, " ")
}


// TokenMacroCD
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroCD struct {

}

func ParseTokenMacroCD(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroCD arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroCD arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroCD{
	
	}, nil
}

func (t *TokenMacroCD) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".CD"
	}
	return ".CD " + strings.Join(args, " ")
}


// TokenMacroBD
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroBD struct {

}

func ParseTokenMacroBD(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBD arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBD arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroBD{
	
	}, nil
}

func (t *TokenMacroBD) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".BD"
	}
	return ".BD " + strings.Join(args, " ")
}


// TokenMacroID
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroID struct {

}

func ParseTokenMacroID(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroID arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroID arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroID{
	
	}, nil
}

func (t *TokenMacroID) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".ID"
	}
	return ".ID " + strings.Join(args, " ")
}


// TokenMacroLD
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroLD struct {

}

func ParseTokenMacroLD(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroLD arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroLD arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroLD{
	
	}, nil
}

func (t *TokenMacroLD) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".LD"
	}
	return ".LD " + strings.Join(args, " ")
}


// TokenMacroQS
// The registers ps4html and www-image-template are defined
// by the pre-grohtml preprocessor.  pre-grohtml reads in the
// troff input, marks up the inline equations and passes the result
// firstly to
// 
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// 
// and secondly to
// 
// 
// troff -Thtml
// 
// 
// or
// 
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for
// -Thtml; if -Txhtml is used, all equations are passed
// to geqn to produce MathML, and the register
// ps4html enables the macro sets to ignore floating keeps, footers,
// and headings.
// 
// The register www-image-template is set to the user specified
// template name or the default name.
type TokenMacroQS struct {

}

func ParseTokenMacroQS(argText string) (Token, error) {
    args, _, err := ArgumentParser(argText)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroQS arguments: %s, %w", argText, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroQS arguments: %s, %w", argText, err)
    }
	
	return &TokenMacroQS{
	
	}, nil
}

func (t *TokenMacroQS) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".QS"
	}
	return ".QS " + strings.Join(args, " ")
}

