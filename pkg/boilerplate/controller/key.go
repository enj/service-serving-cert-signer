package controller

type Key interface {
	GetNamespace() string
	GetName() string
}

type ProcessFunc func(Key) error

type QueueKey struct {
	Namespace string
	Name      string
}

func (o QueueKey) GetNamespace() string {
	return o.Namespace
}

func (o QueueKey) GetName() string {
	return o.Name
}
