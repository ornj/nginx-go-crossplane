package nginxhttp

const (
	HTTPMainConf = 0x02000000 << iota
	HTTPSrvConf
	HTTPLocConf
	HTTPUpsConf
	HTTPSifConf
	HTTPLifConf
	HTTPLmtConf
)
