package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/goharbor/harbor/src/lib/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"kubegems.io/kubegems/pkg/model/store/types"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type Models struct {
	Collection *mongo.Collection
}

type ModelListOptions struct {
	CommonListOptions
	Source    string   `json:"source"`
	Tags      []string `json:"tags,omitempty"`
	Framework string   `json:"framework,omitempty"`
}

func (o *ModelListOptions) ToConditionAndFindOptions() (bson.D, *options.FindOptions) {
	cond := bson.D{}
	if o.Source != "" {
		cond = append(cond, bson.E{Key: "source", Value: o.Source})
	}
	if o.Search != "" {
		cond = append(cond, bson.E{Key: "name", Value: bson.M{"$regex": o.Search}})
	}
	if o.Tags != nil {
		cond = append(cond, bson.E{Key: "tags", Value: bson.M{"$all": o.Tags}})
	}
	if o.Framework != "" {
		cond = append(cond, bson.E{Key: "framework", Value: o.Framework})
	}

	sort := bson.D{}
	for _, item := range strings.Split(o.Sort, ",") {
		if item == "" {
			continue
		}
		if item[0] == '-' {
			sort = append(sort, bson.E{Key: item[1:], Value: -1})
		} else {
			sort = append(sort, bson.E{Key: item, Value: 1})
		}
	}

	if o.Page <= 0 {
		o.Page = 1
	}
	if o.Size <= 0 {
		o.Size = 10
	}
	return cond, options.Find().SetSort(sort).SetLimit(o.Size).SetSkip((o.Page - 1) * o.Size)
}

func (m *Models) Get(ctx context.Context, registry string, name string) (types.Model, error) {
	cond := bson.D{
		{Key: "registry", Value: registry}, {Key: "name", Value: name},
	}
	ret := types.Model{}
	err := m.Collection.FindOne(ctx, cond).Decode(&ret)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ret, response.NewError(http.StatusNotFound, fmt.Sprintf("model %s not found", name))
		}
		return types.Model{}, err
	}
	return ret, nil
}

func (m *Models) Count(ctx context.Context, opts ModelListOptions) (int64, error) {
	cond, _ := opts.ToConditionAndFindOptions()
	return m.Collection.CountDocuments(ctx, cond)
}

func (m *Models) List(ctx context.Context, opts ModelListOptions) ([]types.Model, error) {
	cond, options := opts.ToConditionAndFindOptions()
	cur, err := m.Collection.Find(ctx, cond, options)
	if err != nil {
		return nil, err
	}
	result := []types.Model{}
	err = cur.All(ctx, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *Models) Delete(ctx context.Context, registry string, name string) error {
	cond := bson.D{
		{Key: "registry", Value: registry}, {Key: "name", Value: name},
	}
	_, err := m.Collection.DeleteOne(ctx, cond)
	return err
}

type Selectors struct {
	Tags      []string `json:"tags"`
	Libraries []string `json:"libraries"`
	Licenses  []string `json:"licenses"`
}

func (m *Models) ListSelectors(ctx context.Context, registry string, listopts ModelListOptions) (*Selectors, error) {
	selectors := &Selectors{
		Tags:      []string{},
		Libraries: []string{},
		Licenses:  []string{},
	}

	cond, _ := listopts.ToConditionAndFindOptions()
	distincttags, _ := m.Collection.Distinct(ctx, "tags", cond)
	distinctlibraries, _ := m.Collection.Distinct(ctx, "library", cond)
	distinctlicenses, _ := m.Collection.Distinct(ctx, "license", cond)

	if tags, ok := any(distincttags).([]string); ok {
		selectors.Tags = tags
	}
	if libraries, ok := any(distinctlibraries).([]string); ok {
		selectors.Libraries = libraries
	}
	if licenses, ok := any(distinctlicenses).([]string); ok {
		selectors.Licenses = licenses
	}
	return selectors, nil
}

func (m *Models) AddToWebService(rg *route.Group) {
	list := func(req *restful.Request, resp *restful.Response) {
		listOptions := ModelListOptions{
			CommonListOptions: ParseCommonListOptions(req),
			Tags:              strings.Split(req.QueryParameter("tags"), ","),
			Framework:         req.QueryParameter("framework"),
			Source:            req.QueryParameter("source"),
		}
		list, err := m.List(req.Request.Context(), listOptions)
		if err != nil {
			response.BadRequest(resp, err.Error())
			return
		}
		// ignore total count error
		total, _ := m.Count(req.Request.Context(), listOptions)
		response.OK(resp, response.Page{
			List:  list,
			Total: total,
			Page:  listOptions.Page,
			Size:  listOptions.Size,
		})
	}

	get := func(req *restful.Request, resp *restful.Response) {
		modelid := req.PathParameter("model")
		ret, err := m.Get(req.Request.Context(), req.PathParameter("source"), modelid)
		if err != nil {
			response.ErrorResponse(resp, err)
			return
		}
		response.OK(resp, ret)
	}

	listSelector := func(req *restful.Request, resp *restful.Response) {
		listOptions := ModelListOptions{
			CommonListOptions: ParseCommonListOptions(req),
			Tags:              strings.Split(req.QueryParameter("tags"), ","),
			Framework:         req.QueryParameter("framework"),
			Source:            req.QueryParameter("source"),
		}
		selectors, err := m.ListSelectors(req.Request.Context(), req.PathParameter("source"), listOptions)
		if err != nil {
			response.BadRequest(resp, err.Error())
			return
		}
		response.OK(resp, selectors)
	}

	rg.AddSubGroup(route.
		NewGroup("/sources/{source}").
		Parameters(route.PathParameter("source", "model source name")).
		Tag("models").
		AddRoutes(
			route.GET("/selectors").To(listSelector).ShortDesc("list selectors"),
			route.GET("/models/{model}").To(get).ShortDesc("get model"),
			route.GET("/models").To(list).ShortDesc("list models"),
		),
	)
}
