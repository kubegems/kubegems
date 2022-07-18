package kube

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NoopClient struct{}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (NoopClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return nil
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (NoopClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

// Create saves the object obj in the Kubernetes cluster.
func (NoopClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return nil
}

// Delete deletes the given obj from Kubernetes cluster.
func (NoopClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (NoopClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return nil
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (NoopClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (NoopClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (c NoopClient) Status() client.StatusWriter {
	return c
}

// Scheme returns the scheme this client is using.
func (NoopClient) Scheme() *runtime.Scheme {
	return scheme.Scheme
}

// RESTMapper returns the rest this client is using.
func (NoopClient) RESTMapper() meta.RESTMapper {
	return nil
}
