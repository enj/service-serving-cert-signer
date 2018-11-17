package operator

import "k8s.io/apimachinery/pkg/apis/meta/v1"

type Filter interface {
	Add(obj v1.Object) bool
	Update(oldObj, newObj v1.Object) bool
	Delete(obj v1.Object) bool
}
