package host

// Option configures a Module.
type Option func(*Module)

// WithResolver replaces the filesystem resolver used to open vhost content.
func WithResolver(resolver Resolver) Option {
	return func(m *Module) {
		m.resolver = resolver
	}
}
