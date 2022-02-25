package client

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/pkg/apis/application/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testdata = []client.Object{
	&v1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test2",
			Namespace: "test",
			Labels: map[string]string{
				"test": "test",
				"app":  "test",
			},
		},
		Spec: v1beta1.ApplicationSpec{
			Remark: "this is a temporary application",
			Kind:   "Deployment",
			Images: []string{"nginx:1.15"},
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	},
	&v1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: v1beta1.ApplicationSpec{
			Remark: "this is a temporary application",
			Kind:   "Deployment",
			Images: []string{"nginx:1.14"},
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	},
	&v1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test3",
			Namespace: "test",
		},
		Spec: v1beta1.ApplicationSpec{
			Remark: "this is a temporary application",
			Kind:   "Deployment",
			Images: []string{"nginx:1.14"},
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	},
}

func TestDefaultClient_Create(t *testing.T) {
	// IMPORTANT NOTE:
	// add:
	//
	//   "go.buildFlags": ["-tags=json1"],
	//
	// to your .vscode/settings.json before running this test
	//
	// add json_extract support on :memory: sqlite3
	// https://github.com/mattn/go-sqlite3/issues/410
	dial := sqlite.Open(":memory:")
	db, err := gorm.Open(dial, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db = db.Debug()

	cli := NewTypedClient(db)

	ctx := context.Background()

	obj := &v1beta1.Application{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	if err := cli.Migrate(ctx, obj); err != nil {
		t.Fatal(err)
		return
	}

	for _, obj := range testdata {
		if err := cli.Create(ctx, obj); err != nil {
			t.Errorf("DefaultClient.Create() error = %v", err)
		}
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		t.Errorf("DefaultClient.Get() error = %v", err)
	}
	list := &v1beta1.ApplicationList{}
	if err := cli.List(ctx, list,
		client.InNamespace("test"),
		client.MatchingLabels{
			"app":  "test",
			"test": "test",
		},
	); err != nil {
		t.Errorf("DefaultClient.List() error = %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("DefaultClient.List() error = %v", errors.New("invalid list"))
	}

	if err := cli.Delete(ctx, obj); err != nil {
		t.Errorf("DefaultClient.Delete() error = %v", err)
	}

	if err := cli.List(ctx, list); err != nil {
		t.Errorf("DefaultClient.List() error = %v", err)
	}
	if len(list.Items) != len(testdata)-1 {
		t.Errorf("DefaultClient.List() error = %v", errors.New("invalid list items"))
	}
	if err := cli.DeleteAllOf(ctx, &v1beta1.Application{},
		client.InNamespace("test"),
		&client.DeleteAllOfOptions{
			ListOptions: client.ListOptions{
				LabelSelector: labels.Everything(),
			},
		}); err != nil {
		t.Errorf("DefaultClient.DeleteAllOf() error = %v", err)
	}
	if err := cli.List(context.Background(), list); err != nil {
		t.Errorf("DefaultClient.List() error = %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("DefaultClient.List() error = %v", errors.New("invalid list items"))
	}

	t.Log(list)
}
