package config_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/config"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "docs.example.com", "docs.example.com"},
		{"uppercase", "Docs.Example.COM", "docs.example.com"},
		{"port", "docs.example.com:8080", "docs.example.com"},
		{"fully qualified", "docs.example.com.", "docs.example.com"},
		{"fully qualified with port", "docs.example.com.:8080", "docs.example.com"},
		{"whitespace", "  docs.example.com  ", "docs.example.com"},
		{"ipv6", "[::1]", "[::1]"},
		{"ipv6 with port", "[::1]:8080", "[::1]"},
		{"empty", "", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, config.NormalizeHost(test.in))
		})
	}
}

func TestModeValid(t *testing.T) {
	for _, mode := range config.Modes() {
		assert.True(t, mode.Valid(), mode.String())
	}
	assert.False(t, config.Mode("blog").Valid())
	assert.False(t, config.Mode("").Valid())
}

func TestDecode(t *testing.T) {
	cfg, err := config.Decode([]byte(`
addr: :9000
trust_forwarded_host: true
vhost:
  - domain: docs.example.com
    path: ./content/docs
    mode: docs
  - domain: tour.example.com
    path: /srv/tour
    mode: tour
`))
	require.NoError(t, err)
	assert.Equal(t, ":9000", cfg.Addr)
	assert.True(t, cfg.TrustForwardedHost)
	require.Len(t, cfg.VHosts, 2)
	assert.Equal(t, "docs.example.com", cfg.VHosts[0].Domain)
	assert.Equal(t, "./content/docs", cfg.VHosts[0].Path)
	assert.Equal(t, config.ModeDocs, cfg.VHosts[0].Mode)
	assert.Equal(t, config.ModeTour, cfg.VHosts[1].Mode)
}

func TestDecode_Invalid(t *testing.T) {
	_, err := config.Decode([]byte("vhost: [oh dear"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config")
}

func TestResolve(t *testing.T) {
	cfg := &config.Config{
		VHosts: []config.VHost{
			{Domain: "Docs.Example.COM", Path: "content/docs", Mode: config.ModeDocs},
			{Domain: "abs.example.com", Path: "/srv/tour", Mode: config.ModeTour},
		},
	}
	cfg.Resolve("/app")

	assert.Equal(t, "docs.example.com", cfg.VHosts[0].Domain)
	assert.Equal(t, "/app/content/docs", cfg.VHosts[0].Path)
	assert.Equal(t, "/srv/tour", cfg.VHosts[1].Path, "absolute paths are left alone")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		vhosts []config.VHost
		errmsg string
	}{
		{
			name:   "no entries",
			vhosts: nil,
			errmsg: "no vhost entries configured",
		},
		{
			name:   "missing domain",
			vhosts: []config.VHost{{Path: "/site", Mode: config.ModeDocs}},
			errmsg: "domain is required",
		},
		{
			name: "duplicate domain",
			vhosts: []config.VHost{
				{Domain: "a.example.com", Path: "/a", Mode: config.ModeDocs},
				{Domain: "a.example.com", Path: "/b", Mode: config.ModeTour},
			},
			errmsg: "already declared by vhost[0]",
		},
		{
			name:   "missing path",
			vhosts: []config.VHost{{Domain: "a.example.com", Mode: config.ModeDocs}},
			errmsg: "path is required",
		},
		{
			name:   "unknown mode",
			vhosts: []config.VHost{{Domain: "a.example.com", Path: "/a", Mode: "blog"}},
			errmsg: `unknown mode "blog", want one of docs, tour, serve`,
		},
		{
			name:   "empty mode",
			vhosts: []config.VHost{{Domain: "a.example.com", Path: "/a", Mode: ""}},
			errmsg: "unknown mode",
		},
		{
			name:   "valid",
			vhosts: []config.VHost{{Domain: "a.example.com", Path: "/a", Mode: config.ModeServe}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{VHosts: test.vhosts}
			err := cfg.Validate()
			if test.errmsg == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errmsg)
		})
	}
}

func TestFindFS(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		_, err := config.FindFS(fstest.MapFS{"other.yml": &fstest.MapFile{}})
		require.ErrorIs(t, err, config.ErrNotFound)
	})

	t.Run("yml", func(t *testing.T) {
		name, err := config.FindFS(fstest.MapFS{"vuego-cli.yml": &fstest.MapFile{}})
		require.NoError(t, err)
		assert.Equal(t, "vuego-cli.yml", name)
	})

	t.Run("yaml", func(t *testing.T) {
		name, err := config.FindFS(fstest.MapFS{"vuego-cli.yaml": &fstest.MapFile{}})
		require.NoError(t, err)
		assert.Equal(t, "vuego-cli.yaml", name)
	})

	t.Run("yml wins over yaml", func(t *testing.T) {
		name, err := config.FindFS(fstest.MapFS{
			"vuego-cli.yaml": &fstest.MapFile{},
			"vuego-cli.yml":  &fstest.MapFile{},
		})
		require.NoError(t, err)
		assert.Equal(t, "vuego-cli.yml", name)
	})

	t.Run("directory is not a config", func(t *testing.T) {
		_, err := config.FindFS(fstest.MapFS{
			"vuego-cli.yml/keep": &fstest.MapFile{},
		})
		require.ErrorIs(t, err, config.ErrNotFound)
	})
}

func TestLoadFS(t *testing.T) {
	fsys := fstest.MapFS{
		"vuego-cli.yml": &fstest.MapFile{Data: []byte(`
vhost:
  - domain: Docs.Example.com
    path: ./docs
    mode: docs
  - domain: tour.example.com
    path: /srv/tour
    mode: tour
`)},
	}

	cfg, err := config.LoadFS(fsys, "vuego-cli.yml", "/app")
	require.NoError(t, err)
	require.Len(t, cfg.VHosts, 2)

	assert.Equal(t, "docs.example.com", cfg.VHosts[0].Domain, "domains are normalized")
	assert.Equal(t, "/app/docs", cfg.VHosts[0].Path, "relative paths anchor at the config file")
	assert.Equal(t, "/srv/tour", cfg.VHosts[1].Path)
}

func TestLoadFS_Missing(t *testing.T) {
	_, err := config.LoadFS(fstest.MapFS{}, "vuego-cli.yml", "/app")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config")
}

func TestLoadFS_ValidationFailure(t *testing.T) {
	fsys := fstest.MapFS{
		"vuego-cli.yml": &fstest.MapFile{Data: []byte("vhost: []")},
	}

	_, err := config.LoadFS(fsys, "vuego-cli.yml", "/app")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no vhost entries configured")
}

func TestLoad_NamesTheFile(t *testing.T) {
	// Load wraps LoadFS with the path it was given, so a failure points at
	// the file rather than at a bare parse error.
	_, err := config.Load("testdata/absent/vuego-cli.yml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "testdata/absent/vuego-cli.yml")
	assert.Contains(t, err.Error(), "reading config")
}
