package openresty

import (
	"errors"
	"fmt"
	"io"
	"strings"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttp"
)

func isEOL(s string) bool   { return strings.HasSuffix(s, "\n") }
func isSpace(s string) bool { return len(strings.TrimSpace(s)) == 0 }

// LuaScanner implements crossplane.SubScanner by implementing a nieve Lua scanner. This is a proof of concept and will
// not parse the entire Lua grammar. What is implemented is likely to be incorrect to some degree.
type LuaScanner struct {
	ext            *crossplane.SubScanner
	tokenDepth     int
	tokenStartLine int
}

func (s *LuaScanner) Scan() (crossplane.Token, error) {
	var tok strings.Builder
	for {
		if !s.ext.Scan() {
			if tok.Len() > 0 {
				return crossplane.Token{Text: tok.String(), Line: s.tokenStartLine}, nil
			}

			if s.tokenDepth > 0 {
				return crossplane.Token{}, crossplane.NewScannerErrf(s.tokenStartLine, "unexpected end of file")
			}

			return crossplane.Token{}, io.EOF
		}

		next := s.ext.Text()

		switch {
		case isEOL(next):
			s.tokenStartLine++
			continue
		case next == "{":
			s.tokenDepth++
		case next == "}":
			s.tokenDepth--
			if s.tokenDepth < 0 {
				return crossplane.Token{}, crossplane.NewScannerErrf(s.tokenStartLine, `unexpected "}"`)
			}
		case isSpace(next):
			if tok.Len() == 0 {
				continue
			}
			return crossplane.Token{Text: tok.String(), Line: s.tokenStartLine}, nil
		default:
			tok.WriteString(next)
		}

		if s.tokenDepth == 0 {
			return crossplane.Token{}, crossplane.EndScanWith
		}
	}
}

// ContentByLuaBlock is a command for parsing content_by_lua_block directives. The body of this directive does not
// follow the usual NGINX grammar. When the directive is encountered this command will assume responsibility of
// advancing the scanner to tokenize and parse the Lua contents of the block. Since crossplane.Directive can not
// accommodate non-directive content in the Block field, the Lua contents of the directive are sent as setings to Args.
var ContentByLuaBlock = crossplane.Command{
	Name:  "content_by_lua_block",
	Flags: nginxhttp.HTTPLocConf | nginxhttp.HTTPLifConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(d *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		s, err := crossplane.NewSubScanner(p.Scanner)
		if err != nil {
			return err
		}

		ls := &LuaScanner{ext: s, tokenDepth: 1, tokenStartLine: d.Line}

		for {
			// TODO: can this be done outside the for loop to setup a subtype, or would that be more error prone?
			tok, err := p.Scanner.ScanWith(ls)

			if err != nil {
				if errors.Is(err, crossplane.EndScanWith) {
					return next(p)
				}
				return err
			}

			// TODO: I haven't spent more time to try and figure out a better way to handle lua content than what
			// python crossplane does. For now I'll just put the lua content into the args.
			switch tok.Text {
			default:
				if len(d.Args) == 0 {
					d.Args = append(d.Args, tok.Text)
				} else {
					d.Args[0] = fmt.Sprintf("%s %s", d.Args[0], tok.Text)
				}
			}
		}
	},
}

var LuaModule = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	if name == ContentByLuaBlock.Name {
		return ContentByLuaBlock, true
	}

	return crossplane.Command{}, false
})
