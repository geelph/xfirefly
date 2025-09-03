package finger

import (
	"fmt"
	"net/url"
	"strings"
	"xfirefly/pkg/cel"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/config"
	"xfirefly/pkg/utils/proto"

	"github.com/google/cel-go/checker/decls"
	"gopkg.in/yaml.v2"

	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type Checker struct {
	VariableMap map[string]any
	CustomLib   *cel.CustomLib
}

type ListItem struct {
	Key   string
	Value []string
}
type ListMap []ListItem

// IsFuzzSet 解析Set中的定义变量
func IsFuzzSet(args yaml.MapSlice, variableMap map[string]any, customLib *cel.CustomLib) {
	for _, arg := range args {
		key := arg.Key.(string)
		value := arg.Value.(string)
		// 处理dns反连
		if value == "newReverse()" {
			variableMap[key] = newReverse()
			customLib.UpdateCompileOption(key, decls.NewObjectType("proto.Reverse"))
			continue
		}
		// 处理jndi连接
		if value == "newJNDI()" {
			variableMap[key] = newJNDI()
			customLib.UpdateCompileOption(key, decls.NewObjectType("proto.Reverse"))
			continue
		}

		out, err := customLib.Evaluate(value, variableMap)
		if err != nil {
			variableMap[key] = fmt.Sprintf("%v", value)
			customLib.UpdateCompileOption(key, decls.String)
			continue
		}
		switch value := out.Value().(type) {
		case *proto.UrlType:
			variableMap[key] = common.UrlTypeToString(value)
			customLib.UpdateCompileOption(key, decls.NewObjectType("proto.UrlType"))
		case int64:
			variableMap[key] = int(value)
			customLib.UpdateCompileOption(key, decls.Int)
		case map[string]string:
			variableMap[key] = value
			customLib.UpdateCompileOption(key, cel.StrStrMapType)
		default:
			variableMap[key] = fmt.Sprintf("%v", out)
			customLib.UpdateCompileOption(key, decls.String)
		}
	}
}

// SetVariableMap 处理解析set中变量
func SetVariableMap(find string, variableMap map[string]any) string {
	for k, v := range variableMap {
		_, isMap := v.(map[string]string)
		if isMap {
			continue
		}
		newStr := fmt.Sprintf("%v", v)
		oldStr := "{{" + k + "}}"
		if !strings.Contains(find, oldStr) {
			continue
		}
		find = strings.ReplaceAll(find, oldStr, newStr)
	}
	return find
}

// newReverse 处理dns反连
func newReverse() *proto.Reverse {
	sub := common.RandomString(12)
	urlStr := fmt.Sprintf("http://%s.%s", sub, config.ReverseCeyeDomain)
	u, _ := url.Parse(urlStr)
	return &proto.Reverse{
		Url:                common.ParseUrl(u),
		Domain:             u.Hostname(),
		Ip:                 u.Host,
		IsDomainNameServer: false,
	}
}

// newJNDI 处理jndi连接
func newJNDI() *proto.Reverse {
	randomStr := common.RandomString(22)
	urlStr := fmt.Sprintf("http://%s:%s/%s", config.ReverseJndi, config.ReverseLdapPort, randomStr)
	u, _ := url.Parse(urlStr)
	parseUrl := common.ParseUrl(u)
	return &proto.Reverse{
		Url:                parseUrl,
		Domain:             u.Hostname(),
		Ip:                 config.ReverseJndi,
		IsDomainNameServer: false,
	}
}

// BatchFuzzSet 批量处理多个Set中的定义变量，优化性能
func BatchFuzzSet(args []interface{}, variableMap map[string]any, customLib *cel.CustomLib) {
	// 如果没有规则集，直接返回
	if len(args) == 0 {
		return
	}

	// 预处理所有可以直接处理的规则
	declarations := make(map[string]*exprpb.Decl)

	// 批量处理每个规则集
	for _, argItem := range args {
		switch argValue := argItem.(type) {
		case yaml.MapSlice:
			for _, arg := range argValue {
				key := arg.Key.(string)
				value := arg.Value.(string)

				// 处理特殊规则类型
				if value == "newReverse()" {
					variableMap[key] = newReverse()
					declarations[key] = decls.NewVar(key, decls.NewObjectType("proto.Reverse"))
					continue
				}

				// 处理jndi连接
				if value == "newJNDI()" {
					variableMap[key] = newJNDI()
					declarations[key] = decls.NewVar(key, decls.NewObjectType("proto.Reverse"))
					continue
				}

				// 评估表达式
				out, err := customLib.Evaluate(value, variableMap)
				if err != nil {
					variableMap[key] = fmt.Sprintf("%v", value)
					declarations[key] = decls.NewVar(key, decls.String)
					continue
				}

				// 根据类型设置变量和声明
				switch val := out.Value().(type) {
				case *proto.UrlType:
					variableMap[key] = common.UrlTypeToString(val)
					declarations[key] = decls.NewVar(key, decls.NewObjectType("proto.UrlType"))
				case int64:
					variableMap[key] = int(val)
					declarations[key] = decls.NewVar(key, decls.Int)
				case map[string]string:
					variableMap[key] = val
					declarations[key] = decls.NewVar(key, cel.StrStrMapType)
				default:
					variableMap[key] = fmt.Sprintf("%v", out)
					declarations[key] = decls.NewVar(key, decls.String)
				}
			}
		}
	}

	// 批量更新编译选项
	if len(declarations) > 0 {
		customLib.BatchUpdateCompileOptions(declarations)
	}
}
