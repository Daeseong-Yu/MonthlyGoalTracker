package principal

import (
	"context"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

type userContextKey struct{}

func WithUser(ctx context.Context, user domain.User) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) (domain.User, bool) {
	if ctx == nil {
		return domain.User{}, false
	}

	user, ok := ctx.Value(userContextKey{}).(domain.User)
	if !ok || user.ID == 0 {
		return domain.User{}, false
	}

	return user, true
}
