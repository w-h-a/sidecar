package resources

import pbSidecar "github.com/w-h-a/pkg/proto/sidecar"

type MethodEvent struct {
	Method string
	Event  *pbSidecar.Event
}
