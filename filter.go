package filter

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/xmdas-link/auth"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/iancoleman/strcase"
	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	"github.com/xmdas-link/filter/model"
)

var (
	ContextTmpObjKey = "filter_tmp_obj_key"
)

func NewDataFilter(filter *Filter) gin.HandlerFunc {

	return func(context *gin.Context) {
		// replace gin.context default responseWrite
		fw := &filterWriter{ResponseWriter: context.Writer, body: bytes.NewBufferString(""), fakeWrite: true}
		context.Writer = fw

		// call next handler
		context.Next()

		fw.fakeWrite = false

		obj, ok := context.Get(ContextTmpObjKey)
		if !ok {
			context.Data(http.StatusOK, "application/json; charset=utf-8", fw.body.Bytes())
			return
		}

		// get current request user/role
		json := filter.Process(context, obj)

		ret, _ := json.Marshal(obj)

		context.Data(http.StatusOK, "application/json; charset=utf-8", ret)
	}
}

type H struct {
	Ctx  *gin.Context
	Data interface{}
}

func (h H) MarshalJSON() ([]byte, error) {
	h.Ctx.Set(ContextTmpObjKey, h.Data)

	hasFilterMiddleware := false
	list := h.Ctx.HandlerNames()
	for _, handlerName := range list {
		if strings.Contains(handlerName, "filter.NewDataFilter") {
			hasFilterMiddleware = true
			break
		}
	}

	if hasFilterMiddleware {
		return []byte("[]"), nil
	} else {
		return json.Marshal(h.Data)
	}
}

type filterWriter struct {
	gin.ResponseWriter
	body      *bytes.Buffer
	fakeWrite bool
}

func (fw filterWriter) Write(b []byte) (int, error) {
	if fw.fakeWrite {
		log.Printf("fake write: %s \n", b)
		fw.body.Write(b)
		return 0, nil
	} else {
		log.Printf("real write: %s \n", b)
		return fw.ResponseWriter.Write(b)
	}
}

type Filter struct {
	encoderMaps model.EncoderMap
	modelPath   string
	model       *model.Model
	enforcer    *casbin.Enforcer
}

func NewFilter(modelPath string, enforcer *casbin.Enforcer) (*Filter, error) {
	if modelPath == "" {
		return nil, errors.New("invalid parameters for filter")
	}

	f := &Filter{}
	f.modelPath = modelPath
	f.encoderMaps = model.LoadEncoderMap()
	f.enforcer = enforcer

	m, err := model.NewModelFromFile(f.modelPath)
	if err != nil {
		return nil, err
	}
	f.model = m

	return f, nil
}

func (f *Filter) GetUserName(ctx *gin.Context) string {
	user := ctx.GetStringMapString(auth.CtxKeyAuthUser)
	if userName, ok := user["user"]; ok {
		return userName
	}
	return ""
}

func (f *Filter) GetUserRole(ctx *gin.Context) string {
	userRole := ctx.GetString(auth.CtxKeyUserRole)
	return userRole
}

// sub, obj
func (f *Filter) Process(ctx *gin.Context, obj interface{}) jsoniter.API {
	jsonapi := jsoniter.Config{}.Froze()

	if len(f.model.Policy) == 0 {
		return jsonapi
	}

	sub := f.GetUserName(ctx)

	var applyRule []model.PolicyRule
	for _, policy := range f.model.Policy {
		// username/role match
		if !f.hasLink(sub, policy[0]) {
			continue
		}

		// encoder registed
		if !f.hasProcessEncoder(policy[3]) {
			log.Println("not register process encoder:" + policy[3])
			continue
		}

		if strings.Index(policy[2], "|") != -1 {
			fields := strings.Split(policy[2], "|")
			for i := 0; i < len(fields); i++ {
				fields[i] = strings.TrimSpace(fields[i])
			}

			applyRule = append(applyRule, model.PolicyRule{
				Sub:     policy[0],
				Model:   policy[1],
				Fields:  fields,
				Encoder: f.encoderMaps[policy[3]],
			})
		} else {
			applyRule = append(applyRule, model.PolicyRule{
				Sub:     policy[0],
				Model:   policy[1],
				Fields:  []string{policy[2]},
				Encoder: f.encoderMaps[policy[3]],
			})
		}
	}

	if len(applyRule) == 0 {
		return jsonapi
	}

	for _, rule := range applyRule {
		jsonapi.RegisterExtension(&FieldsExtension{
			jsoniter.DummyExtension{},
			rule.Model,
			rule.Fields,
			rule.Encoder,
		})
	}

	return jsonapi
}

// 判断角色/用户关系
func (f *Filter) hasLink(sub1, sub2 string) bool {
	if sub1 == sub2 {
		return true
	}

	if f.enforcer == nil {
		return false
	}

	roles, _ := f.enforcer.GetImplicitRolesForUser(sub1)
	hasRole := false
	for _, r := range roles {
		if r == sub2 {
			hasRole = true
			break
		}
	}

	return hasRole
}

func (f *Filter) hasProcessEncoder(encoderName string) bool {
	_, ok := f.encoderMaps[encoderName]
	return ok
}

func (f *Filter) AddEncoder(name string, encoder model.Encoder) {
	f.encoderMaps.AddEncoder(name, encoder)
}

// JSON 过滤扩展
type FieldsExtension struct {
	jsoniter.DummyExtension
	ModelName string
	Fields    []string
	Func      model.Encoder
}

func (extension *FieldsExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	if extension.ModelName == "" || len(extension.Fields) == 0 {
		// nothing need to filter, skip
		return
	}

	for _, binding := range structDescriptor.Fields {
		binding.ToNames = []string{strcase.ToSnake(binding.Field.Name())}
		binding.FromNames = []string{strcase.ToSnake(binding.Field.Name())}
	}

	structType := structDescriptor.Type.(*reflect2.UnsafeStructType)
	if extension.ModelName != structType.Name() {
		// not process model
		//log.Printf("except Model name: %v, real model name: %v\n", extension.ModelName, structType.Name())
		return
	}

	for _, binding := range structDescriptor.Fields {
		name := binding.Field.Name()
		for _, v := range extension.Fields {
			//log.Printf("compare field, %v == %v\n", name, v)
			if name == v {
				binding.Encoder = extension.Func
				break
			}
		}
	}

	return
}
