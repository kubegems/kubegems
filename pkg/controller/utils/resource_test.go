package utils

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestSubResource(t *testing.T) {
	type args struct {
		oldres corev1.ResourceList
		newres corev1.ResourceList
	}
	tests := []struct {
		name string
		args args
		want corev1.ResourceList
	}{
		{
			name: "normal1",
			args: args{oldres: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewQuantity(1, resource.DecimalSI),
			}, newres: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewQuantity(1, resource.DecimalSI),
			}},
			want: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewQuantity(0, resource.DecimalSI),
			},
		},
		{
			name: "normal2",
			args: args{oldres: corev1.ResourceList{
				corev1.ResourceMemory: *resource.NewQuantity(1, resource.BinarySI),
				corev1.ResourceCPU:    *resource.NewQuantity(10, resource.BinarySI),
			}, newres: corev1.ResourceList{
				corev1.ResourceMemory: *resource.NewQuantity(10, resource.BinarySI),
				corev1.ResourceCPU:    *resource.NewQuantity(9, resource.BinarySI),
			}},
			want: corev1.ResourceList{
				corev1.ResourceMemory: *resource.NewQuantity(9, resource.BinarySI),
				corev1.ResourceCPU:    *resource.NewQuantity(-1, resource.BinarySI),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubResource(tt.args.oldres, tt.args.newres); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceEnough(t *testing.T) {
	type args struct {
		total corev1.ResourceList
		used  corev1.ResourceList
		need  corev1.ResourceList
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		{
			name: "test1",
			args: args{
				total: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(1, resource.BinarySI),
					corev1.ResourceCPU:    *resource.NewQuantity(10, resource.BinarySI),
				},
				used: corev1.ResourceList{},
				need: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(1, resource.BinarySI),
					corev1.ResourceCPU:    *resource.NewQuantity(10, resource.BinarySI),
				},
			},
			want:  true,
			want1: []string{},
		},
		{
			name: "error",
			args: args{
				total: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(1, resource.BinarySI),
					corev1.ResourceCPU:    *resource.NewQuantity(10, resource.BinarySI),
				},
				used: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(1, resource.BinarySI),
					corev1.ResourceCPU:    *resource.NewQuantity(10, resource.BinarySI),
				},
				need: corev1.ResourceList{
					corev1.ResourceMemory: *resource.NewQuantity(1, resource.BinarySI),
					corev1.ResourceCPU:    *resource.NewQuantity(10, resource.BinarySI),
				},
			},
			want:  false,
			want1: []string{"memory left 0 but need 1", "cpu left 0 but need 10"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ResourceEnough(tt.args.total, tt.args.used, tt.args.need)
			if got != tt.want {
				t.Errorf("ResourceEnough() got = %v, want %v", got, tt.want)
			}
			sort.Slice(tt.want1, func(i, j int) bool { return strings.Compare(tt.want1[i], tt.want1[j]) == 1 })
			sort.Slice(got1, func(i, j int) bool { return strings.Compare(got1[i], got1[j]) == 1 })
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ResourceEnough() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
