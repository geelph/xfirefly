package cel

import (
	"xfirefly/pkg/utils/proto"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

// StrStrMapType 定义了一个键和值都为字符串的映射类型
var StrStrMapType = decls.NewMapType(decls.String, decls.String)

// NewEnvOptions 定义了默认的 CEL 环境选项（仅类型与变量声明，函数实现改为使用 cel.Function 绑定）
var NewEnvOptions = []cel.EnvOption{
	cel.Container("proto"),
	cel.Types(
		&proto.UrlType{},
		&proto.Request{},
		&proto.Response{},
		&proto.Reverse{},
		StrStrMapType,
	),
	cel.Declarations(
		decls.NewVar("request", decls.NewObjectType("proto.Request")),
		decls.NewVar("response", decls.NewObjectType("proto.Response")),
	),
}

// ReadCompileOptions 返回 CEL 环境选项（类型、变量、函数实现）
// 注：不再使用已弃用的 TypeRegistry 组合适配器/提供者，统一依赖 cel.Types 注册类型
func ReadCompileOptions() []cel.EnvOption {
	allEnvOptions := make([]cel.EnvOption, 0, len(NewEnvOptions)+len(FunctionEnvOptions))
	allEnvOptions = append(allEnvOptions, NewEnvOptions...)
	allEnvOptions = append(allEnvOptions, FunctionEnvOptions...)
	return allEnvOptions
}
