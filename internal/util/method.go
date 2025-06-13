package util

import (
	"github.com/stoewer/go-strcase"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
)

var (
	ConnectNewRequest = protogen.GoIdent{
		GoName:       "NewRequest",
		GoImportPath: "connectrpc.com/connect",
	}
)

type MethodCreate struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	Parent         string
	Message        string
	UserSettableID string
}

func (m MethodCreate) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	userSetID := m.UserSettableID
	if userSetID == "" && HasUserSettableIDField(m.Resource, m.Method.Input.Desc) {
		userSetID = "userSetID"
		f.P(userSetID + " := \"\"")
		f.P("if fx.IDGenerator != nil {")
		f.P(userSetID + " = fx.IDGenerator()")
		f.P("}")
	}

	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("createResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	if HasParent(m.Resource) {
		f.P("Parent: ", m.Parent, ",")
	}

	upper := strcase.UpperCamelCase(string(FindResourceField(
		m.Method.Input.Desc,
		m.Resource,
	).Name()))

	switch {
	case m.Message != "":
		f.P(upper, ": ", m.Message, ",")
	case !HasParent(m.Resource):
		f.P(upper, ": fx.Create(),")
	default:
		f.P(upper, ": fx.Create(", m.Parent, "),")
	}

	if userSetID != "" && HasUserSettableIDField(m.Resource, m.Method.Input.Desc) {
		f.P(upper, "Id: ", userSetID, ",")
	}

	f.P("}))")
	if response != "_" {
		f.P(response, assign, "createResp.Msg")
	}
}

type MethodGet struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	Name string
}

func (m MethodGet) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("getResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	f.P("Name: ", m.Name, ",")
	f.P("}))")
	if response != "_" {
		f.P(response, assign, "getResp.Msg")
	}
}

type MethodBatchGet struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	Parent string
	Names  []string
}

func (m MethodBatchGet) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("batchGetResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	if HasParent(m.Resource) {
		f.P("Parent: ", m.Parent, ",")
	}
	f.P("Names: []string{")
	for _, name := range m.Names {
		f.P(name, ",")
	}
	f.P("},")
	f.P("}))")
	if response != "_" {
		f.P(response, assign, "batchGetResp.Msg")
	}
}

type MethodUpdate struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	// set either Parent + Name, or Msg
	Name       string
	Parent     string
	Msg        string
	UpdateMask []string
	Etag       string
	EtagTest   bool
}

func (m MethodUpdate) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	upper := strcase.UpperCamelCase(string(FindResourceField(
		m.Method.Input.Desc,
		m.Resource,
	).Name()))

	if m.Msg == "" {
		if HasParent(m.Resource) {
			f.P("msg := fx.Update(", m.Parent, ")")
		} else {
			f.P("msg := fx.Update()")
		}
		f.P("msg.Name = ", m.Name)
	}
	if m.EtagTest && !HasEtagField(m.Method.Input.Desc) && HasEtagField(m.Method.Output.Desc) {
		// Request object does not have an etag field, but the resource has.
		if m.Etag != "" {
			f.P("msg.Etag = ", m.Etag)
		} else {
			f.P(`msg.Etag = created.Etag // assign etag from the created resource`)
		}
	}
	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("updateResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	if m.Msg != "" {
		f.P(upper, ":", m.Msg, ",")
	} else {
		f.P(upper, ": msg,")
	}
	if HasUpdateMask(m.Method.Desc) && len(m.UpdateMask) > 0 {
		fieldmaskpbFieldMask := f.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "FieldMask",
			GoImportPath: "google.golang.org/protobuf/types/known/fieldmaskpb",
		})
		f.P("UpdateMask: &", fieldmaskpbFieldMask, "{")
		f.P("Paths: []string{")
		for _, path := range m.UpdateMask {
			f.P(path, ",")
		}
		f.P("},")
		f.P("},")
	}
	switch {
	case HasEtagField(m.Method.Input.Desc) && m.Etag != "":
		f.P("Etag: ", m.Etag, ",")
	case HasRequiredEtagField(m.Method.Input.Desc):
		if m.Msg != "" {
			// Delete request has an required etag field.
			f.P("Etag: ", m.Msg, ".Etag,")
		} else {
			f.P("Etag: msg.Etag,")
		}
	}
	f.P("}))")
	if response != "_" {
		f.P(response, assign, "updateResp.Msg")
	}
}

type MethodList struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	Parent    string
	PageSize  string
	PageToken string
}

func (m MethodList) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("listResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	if HasParent(m.Resource) {
		f.P("Parent: ", m.Parent, ",")
	}
	if m.PageSize != "" {
		f.P("PageSize: ", m.PageSize, ",")
	}
	if m.PageToken != "" {
		f.P("PageToken: ", m.PageToken, ",")
	}
	f.P("}))")
	if response != "_" {
		f.P(response, assign, "listResp.Msg")
	}
}

type MethodSearch struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	Parent    string
	PageSize  string
	PageToken string
}

func (m MethodSearch) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("searchResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	if HasParent(m.Resource) {
		f.P("Parent: ", m.Parent, ",")
	}
	if m.PageSize != "" {
		f.P("PageSize: ", m.PageSize, ",")
	}
	if m.PageToken != "" {
		f.P("PageToken: ", m.PageToken, ",")
	}
	f.P("}))")
	if response != "_" {
		f.P(response, assign, "searchResp.Msg")
	}
}

type MethodDelete struct {
	Resource *annotations.ResourceDescriptor
	Method   *protogen.Method

	ResourceVar string // variable name of the resource.
	Name        string
	Etag        string
}

func (m MethodDelete) Generate(f *protogen.GeneratedFile, response, err, assign string) {
	if response == "_" {
		f.P(response, ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	} else {
		f.P("deleteResp", ", ", err, " ", assign, " fx.Service().", m.Method.GoName, "(fx.Context(), ", ConnectNewRequest, "(&", m.Method.Input.GoIdent, "{") //nolint:lll
	}
	if m.Name != "" {
		f.P("Name: ", m.Name, ",")
	} else {
		f.P("Name: ", m.ResourceVar, ".Name,")
	}
	switch {
	case HasEtagField(m.Method.Input.Desc) && m.Etag != "":
		f.P("Etag: ", m.Etag, ",")
	case HasRequiredEtagField(m.Method.Input.Desc):
		if m.ResourceVar != "" {
			// Delete request has an required etag field.
			f.P("Etag: ", m.ResourceVar, ".Etag,")
		} else {
			f.P("Etag: \"\",")
		}
	}
	f.P("}))")
	if response != "_" {
		f.P(response, assign, "deleteResp.Msg")
	}
}
