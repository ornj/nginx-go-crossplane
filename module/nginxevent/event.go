package nginxevent

import (
	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

const (
	EventConf = 0x02000000 << iota // TODO: Not unique, equals nginxhttp.HTTPMainConf
	_
)

var events = crossplane.Command{
	Name:  "events",
	Flags: crossplane.ConfMainConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		// TODO: Check for  duplicate
		p.CommandType = EventConf
		return next(p)
	},
}

var workerConnections = crossplane.Command{
	Name:  "worker_connections",
	Flags: EventConf | crossplane.ConfTake1,
}

var Module = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	switch name {
	case "events":
		return events, true
	case "worker_connections":
		return workerConnections, true
	}

	return crossplane.Command{}, false
})
