package crossplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxevent"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttp"
	"github.com/nginxinc/nginx-go-crossplane/module/nginxhttprewrite"
	"github.com/stretchr/testify/require"
)

func pStr(s string) *string { return &s }

func getTestConfigPath(parts ...string) string {
	return filepath.Join("testdata", "configs", filepath.Join(parts...))
}

type expectError struct {
	Filename string
	Line     int
}

type testParserCase struct {
	Want    crossplane.Package
	WantErr *expectError
}

func TestParserWithComments(t *testing.T) {
	t.Parallel()

	testParser(t, "with-comments", testParserCase{
		Want: crossplane.Package{
			Name: getTestConfigPath("with-comments", "nginx.conf"),
			Files: map[string]crossplane.File{
				getTestConfigPath("with-comments", "nginx.conf"): {
					Name: getTestConfigPath("with-comments", "nginx.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "events",
							Line:      1,
							Block: crossplane.Directives{
								{
									Directive: "worker_connections",
									Args:      []string{"1024"},
									Line:      2,
								},
							},
						},
						{
							Directive: "#",
							Line:      4,
							Comment:   pStr("comment"),
						},
						{
							Directive: "http",
							Line:      5,
							Block: crossplane.Directives{
								{
									Directive: "server",
									Line:      6,
									Block: crossplane.Directives{
										{
											Directive: "listen",
											Args:      []string{"127.0.0.1:8080"},
											Line:      7,
										},
										{
											Directive: "#",
											Line:      7,
											Comment:   pStr("listen"),
										},
										{
											Directive: "server_name",
											Args:      []string{"default_server"},
											Line:      8,
										},
										{
											Directive: "location",
											Args:      []string{"/"},
											Line:      9,
											Block: crossplane.Directives{
												{
													Directive: "#",
													Line:      9,
													Comment:   pStr("# this is brace"),
												},
												{
													Directive: "#",
													Line:      10,
													Comment:   pStr(" location /"),
												},
												{
													Directive: "return",
													Args:      []string{"200", "foo bar baz"},
													Line:      11,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
}

func TestParserIncludesGlobbed(t *testing.T) {
	t.Parallel()

	testParser(t, "includes-globbed", testParserCase{
		Want: crossplane.Package{
			Name: getTestConfigPath("includes-globbed", "nginx.conf"),
			Files: map[string]crossplane.File{
				getTestConfigPath("includes-globbed", "nginx.conf"): {
					Name: getTestConfigPath("includes-globbed", "nginx.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "events",
							Line:      1,
						},
						{
							Directive: "include",
							Args:      []string{"http.conf"},
							Line:      2,
							Includes:  []int{1},
						},
					},
				},
				getTestConfigPath("includes-globbed", "http.conf"): {
					Name: getTestConfigPath("includes-globbed", "http.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "http",
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "include",
									Args:      []string{"servers/*.conf"},
									Line:      2,
									Includes:  []int{2, 3},
								},
							},
						},
					},
				},
				getTestConfigPath("includes-globbed", "servers", "server1.conf"): {
					Name: getTestConfigPath("includes-globbed", "servers", "server1.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "server",
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "listen",
									Args:      []string{"8080"},
									Line:      2,
								},
								{
									Directive: "include",
									Args:      []string{"locations/*.conf"},
									Line:      3,
									Includes:  []int{4, 5},
								},
							},
						},
					},
				},
				getTestConfigPath("includes-globbed", "servers", "server2.conf"): {
					Name: getTestConfigPath("includes-globbed", "servers", "server2.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "server",
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "listen",
									Args:      []string{"8081"},
									Line:      2,
								},
								{
									Directive: "include",
									Args:      []string{"locations/*.conf"},
									Line:      3,
									Includes:  []int{4, 5},
								},
							},
						},
					},
				},
				getTestConfigPath("includes-globbed", "locations", "location1.conf"): {
					Name: getTestConfigPath("includes-globbed", "locations", "location1.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "location",
							Args:      []string{"/foo"},
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "return",
									Args:      []string{"200", "foo"},
									Line:      2,
								},
							},
						},
					},
				},
				getTestConfigPath("includes-globbed", "locations", "location2.conf"): {
					Name: getTestConfigPath("includes-globbed", "locations", "location2.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "location",
							Args:      []string{"/bar"},
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "return",
									Args:      []string{"200", "bar"},
									Line:      2,
								},
							},
						},
					},
				},
			},
		},
	})
}

func TestIncludesRegularFailed(t *testing.T) {
	t.Parallel()

	testParser(t, "includes-regular", testParserCase{
		WantErr: &expectError{
			Filename: getTestConfigPath("includes-regular", "conf.d", "server.conf"),
			Line:     5,
		},
		Want: crossplane.Package{
			Name: getTestConfigPath("includes-regular", "nginx.conf"),
			Files: map[string]crossplane.File{
				getTestConfigPath("includes-regular", "nginx.conf"): {
					Name: getTestConfigPath("includes-regular", "nginx.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "events",
							Line:      1,
						},
						{
							Directive: "http",
							Line:      2,
							Block: []*crossplane.Directive{
								{
									Directive: "include",
									Args:      []string{"conf.d/server.conf"},
									Line:      3,
									Includes:  []int{1},
								},
							},
						},
					},
				},
				getTestConfigPath("includes-regular", "conf.d", "server.conf"): {
					Name: getTestConfigPath("includes-regular", "conf.d", "server.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "server",
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "listen",
									Args:      []string{"127.0.0.1:8080"},
									Line:      2,
								},
								{
									Directive: "server_name",
									Args:      []string{"default_server"},
									Line:      3,
								},
								{
									Directive: "include",
									Args:      []string{"foo.conf"},
									Line:      4,
									Includes:  []int{2},
								},
								{
									Directive: "include",
									Args:      []string{"bar.conf"},
									Line:      5,
								},
							},
						},
					},
				},
				getTestConfigPath("includes-regular", "foo.conf"): {
					Name: getTestConfigPath("includes-regular", "foo.conf"),
					Directives: []*crossplane.Directive{
						{
							Directive: "location",
							Args:      []string{"/foo"},
							Line:      1,
							Block: []*crossplane.Directive{
								{
									Directive: "return",
									Args:      []string{"200", "foo"},
									Line:      2,
								},
							},
						},
					},
				},
			},
		},
	})
}

func testParser(t *testing.T, name string, tc testParserCase) {
	f, err := os.Open(getTestConfigPath(name, "nginx.conf"))
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	p := &crossplane.Parser{
		Modules: []crossplane.Module{
			crossplane.CoreModule, // TODO: hidden default
			nginxevent.Module,     // TODO: autoinclude core modules in the set?
			nginxhttp.Module,
			nginxhttprewrite.Module,
		},
		ParseComments: true,
	}

	pkg, err := p.Parse(f.Name(), crossplane.NewScanner(f))
	if err != nil {
		if tc.WantErr == nil {
			t.Fatalf("unexpected error: %v", err)
		}

		gotLine, _ := crossplane.LineNumber(err)
		gotFilename, _ := crossplane.Filename(err)

		require.Equal(t, tc.WantErr.Line, gotLine)
		require.Equal(t, tc.WantErr.Filename, gotFilename)
		return
	}

	if err == nil && tc.WantErr != nil {
		t.Fatalf("expected error in %s:%d", tc.WantErr.Filename, tc.WantErr.Line)
	}

	require.Equal(t, tc.Want, pkg)

	have, _ := json.MarshalIndent(pkg, "", "  ")
	t.Logf("parse: err=%v have=%s", err, have)
}

func benchmarkParser(b *testing.B, name string, expectErr bool) {
	f, err := os.Open(getTestConfigPath(name, "nginx.conf"))
	if err != nil {
		b.Fatal(err)
	}

	defer f.Close()

	p := &crossplane.Parser{
		Modules: []crossplane.Module{
			crossplane.CoreModule,
			nginxevent.Module, // TODO: autoinclude core modules in the set?
			nginxhttp.Module,
			nginxhttprewrite.Module,
		},
		ParseComments: true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := f.Seek(0, 0); err != nil {
			b.Fatal(err)
		}

		_, err := p.Parse(f.Name(), crossplane.NewScanner(f))
		if err != nil && !expectErr {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserWithComments(b *testing.B) {
	benchmarkParser(b, "with-comments", false)
}

func BenchmarkParserIncludesGlobbed(b *testing.B) {
	benchmarkParser(b, "includes-globbed", false)
}

func BenchmarkParserIncludesRegular(b *testing.B) {
	benchmarkParser(b, "includes-regular", true)
}
