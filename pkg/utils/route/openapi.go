package route

import (
	"net/http"
	"path"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
)

func BuildOpenAPIWebService(wss []*restful.WebService, path string, postfun func(swagger *spec.Swagger)) *restful.WebService {
	ws := new(restful.WebService).
		Path(path).
		Produces(restful.MIME_JSON)

	cros := &restful.CrossOriginResourceSharing{AllowedHeaders: []string{"*"}, AllowedMethods: []string{"*"}}
	ws.Filter(cros.Filter)

	builder := &Builder{
		InterfaceBuildOption: InterfaceBuildOptionOverride,
	}
	swagger := builder.buildOpenAPI(wss, postfun)

	ws.Route(ws.GET("/").To(func(r *restful.Request, w *restful.Response) {
		w.WriteAsJson(swagger)
	}))
	return ws
}

type SwaggerBuilder struct {
	openapi *Builder
}

func (b *Builder) buildOpenAPI(wss []*restful.WebService, afterFunc func(swagger *spec.Swagger)) *spec.Swagger {
	swagger := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger: "2.0",
			Schemes: []string{"http", "https"},
			Paths:   &spec.Paths{},
		},
	}

	paths := map[string]spec.PathItem{}
	for _, ws := range wss {
		rootPath, patterns := sanitizePath(ws.RootPath())

		var commonPathParameters []spec.Parameter
		for _, pathparam := range ws.PathParameters() {
			parameter := convertParameter(pathparam.Data())
			for name, pattern := range patterns {
				if parameter.Name == name && parameter.In == "path" && parameter.Pattern == "" {
					parameter.Pattern = pattern
					break
				}
			}
			commonPathParameters = append(commonPathParameters, parameter)
		}

		for _, route := range ws.Routes() {
			pathItem := spec.PathItem{}

			routePath, patterns := sanitizePath(route.Path)
			routePath = path.Join(rootPath, routePath)

			if exist, ok := paths[routePath]; ok {
				pathItem = exist
			}
			// common parameters
			pathItem.Parameters = commonPathParameters
			// fill pattern
			operation := b.buildOperation(route, patterns)

			switch route.Method {
			case http.MethodGet, "":
				pathItem.Get = operation
			case http.MethodPost:
				pathItem.Post = operation
			case http.MethodPut:
				pathItem.Put = operation
			case http.MethodDelete:
				pathItem.Delete = operation
			case http.MethodPatch:
				pathItem.Patch = operation
			case http.MethodHead:
				pathItem.Head = operation
			case http.MethodOptions:
				pathItem.Options = operation
			}

			paths[routePath] = pathItem
		}
	}

	swagger.Paths.Paths = paths
	swagger.Definitions = b.Definitions

	if afterFunc != nil {
		afterFunc(swagger)
	}
	return swagger
}

func (b *Builder) buildOperation(route restful.Route, pathPatterns map[string]string) *spec.Operation {
	op := &spec.Operation{
		OperationProps: spec.OperationProps{
			Summary:     route.Doc,
			Description: route.Doc,
			Consumes:    route.Consumes,
			Produces:    route.Produces,
			Deprecated:  route.Deprecated,
		},
	}
	// ID keep empty
	op.ID = ""

	// parameters
	params := make([]spec.Parameter, 0, len(route.ParameterDocs))
	for _, param := range route.ParameterDocs {
		params = append(params, convertParameter(param.Data()))
	}
	op.Parameters = params

	// tags
	if val, ok := route.Metadata["openapi.tags"].([]string); ok {
		op.Tags = val
	}

	// responses
	responses := spec.Responses{
		ResponsesProps: spec.ResponsesProps{
			StatusCodeResponses: make(map[int]spec.Response),
		},
	}
	if route.DefaultResponse != nil {
		defaultresponse := b.convertResponse(*route.DefaultResponse)
		responses.Default = &defaultresponse
	}
	for code, resp := range route.ResponseErrors {
		responses.StatusCodeResponses[code] = b.convertResponse(resp)
	}
	if len(responses.StatusCodeResponses) == 0 {
		responses.StatusCodeResponses[200] = spec.Response{ResponseProps: spec.ResponseProps{Description: "OK"}}
	}
	op.Responses = &responses

	// fill body parameter schema
	if route.WriteSample != nil {
		b.Build(route.WriteSample)
	}
	if route.ReadSample != nil {
		b.Build(route.ReadSample)
	}
	return op
}

func (b *Builder) convertResponse(resp restful.ResponseError) spec.Response {
	response := spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: resp.Message,
		},
	}

	// headers
	if len(resp.Headers) > 0 {
		headers := map[string]spec.Header{}
		for k, h := range resp.Headers {
			headers[k] = convertHeader(h)
		}
		response.Headers = headers
	}

	// schema
	response.Schema = b.Build(resp.Model)
	return response
}

func convertHeader(header restful.Header) spec.Header {
	return buildHeader(header)
}

func ParamIn(kind int) string {
	switch {
	case kind == restful.PathParameterKind:
		return "path"
	case kind == restful.QueryParameterKind:
		return "query"
	case kind == restful.BodyParameterKind:
		return "body"
	case kind == restful.HeaderParameterKind:
		return "header"
	case kind == restful.FormParameterKind:
		return "formData"
	}
	return ""
}

func convertParameter(param restful.ParameterData) spec.Parameter {
	parameter := spec.Parameter{
		SimpleSchema: spec.SimpleSchema{
			CollectionFormat: param.CollectionFormat,
		},
		CommonValidations: spec.CommonValidations{
			Pattern:     param.Pattern,
			MaxLength:   param.MaxLength,
			MinLength:   param.MinLength,
			Maximum:     param.Maximum,
			Minimum:     param.Minimum,
			MaxItems:    param.MaxItems,
			MinItems:    param.MinItems,
			UniqueItems: param.UniqueItems,
		},
		ParamProps: spec.ParamProps{
			Name:            param.Name,
			Description:     param.Description,
			Required:        param.Required,
			In:              ParamIn(param.Kind),
			AllowEmptyValue: param.AllowEmptyValue,
		},
	}

	if param.Kind == restful.BodyParameterKind {
		if strings.HasPrefix(param.DataType, "[]") {
			parameter.Schema = spec.ArrayProperty(spec.RefSchema("#/definitions/" + param.DataType[2:]))
		} else {
			parameter.Schema = spec.RefSchema("#/definitions/" + param.DataType)
		}
	} else {
		parameter.Type = param.DataType
		parameter.Format = param.DataFormat
	}

	// enum
	if len(param.PossibleValues) > 0 {
		// set default value
		parameter.Default = param.DefaultValue
		if param.DefaultValue == "" {
			parameter.Default = param.PossibleValues[0]
		}
		// set enum
		enum := make([]interface{}, len(param.PossibleValues))
		for i, v := range param.PossibleValues {
			enum[i] = v
		}
		parameter.CommonValidations.Enum = enum
	}
	// allow multiple
	if param.AllowMultiple {
		parameter = spec.Parameter{
			CommonValidations: parameter.CommonValidations,
			SimpleSchema: spec.SimpleSchema{
				Type: "array",
				Items: &spec.Items{
					SimpleSchema: parameter.SimpleSchema,
				},
			},
		}
	}

	// schema
	return parameter
}

// sanitizePath removes regex expressions from named path params,
// since openapi only supports setting the pattern as a property named "pattern".
// Expressions like "/api/v1/{name:[a-z]}/" are converted to "/api/v1/{name}/".
// The second return value is a map which contains the mapping from the path parameter
// name to the extracted pattern
func sanitizePath(restfulPath string) (string, map[string]string) {
	openapiPath := ""
	patterns := map[string]string{}
	for _, fragment := range strings.Split(restfulPath, "/") {
		if fragment == "" {
			continue
		}
		if strings.HasPrefix(fragment, "{") && strings.Contains(fragment, ":") {
			split := strings.Split(fragment, ":")
			fragment = split[0][1:]
			pattern := split[1][:len(split[1])-1]
			patterns[fragment] = pattern
			fragment = "{" + fragment + "}"
		}
		openapiPath += "/" + fragment
	}
	return openapiPath, patterns
}

// buildHeader builds a specification header structure from restful.Header
func buildHeader(header restful.Header) spec.Header {
	responseHeader := spec.Header{}
	responseHeader.Type = header.Type
	responseHeader.Description = header.Description
	responseHeader.Format = header.Format
	responseHeader.Default = header.Default

	// If type is "array" items field is required
	if header.Type == "array" {
		responseHeader.CollectionFormat = header.CollectionFormat
		responseHeader.Items = buildHeadersItems(header.Items)
	}

	return responseHeader
}

// buildHeadersItems builds
func buildHeadersItems(items *restful.Items) *spec.Items {
	responseItems := spec.NewItems()
	responseItems.Format = items.Format
	responseItems.Type = items.Type
	responseItems.Default = items.Default
	responseItems.CollectionFormat = items.CollectionFormat
	if items.Items != nil {
		responseItems.Items = buildHeadersItems(items.Items)
	}
	return responseItems
}
