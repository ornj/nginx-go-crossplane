package crossplane

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Token is a lexical token of the NGINX configuration syntax.
type Token struct {
	// Text is the string corresponding to the token. It could be a directive or symbol. The value is the actual token
	// sequence in order to support defining directives in modules other than the core NGINX module set.
	Text string
	// Line is the source starting line number the token within a file.
	Line int
	// IsQuoted signifies if the token is wrapped by quotes (", '). Quotes are not usually necessary in an NGINX
	// configuration and mostly serve to help make the config less ambiguous.
	IsQuoted bool
}

type scannerError struct {
	msg  string
	line int
}

func (e *scannerError) Error() string { return e.msg }
func (e *scannerError) Line() int     { return e.line }

func NewScannerErrf(line int, format string, a ...any) *scannerError {
	return &scannerError{line: line, msg: fmt.Sprintf(format, a...)}
}

// LineNumber reports the line on which the error occurred by finding the first error in
// the errs chain that returns a line number. Otherwise, it returns 0, false.
//
// An error type should provide a Line() int method to return a line number.
func LineNumber(err error) (int, bool) {
	var e interface{ Line() int }
	if !errors.As(err, &e) {
		return 0, false
	}

	return e.Line(), true
}

// Scanner provides an interface for tokenizing an NGINX configuration. Successive calls to the Scane method will step
// through the 'tokens; of an NGINX configuration.
//
// Scanning stops unrecoverably at EOF, the first I/O error, or an unexpected token.
//
// Use NewScanner to construct a Scanner.
type Scanner struct {
	scanner            *bufio.Scanner
	lineno             int
	tokenStartLine     int
	tokenDepth         int
	repeateSpecialChar bool //  only '}' can be repeated
	prev               string
	err                error
}

// NewScanner returns a new Scanner to read from r.
func NewScanner(r io.Reader) *Scanner {
	s := &Scanner{
		scanner:            bufio.NewScanner(r),
		lineno:             1,
		tokenStartLine:     1,
		tokenDepth:         0,
		repeateSpecialChar: false,
	}

	s.scanner.Split(bufio.ScanRunes)

	return s
}

// Err returns the first non-EOF error that was encountered by the Scanner.
func (s *Scanner) Err() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}

func (s *Scanner) setErr(err error) {
	if s.err == nil || s.err != io.EOF {
		s.err = err
	}
}

// Scan reads the next token from source and returns it.. It returns io.EOF at the end of the source. Scanner errors are
// returned when encountered.
func (s *Scanner) Scan() (Token, error) { //nolint: funlen, gocognit, gocyclo
	var tok strings.Builder

	lexState := skipSpace
	newToken := false
	readNext := true
	esc := false

	var r, quote string

	for {
		if s.err != nil {
			return Token{}, s.err
		}

		switch {
		case s.prev != "":
			r, s.prev = s.prev, ""
		case readNext:
			if !s.scanner.Scan() {
				if tok.Len() > 0 {
					return Token{Text: tok.String(), Line: s.tokenStartLine, IsQuoted: lexState == inQuote}, nil
				}

				if s.tokenDepth > 0 {
					s.setErr(&scannerError{line: s.tokenStartLine, msg: "unexpected end of file, expecting }"})
					return Token{}, s.err
				}

				s.setErr(io.EOF)
				return Token{}, s.err
			}

			nextRune := s.scanner.Text()
			r = nextRune
			if isEOL(r) {
				s.lineno++
			}
		default:
			readNext = true
		}

		// skip CRs
		if r == "\r" || r == "\\\r" {
			continue
		}

		if r == "\\" && !esc {
			esc = true
			continue
		}

		if esc {
			esc = false
			r = "\\" + r
		}

		switch lexState {
		case skipSpace:
			if !isSpace(r) {
				lexState = inWord
				newToken = true
				readNext = false // re-eval
				s.tokenStartLine = s.lineno
			}
			continue

		case inWord:
			if newToken {
				newToken = false
				if r == "#" {
					tok.WriteString(r)
					lexState = inComment
					s.tokenStartLine = s.lineno
					continue
				}
			}

			if isSpace(r) {
				return Token{Text: tok.String(), Line: s.tokenStartLine}, nil
			}

			// parameter expansion syntax (ex: "${var[@]}")
			if tok.Len() > 0 && strings.HasSuffix(tok.String(), "$") && r == "{" {
				tok.WriteString(r)
				lexState = inVar
				s.repeateSpecialChar = false
				continue
			}

			// add entire quoted string to the token buffer
			if r == `"` || r == "'" {
				if tok.Len() > 0 {
					// if a quote is inside a token, treat it like any other char
					tok.WriteString(r)
				} else {
					quote = r
					lexState = inQuote
					s.tokenStartLine = s.lineno
				}
				s.repeateSpecialChar = false
				continue
			}

			// special characters treated as full tokens
			if isSpecialChar(r) {
				if tok.Len() > 0 {
					s.prev = r
					return Token{Text: tok.String(), Line: s.tokenStartLine}, nil
				}

				// only } can be repeated
				if s.repeateSpecialChar && r != "}" {
					s.setErr(NewScannerErrf(s.tokenStartLine, "unxpected %q", r))
					return Token{}, s.err
				}

				s.repeateSpecialChar = true
				if r == "{" {
					s.tokenDepth++
				}

				if r == "}" {
					s.tokenDepth--
					if s.tokenDepth < 0 {
						s.setErr(&scannerError{line: s.tokenStartLine, msg: `unexpected "}"`})
						return Token{}, s.err
					}
				}

				tok.WriteString(r)
				return Token{Text: tok.String(), Line: s.tokenStartLine}, nil
			}

			s.repeateSpecialChar = false
			tok.WriteString(r)
		case inComment:
			if isEOL(r) {
				return Token{Text: tok.String(), Line: s.tokenStartLine}, nil
			}
			tok.WriteString(r)
		case inVar:
			tok.WriteString(r)
			if r != "}" && !isSpace(r) {
				continue
			}
			lexState = inWord
		case inQuote:
			if r == quote {
				return Token{Text: tok.String(), Line: s.tokenStartLine}, nil
			}
			if r == "\\"+quote {
				r = quote
			}
			tok.WriteString(r)
		}
	}
}

// Extend returns a scanner extension that can be used to advance the scanner while applying custom lexical rules that
// differ from the usual NGINX configuration grammar. Only one extension is to be used to scan at a time and discarded
// when no longer needed. Do not advance the scanner until the extension is no longer need and discarded. If the
// extension encounters an error while scanning, the error will stop the parent scanner unrecoverably.
func (s *Scanner) Extend() (*SubScanner, error) {
	if s.err != nil {
		return nil, s.err
	}

	rv := &SubScanner{parent: s}
	return rv, nil
}

func NewSubScanner(s *Scanner) (*SubScanner, error) {
	if s.err != nil {
		return nil, fmt.Errorf("scanner has been stopped with error: %w", s.err)
	}

	return &SubScanner{parent: s}, nil
}

func (s *Scanner) ScanWith(ext Extension) (Token, error) {
	if s.err != nil {
		return Token{}, s.err
	}

	tok, err := ext.Scan()
	if err != nil && !errors.Is(err, EndScanWith) {
		s.setErr(err)
		return tok, err
	}

	if err == nil && tok.Line < s.lineno {
		return tok, NewScannerErrf(tok.Line, "invalid token: token is before current scanner position %d", s.lineno)
	} else if err == nil {
		s.lineno = tok.Line
	}

	return tok, err
}

var EndScanWith = errors.New("end sub scanning")

type Extension interface { // TODO: Ineffective name
	Scan() (Token, error)
}

// SubScanner provides an interface for scanning alternative grammars within NGINX configuration data. If the extension
// encounters an error while scanning, the error will stop the parent scanner unrecoverably.
// Use Scanner.Extend to create an extension.
type SubScanner struct {
	parent *Scanner
}

// Text returns the most recent token generated by a call to Scan.
func (e *SubScanner) Text() string { return e.parent.prev }

// Err returns the fist non-EOF error encountered by the Extension or parent Scanner.
func (e *SubScanner) Err() error { return e.parent.err }

// Scan advances the parent scanner to the next token which will be availabvle though the Text method. It returns false
// when the scan stops, either by reaching the end of input or an error.
func (e *SubScanner) Scan() bool {
	if e.parent.err != nil {
		return false
	}

	if !e.parent.scanner.Scan() {
		if err := e.parent.scanner.Err(); err != nil {
			e.parent.setErr(err)
		}
		return false
	}

	e.parent.prev = e.parent.scanner.Text()
	return true
}