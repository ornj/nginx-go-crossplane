package nginxhttprewrite

import (
	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttp"
)

var Return = crossplane.Command{
	Name:  "return",
	Flags: nginxhttp.HTTPSrvConf | nginxhttp.HTTPSifConf | nginxhttp.HTTPLocConf | nginxhttp.HTTPLifConf | crossplane.ConfTake12,
}

var Set = crossplane.Command{
	Name:  "set",
	Flags: nginxhttp.HTTPSrvConf | nginxhttp.HTTPLocConf | nginxhttp.HTTPLifConf | crossplane.ConfTake2,
}

var Module = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	switch name {
	case Return.Name:
		return Return, true
	case Set.Name:
		return Set, true
	}

	return crossplane.Command{}, false
})
