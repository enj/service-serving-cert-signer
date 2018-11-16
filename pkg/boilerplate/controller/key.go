package controller

import "k8s.io/apimachinery/pkg/apis/meta/v1"

type KeyFunc func(namespace, name string) (v1.Object, error)
type ProcessFunc func(v1.Object) error

type queueKey struct {
	namespace string
	name      string
}
