/*
  - Package cel
    @Author: zhizhuo
    @IDE：GoLand
    @File: cel.go
    @Date: 2025/2/7 上午8:57*
*/
package cel

import (
	"fmt"
	"strings"
	"sync"

	"github.com/donnie4w/go-logger/logger"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"gopkg.in/yaml.v2"
)

// 全局CEL环境互斥锁，确保每次只有一个goroutine可以配置环境
var globalCELEnvMutex sync.Mutex

// CustomLib 自定义CEL库结构体
type CustomLib struct {
	envOptions  []cel.EnvOption
	env         *cel.Env // 缓存的CEL环境
	initialized bool     // 标记是否已初始化
}

// CompileOptions 返回环境选项
func (c *CustomLib) CompileOptions() []cel.EnvOption {
	return c.envOptions
}

// Evaluate 执行CEL表达式并返回结果
func (c *CustomLib) Evaluate(expression string, variables map[string]any) (ref.Val, error) {
	var env *cel.Env
	var err error

	if c.initialized && c.env != nil {
		env = c.env
	} else {
		// 如果没有预初始化环境，创建一个新环境
		globalCELEnvMutex.Lock()
		env, err = cel.NewEnv(c.envOptions...)
		globalCELEnvMutex.Unlock()

		if err != nil {
			return nil, fmt.Errorf("创建CEL环境失败: %v", err)
		}
	}

	// 复制一份变量映射，避免潜在的并发修改
	varsCopy := make(map[string]any, len(variables))
	for k, v := range variables {
		varsCopy[k] = v
	}

	// 编译和评估表达式
	return Eval(env, expression, varsCopy)
}

// NewCelEnv 创建新的CEL环境并缓存
func (c *CustomLib) NewCelEnv() (*cel.Env, error) {
	// 首先检查是否有预初始化的环境
	if c.initialized && c.env != nil {
		env := c.env
		return env, nil
	}
	env, err := cel.NewEnv(c.envOptions...)
	if err != nil {
		return nil, err
	}
	// 缓存创建的环境
	c.env = env
	c.initialized = true
	return env, nil
}

// NewCustomLib 创建新的CustomLib实例
func NewCustomLib() *CustomLib {
	c := &CustomLib{}
	c.envOptions = ReadCompileOptions()
	return c
}

// Eval 执行CEL表达式
func Eval(env *cel.Env, expression string, params map[string]any) (ref.Val, error) {
	ast, issues := env.Compile(expression)
	if issues.Err() != nil {
		logger.Error(fmt.Sprintf("CEL编译错误: %s", issues.Err()))
		return nil, issues.Err()
	}

	prg, err := env.Program(ast)
	if err != nil {
		logger.Error(fmt.Sprintf("CEL程序创建错误: %s", err))
		return nil, err
	}

	out, _, err := prg.Eval(params)
	if err != nil {
		logger.Error(fmt.Sprintf("CEL执行错误: %s", err))
		return nil, err
	}

	return out, nil
}

// WriteRuleSetOptions 从YAML配置中添加变量声明
func (c *CustomLib) WriteRuleSetOptions(args yaml.MapSlice) {

	for _, v := range args {
		key := v.Key.(string)
		value := v.Value

		var declaration *exprpb.Decl
		switch val := value.(type) {
		case int64:
			declaration = decls.NewVar(key, decls.Int)
		case string:
			if strings.HasPrefix(val, "newReverse") {
				declaration = decls.NewVar(key, decls.NewObjectType("proto.Reverse"))
			} else if strings.HasPrefix(val, "randomInt") {
				declaration = decls.NewVar(key, decls.Int)
			} else {
				declaration = decls.NewVar(key, decls.String)
			}
		case map[string]string:
			declaration = decls.NewVar(key, StrStrMapType)
		default:
			declaration = decls.NewVar(key, decls.String)
		}
		c.envOptions = append(c.envOptions, cel.Declarations(declaration))
	}
}

// WriteRuleFunctionsROptions 注册用于处理r0 || r1规则解析的函数
func (c *CustomLib) WriteRuleFunctionsROptions(funcName string, returnBool bool) {

	c.envOptions = append(c.envOptions, cel.Function(
		funcName,
		cel.Overload(
			funcName+"_bool",
			[]*cel.Type{},
			cel.BoolType,
			cel.FunctionBinding(func(values ...ref.Val) ref.Val {
				return types.Bool(returnBool)
			}),
		),
	))
}

// BatchUpdateCompileOptions 批量更新编译选项，减少锁竞争
func (c *CustomLib) BatchUpdateCompileOptions(declarations map[string]*exprpb.Decl) {
	if len(declarations) == 0 {
		return
	}

	// 将所有声明合并为一个环境选项
	allDecls := make([]*exprpb.Decl, 0, len(declarations))
	for _, decl := range declarations {
		allDecls = append(allDecls, decl)
	}

	// 一次性添加所有声明
	c.envOptions = append(c.envOptions, cel.Declarations(allDecls...))

	// 重置环境缓存，强制下次使用时重新创建
	c.env = nil
	c.initialized = false
}

// UpdateCompileOption 更新单个编译选项
func (c *CustomLib) UpdateCompileOption(name string, t *exprpb.Type) {
	// 添加单个声明
	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(name, t)))

	// 重置环境缓存
	c.env = nil
	c.initialized = false
}

// Reset 重置CEL库状态，释放资源
func (c *CustomLib) Reset() {
	// 释放环境，让GC回收资源
	c.env = nil
	c.initialized = false
}

// WriteRuleIsVulOptions 添加漏洞检测函数声明
func (c *CustomLib) WriteRuleIsVulOptions(key string) {
	c.envOptions = append(c.envOptions, cel.Declarations(decls.NewVar(key+"()", decls.Bool)))
}
