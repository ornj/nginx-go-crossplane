package crossplane

import (
	"path/filepath"
	"sort"
)

const (
	ConfNoArgs = 1 << iota
	ConfTake1
	ConfTake2
	ConfTake3
	ConfTake4
	ConfTake5
	ConfTake6
	ConfTake7
	ConfBlock
	ConfFlag
	ConfAny
	Conf1More
	Conf2More

	ConfMaxArgs = 8

	ConfTake12 = ConfTake1 | ConfTake2
	ConfTake13 = ConfTake1 | ConfTake3
	ConfTake23 = ConfTake2 | ConfTake3

	ConfTake123  = ConfTake1 | ConfTake2 | ConfTake3
	ConfTake1234 = ConfTake1 | ConfTake2 | ConfTake3 | ConfTake4

	ConfArgsNumber = 0x000000ff
	ConfDirectConf = 0x00010000
	ConfMainConf   = 0x01000000
	ConfAnyConf    = 0xFF000000
)

var ArgumentNumber = []int{
	ConfNoArgs,
	ConfTake1,
	ConfTake2,
	ConfTake3,
	ConfTake4,
	ConfTake5,
	ConfTake6,
	ConfTake7,
}

var include = Command{
	Name:  "include",
	Flags: ConfAnyConf | ConfTake1,
	Run: func(d *Directive, p *Pass, next func(*Pass) error) (err error) {
		pat := d.Args[0]
		if !filepath.IsAbs(pat) {
			pat = filepath.Join(p.ConfigDir, pat)
		}

		var names []string
		if hasMagic.MatchString(pat) {
			if names, err = p.parser.glob(pat); err != nil {
				return err
			}
			sort.Strings(names)
		} else {
			// Checking that the file can be opened and read
			// TODO: Who cares? Why not just open it and read it (or fail?) Why do this twice? This seems like an
			// unnecessary optimization for a failure case, unless we are also going to parse it now.
			// var f io.ReadCloser
			// if f, err = p.parser.open(pat); err != nil {
			// 	return err
			// }

			// _ = f.Close()

			names = append(names, pat)
		}

		for _, name := range names {
			// TODO: The original parser tries to avoid parsing the same include twice, but I think we may have to
			// when the "block context" of the duplicate include is different. It might not be valid to include the
			// directives in the subsequent contexts.
			if _, ok := p.included[name]; !ok {
				p.included[name] = len(p.included)
				p.includes = append(p.includes, Include{
					CommandType: p.CommandType,
					File:        File{Name: name},
					Filename:    p.Filename,
					Line:        d.Line,
				})
			}

			d.Includes = append(d.Includes, p.included[name]) // TODO: should this be names or indexes?
		}

		// TODO: NGINX would parse this immediately, but crossplane defers to later.

		return next(p)
	},
}

var CoreModule = ModuleFunc(func(name string) (Command, bool) {
	switch name {
	case "include":
		return include, true
	}
	return Command{}, false
})
