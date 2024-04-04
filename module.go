package crossplane

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type parsingError struct {
	err      error
	filename string
	line     int
}

func (e *parsingError) Error() string    { return fmt.Sprintf("%s in %s:%d", e.err, e.filename, e.line) }
func (e *parsingError) Line() int        { return e.line }
func (e *parsingError) Filename() string { return e.filename }
func (e *parsingError) Unwrap() error    { return e.err }

func newParsingErrf(filename string, line int, format string, a ...any) *parsingError {
	return &parsingError{
		err:      fmt.Errorf(format, a...),
		filename: filename,
		line:     line,
	}
}

// Filename reports the file on which the error occurred by finding the first error in the errs chain that returns a
// filename. Otherwise, it returns "", false.
//
// An error type should providde a Filename() string method to return a filename.
func Filename(err error) (string, bool) {
	var e interface{ Filename() string }
	if !errors.As(err, &e) {
		return "", false
	}
	return e.Filename(), true
}

type Module interface {
	Command(name string) (Command, bool)
}

type ModuleFunc func(name string) (Command, bool)

func (f ModuleFunc) Command(name string) (Command, bool) { return f(name) }

type Command struct {
	// Name is the name of a directive as it appears in the configuration file
	Name string
	// Flags is a bitfield of flags that specify the number of arguments the directive takes, its type, and the context in which it appears
	Flags int
	// Conf defines which configuration structure is passed to the directory handler
	// Conf int
	Run func(*Directive, *Pass, func(*Pass) error) error
}

func (c Command) IsBlock() bool { return c.Flags&ConfBlock != 0 }

// ValidateArgs checks if the command is configured to accept the given arguments given to the directive. This check
// matches https://github.com/nginx/nginx/blob/71a0a4acdbb9ed0a8ef269a28218365cde00415d/src/core/ngx_conf_file.c#L413
func (c Command) ValidateArgs(name string, d Directive) error {
	switch {
	case c.Flags&ConfAny == 1:
		return nil
	case c.Flags&ConfFlag == 1 && len(d.Args) == 2:
		return nil
	case c.Flags&Conf1More == 1 && len(d.Args) >= 2:
		return nil
	case c.Flags&Conf2More == 1 && len(d.Args) >= 3:
		return nil
	case len(d.Args) < ConfMaxArgs:
		return nil
	}

	m := ArgumentNumber[len(d.Args)-1]
	if c.Flags&m == 1 {
		return nil
	}

	return newParsingErrf(name, d.Line, "invalid number of arguments in %q directive", d.Directive)
}

type Ignore []Command

func (i Ignore) Ignore(name string, commandType int) bool {
	for _, id := range i {
		if id.Name == name && id.Flags&commandType != 0 {
			return true
		}
	}

	return false
}

type Parser struct {
	Modules []Module
	Open    func(path string) (io.ReadCloser, error)
	Glob    func(path string) ([]string, error)
	// IgnoreCommands is a list of commands to skip parsing and omit from the result. If a skipped command is a block
	// directive then the contents of the directive are skipped.
	IgnoreCommands Ignore
	// ParseComments enables the parsing of comments. If true comments will be included in the result as # directives.
	ParseComments bool
	// ParseIncludes enables the parsing of configuration files included with the "include" directive. Glob will be
	// used to resolve any glob patterns and Open is used to open the included files.
	ParseIncludes bool
}

type Pass struct {
	CommandType int
	Directives  []*Directive
	Filename    string
	ConfigDir   string
	parser      *Parser
	Scanner     *Scanner
	includes    []Include
	included    map[string]int // keys files from being parsed twice
}

// next creates a shallow copy of the pass without directives to be given to the next context.
func (p Pass) next() Pass {
	return Pass{
		CommandType: p.CommandType,
		ConfigDir:   p.ConfigDir,
		Filename:    p.Filename,
		included:    p.included,
		includes:    p.includes,
		parser:      p.parser,
		Scanner:     p.Scanner,
	}
}

type File struct {
	Name       string
	Directives []*Directive
}

type Include struct {
	CommandType int // Block context
	File        File
	Filename    string // Filename is the name of the file that has the include directive.
	Line        int    // Line is the line number of the include directive.
}

type Package struct {
	Name  string
	Files map[string]File
}

func (p *Parser) Parse(basename string, s *Scanner) (Package, error) {
	pkg := Package{
		Name:  basename,
		Files: make(map[string]File),
	}

	pass := &Pass{
		ConfigDir: filepath.Dir(basename),
		parser:    p,
		Scanner:   s,
		included:  map[string]int{basename: 0},
		includes: []Include{
			{
				CommandType: ConfMainConf, File: File{Name: basename},
			},
		},
	}

	for i := 0; i < len(pass.includes); i++ {
		name := pass.includes[i].File.Name
		// TODO: Already parsed, but do we need to traverse it again to make sure it's valid in the current context?
		// if _, found := pkg.Files[name]; found {
		// 	continue
		// }

		// Setting the block context based on the current file. If the file was included this records the context in
		// which it was included.
		if pass.includes[i].CommandType == 0 {
			return pkg, fmt.Errorf("unknown context for included file %s", name)
		}

		pass.CommandType = pass.includes[i].CommandType
		pass.Filename = name

		// POC: wrapped in a function to defer closing opened files
		err := func() error {
			if pass.Scanner == nil {
				rc, err := p.open(name)
				if err != nil {
					return newParsingErrf(pass.includes[i].Filename, pass.includes[i].Line, "%w", err)
				}

				defer rc.Close()
				pass.Scanner = NewScanner(rc)
			}

			for {
				err := p.parse(pass)
				if err == io.EOF {
					pass.includes[i].File.Directives = pass.Directives
					pass.Directives = nil
					pass.Scanner = nil
					pkg.Files[name] = pass.includes[i].File
					break
				}

				if err != nil {
					return err
				}
			}

			return nil
		}()

		if err != nil {
			return pkg, err
		}
	}

	return pkg, nil
}

func (p *Parser) parse(pass *Pass) error {
	for {
		// TODO: I'm not a fan of labels and would rather refactor this out, but it works. This method is too long and
		// should probably be split up.
	outer:
		tok, err := pass.Scanner.Scan()
		if err != nil {
			return err
		}

		// we are parsing a block, so stop if it's closing
		if tok.Text == "}" && !tok.IsQuoted {
			return nil
		}

		d := Directive{
			Directive: tok.Text,
			Line:      tok.Line,
		}

		if strings.HasPrefix(tok.Text, "#") && !tok.IsQuoted {
			if p.ParseComments {
				d.Directive = "#"
				c := tok.Text[1:]
				d.Comment = &c
				pass.Directives = append(pass.Directives, &d)
			}

			continue
		}

		if tok, err = pass.Scanner.Scan(); err != nil {
			return err
		}

		var commentsInArgs []string

		for tok.IsQuoted || !isSpecialChar(tok.Text) {
			switch {
			case !strings.HasPrefix(tok.Text, "#") || tok.IsQuoted:
				d.Args = append(d.Args, tok.Text)
			case p.ParseComments:
				commentsInArgs = append(commentsInArgs, tok.Text[1:])
			}

			if tok, err = pass.Scanner.Scan(); err != nil {
				return err
			}
		}

		found := false
		for _, m := range p.Modules {
			c, ok := m.Command(d.Directive)
			if !ok {
				continue
			}

			found = true

			if c.Flags&pass.CommandType == 0 {
				// Wrong location
				continue
			}

			// TODO: This seems to continue parsing and ending up in the wrong context.
			if p.IgnoreCommands != nil && p.IgnoreCommands.Ignore(c.Name, pass.CommandType) {
				// TODO: What if it is a block but has no run? Should that be allowed or not legal?
				if c.Run == nil {
					goto outer
				}

				nextPass := pass.next()
				_ = c.Run(&d, &nextPass, func(pass *Pass) error {
					if !c.IsBlock() {
						return nil
					}

					return p.parse(&nextPass)
				})

				goto outer
			}

			// Check if it's not a block and terminated by ;
			if !c.IsBlock() && tok.Text != ";" {
				return newParsingErrf(pass.Filename, d.Line, `directive %q is not terminated by ";"`, d.Directive)
			}

			// Check if it's a block and is missing opening {
			if c.IsBlock() && tok.Text != "{" {
				return newParsingErrf(pass.Filename, d.Line, `directive %q has no opening "{"`, d.Directive)
			}

			if err := c.ValidateArgs(pass.Filename, d); err != nil {
				return err
			}

			// TODO: What if it is a block but has no run? Should that be allowed or not legal?
			if c.Run == nil {
				pass.Directives = append(pass.Directives, &d)
				goto outer
			}

			nextPass := pass.next()
			err := c.Run(&d, &nextPass, func(pass *Pass) error {
				if !c.IsBlock() {
					return nil
				}
				return p.parse(pass)
			})

			if err != nil {
				return err
			}

			if c.IsBlock() {
				d.Block = nextPass.Directives
			}

			pass.Directives = append(pass.Directives, &d)

			for _, c := range commentsInArgs {
				c := c
				pass.Directives = append(pass.Directives, &Directive{
					Directive: "#",
					Line:      d.Line,
					Comment:   &c,
				})
			}

			pass.included = nextPass.included
			pass.includes = nextPass.includes

			goto outer
		}

		if found {
			// TODO: Ignore "unknown directives" option.
			return newParsingErrf(pass.Filename, d.Line, "%q is not allowed here", d.Directive)
		}

		return newParsingErrf(pass.Filename, d.Line, "unknown directive %q", d.Directive)
	}
}

func (p *Parser) open(name string) (io.ReadCloser, error) {
	if p.Open != nil {
		return p.Open(name)
	}

	return os.Open(name)
}

func (p *Parser) glob(pattern string) ([]string, error) {
	if p.Glob != nil {
		return p.Glob(pattern)
	}

	return filepath.Glob(pattern)
}
