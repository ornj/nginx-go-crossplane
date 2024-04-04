package nginxhttp

import (
	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

var HTTP = crossplane.Command{
	Name:  "http",
	Flags: crossplane.ConfMainConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		// TODO: Check for duplicate
		p.CommandType = HTTPMainConf
		return next(p)
	},
}

var Listen = crossplane.Command{
	Name:  "listen",
	Flags: HTTPSrvConf | crossplane.Conf1More,
}

var Location = crossplane.Command{
	Name:  "location",
	Flags: HTTPSrvConf | HTTPLocConf | crossplane.ConfBlock | crossplane.ConfTake1,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		p.CommandType = HTTPLocConf
		return next(p)
	},
}

var Server = crossplane.Command{
	Name:  "server",
	Flags: HTTPMainConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		p.CommandType = HTTPSrvConf
		return next(p)
	},
}

var ServerName = crossplane.Command{
	Name:  "server_name",
	Flags: HTTPSrvConf | crossplane.Conf1More,
}

var Module = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	switch name {
	case "http":
		return HTTP, true
	case "listen":
		return Listen, true
	case "location":
		return Location, true
	case "server":
		return Server, true
	case "server_name":
		return ServerName, true
	}

	return crossplane.Command{}, false
})
