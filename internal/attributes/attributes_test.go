package attributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
		want  attribute.KeyValue
	}{
		{name: "string", key: "key1", value: "value1", want: attribute.Key("key1").String("value1")},
		{name: "pointer to string", key: "key2", value: ptr("value2"), want: attribute.Key("key2").String("value2")},
		{name: "bool", key: "key3", value: true, want: attribute.Key("key3").Bool(true)},
		{name: "pointer to bool", key: "key4", value: ptr(true), want: attribute.Key("key4").Bool(true)},
		{name: "int", key: "key5", value: 1, want: attribute.Key("key5").Int(1)},
		{name: "pointer to int", key: "key6", value: ptr(1), want: attribute.Key("key6").Int(1)},
		{name: "int64", key: "key7", value: int64(1), want: attribute.Key("key7").Int64(1)},
		{name: "pointer to int64", key: "key8", value: ptr(int64(1)), want: attribute.Key("key8").Int64(1)},
		{name: "float64", key: "key9", value: 1.0, want: attribute.Key("key9").Float64(1.0)},
		{name: "pointer to float64", key: "key10", value: ptr(1.0), want: attribute.Key("key10").Float64(1.0)},
		{
			name: "string slice", key: "key11", value: []string{"value1", "value2"},
			want: attribute.Key("key11").StringSlice([]string{"value1", "value2"}),
		},
		{name: "bool slice", key: "key12", value: []bool{true, false}, want: attribute.Key("key12").BoolSlice([]bool{true, false})},
		{name: "int slice", key: "key13", value: []int{1, 2}, want: attribute.Key("key13").IntSlice([]int{1, 2})},
		{name: "int64 slice", key: "key14", value: []int64{1, 2}, want: attribute.Key("key14").Int64Slice([]int64{1, 2})},
		{
			name: "float64 slice", key: "key15", value: []float64{1.0, 2.0},
			want: attribute.Key("key15").Float64Slice([]float64{1.0, 2.0}),
		},
		{name: "stringer", key: "key16", value: stringer("value1"), want: attribute.Key("key16").String("value1")},
		{name: "default", key: "key17", value: struct{}{}, want: attribute.Key("key17").String("{}")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, New(tt.key, tt.value))
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

type stringer string

func (s stringer) String() string {
	return string(s)
}
