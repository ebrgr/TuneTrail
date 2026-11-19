package importer

import "context"

type untrustedImportKey struct{}

// WithUntrustedImport is set by authenticated management adapters, never by file content.
func WithUntrustedImport(ctx context.Context) context.Context {
	return context.WithValue(ctx, untrustedImportKey{}, true)
}

// untrustedImport distinguishes owner uploads from trusted operator/root import jobs.
func untrustedImport(ctx context.Context) bool {
	value, _ := ctx.Value(untrustedImportKey{}).(bool)
	return value
}
