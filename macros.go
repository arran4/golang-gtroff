package groff

import (
    "fmt"
    "strings"
)


// TokenMacroTH
// Determine the contents of the page header and footer. The subject of the
// man page is topic and the section of the manual to which it belongs is
// section. See man(1) or intro(1) for the manual sectioning applicable to
// your system. topic and section are positioned together at the left and
// right in the header (with section in parentheses immediately appended to
// topic). footer-middle is centered in the footer. The arrangement of the
// rest of the footer depends on whether double-sided layout is enabled
// with the option -rD1. When disabled (the default), footer-inside is
// positioned at the bottom left. Otherwise, footer-inside appears at the
// bottom left on recto (odd-numbered) pages, and at the bottom right on
// verso (even-numbered) pages. The outside footer is the page number,
// except in the continuous-rendering mode enabled by the option -rcR=1, in
// which case it is the topic and section, as in the header. header-middle
// is centered in the header. If section is an integer between 1 and 9
// (inclusive), there is no need to specify header-middle; an.tmac will
// supply text for it. The macro package may also abbreviate topic and
// footer-inside with ellipses if they would overrun the space available in
// the header and footer, respectively. For HTML output, headers and
// footers are suppressed.
// 
// Additionally, this macro breaks the page, resetting the number to 1
// (unless the -rC1 option is given). This feature is intended only for
// formatting multiple man documents in sequence.
// 
// A valid man document calls .TH once, early in the file, prior to any
// other macro calls.
type TokenMacroTH struct {


    Title string


    Section string


    Extra1 string


    Extra2 string


    Extra3 string

}

func ParseTokenMacroTH(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTH arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            false,
            
            true,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTH arguments: %s, %w", lineArgs, err)
    }
	
        argTitle := matched[0]
    
        argSection := matched[1]
    
        argExtra1 := matched[2]
    
        argExtra2 := matched[3]
    
        argExtra3 := matched[4]
    
	return &TokenMacroTH{
	
        Title: argTitle,
    
        Section: argSection,
    
        Extra1: argExtra1,
    
        Extra2: argExtra2,
    
        Extra3: argExtra3,
    
	}, nil
}

func (t *TokenMacroTH) String() string {
	var args []string
	
	args = append(args, t.Title)
	
	args = append(args, t.Section)
	
	args = append(args, t.Extra1)
	
	args = append(args, t.Extra2)
	
	args = append(args, t.Extra3)
	
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
// Set heading-text as a section heading. If no argument is given, a
// one-line input trap is planted; text on the next line becomes
// heading-text. The left margin is reset to zero to set the heading text
// in bold (or the font specified by the string HF), and, on typesetting
// devices, slightly larger than the base type size. If the heading font
// *[HF] is bold, use of an italic style in heading-text is mapped to the
// bold-italic style if available in the font family. The inset level is
// reset to 1, setting the left margin to the value of the IN register.
// Text after heading-text is set as an ordinary paragraph (.P).
type TokenMacroSH struct {


    Heading string

}

func ParseTokenMacroSH(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSH arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSH arguments: %s, %w", lineArgs, err)
    }
	
        argHeading := matched[0]
    
	return &TokenMacroSH{
	
        Heading: argHeading,
    
	}, nil
}

func (t *TokenMacroSH) String() string {
	var args []string
	
	args = append(args, t.Heading)
	
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
// Set subheading-text as a subsection heading indented between a section
// heading and an ordinary paragraph (.P). If no argument is given, a
// one-line input trap is planted; text on the next line becomes
// subheading-text. The left margin is reset to the value of the SN
// register to set the heading text in bold (or the font specified by the
// string HF). If the heading font *[HF] is bold, use of an italic style in
// subheading-text is mapped to the bold-italic style if available in the
// font family. The inset level is reset to 1, setting the left margin to
// the value of the IN register. Text after subheading-text is set as an
// ordinary paragraph (.P).
type TokenMacroSS struct {


    Heading string

}

func ParseTokenMacroSS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSS arguments: %s, %w", lineArgs, err)
    }
	
        argHeading := matched[0]
    
	return &TokenMacroSS{
	
        Heading: argHeading,
    
	}, nil
}

func (t *TokenMacroSS) String() string {
	var args []string
	
	args = append(args, t.Heading)
	
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
// Set a paragraph with a leading tag, and the remainder of the paragraph
// indented. A one-line input trap is planted; text on the next line, which
// can be formatted with a macro, becomes the tag, which is placed at the
// current left margin. The tag can be extended with the \c escape
// sequence. Subsequent text is indented by indentation, if specified, and
// by the amount of the IN register otherwise. If the tag is not as wide as
// the indentation, the paragraph starts on the same line as the tag, at
// the applicable indentation, and continues on the following lines.
// Otherwise, the descriptive part of the paragraph begins on the line
// following the tag.
type TokenMacroTP struct {


    Nnn string

}

func ParseTokenMacroTP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTP arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTP arguments: %s, %w", lineArgs, err)
    }
	
        argNnn := matched[0]
    
	return &TokenMacroTP{
	
        Nnn: argNnn,
    
	}, nil
}

func (t *TokenMacroTP) String() string {
	var args []string
	
	args = append(args, t.Nnn)
	
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
// Begin a new paragraph; these macros are synonymous. The indentation is
// reset to the default value; the left margin, as affected by .RS and .RE,
// is not.
type TokenMacroP struct {

}

func ParseTokenMacroP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroP arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroP arguments: %s, %w", lineArgs, err)
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
// Set an indented paragraph with an optional tag. The tag and indentation
// arguments, if present, are handled as with .TP, with the exception that
// the tag argument to .IP cannot include a macro call.
type TokenMacroIP struct {


    Designator string


    Nnn string

}

func ParseTokenMacroIP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIP arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIP arguments: %s, %w", lineArgs, err)
    }
	
        argDesignator := matched[0]
    
        argNnn := matched[1]
    
	return &TokenMacroIP{
	
        Designator: argDesignator,
    
        Nnn: argNnn,
    
	}, nil
}

func (t *TokenMacroIP) String() string {
	var args []string
	
	args = append(args, t.Designator)
	
	args = append(args, t.Nnn)
	
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
// Set up a paragraph with a hanging left indentation. The indentation
// argument, if present, is handled as with .TP.
// 
// Use of this presentation-oriented macro is deprecated. A hanging
// indentation cannot be expressed naturally under HTML, and non-roff-based
// man page interpreters may treat .HP as an ordinary paragraph. Thus,
// information or distinctions you mean to express with indentation may be
// lost.
type TokenMacroHP struct {


    Nnn string

}

func ParseTokenMacroHP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroHP arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroHP arguments: %s, %w", lineArgs, err)
    }
	
        argNnn := matched[0]
    
	return &TokenMacroHP{
	
        Nnn: argNnn,
    
	}, nil
}

func (t *TokenMacroHP) String() string {
	var args []string
	
	args = append(args, t.Nnn)
	
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
// Start a new relative inset level. The position of the left margin is
// saved, then moved right by inset-amount, if specified, and by the amount
// of the IN register otherwise. Calls to .RS can be nested; each
// increments by 1 the inset level used by .RE. The level prior to any .RS
// calls is 1.
type TokenMacroRS struct {


    Nnn string

}

func ParseTokenMacroRS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRS arguments: %s, %w", lineArgs, err)
    }
	
        argNnn := matched[0]
    
	return &TokenMacroRS{
	
        Nnn: argNnn,
    
	}, nil
}

func (t *TokenMacroRS) String() string {
	var args []string
	
	args = append(args, t.Nnn)
	
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
// End a relative inset. The left margin corresponding to inset level level
// is restored. If no argument is given, the inset level is reduced by 1.
type TokenMacroRE struct {


    Nnn string

}

func ParseTokenMacroRE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRE arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRE arguments: %s, %w", lineArgs, err)
    }
	
        argNnn := matched[0]
    
	return &TokenMacroRE{
	
        Nnn: argNnn,
    
	}, nil
}

func (t *TokenMacroRE) String() string {
	var args []string
	
	args = append(args, t.Nnn)
	
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
// Set text one point smaller than the default type size on typesetting
// devices. If no argument is given, a one-line input trap is planted; text
// on the next line, which can be further formatted with a macro, is set
// smaller.
type TokenMacroSM struct {


    Text string

}

func ParseTokenMacroSM(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSM arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSM arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroSM{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroSM) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set text in bold and (on typesetting devices) one point smaller than the
// default type size. If no argument is given, a one-line input trap is
// planted; text on the next line, which can be further formatted with a
// macro, is set smaller and in bold. This macro is an extension introduced
// in SunOS 4.0.
type TokenMacroSB struct {


    Text string

}

func ParseTokenMacroSB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSB arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroSB{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroSB) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set each argument in bold and italics, alternately.
type TokenMacroBI struct {


    Text string

}

func ParseTokenMacroBI(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBI arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBI arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroBI{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroBI) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set each argument in italics and bold, alternately.
type TokenMacroIB struct {


    Text string

}

func ParseTokenMacroIB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIB arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroIB{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroIB) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set each argument in roman and italics, alternately.
type TokenMacroRI struct {


    Text string

}

func ParseTokenMacroRI(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRI arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRI arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroRI{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroRI) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set each argument in italics and roman, alternately.
type TokenMacroIR struct {


    Text string

}

func ParseTokenMacroIR(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIR arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIR arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroIR{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroIR) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set each argument in bold and roman, alternately.
type TokenMacroBR struct {


    Text string

}

func ParseTokenMacroBR(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBR arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBR arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroBR{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroBR) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set each argument in roman and bold, alternately.
type TokenMacroRB struct {


    Text string

}

func ParseTokenMacroRB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRB arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroRB{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroRB) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set text in bold. If no argument is given, a one-line input trap is
// planted; text on the next line, which can be further formatted with a
// macro, is set in bold.
type TokenMacroB struct {


    Text string

}

func ParseTokenMacroB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroB arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroB{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroB) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set text in an italic or oblique face. If no argument is given, a
// one-line input trap is planted; text on the next line, which can be
// further formatted with a macro, is set in an italic or oblique face.
type TokenMacroI struct {


    Text string

}

func ParseTokenMacroI(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroI arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroI arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroI{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroI) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Reset tab stops to the default (every 0.5i). Use of this
// presentation-oriented macro is deprecated. It translates poorly to HTML,
// under which exact space control and tabulation are not readily
// available. Thus, information or distinctions that you use tab stops to
// express are likely to be lost. If you feel tempted to change the tab
// stops such that calling this macro later is desirable to restore them,
// you should probably be composing a table using tbl(1) instead.
type TokenMacroDT struct {

}

func ParseTokenMacroDT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDT arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDT arguments: %s, %w", lineArgs, err)
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
// Define the vertical space between paragraphs or (sub)sections. The
// optional argument vertical-space specifies the amount; the default
// scaling unit is “v”. Without an argument, the spacing is reset to
// its default value; see subsection “Horizontal and vertical spacing”
// above. Use of this presentation-oriented macro is deprecated. It
// translates poorly to HTML, under which exact control of inter-paragraph
// spacing is not readily available. Thus, information or distinctions that
// you use .PD to express are likely to be lost.
type TokenMacroPD struct {


    Nnn string

}

func ParseTokenMacroPD(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPD arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPD arguments: %s, %w", lineArgs, err)
    }
	
        argNnn := matched[0]
    
	return &TokenMacroPD{
	
        Nnn: argNnn,
    
	}, nil
}

func (t *TokenMacroPD) String() string {
	var args []string
	
	args = append(args, t.Nnn)
	
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
// Alter the footer for use with legacy AT&T man pages, overriding any
// definition of the footer-inside argument to .TH. This macro exists only
// to render man pages from historical systems. system can be any of the
// following: 3 (7th edition), 4 (System III), 5 (System V). The optional
// release argument specifies the release number, as in “System V Release
// 3”.
type TokenMacroAT struct {


    System string


    Release string

}

func ParseTokenMacroAT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAT arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAT arguments: %s, %w", lineArgs, err)
    }
	
        argSystem := matched[0]
    
        argRelease := matched[1]
    
	return &TokenMacroAT{
	
        System: argSystem,
    
        Release: argRelease,
    
	}, nil
}

func (t *TokenMacroAT) String() string {
	var args []string
	
	args = append(args, t.System)
	
	args = append(args, t.Release)
	
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
// Alter the footer for use with legacy BSD man pages, overriding any
// definition of the footer-inside argument to .TH. This macro exists only
// to render man pages from historical systems. version can be any of the
// following: 3 (3rd Berkeley Distribution), 4 (4th Berkeley Distribution),
// 5 (4.2 Berkeley Distribution), 6 (4.3 Berkeley Distribution), 7 (4.4
// Berkeley Distribution).
type TokenMacroUC struct {


    Version string

}

func ParseTokenMacroUC(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroUC arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroUC arguments: %s, %w", lineArgs, err)
    }
	
        argVersion := matched[0]
    
	return &TokenMacroUC{
	
        Version: argVersion,
    
	}, nil
}

func (t *TokenMacroUC) String() string {
	var args []string
	
	args = append(args, t.Version)
	
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
// Set the page header text (“page trap”).
type TokenMacroPT struct {

}

func ParseTokenMacroPT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPT arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPT arguments: %s, %w", lineArgs, err)
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
// Set the page footer text (“bottom trap”).
type TokenMacroBT struct {

}

func ParseTokenMacroBT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBT arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBT arguments: %s, %w", lineArgs, err)
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


    Key string

}

func ParseTokenMacroCT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroCT arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroCT arguments: %s, %w", lineArgs, err)
    }
	
        argKey := matched[0]
    
	return &TokenMacroCT{
	
        Key: argKey,
    
	}, nil
}

func (t *TokenMacroCT) String() string {
	var args []string
	
	args = append(args, t.Key)
	
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

func ParseTokenMacroCW(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroCW arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroCW arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacroDs(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDs arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDs arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacroDe(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDe arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDe arguments: %s, %w", lineArgs, err)
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
// Begin and end example. After .EX, filling is disabled and a
// constant-width (monospaced) font is selected. Calling .EE enables
// filling and restores the previous font. These macros are extensions
// introduced in Ninth Edition Research Unix.
type TokenMacroEX struct {


    Indent string

}

func ParseTokenMacroEX(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEX arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEX arguments: %s, %w", lineArgs, err)
    }
	
        argIndent := matched[0]
    
	return &TokenMacroEX{
	
        Indent: argIndent,
    
	}, nil
}

func (t *TokenMacroEX) String() string {
	var args []string
	
	args = append(args, t.Indent)
	
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

func ParseTokenMacroEE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEE arguments: %s, %w", lineArgs, err)
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
// Set text in Helvetica. If no text is present on the line where the macro
// is called, then the text of the next line appears in Helvetica.
type TokenMacroG struct {


    Text string

}

func ParseTokenMacroG(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroG arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroG arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroG{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroG) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set text in Helvetica Oblique. If no text is present on the line where
// the macro is called, then the text of the next line appears in Helvetica
// Oblique.
type TokenMacroGL struct {


    Text string

}

func ParseTokenMacroGL(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroGL arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroGL arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroGL{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroGL) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set text in Helvetica Bold. If no text is present on the line where the
// macro is called, then all text up to the next HB appears in Helvetica
// Bold.
type TokenMacroHB struct {


    Text string

}

func ParseTokenMacroHB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroHB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroHB arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroHB{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroHB) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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


    Text string

}

func ParseTokenMacroTB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTB arguments: %s, %w", lineArgs, err)
    }
	
        argText := matched[0]
    
	return &TokenMacroTB{
	
        Text: argText,
    
	}, nil
}

func (t *TokenMacroTB) String() string {
	var args []string
	
	args = append(args, t.Text)
	
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
// Set a manpage reference in Ultrix format. The title is in Courier
// instead of italic. Optional punctuation follows the section number
// without an intervening space.
type TokenMacroMS struct {


    Title string


    Sect string


    Punct string

}

func ParseTokenMacroMS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroMS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroMS arguments: %s, %w", lineArgs, err)
    }
	
        argTitle := matched[0]
    
        argSect := matched[1]
    
        argPunct := matched[2]
    
	return &TokenMacroMS{
	
        Title: argTitle,
    
        Sect: argSect,
    
        Punct: argPunct,
    
	}, nil
}

func (t *TokenMacroMS) String() string {
	var args []string
	
	args = append(args, t.Title)
	
	args = append(args, t.Sect)
	
	args = append(args, t.Punct)
	
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
// Begin a note. Print the optional title, or the word “Note”, centered
// on the page. Text following the macro makes up the body of the note, and
// is indented on both sides. If the first argument is C, the body of the
// note is printed centered (the second argument replaces the word
// “Note” if specified).
type TokenMacroNT struct {


    C string


    Title string

}

func ParseTokenMacroNT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNT arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNT arguments: %s, %w", lineArgs, err)
    }
	
        argC := matched[0]
    
        argTitle := matched[1]
    
	return &TokenMacroNT{
	
        C: argC,
    
        Title: argTitle,
    
	}, nil
}

func (t *TokenMacroNT) String() string {
	var args []string
	
	args = append(args, t.C)
	
	args = append(args, t.Title)
	
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

func ParseTokenMacroNE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNE arguments: %s, %w", lineArgs, err)
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


    Path string


    Punct string

}

func ParseTokenMacroPN(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPN arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPN arguments: %s, %w", lineArgs, err)
    }
	
        argPath := matched[0]
    
        argPunct := matched[1]
    
	return &TokenMacroPN{
	
        Path: argPath,
    
        Punct: argPunct,
    
	}, nil
}

func (t *TokenMacroPN) String() string {
	var args []string
	
	args = append(args, t.Path)
	
	args = append(args, t.Punct)
	
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
// If called with two arguments, identical to PN. If called with three
// arguments, set the second argument in constant width (Courier),
// bracketed by the first and third arguments in the current font.
type TokenMacroPn struct {


    Punct string


    Path string


    Punct2 string

}

func ParseTokenMacroPn(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPn arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            true,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPn arguments: %s, %w", lineArgs, err)
    }
	
        argPunct := matched[0]
    
        argPath := matched[1]
    
        argPunct2 := matched[2]
    
	return &TokenMacroPn{
	
        Punct: argPunct,
    
        Path: argPath,
    
        Punct2: argPunct2,
    
	}, nil
}

func (t *TokenMacroPn) String() string {
	var args []string
	
	args = append(args, t.Punct)
	
	args = append(args, t.Path)
	
	args = append(args, t.Punct2)
	
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

func ParseTokenMacroR(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroR arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroR arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacroRN(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRN arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRN arguments: %s, %w", lineArgs, err)
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
// Start printing a change bar in the margin if the number 4 is specified.
// Otherwise, this macro does nothing.
type TokenMacroVS struct {


    Arg4 string

}

func ParseTokenMacroVS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroVS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroVS arguments: %s, %w", lineArgs, err)
    }
	
        argArg4 := matched[0]
    
	return &TokenMacroVS{
	
        Arg4: argArg4,
    
	}, nil
}

func (t *TokenMacroVS) String() string {
	var args []string
	
	args = append(args, t.Arg4)
	
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

func ParseTokenMacroVE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroVE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroVE arguments: %s, %w", lineArgs, err)
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
// Use the “report” (AT&T: “released paper”) format for your
// document, creating a separate cover page. The default arrangement is to
// place most of the document description (title, author names and
// institutions, and abstract, but not the date) at the top of the first
// page. If the optional no-repeat-info argument is given, ms produces a
// cover page but does not repeat any of its information on subsequently
// (but see the DA macro below regarding the date). Normally, .RP sets the
// page number following the cover page to 1. Specifying the optional
// no-renumber argument suppresses this alteration. Optional arguments can
// occur in any order. “no” is recognized as a synonym of
// no-repeat-info for AT&T compatibility.
type TokenMacroRP struct {


    No string

}

func ParseTokenMacroRP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRP arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRP arguments: %s, %w", lineArgs, err)
    }
	
        argNo := matched[0]
    
	return &TokenMacroRP{
	
        No: argNo,
    
	}, nil
}

func (t *TokenMacroRP) String() string {
	var args []string
	
	args = append(args, t.No)
	
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
// Typeset the header even on page 1. To be effective, this macro must be
// called before the header trap is sprung on any page numbered “1”.
// This is a Berkeley extension.
type TokenMacroP1 struct {

}

func ParseTokenMacroP1(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroP1 arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroP1 arguments: %s, %w", lineArgs, err)
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
// Typeset the current date, or any arguments x, in the center footer, and,
// if .RP is also called, left-aligned at the end of the document
// description on the cover page.
type TokenMacroDA struct {


    Ellipse string

}

func ParseTokenMacroDA(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDA arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDA arguments: %s, %w", lineArgs, err)
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
// Typeset the current date, or any arguments x, if .RP is also called,
// left-aligned at the end of the document description on the cover page.
// This is groff ms's default.
type TokenMacroND struct {


    Ellipse string

}

func ParseTokenMacroND(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroND arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroND arguments: %s, %w", lineArgs, err)
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
// Specify the document title. ms collects text on input lines following
// this call into the title until reaching .AU, .AB, or a heading or
// paragraphing macro call.
type TokenMacroTL struct {

}

func ParseTokenMacroTL(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTL arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTL arguments: %s, %w", lineArgs, err)
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
// Specify an author's name. ms collects text on input lines following this
// call into the author's name until reaching .AI, .AB, another .AU, or a
// heading or paragraphing macro call. Call it repeatedly to specify
// multiple authors.
type TokenMacroAU struct {

}

func ParseTokenMacroAU(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAU arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAU arguments: %s, %w", lineArgs, err)
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
// Specify the preceding author's institution. An .AU call is usefully
// followed by at most one .AI call; if there are more, the last .AI call
// controls. ms collects text on input lines following this call into the
// author's institution until reaching .AU, .AB, or a heading or
// paragraphing macro call.
type TokenMacroAI struct {

}

func ParseTokenMacroAI(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAI arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAI arguments: %s, %w", lineArgs, err)
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
// Begin the abstract. ms collects text on input lines following this call
// into the abstract until reaching an .AE call. By default, ms places the
// word “ABSTRACT” centered and in italics above the text of the
// abstract. The optional argument “no” suppresses this heading.
type TokenMacroAB struct {


    No string

}

func ParseTokenMacroAB(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAB arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAB arguments: %s, %w", lineArgs, err)
    }
	
        argNo := matched[0]
    
	return &TokenMacroAB{
	
        No: argNo,
    
	}, nil
}

func (t *TokenMacroAB) String() string {
	var args []string
	
	args = append(args, t.No)
	
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
// End the abstract.
type TokenMacroAE struct {

}

func ParseTokenMacroAE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAE arguments: %s, %w", lineArgs, err)
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
// Set a paragraph with a first-line left indentation in the amount stored
// in the PI register.
type TokenMacroPP struct {

}

func ParseTokenMacroPP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPP arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPP arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacroLP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroLP arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroLP arguments: %s, %w", lineArgs, err)
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
// Sets a paragraph that is indented at both left and right margins by the
// amount of the register QI. The effect is identical to the HTML
// <BLOCKQUOTE> element. The next paragraph or heading returns margins to
// normal. QP inserts vertical space of amount set by register PD before
// the paragraph.
type TokenMacroQP struct {

}

func ParseTokenMacroQP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroQP arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroQP arguments: %s, %w", lineArgs, err)
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
// These macros begin and end a quoted section. The QI register controls
// the amount of indentation. Both QS and QE insert inter-paragraph
// vertical space set by register PD. The text between QS and QE can be
// structured further by use of the macros LP or PP.
type TokenMacroQE struct {

}

func ParseTokenMacroQE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroQE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroQE arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacroXP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXP arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXP arguments: %s, %w", lineArgs, err)
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
// Numbered heading. The argument is either a numeric argument to indicate
// the level of the heading, or the letter S followed by numeric arguments
// to set the heading level explicitly.
// 
// If you specify heading levels out of sequence, such as invoking ‘.NH
// 3’ after ‘.NH 1’, groff prints a warning on standard error.
type TokenMacroNH struct {


    S string


    Level0 string


    Ellipse string

}

func ParseTokenMacroNH(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNH arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            
            false,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNH arguments: %s, %w", lineArgs, err)
    }
	
        argS := matched[0]
    
        argLevel0 := matched[1]
    
        argEllipse := matched[2]
    
	return &TokenMacroNH{
	
        S: argS,
    
        Level0: argLevel0,
    
        Ellipse: argEllipse,
    
	}, nil
}

func (t *TokenMacroNH) String() string {
	var args []string
	
	args = append(args, t.S)
	
	args = append(args, t.Level0)
	
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
// Prints its argument and draws a box around it. If you want to box a
// string that contains spaces, use a digit-width space (\0).
type TokenMacroBX struct {


    Txt string

}

func ParseTokenMacroBX(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBX arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBX arguments: %s, %w", lineArgs, err)
    }
	
        argTxt := matched[0]
    
	return &TokenMacroBX{
	
        Txt: argTxt,
    
	}, nil
}

func (t *TokenMacroBX) String() string {
	var args []string
	
	args = append(args, t.Txt)
	
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
// Prints its first argument with an underline. If you specify a second
// argument, groff prints it in the previous font after the underlined
// text, with no intervening space.
type TokenMacroUL struct {


    Txt string


    Post string

}

func ParseTokenMacroUL(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroUL arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroUL arguments: %s, %w", lineArgs, err)
    }
	
        argTxt := matched[0]
    
        argPost := matched[1]
    
	return &TokenMacroUL{
	
        Txt: argTxt,
    
        Post: argPost,
    
	}, nil
}

func (t *TokenMacroUL) String() string {
	var args []string
	
	args = append(args, t.Txt)
	
	args = append(args, t.Post)
	
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
// or heading macro. You can specify this macro multiple times to enlarge
// the point size as needed.
type TokenMacroLG struct {

}

func ParseTokenMacroLG(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroLG arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroLG arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacroNL(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroNL arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroNL arguments: %s, %w", lineArgs, err)
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
// Use this macro to reset the tab stops to the default for ms (every 5n).
// You can redefine the TA macro to create a different set of default tab
// stops.
type TokenMacroTA struct {

}

func ParseTokenMacroTA(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTA arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTA arguments: %s, %w", lineArgs, err)
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
// Left-justified display. The ‘.DS L’ call generates a page break, if
// necessary, to keep the entire display on one page. The LD macro allows
// the display to break across pages. The DE macro ends the display.
type TokenMacroDE struct {

}

func ParseTokenMacroDE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDE arguments: %s, %w", lineArgs, err)
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
// The KS macro begins a block of text to be kept on a single page, and the
// KE macro ends the block.
type TokenMacroKE struct {

}

func ParseTokenMacroKE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroKE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroKE arguments: %s, %w", lineArgs, err)
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
// around it. The B1 macro begins the box; the B2 macro ends it. Text in
// the box is automatically placed in a diversion (keep).
type TokenMacroB2 struct {

}

func ParseTokenMacroB2(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroB2 arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroB2 arguments: %s, %w", lineArgs, err)
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
// Denotes a table, to be processed by the tbl preprocessor. The optional
// argument H to TS instructs groff to create a running header with the
// information up to the TH macro. groff prints the header at the beginning
// of the table; if the table runs onto another page, groff prints the
// header on the next page as well.
type TokenMacroTE struct {

}

func ParseTokenMacroTE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTE arguments: %s, %w", lineArgs, err)
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
// Denotes a graphic, to be processed by the pic preprocessor. You can
// create a pic file by hand, using the AT&T pic manual available on the
// Web as a reference, or by using a graphics program such as xfig.
type TokenMacroPE struct {

}

func ParseTokenMacroPE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPE arguments: %s, %w", lineArgs, err)
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
// Denotes an equation, to be processed by the eqn preprocessor. The
// optional align argument can be C, L, or I to center (the default),
// left-justify, or indent the equation.
type TokenMacroEN struct {

}

func ParseTokenMacroEN(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEN arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEN arguments: %s, %w", lineArgs, err)
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
// Denotes a reference, to be processed by the refer preprocessor. The GNU
// refer(1) man page provides a comprehensive reference to the preprocessor
// and the format of the bibliographic database.
type TokenMacroSquareBracketClose struct {

}

func ParseTokenMacroSquareBracketClose(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSquareBracketClose arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSquareBracketClose arguments: %s, %w", lineArgs, err)
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
// Specifies the text of the footnote. The default action is to create a
// numbered footnote; you can create a symbolic footnote by specifying a
// mark glyph (such as \[dg] for the dagger glyph) in the body text and as
// an argument to the FS macro, followed by the text of the footnote and
// the FE macro.
type TokenMacroFE struct {

}

func ParseTokenMacroFE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroFE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroFE arguments: %s, %w", lineArgs, err)
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
// The OH and EH macros define headers for the odd and even pages; the OF
// and EF macros define footers for the odd and even pages. This is more
// flexible than defining the individual strings.
// 
// You can replace the quote (') marks with any character not appearing in
// the header or footer text.
type TokenMacroEF struct {


    Leftcenterright string

}

func ParseTokenMacroEF(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEF arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEF arguments: %s, %w", lineArgs, err)
    }
	
        argLeftcenterright := matched[0]
    
	return &TokenMacroEF{
	
        Leftcenterright: argLeftcenterright,
    
	}, nil
}

func (t *TokenMacroEF) String() string {
	var args []string
	
	args = append(args, t.Leftcenterright)
	
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

func ParseTokenMacro1C(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacro1C arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacro1C arguments: %s, %w", lineArgs, err)
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

func ParseTokenMacro2C(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacro2C arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacro2C arguments: %s, %w", lineArgs, err)
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
// Multi-column mode. If you specify no arguments, it is equivalent to the
// 2C macro. Otherwise, width is the width of each column and gutter is the
// space between columns. The MINGW number register controls the default
// gutter width.
type TokenMacroMC struct {


    Width string


    Gutter string

}

func ParseTokenMacroMC(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroMC arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroMC arguments: %s, %w", lineArgs, err)
    }
	
        argWidth := matched[0]
    
        argGutter := matched[1]
    
	return &TokenMacroMC{
	
        Width: argWidth,
    
        Gutter: argGutter,
    
	}, nil
}

func (t *TokenMacroMC) String() string {
	var args []string
	
	args = append(args, t.Width)
	
	args = append(args, t.Gutter)
	
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
// table of contents, depending on their use. The macros are very simple;
// they cannot indent a heading based on its level. The easiest way to work
// around this is to add tabs to the table of contents string. The
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
// You can manually create a table of contents by beginning with the XS
// macro for the first entry, specifying the page number for that entry as
// the argument to XS. Add subsequent entries using the XA macro,
// specifying the page number for that entry as the argument to XA. The
// following is an example:
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

func ParseTokenMacroXE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXE arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXE arguments: %s, %w", lineArgs, err)
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
// Prints the table of contents on a new page, setting the page number to i
// (Roman lowercase numeral one). You should usually place this macro at
// the end of the file, since groff is a single-pass formatter and can only
// print what has been collected up to the point that the TC macro appears.
// 
// The optional argument no suppresses printing the title specified by the
// string register TOC.
type TokenMacroTC struct {


    No string

}

func ParseTokenMacroTC(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTC arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTC arguments: %s, %w", lineArgs, err)
    }
	
        argNo := matched[0]
    
	return &TokenMacroTC{
	
        No: argNo,
    
	}, nil
}

func (t *TokenMacroTC) String() string {
	var args []string
	
	args = append(args, t.No)
	
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
// numbering sequence. Use this macro to print a manually generated table
// of contents at the beginning of your document.
// 
// The optional argument no suppresses printing the title specified by the
// string register TOC.
type TokenMacroPX struct {


    No string

}

func ParseTokenMacroPX(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPX arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPX arguments: %s, %w", lineArgs, err)
    }
	
        argNo := matched[0]
    
	return &TokenMacroPX{
	
        No: argNo,
    
	}, nil
}

func (t *TokenMacroPX) String() string {
	var args []string
	
	args = append(args, t.No)
	
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
// accent marks and special characters. This is a Berkeley extension.
// 
// To use the accent marks, place them after the character being accented.
// 
// Note that groff’s native support for accents is superior to the
// following definitions.
type TokenMacroAM struct {

}

func ParseTokenMacroAM(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroAM arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroAM arguments: %s, %w", lineArgs, err)
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
// Indented display. The default behavior of AT&T troff -ms was to indent;
// the groff default prints displays flush left with the body text.
type TokenMacroDS struct {


    I string

}

func ParseTokenMacroDS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroDS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroDS arguments: %s, %w", lineArgs, err)
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
// Indexing term (printed on standard error). You can write a script to
// capture and process an index generated in this manner.
type TokenMacroIX struct {

}

func ParseTokenMacroIX(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroIX arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroIX arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroXA struct {


    Page string

}

func ParseTokenMacroXA(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXA arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXA arguments: %s, %w", lineArgs, err)
    }
	
        argPage := matched[0]
    
	return &TokenMacroXA{
	
        Page: argPage,
    
	}, nil
}

func (t *TokenMacroXA) String() string {
	var args []string
	
	args = append(args, t.Page)
	
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroXS struct {


    Page string

}

func ParseTokenMacroXS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroXS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroXS arguments: %s, %w", lineArgs, err)
    }
	
        argPage := matched[0]
    
	return &TokenMacroXS{
	
        Page: argPage,
    
	}, nil
}

func (t *TokenMacroXS) String() string {
	var args []string
	
	args = append(args, t.Page)
	
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroHD struct {

}

func ParseTokenMacroHD(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroHD arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroHD arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroOF struct {


    Leftcenterright string

}

func ParseTokenMacroOF(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroOF arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroOF arguments: %s, %w", lineArgs, err)
    }
	
        argLeftcenterright := matched[0]
    
	return &TokenMacroOF{
	
        Leftcenterright: argLeftcenterright,
    
	}, nil
}

func (t *TokenMacroOF) String() string {
	var args []string
	
	args = append(args, t.Leftcenterright)
	
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroEH struct {


    Leftcenterright string

}

func ParseTokenMacroEH(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEH arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEH arguments: %s, %w", lineArgs, err)
    }
	
        argLeftcenterright := matched[0]
    
	return &TokenMacroEH{
	
        Leftcenterright: argLeftcenterright,
    
	}, nil
}

func (t *TokenMacroEH) String() string {
	var args []string
	
	args = append(args, t.Leftcenterright)
	
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroOH struct {


    Leftcenterright string

}

func ParseTokenMacroOH(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroOH arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroOH arguments: %s, %w", lineArgs, err)
    }
	
        argLeftcenterright := matched[0]
    
	return &TokenMacroOH{
	
        Leftcenterright: argLeftcenterright,
    
	}, nil
}

func (t *TokenMacroOH) String() string {
	var args []string
	
	args = append(args, t.Leftcenterright)
	
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroFS struct {

}

func ParseTokenMacroFS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroFS arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroFS arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroSquareBracketOpen struct {

}

func ParseTokenMacroSquareBracketOpen(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSquareBracketOpen arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSquareBracketOpen arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroEQ struct {


    Align string

}

func ParseTokenMacroEQ(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroEQ arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroEQ arguments: %s, %w", lineArgs, err)
    }
	
        argAlign := matched[0]
    
	return &TokenMacroEQ{
	
        Align: argAlign,
    
	}, nil
}

func (t *TokenMacroEQ) String() string {
	var args []string
	
	args = append(args, t.Align)
	
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroPS struct {

}

func ParseTokenMacroPS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroPS arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroPS arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroTS struct {


    H string

}

func ParseTokenMacroTS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTS arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTS arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroB1 struct {

}

func ParseTokenMacroB1(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroB1 arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroB1 arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroKF struct {

}

func ParseTokenMacroKF(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroKF arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroKF arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroKS struct {

}

func ParseTokenMacroKS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroKS arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroKS arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroRD struct {

}

func ParseTokenMacroRD(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroRD arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroRD arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroCD struct {

}

func ParseTokenMacroCD(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroCD arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroCD arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroBD struct {

}

func ParseTokenMacroBD(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroBD arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroBD arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroID struct {

}

func ParseTokenMacroID(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroID arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroID arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroLD struct {

}

func ParseTokenMacroLD(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroLD arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroLD arguments: %s, %w", lineArgs, err)
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
// The registers ps4html and www-image-template are defined by the
// pre-grohtml preprocessor. pre-grohtml reads in the troff input, marks up
// the inline equations and passes the result firstly to
// 
// troff -Tps -rps4html=1 -dwww-image-template=template
// 
// and secondly to
// 
// troff -Thtml
// 
// or
// 
// troff -Txhtml
// 
// 
// 
// The POSTSCRIPT device is used to create all the image files (for -Thtml;
// if -Txhtml is used, all equations are passed to geqn to produce MathML,
// and the register ps4html enables the macro sets to ignore floating
// keeps, footers, and headings.
// 
// The register www-image-template is set to the user specified template
// name or the default name.
type TokenMacroQS struct {

}

func ParseTokenMacroQS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroQS arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroQS arguments: %s, %w", lineArgs, err)
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


// TokenMacroSY
// Begin synopsis. A new paragraph begins at the left margin unless .SY has
// already been called without a corresponding .YS, in which case only a
// break is performed. Adjustment and automatic hyphenation are disabled.
// command is set in bold. If a break is required, lines after the first
// are indented by the width of command plus a space.
type TokenMacroSY struct {


    Command string

}

func ParseTokenMacroSY(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroSY arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroSY arguments: %s, %w", lineArgs, err)
    }
	
        argCommand := matched[0]
    
	return &TokenMacroSY{
	
        Command: argCommand,
    
	}, nil
}

func (t *TokenMacroSY) String() string {
	var args []string
	
	args = append(args, t.Command)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".SY"
	}
	return ".SY " + strings.Join(args, " ")
}


// TokenMacroYS
// End synopsis. Indentation, adjustment, and hyphenation are restored to
// their previous states.
type TokenMacroYS struct {

}

func ParseTokenMacroYS(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroYS arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroYS arguments: %s, %w", lineArgs, err)
    }
	
	return &TokenMacroYS{
	
	}, nil
}

func (t *TokenMacroYS) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".YS"
	}
	return ".YS " + strings.Join(args, " ")
}


// TokenMacroMR
// Set a man page cross reference as “topic(manual-section)”. If
// trailing-text (typically punctuation) is specified, it follows the
// closing parenthesis without intervening space. Hyphenation is disabled
// while the cross reference is set. topic is set in the font specified by
// the MF string. The cross reference hyperlinks to a URI of the form
// “man:topic(manual-section)”.
type TokenMacroMR struct {


    Topic string


    Manualsection string


    Trailingtext string

}

func ParseTokenMacroMR(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroMR arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroMR arguments: %s, %w", lineArgs, err)
    }
	
        argTopic := matched[0]
    
        argManualsection := matched[1]
    
        argTrailingtext := matched[2]
    
	return &TokenMacroMR{
	
        Topic: argTopic,
    
        Manualsection: argManualsection,
    
        Trailingtext: argTrailingtext,
    
	}, nil
}

func (t *TokenMacroMR) String() string {
	var args []string
	
	args = append(args, t.Topic)
	
	args = append(args, t.Manualsection)
	
	args = append(args, t.Trailingtext)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".MR"
	}
	return ".MR " + strings.Join(args, " ")
}


// TokenMacroMT
// Identify address as an RFC 6068 addr-spec for a “mailto:” URI with
// the text between the two macro calls as the link text. An argument to
// .ME is placed after the link text without intervening space. address may
// not be visible in the rendered document if hyperlinks are enabled and
// supported by the output driver. If they are not, address is set in angle
// brackets after the link text and before trailing-text. If hyperlinking
// is enabled but there is no link text, address is formatted and
// hyperlinked without angle brackets.
type TokenMacroMT struct {


    Address string

}

func ParseTokenMacroMT(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroMT arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroMT arguments: %s, %w", lineArgs, err)
    }
	
        argAddress := matched[0]
    
	return &TokenMacroMT{
	
        Address: argAddress,
    
	}, nil
}

func (t *TokenMacroMT) String() string {
	var args []string
	
	args = append(args, t.Address)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".MT"
	}
	return ".MT " + strings.Join(args, " ")
}


// TokenMacroME
// End a mail-to hyperlink.
type TokenMacroME struct {


    Trailingtext string

}

func ParseTokenMacroME(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroME arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroME arguments: %s, %w", lineArgs, err)
    }
	
        argTrailingtext := matched[0]
    
	return &TokenMacroME{
	
        Trailingtext: argTrailingtext,
    
	}, nil
}

func (t *TokenMacroME) String() string {
	var args []string
	
	args = append(args, t.Trailingtext)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".ME"
	}
	return ".ME " + strings.Join(args, " ")
}


// TokenMacroUR
// Identify uri as an RFC 3986 URI hyperlink with the text between the two
// macro calls as the link text. An argument to .UE is placed after the
// link text without intervening space. uri may not be visible in the
// rendered document if hyperlinks are enabled and supported by the output
// driver. If they are not, uri is set in angle brackets after the link
// text and before trailing-text. If hyperlinking is enabled but there is
// no link text, uri is formatted and hyperlinked without angle brackets.
type TokenMacroUR struct {


    Uri string

}

func ParseTokenMacroUR(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroUR arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  BackwardMatcher(args,
            
            false,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroUR arguments: %s, %w", lineArgs, err)
    }
	
        argUri := matched[0]
    
	return &TokenMacroUR{
	
        Uri: argUri,
    
	}, nil
}

func (t *TokenMacroUR) String() string {
	var args []string
	
	args = append(args, t.Uri)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".UR"
	}
	return ".UR " + strings.Join(args, " ")
}


// TokenMacroUE
// End a URI hyperlink.
type TokenMacroUE struct {


    Trailingtext string

}

func ParseTokenMacroUE(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroUE arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroUE arguments: %s, %w", lineArgs, err)
    }
	
        argTrailingtext := matched[0]
    
	return &TokenMacroUE{
	
        Trailingtext: argTrailingtext,
    
	}, nil
}

func (t *TokenMacroUE) String() string {
	var args []string
	
	args = append(args, t.Trailingtext)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".UE"
	}
	return ".UE " + strings.Join(args, " ")
}


// TokenMacroTQ
// Set an additional tag for a paragraph tagged with .TP. An input trap is
// planted as with .TP. This macro is a GNU extension not defined on
// systems running AT&T, Plan 9, or Solaris troff.
type TokenMacroTQ struct {

}

func ParseTokenMacroTQ(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroTQ arguments: %s, %w", lineArgs, err)
    }
    
      _, err =  BackwardMatcher(args,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroTQ arguments: %s, %w", lineArgs, err)
    }
	
	return &TokenMacroTQ{
	
	}, nil
}

func (t *TokenMacroTQ) String() string {
	var args []string
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".TQ"
	}
	return ".TQ " + strings.Join(args, " ")
}


// TokenMacroOP
// Indicate an optional command parameter called option-name, which is set
// in bold. If the option takes an argument, specify option-argument using
// a noun, abbreviation, or hyphenated noun phrase. If present,
// option-argument is preceded by a space and set in italics. Square
// brackets in roman surround both arguments. Use of this quasi-semantic
// macro is deprecated.
type TokenMacroOP struct {


    Optionname string


    Optionargument string

}

func ParseTokenMacroOP(lineArgs string) (Token, error) {
    args, _, err := ArgumentParser(lineArgs)
    if err != nil {
        return nil, fmt.Errorf("parse TokenMacroOP arguments: %s, %w", lineArgs, err)
    }
    
      matched, err :=  ForwardMatcher(args,
            
            false,
            
            true,
            )
    
    if err != nil {
        return nil, fmt.Errorf("fitting TokenMacroOP arguments: %s, %w", lineArgs, err)
    }
	
        argOptionname := matched[0]
    
        argOptionargument := matched[1]
    
	return &TokenMacroOP{
	
        Optionname: argOptionname,
    
        Optionargument: argOptionargument,
    
	}, nil
}

func (t *TokenMacroOP) String() string {
	var args []string
	
	args = append(args, t.Optionname)
	
	args = append(args, t.Optionargument)
	
	for len(args) > 0 && args[len(args)-1] == "" {
		args = args[:len(args)-1]
	}
	for i, arg := range args {
		args[i] = Quote(arg)
	}
	if len(args) == 0 {
		return ".OP"
	}
	return ".OP " + strings.Join(args, " ")
}

