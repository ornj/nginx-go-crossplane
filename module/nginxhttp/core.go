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

var ProxyPass = crossplane.Command{
	Name:  "proxy_pass",
	Flags: HTTPLocConf | HTTPLifConf | HTTPLmtConf | crossplane.ConfTake1,
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
	case HTTP.Name:
		return HTTP, true
	case Listen.Name:
		return Listen, true
	case Location.Name:
		return Location, true
	case ProxyPass.Name:
		return ProxyPass, true
	case Server.Name:
		return Server, true
	case ServerName.Name:
		return ServerName, true
	}

	return crossplane.Command{}, false
})
