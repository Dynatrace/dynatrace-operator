package kubeobjects

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestKey(t *testing.T) {
	type args struct {
		object client.Object
	}
	tests := []struct {
		name string
		args args
		want client.ObjectKey
	}{
		{name: "handle nil value", args: args{object: nil}, want: client.ObjectKey{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, Key(tt.args.object), "Key(%v)", tt.args.object)
		})
	}
}
