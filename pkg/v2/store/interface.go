package store

import "context"

type StoreKind string

type EntityObject interface {
	Store() string
	Kind() string
}

type EntityObjectList interface {
	Store() string
	Kind() string
}

type ListOptions interface {
	Apply(*ListOption)
}

type CreteOptions interface {
	Apply(*CreateOption)
}

type RetrieveOptions interface {
	Apply(*RetrieveOption)
}

type UpdateOptions interface {
	Apply(*UpdateOption)
}

type DeleteOptions interface {
	Apply(*DeleteOption)
}

type StoreIface interface {
	List(context.Context, EntityObjectList, ...ListOptions) error
	Create(context.Context, EntityObject, ...CreateOption) error
	CreateMany(context.Context, EntityObjectList, ...CreateOption) error
	Retrieve(context.Context, EntityObject, ...RetrieveOptions) error
	Update(context.Context, EntityObject, ...UpdateOptions) error
	Delete(context.Context, EntityObject, DeleteOptions) error
}
