package runtime

import "context"

type appContextKey struct{}

// WithApp stores the runtime app on the context.
func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appContextKey{}, app)
}

// FromContext returns the runtime app from the context.
func FromContext(ctx context.Context) *App {
	app, _ := ctx.Value(appContextKey{}).(*App)
	return app
}
