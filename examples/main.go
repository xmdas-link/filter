package main

import (
	"unsafe"

	"github.com/casbin/casbin/v2"
	jsoniter "github.com/json-iterator/go"

	"github.com/gin-gonic/gin"
	"github.com/xmdas-link/authorization"
	"github.com/xmdas-link/filter"
)

type User struct {
	UserName   string  `json:"user_name"`
	UserPwd    string  `json:"-"`
	UserSalary float64 `json:"user_salary, omitempty"`
	UserAge    int     `json:"user_age"`
	UserMobile string  `json:"user_mobile"`

	Profile Profile `json:"profile"`
}

type Profile struct {
	ID    int    `json:"-"`
	Grade int    `json:"grade"`
	Photo string `json:"photo"`
}

type School struct {
	SchoolId   int    `json:"school_id"`
	SchoolName string `json:"school_name"`
	SchoolCode string `json:"school_code"`
}

/**
数据过滤中间件测试：
http://localhost:8080/school/add?username=alice
http://localhost:8080/school/add?username=bob
http://localhost:8080/school/add?username=foo
*/

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

func helloHandler(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"message": "hello",
	})
}

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
		Ctx: ctx,
		Data: gin.H{
			"code":    1,
			"message": "",
			"data":    user,
		},
	})
}

type myEncoder2 struct {
}

func (encoder *myEncoder2) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	str := *((*string)(ptr))
	newstr := "[======]" + str
	stream.WriteString(newstr)
}
func (encoder *myEncoder2) IsEmpty(ptr unsafe.Pointer) bool {
	return false
}

func postSchool(ctx *gin.Context) {
	school := School{
		SchoolId:   1,
		SchoolName: "学校",
		SchoolCode: "codetest",
	}

	ctx.JSON(200, filter.H{
		Ctx: ctx,
		Data: gin.H{
			"code":    1,
			"message": "",
			"data":    school,
		},
	})

	//// fail response
	//ctx.JSON(200, filter.H{
	//	Ctx: ctx,
	//	Data: gin.H{
	//		"code":    0,
	//		"message": "err info",
	//	},
	//})
}

func getPerson(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"message": "getPerson",
	})
}
func postPerson(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"message": "postPerson",
	})
}
