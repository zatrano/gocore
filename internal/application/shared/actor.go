package shared

import "context"

// ActorType, denetim kaydındaki aktör sınıfı.
const (
	ActorTypeUser      = "user"
	ActorTypeAnonymous = "anonymous"
	ActorTypeSystem    = "system"
	ActorTypeProvider  = "provider"
	ActorTypeScheduler = "scheduler"
)

// Source, isteğin geldiği yüzey.
const (
	SourceAPI       = "api"
	SourceWeb       = "web"
	SourceWebhook   = "webhook"
	SourceScheduler = "scheduler"
	SourceStartup   = "startup"
)

// ActorContext, istek/sistem aktörünün anlık görüntüsüdür.
type ActorContext struct {
	ActorID       string
	ActorType     string
	ActorEmail    string
	Source        string
	CorrelationID string
	IP            string
	UserAgent     string
}

type actorCtxKey struct{}

// WithActor, context'e aktör zarfını ekler.
func WithActor(ctx context.Context, a ActorContext) context.Context {
	if a.ActorType == "" {
		if a.ActorID == "" {
			a.ActorType = ActorTypeAnonymous
		} else {
			a.ActorType = ActorTypeUser
		}
	}
	return context.WithValue(ctx, actorCtxKey{}, a)
}

// ActorFromContext, context'teki aktör zarfını döner.
func ActorFromContext(ctx context.Context) ActorContext {
	if a, ok := ctx.Value(actorCtxKey{}).(ActorContext); ok {
		return a
	}
	return ActorContext{ActorType: ActorTypeAnonymous}
}
