package nginxhttp

import (
	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

var http = crossplane.Command{
	Name:  "http",
	Flags: crossplane.ConfMainConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		// TODO: Check for duplicate
		p.CommandType = HTTPMainConf
		return next(p)
	},
}

var listen = crossplane.Command{
	Name:  "listen",
	Flags: HTTPSrvConf | crossplane.Conf1More,
}

var location = crossplane.Command{
	Name:  "location",
	Flags: HTTPSrvConf | HTTPLocConf | crossplane.ConfBlock | crossplane.ConfTake1,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		p.CommandType = HTTPLocConf
		return next(p)
	},
}

var server = crossplane.Command{
	Name:  "server",
	Flags: HTTPMainConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		p.CommandType = HTTPSrvConf
		return next(p)
	},
}

var serverName = crossplane.Command{
	Name:  "server_name",
	Flags: HTTPSrvConf | crossplane.Conf1More,
}

var Module = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	switch name {
	case "http":
		return http, true
	case "listen":
		return listen, true
	case "location":
		return location, true
	case "server":
		return server, true
	case "server_name":
		return serverName, true
	}

	return crossplane.Command{}, false
})
