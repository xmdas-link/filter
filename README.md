# filter
一个gin的中间件，基于jsoniter库，类似casbin定义数据过滤规则后，在JSON输出之前，
会根据定义的过滤策略对返回数据进行移除、脱敏等操作，也支持自定义处理方法。
如果引入authorization的话，可以使用casbin的角色/用户继承关系。

## 用法
```go
import (
    "github.com/xmdas-link/authorization"
    "github.com/xmdas-link/filter"
)

var dataFilter *filter.Filter
var enforcer *casbin.Enforcer

func main() {
	route := gin.Default()

	enforcer, _ = casbin.NewEnforcer("model.conf", "policy.csv")
	route.Use(authorization.NewAuthorizer(enforcer))

	var err error
	dataFilter, err = filter.NewFilter("filter_policy.csv", enforcer)
	if err == nil {
		route.Use(filter.NewDataFilter(dataFilter))
		// 全局注册
		dataFilter.AddEncoder("my_custom", &myEncoder{})
		dataFilter.AddEncoder("my_custom2", &myEncoder2{})
	}

	route.GET("/", helloHandler)
	route.GET("/school/list", getSchool)
	route.GET("/school/add", postSchool)
	route.GET("/person/list", getPerson)
	route.POST("/person/add", postPerson)

	route.Run()
}
```
```go
func getSchool(ctx *gin.Context) {
	user := User{
		UserName:   "test",
		UserPwd:    "123456",
		UserSalary: 342342,
		UserAge:    26,
		UserMobile: ctx.DefaultQuery("mobile", "123456789"),
		Profile: Profile{
			ID:    4,
			Grade: 5,
			Photo: "nnnnnnn",
		},
	}

	ctx.JSON(200, filter.H{
		Ctx:  ctx,
		Data: user,
	})
}
```

**注意:** 
在控制器最终ctx.JSON输出的时候，要调用**filter.H**

## 数据过滤规则策略文件
```ini
# sub, model, field1|field2|filed3, action

user, User, UserSalary|UserPwd, remove
admin, User, UserMobile, sensitive

alice, Profile, Photo, my_custom
alice, School, SchoolCode, my_custom2
```
- sub: 要应用的用户名或者角色名
- model: 要应用在哪个模型上
- field1: 要应用的字段，支持多个字段，以“|”分隔
- action: 具体的处理规则，目前内置移除与脱敏，支持自定义  
    - remove: 字段移除
    - sensitive: 字段脱敏处理
    
### 自定义过滤规则
在policy.csv文件中，action设置为自定义的规则名，全局唯一，my_custom  
然后在代码中定义my_custom的编码规则，并进行注册
```go
// 定义自定义编码规则 
type myEncoder struct {
}

func (encoder *myEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	str := *((*string)(ptr))
	newstr := "======>" + str
	stream.WriteString(newstr)
}
func (encoder *myEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	return false
}

// 全局注册
dataFilter.AddEncoder("my_custom", &myEncoder{})
```

