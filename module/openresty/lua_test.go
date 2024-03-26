package openresty_test

import (
	"io"
	"strings"
	"testing"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxevent"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttp"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttprewrite"
	"github.com/nginxinc/nginx-go-crossplane/module/openresty"
	"github.com/stretchr/testify/require"
)

func TestContentByLuaBlock(t *testing.T) {
	t.Parallel()

	p := &crossplane.Parser{
		ParseComments: true,
		Modules: []crossplane.Module{
			crossplane.CoreModule,
			nginxevent.Module,
			nginxhttp.Module,
			nginxhttprewrite.Module,
			openresty.LuaModule,
		},
	}

	r := io.NopCloser(strings.NewReader(`
		http {
			server {
				location / {
					content_by_lua_block {
						ngx.say('Hello, world!')
					}
					return 200, "OK";
				}

				listen 8080; # check that lua hasn't messed up context/depth and lineno
			}
		}
	`))

	got, err := p.Parse("nginx.conf", crossplane.NewScanner(r))
	if err != nil {
		t.Fatal(err)
	}

	want := crossplane.Package{
		Name: "nginx.conf",
		Files: map[string]crossplane.File{
			"nginx.conf": {
				Name: "nginx.conf",
				Directives: []*crossplane.Directive{
					{
						Directive: "http",
						Line:      2,
						Block: []*crossplane.Directive{
							{
								Directive: "server",
								Line:      3,
								Block: []*crossplane.Directive{
									{
										Directive: "location",
										Line:      4,
										Args:      []string{"/"},
										Block: []*crossplane.Directive{
											{
												Directive: "content_by_lua_block",
												Line:      5,
												Args:      []string{"ngx.say('Hello, world!')"},
											},
											{
												Directive: "return",
												Line:      8,
												Args:      []string{"200,", "OK"}, // TODO: check if this is correct
											},
										},
									},
									{
										Directive: "listen",
										Line:      11,
										Args:      []string{"8080"},
									},
									{
										Directive: "#",
										Line:      11,
										Comment:   pointer(" check that lua hasn't messed up context/depth and lineno"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	require.Equal(t, want, got)
}

func pointer[T any](v T) *T { return &v }
