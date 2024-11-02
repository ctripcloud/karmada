package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGenOneLabelEqualIndexerFunc(t *testing.T) {
	type args struct {
		key string
		obj client.Object
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "cache hit",
			args: args{
				key: "a",
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"a": "a",
						},
					},
				},
			},
			want: []string{"a"},
		},
		{
			name: "cache missed",
			args: args{
				key: "a",
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"b": "b",
						},
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := GenOneLabelEqualIndexerFunc(tt.args.key)
			assert.NotNil(t, fn)
			assert.Equalf(t, tt.want, fn(tt.args.obj), "GenOneLabelEqualIndexerFunc(%v)", tt.args.key)
		})
	}
}
