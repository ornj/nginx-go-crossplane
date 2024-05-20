package nginxevent

import (
	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

const (
	EventConf = 0x02000000 << iota // TODO: Not unique, equals nginxhttp.HTTPMainConf
	_
)

var Events = crossplane.Command{
	Name:  "events",
	Flags: crossplane.ConfMainConf | crossplane.ConfBlock | crossplane.ConfNoArgs,
	Run: func(_ *crossplane.Directive, p *crossplane.Pass, next func(*crossplane.Pass) error) error {
		// TODO: Check for  duplicate
		p.CommandType = EventConf
		return next(p)
	},
}

var WorkerConnections = crossplane.Command{
	Name:  "worker_connections",
	Flags: EventConf | crossplane.ConfTake1,
}

var Module = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	switch name {
	case Events.Name:
		return Events, true
	case WorkerConnections.Name:
		return WorkerConnections, true
	}

	return crossplane.Command{}, false
})
