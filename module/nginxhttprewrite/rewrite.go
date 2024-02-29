package nginxhttprewrite

import (
	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttp"
)

var returnCommand = crossplane.Command{
	Name:  "return",
	Flags: nginxhttp.HTTPSrvConf | nginxhttp.HTTPSifConf | nginxhttp.HTTPLocConf | nginxhttp.HTTPLifConf | crossplane.ConfTake12,
}

var Module = crossplane.ModuleFunc(func(name string) (crossplane.Command, bool) {
	switch name {
	case "return":
		return returnCommand, true
	}

	return crossplane.Command{}, false
})
