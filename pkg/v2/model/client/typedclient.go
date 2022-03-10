package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	_ "kubegems.io/pkg/utils/kube" // register schema
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type DefaultClient struct {
	isStatus   bool
	db         *gorm.DB
	scheme     *runtime.Scheme
	restmapper apimeta.RESTMapper
}

func NewTypedClient(db *gorm.DB) *DefaultClient {
	return &DefaultClient{db: db, scheme: scheme.Scheme}
}

type Common struct {
	Name              string    `gorm:"primaryKey;type:varchar(60);not null"`
	Namespace         string    `gorm:"primaryKey;type:varchar(60);not null"`
	UID               string    `gorm:"uniqueIndex;type:varchar(36)"`
	CreationTimestamp time.Time `gorm:"not null"`
	DeletionTimestamp time.Time
	Labels            datatypes.JSON
	Annotations       datatypes.JSON
	OwnerReferences   datatypes.JSON
	Spec              datatypes.JSON
	Status            datatypes.JSON
}

func (c *Common) DecodeInto(obj client.Object) error {
	obj.SetName(c.Name)
	obj.SetNamespace(c.Namespace)
	obj.SetUID(types.UID(c.UID))
	obj.SetCreationTimestamp(metav1.Time{Time: c.CreationTimestamp})

	if !c.DeletionTimestamp.IsZero() {
		obj.SetDeletionTimestamp(&metav1.Time{Time: c.DeletionTimestamp})
	}

	labels := map[string]string{}
	json.Unmarshal(c.Labels, &labels)
	obj.SetLabels(labels)

	annotations := map[string]string{}
	json.Unmarshal(c.Annotations, &annotations)
	obj.SetAnnotations(annotations)

	references := []metav1.OwnerReference{}
	json.Unmarshal(c.OwnerReferences, &references)
	obj.SetOwnerReferences(references)

	json.Unmarshal(c.Spec, getFieldPointer(obj, "Spec"))
	json.Unmarshal(c.Status, getFieldPointer(obj, "Status"))
	return nil
}

func (c *Common) From(obj client.Object) error {
	c.Name = obj.GetName()
	c.Namespace = obj.GetNamespace()
	c.UID = string(obj.GetUID())
	c.OwnerReferences = toJsonRaw(obj.GetOwnerReferences())
	c.CreationTimestamp = obj.GetCreationTimestamp().Time
	if deletionTimestamp := obj.GetDeletionTimestamp(); deletionTimestamp != nil {
		c.DeletionTimestamp = deletionTimestamp.Time
	}
	c.Labels = toJsonRaw(obj.GetLabels())
	c.Annotations = toJsonRaw(obj.GetAnnotations())
	c.Spec = toJsonRaw(getFieldPointer(obj, "Spec"))
	c.Status = toJsonRaw(getFieldPointer(obj, "Status"))
	return nil
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (c *DefaultClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return c.onQuery(ctx, obj, func(db *gorm.DB, common *Common, gvk schema.GroupVersionKind) error {
		if err := db.Take(common).Error; err != nil {
			return err
		}
		if err := common.DecodeInto(obj); err != nil {
			return err
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		return nil
	})
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (c *DefaultClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, c.Scheme())
	if err != nil {
		return toStatusError(nil, err)
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	emptyitem, err := c.scheme.New(gvk)
	if err != nil {
		return toStatusError(nil, err)
	}

	options := client.ListOptions{}
	options.ApplyOptions(opts)

	// namespace filter
	db := c.db.Table(strings.ToLower(gvk.Kind)).Where(&Common{Namespace: options.Namespace})
	// limit
	if options.Limit != 0 {
		db = db.Limit(int(options.Limit))
	}
	// offset
	offset, _ := strconv.Atoi(options.Continue)
	if options.Continue != "" {
		db = db.Offset(offset)
	}
	// labelselector in query
	applyLabelSelector(db, options.LabelSelector)

	items := []Common{}
	if err := db.Find(&items).Error; err != nil {
		return toStatusError(nil, err)
	}
	runtimeObjs := make([]runtime.Object, 0, len(items))
	for _, item := range items {
		outObj, ok := emptyitem.DeepCopyObject().(client.Object)
		if !ok {
			return apierrors.NewBadRequest(fmt.Sprintf("%T is not a client.Object", emptyitem))
		}
		item.DecodeInto(outObj)
		// apply selector after query
		if options.LabelSelector != nil {
			if !options.LabelSelector.Matches(labels.Set(outObj.GetLabels())) {
				continue
			}
		}
		outObj.GetObjectKind().SetGroupVersionKind(gvk)
		runtimeObjs = append(runtimeObjs, outObj)
	}
	if limit := options.Limit; limit != 0 {
		continuekey := strconv.Itoa(offset + int(options.Limit))
		list.SetContinue(continuekey)
	}
	return apimeta.SetList(list, runtimeObjs)
}

// Create saves the object obj in the Kubernetes cluster.
func (c *DefaultClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	obj.SetUID(types.UID(uuid.NewString()))
	obj.SetCreationTimestamp(metav1.Now())
	return c.onQuery(ctx, obj, func(db *gorm.DB, common *Common, _ schema.GroupVersionKind) error {
		common.From(obj)
		return db.Create(common).Error
	})
}

// Delete deletes the given obj from Kubernetes cluster.
func (c *DefaultClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if obj.GetName() == "" {
		return apierrors.NewBadRequest("name is required")
	}
	return c.onQuery(ctx, obj, func(db *gorm.DB, common *Common, _ schema.GroupVersionKind) error {
		return db.Delete(common).Error
	})
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *DefaultClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if obj.GetUID() == "" {
		return apierrors.NewBadRequest("uid must be set")
	}
	if obj.GetName() == "" {
		return apierrors.NewBadRequest("name must be set")
	}
	return c.onQuery(ctx, obj, func(db *gorm.DB, common *Common, _ schema.GroupVersionKind) error {
		toupdate := &Common{
			// OwnerReferences: toJsonRaw(obj.GetOwnerReferences()), // can not change owner reference after creation
			Labels:      toJsonRaw(obj.GetLabels()),
			Annotations: toJsonRaw(obj.GetAnnotations()),
			Status:      toJsonRaw(getFieldPointer(obj, "Status")),
			Spec:        toJsonRaw(getFieldPointer(obj, "Spec")),
		}
		if !c.isStatus {
			toupdate.Status = toJsonRaw(getFieldPointer(obj, "Status"))
		}
		return db.Where(common).Updates(toupdate).Error
	})
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (c *DefaultClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	options := client.PatchOptions{}
	options.ApplyOptions(opts)

	patchcontent, err := patch.Data(obj)
	if err != nil {
		return err
	}
	// nolint: forcetypeassert
	exist := obj.DeepCopyObject().(client.Object)
	if err := c.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil {
		return err
	}
	rawjson, err := json.Marshal(exist)
	if err != nil {
		return err
	}
	var patchedjson []byte
	// https://github.com/kubernetes/kubernetes/blob/release-1.20/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L337
	switch patch.Type() {
	case types.JSONPatchType:
		jsonpatchpatch, err := jsonpatch.DecodePatch(patchcontent)
		if err != nil {
			return err
		}
		patchedjson, err = jsonpatchpatch.Apply(rawjson)
		if err != nil {
			return err
		}
	case types.MergePatchType:
		patchedjson, err = jsonpatch.MergePatch(rawjson, patchcontent)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported patch type %s", patch.Type())
	}
	if err := json.Unmarshal(patchedjson, obj); err != nil {
		return err
	}
	return c.Update(ctx, obj)
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (c *DefaultClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	options := client.DeleteAllOfOptions{}
	options.ApplyOptions(opts)

	return c.onQuery(ctx, obj, func(db *gorm.DB, common *Common, _ schema.GroupVersionKind) error {
		db = db.Where("namespace = ?", common.Namespace)
		applyLabelSelector(db, options.LabelSelector)
		return db.Delete(common).Error
	})
}

func (c *DefaultClient) Status() client.StatusWriter {
	cc := *c
	cc.isStatus = true
	return &cc
}

// Scheme returns the scheme this client is using.
func (c *DefaultClient) Scheme() *runtime.Scheme {
	return c.scheme
}

// RESTMapper returns the rest this client is using.
func (c *DefaultClient) RESTMapper() meta.RESTMapper {
	if c.restmapper == nil {
		c.restmapper = apimeta.NewDefaultRESTMapper(c.scheme.PreferredVersionAllGroups())
	}
	return c.restmapper
}

// Metadata create/update the table  in database
func (c *DefaultClient) Migrate(ctx context.Context, obj client.Object) error {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return toStatusError(obj, err)
	}
	tablename := strings.ToLower(gvk.Kind)
	return c.db.Table(tablename).AutoMigrate(&Common{})
}

func (c *DefaultClient) onQuery(ctx context.Context, obj client.Object, fun func(db *gorm.DB, common *Common, gvk schema.GroupVersionKind) error) error {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return toStatusError(obj, err)
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	common := &Common{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	db := c.db.Table(strings.ToLower(gvk.Kind))

	if err := fun(db, common, gvk); err != nil {
		return toStatusError(obj, err)
	}
	return nil
}

func toStatusError(obj client.Object, err error) *apierrors.StatusError {
	if err == nil {
		return nil
	}

	apie := &apierrors.StatusError{}
	if errors.As(err, &apie) {
		return apie
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apierrors.NewNotFound(schema.GroupResource{
			Group:    obj.GetObjectKind().GroupVersionKind().Group,
			Resource: strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind),
		}, err.Error())
	}

	return apierrors.NewInternalError(err)
}

func applyLabelSelector(db *gorm.DB, labelSelector labels.Selector) *gorm.DB {
	if labelSelector == nil {
		return db
	}
	requirements, selectable := labelSelector.Requirements()
	if !selectable {
		// select nothing
		return db.Where("1=0")
	}
	// using mysql JSON Query instead label selector on eqals query
	// mysql: https://dev.mysql.com/doc/refman/5.7/en/json-search-functions.html#operator_json-column-path
	// sqlite3: https://www.sqlite.org/json1.html
	for _, requirement := range requirements {
		key := requirement.Key()
		op := requirement.Operator()
		db = db.Where(fmt.Sprintf("json_extract(`labels`,'$.%s') %s ?", key, op), requirement.Values().List())
	}
	return db
}

func getFieldPointer(obj interface{}, field string) interface{} {
	v, err := conversion.EnforcePtr(obj)
	if err != nil {
		return nil
	}
	val := v.FieldByName(field)
	if !val.IsValid() {
		return nil
	}
	if val.CanAddr() {
		return val.Addr().Interface()
	}
	return val.Interface()
}

func toJsonRaw(data interface{}) datatypes.JSON {
	content, _ := json.Marshal(data)
	return content
}
