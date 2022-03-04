package route

import (
	"net/http"
	"reflect"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
)

type Function = func(req *restful.Request, resp *restful.Response)

type Tree struct {
	Group           *Group
	RouteUpdateFunc func(r *Route) // can update route setting before build
}

func (t *Tree) AddToWebService(ws *restful.WebService) {
	t.addWebService(ws, "root", "", nil, t.Group)
}

func toRestfulParam(p Param) *restful.Parameter {
	if p.Type == "" && p.Example != nil {
		p.Type = reflect.TypeOf(p.Example).String()
	}
	if p.Type == "" {
		p.Type = "string"
	}

	var param *restful.Parameter
	switch p.Kind {
	case ParamKindBody:
		param = restful.BodyParameter(p.Name, p.Description)
	case ParamKindForm:
		param = restful.FormParameter(p.Name, p.Description)
	case ParamKindPath:
		param = restful.PathParameter(p.Name, p.Description)
	case ParamKindHeader:
		param = restful.HeaderParameter(p.Name, p.Description)
	case ParamKindQuery:
		param = restful.QueryParameter(p.Name, p.Description)
	default:
		return &restful.Parameter{}
	}

	return param.
		DataType(p.Type).
		Required(!p.IsOptional)
}

func (t *Tree) addWebService(ws *restful.WebService, meta string, basepath string, baseparams []*restful.Parameter, group *Group) {
	for _, params := range group.params {
		baseparams = append(baseparams, toRestfulParam(params))
	}

	basepath = strings.TrimRight(basepath, "/") + "/" + strings.TrimLeft(group.path, "/")
	if group.tag != "" {
		meta = group.tag
	}

	for _, route := range group.routes {
		// run hook before add
		if t.RouteUpdateFunc != nil {
			t.RouteUpdateFunc(route)
		}

		rb := ws.
			Method(route.Method).
			Path(basepath+route.Path).
			To(route.Func).
			Metadata(restfulspec.KeyOpenAPITags, []string{meta}).
			Doc(route.Summary)

		for _, param := range baseparams {
			rb.Param(param)
		}
		for _, param := range route.Params {
			if param.Kind == ParamKindBody {
				rb.Reads(param.Example, param.Description)
			} else {
				p := toRestfulParam(param)
				rb.Param(p)
			}
		}
		for _, ret := range route.Responses {
			rb.Returns(ret.Code, ret.Description, ret.Body)
			if ret.Body != nil {
				rb.Writes(ret.Body)
			}
		}

		ws.Route(rb)
	}
	for _, group := range group.subGroups {
		t.addWebService(ws, meta, basepath, baseparams, group)
	}
}

type Group struct {
	tag       string
	path      string
	params    []Param // common params apply to all routes in the group
	routes    []*Route
	subGroups []*Group // sub groups
}

func NewGroup(path string) *Group {
	return &Group{path: path}
}

func (g *Group) Tag(name string) *Group {
	g.tag = name
	return g
}

func (g *Group) AddRoutes(rs ...*Route) *Group {
	g.routes = append(g.routes, rs...)
	return g
}

func (g *Group) AddSubGroup(groups ...*Group) *Group {
	g.subGroups = append(g.subGroups, groups...)
	return g
}

func (g *Group) Parameters(params ...Param) *Group {
	g.params = append(g.params, params...)
	return g
}

type Route struct {
	Summary    string
	Path       string
	Method     string
	Func       Function
	Params     []Param
	Responses  []ResponseMeta
	Properties map[string]interface{}
}

type ResponseMeta struct {
	Code        int
	Headers     map[string]string
	Body        interface{}
	Description string
}

func Do(method string, path string) *Route {
	return &Route{
		Method: method,
		Path:   path,
	}
}

func GET(path string) *Route {
	return Do(http.MethodGet, path)
}

func POST(path string) *Route {
	return Do(http.MethodPost, path)
}

func PUT(path string) *Route {
	return Do(http.MethodPut, path)
}

func PATCH(path string) *Route {
	return Do(http.MethodPatch, path)
}

func DELETE(path string) *Route {
	return Do(http.MethodDelete, path)
}

func (n *Route) To(fun Function) *Route {
	n.Func = fun
	return n
}

func (n *Route) ShortDesc(summary string) *Route {
	n.Summary = summary
	return n
}

func (n *Route) Paged() *Route {
	n.Params = append(n.Params, QueryParameter("page", "page number").Optional())
	n.Params = append(n.Params, QueryParameter("size", "page size").Optional())
	return n
}

func (n *Route) Parameters(params ...Param) *Route {
	n.Params = append(n.Params, params...)
	return n
}

func (n *Route) Response(body interface{}, descs ...string) *Route {
	n.Responses = append(n.Responses, ResponseMeta{Code: http.StatusOK, Body: body, Description: strings.Join(descs, "")})
	return n
}

func (n *Route) SetProperty(k string, v interface{}) *Route {
	if n.Properties == nil {
		n.Properties = make(map[string]interface{})
	}
	n.Properties[k] = v
	return n
}

type ParamKind string

const (
	ParamKindPath   ParamKind = "path"
	ParamKindQuery  ParamKind = "query"
	ParamKindHeader ParamKind = "header"
	ParamKindForm   ParamKind = "form"
	ParamKindBody   ParamKind = "body"
)

type Param struct {
	Name        string
	Kind        ParamKind
	Type        string
	IsOptional  bool
	Description string
	Example     interface{}
}

func BodyParameter(name string, value interface{}) Param {
	return Param{Kind: ParamKindBody, Name: name, Example: value}
}

func FormParameter(name string, description string) Param {
	return Param{Kind: ParamKindForm, Name: name, Description: description}
}

func PathParameter(name string, description string) Param {
	return Param{Kind: ParamKindPath, Name: name, Description: description}
}

func QueryParameter(name string, description string) Param {
	return Param{Kind: ParamKindQuery, Name: name, Description: description}
}

func (p Param) Optional() Param {
	p.IsOptional = true
	return p
}

func (p Param) DataType(t string) Param {
	p.Type = t
	return p
}
