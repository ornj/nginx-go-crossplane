package nginxhttp_test

import (
	"testing"

	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttp"
	"github.com/stretchr/testify/assert"
)

func TestFlags(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0x02000000, nginxhttp.HTTPMainConf)
	assert.Equal(t, 0x04000000, nginxhttp.HTTPSrvConf)
	assert.Equal(t, 0x08000000, nginxhttp.HTTPLocConf)
	assert.Equal(t, 0x10000000, nginxhttp.HTTPUpsConf)
	assert.Equal(t, 0x20000000, nginxhttp.HTTPSifConf)
	assert.Equal(t, 0x40000000, nginxhttp.HTTPLifConf)
	assert.Equal(t, 0x80000000, nginxhttp.HTTPLmtConf)
}
