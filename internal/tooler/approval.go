package tooler

import "context"

type approvalKey struct{}

func WithApproval(ctx context.Context) context.Context {
	return context.WithValue(ctx, approvalKey{}, true)
}

// Approved fails closed: a missing stamp refuses, never an unapproved run.
func Approved(ctx context.Context) bool {
	yes, _ := ctx.Value(approvalKey{}).(bool)

	return yes
}
