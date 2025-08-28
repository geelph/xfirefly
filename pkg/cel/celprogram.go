/*
  - Package cel
    @Author: zhizhuo
    @IDE：GoLand
    @File: celprogram.go
    @Date: 2025/2/7 上午9:22*
*/
package cel

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"xfirefly/pkg/network"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/config"
	"xfirefly/pkg/utils/proto"

	"github.com/dlclark/regexp2"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// FunctionEnvOptions 将所有函数绑定迁移到 Env 期，避免已弃用 Program 期的 cel.Functions 与 interpreter/functions.Overload
var FunctionEnvOptions = []cel.EnvOption{
	// icontains: instance string method
	cel.Function("icontains",
		cel.MemberOverload("string_icontains_string",
			[]*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				v1, ok := lhs.(types.String)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to contains", lhs.Type())
				}
				v2, ok := rhs.(types.String)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to contains", rhs.Type())
				}
				return types.Bool(strings.Contains(strings.ToLower(string(v1)), strings.ToLower(string(v2))))
			}),
		),
	),
	// substr(s, start, length)
	cel.Function("substr",
		cel.Overload("substr_string_int_int",
			[]*cel.Type{cel.StringType, cel.IntType, cel.IntType}, cel.StringType,
			cel.FunctionBinding(func(values ...ref.Val) ref.Val {
				if len(values) == 3 {
					str, ok := values[0].(types.String)
					if !ok {
						return types.NewErr("invalid string to 'substr'")
					}
					start, ok := values[1].(types.Int)
					if !ok {
						return types.NewErr("invalid start to 'substr'")
					}
					length, ok := values[2].(types.Int)
					if !ok {
						return types.NewErr("invalid length to 'substr'")
					}
					runes := []rune(str)
					if start < 0 || length < 0 || int(start+length) > len(runes) {
						return types.NewErr("invalid start or length to 'substr'")
					}
					return types.String(runes[start : start+length])
				}
				return types.NewErr("too many arguments to 'substr'")
			}),
		),
	),
	// replaceAll(s, old, new)
	cel.Function("replaceAll",
		cel.Overload("replaceAll_string_string_string",
			[]*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.StringType,
			cel.FunctionBinding(func(values ...ref.Val) ref.Val {
				s, ok := values[0].(types.String)
				if !ok {
					return types.ValOrErr(s, "unexpected type '%v' passed to replaceAll", s.Type())
				}
				old, ok := values[1].(types.String)
				if !ok {
					return types.ValOrErr(old, "unexpected type '%v' passed to replaceAll", old.Type())
				}
				newStr, ok := values[2].(types.String)
				if !ok {
					return types.ValOrErr(newStr, "unexpected type '%v' passed to replaceAll", newStr.Type())
				}
				return types.String(strings.ReplaceAll(string(s), string(old), string(newStr)))
			}),
		),
	),
	// printable(s)
	cel.Function("printable",
		cel.Overload("printable_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				s, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(s, "unexpected type '%v' passed to printable", s.Type())
				}
				clean := strings.Map(func(r rune) rune {
					if unicode.IsPrint(r) {
						return r
					}
					return -1
				}, string(s))
				return types.String(clean)
			}),
		),
	),
	// toUintString(s, direction)
	cel.Function("toUintString",
		cel.Overload("toUintString_string_string",
			[]*cel.Type{cel.StringType, cel.StringType}, cel.StringType,
			cel.FunctionBinding(func(values ...ref.Val) ref.Val {
				s1, ok := values[0].(types.String)
				s := string(s1)
				if !ok {
					return types.ValOrErr(s1, "unexpected type '%v' passed to toUintString", s1.Type())
				}
				direction, ok := values[1].(types.String)
				if !ok {
					return types.ValOrErr(direction, "unexpected type '%v' passed to toUintString", direction.Type())
				}
				if direction == "<" {
					s = common.ReverseString(s)
				}
				if _, e := strconv.Atoi(s); e == nil {
					return types.String(s)
				} else {
					return types.NewErr("%v", e)
				}
			}),
		),
	),
	// bytes methods
	cel.Function("bcontains",
		cel.MemberOverload("bytes_bcontains_bytes",
			[]*cel.Type{cel.BytesType, cel.BytesType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				v1, ok := lhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to bcontains", lhs.Type())
				}
				v2, ok := rhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to bcontains", rhs.Type())
				}
				return types.Bool(bytes.Contains(v1, v2))
			}),
		),
	),
	cel.Function("ibcontains",
		cel.MemberOverload("bytes_ibcontains_bytes",
			[]*cel.Type{cel.BytesType, cel.BytesType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				v1, ok := lhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to bcontains", lhs.Type())
				}
				v2, ok := rhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to bcontains", rhs.Type())
				}
				return types.Bool(bytes.Contains(bytes.ToLower(v1), bytes.ToLower(v2)))
			}),
		),
	),
	cel.Function("bstartsWith",
		cel.MemberOverload("bytes_bstartsWith_bytes",
			[]*cel.Type{cel.BytesType, cel.BytesType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				v1, ok := lhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to bstartsWith", lhs.Type())
				}
				v2, ok := rhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to bstartsWith", rhs.Type())
				}
				return types.Bool(bytes.HasPrefix(v1, v2))
			}),
		),
	),
	// encode
	cel.Function("md5",
		cel.Overload("md5_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to md5_string", value.Type())
				}
				return types.String(fmt.Sprintf("%x", md5.Sum([]byte(v))))
			}),
		),
	),
	cel.Function("base64",
		cel.Overload("base64_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to base64_string", value.Type())
				}
				return types.String(base64.StdEncoding.EncodeToString([]byte(v)))
			}),
		),
		cel.Overload("base64_bytes",
			[]*cel.Type{cel.BytesType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.Bytes)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to base64_bytes", value.Type())
				}
				return types.String(base64.StdEncoding.EncodeToString(v))
			}),
		),
	),
	cel.Function("base64Decode",
		cel.Overload("base64Decode_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to base64Decode_string", value.Type())
				}
				decodeBytes, err := base64.StdEncoding.DecodeString(string(v))
				if err != nil {
					return types.NewErr("%v", err)
				}
				return types.String(decodeBytes)
			}),
		),
		cel.Overload("base64Decode_bytes",
			[]*cel.Type{cel.BytesType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.Bytes)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to base64Decode_bytes", value.Type())
				}
				decodeBytes, err := base64.StdEncoding.DecodeString(string(v))
				if err != nil {
					return types.NewErr("%v", err)
				}
				return types.String(decodeBytes)
			}),
		),
	),
	cel.Function("urlencode",
		cel.Overload("urlencode_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to urlencode_string", value.Type())
				}
				return types.String(url.QueryEscape(string(v)))
			}),
		),
		cel.Overload("urlencode_bytes",
			[]*cel.Type{cel.BytesType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.Bytes)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to urlencode_bytes", value.Type())
				}
				return types.String(url.QueryEscape(string(v)))
			}),
		),
	),
	cel.Function("urldecode",
		cel.Overload("urldecode_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to urldecode_string", value.Type())
				}
				decodeString, err := url.QueryUnescape(string(v))
				if err != nil {
					return types.NewErr("%v", err)
				}
				return types.String(decodeString)
			}),
		),
		cel.Overload("urldecode_bytes",
			[]*cel.Type{cel.BytesType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.Bytes)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to urldecode_bytes", value.Type())
				}
				decodeString, err := url.QueryUnescape(string(v))
				if err != nil {
					return types.NewErr("%v", err)
				}
				return types.String(decodeString)
			}),
		),
	),
	cel.Function("faviconHash",
		cel.Overload("faviconHash_stringOrBytes",
			[]*cel.Type{cel.DynType}, cel.IntType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				if b, ok := value.(types.Bytes); ok {
					return types.Int(common.Mmh3Hash32(common.Base64Encode(b)))
				}
				if bStr, ok := value.(types.String); ok {
					return types.Int(common.Mmh3Hash32(common.Base64Encode([]byte(bStr))))
				}
				return types.ValOrErr(value, "unexpected type '%v' passed to faviconHash", value.Type())
			}),
		),
	),
	cel.Function("hexdecode",
		cel.Overload("hexdecode_string",
			[]*cel.Type{cel.StringType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.String)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to hexdecode_string", value.Type())
				}
				dst := make([]byte, hex.DecodedLen(len(v)))
				n, err := hex.Decode(dst, []byte(v))
				if err != nil {
					return types.ValOrErr(value, "unexpected type '%s' passed to hexdecode_string", err.Error())
				}
				return types.String(dst[:n])
			}),
		),
	),
	// random
	cel.Function("randomInt",
		cel.Overload("randomInt_int_int",
			[]*cel.Type{cel.IntType, cel.IntType}, cel.IntType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				from, ok := lhs.(types.Int)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to randomInt", lhs.Type())
				}
				to, ok := rhs.(types.Int)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to randomInt", rhs.Type())
				}
				minStr, maxStr := int(from), int(to)
				return types.Int(rand.Intn(maxStr-minStr) + minStr)
			}),
		),
	),
	cel.Function("randomLowercase",
		cel.Overload("randomLowercase_int",
			[]*cel.Type{cel.IntType}, cel.StringType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				n, ok := value.(types.Int)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to randomLowercase", value.Type())
				}
				return types.String(common.RandLetters(int(n)))
			}),
		),
	),
	// regex
	cel.Function("bmatches",
		cel.MemberOverload("string_bmatches_bytes",
			[]*cel.Type{cel.StringType, cel.BytesType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				var isMatch = false
				var err error
				v1, ok := lhs.(types.String)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to bmatches", lhs.Type())
				}
				v2, ok := rhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to bmatches", rhs.Type())
				}
				re := regexp2.MustCompile(string(v1), 0)
				if isMatch, err = re.MatchString(string(v2)); err != nil {
					return types.NewErr("%v", err)
				}
				return types.Bool(isMatch)
			}),
		),
	),
	cel.Function("submatch",
		cel.MemberOverload("string_submatch_string",
			[]*cel.Type{cel.StringType, cel.StringType}, cel.MapType(cel.StringType, cel.StringType),
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				resultMap := make(map[string]string)
				v1, ok := lhs.(types.String)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to submatch", lhs.Type())
				}
				v2, ok := rhs.(types.String)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to submatch", rhs.Type())
				}
				re := regexp2.MustCompile(string(v1), regexp2.RE2)
				if m, _ := re.FindStringMatch(string(v2)); m != nil {
					gps := m.Groups()
					for n, gp := range gps {
						if n == 0 {
							continue
						}
						resultMap[gp.Name] = gp.String()
					}
				}
				return types.NewStringStringMap(types.DefaultTypeAdapter, resultMap)
			}),
		),
	),
	cel.Function("bsubmatch",
		cel.MemberOverload("string_bsubmatch_bytes",
			[]*cel.Type{cel.StringType, cel.BytesType}, cel.MapType(cel.StringType, cel.StringType),
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				resultMap := make(map[string]string)
				v1, ok := lhs.(types.String)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to bsubmatch", lhs.Type())
				}
				v2, ok := rhs.(types.Bytes)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to bsubmatch", rhs.Type())
				}
				re := regexp2.MustCompile(string(v1), regexp2.RE2)
				if m, _ := re.FindStringMatch(string(v2)); m != nil {
					gps := m.Groups()
					for n, gp := range gps {
						if n == 0 {
							continue
						}
						resultMap[gp.Name] = gp.String()
					}
				}
				return types.NewStringStringMap(types.DefaultTypeAdapter, resultMap)
			}),
		),
	),
	// reverse
	cel.Function("wait",
		cel.MemberOverload("reverse_wait_int",
			[]*cel.Type{cel.DynType, cel.IntType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				reverse, ok := lhs.Value().(*proto.Reverse)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to 'wait'", lhs.Type())
				}
				timeout, ok := rhs.Value().(int64)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to 'wait'", rhs.Type())
				}
				return types.Bool(reverseCheck(reverse, timeout))
			}),
		),
	),
	cel.Function("jndi",
		cel.MemberOverload("reverse_jndi_int",
			[]*cel.Type{cel.DynType, cel.IntType}, cel.BoolType,
			cel.BinaryBinding(func(lhs ref.Val, rhs ref.Val) ref.Val {
				reverse, ok := lhs.Value().(*proto.Reverse)
				if !ok {
					return types.ValOrErr(lhs, "unexpected type '%v' passed to 'wait'", lhs.Type())
				}
				timeout, ok := rhs.Value().(int64)
				if !ok {
					return types.ValOrErr(rhs, "unexpected type '%v' passed to 'wait'", rhs.Type())
				}
				return types.Bool(jndiCheck(reverse, timeout))
			}),
		),
	),
	// other
	cel.Function("sleep",
		cel.Overload("sleep_int",
			[]*cel.Type{cel.IntType}, cel.NullType,
			cel.UnaryBinding(func(value ref.Val) ref.Val {
				v, ok := value.(types.Int)
				if !ok {
					return types.ValOrErr(value, "unexpected type '%v' passed to sleep", value.Type())
				}
				time.Sleep(time.Duration(v) * time.Second)
				return types.NullValue
			}),
		),
	),
	// year helpers (保持与旧签名一致：接受一个 int 参数)
	cel.Function("year",
		cel.Overload("year_string",
			[]*cel.Type{cel.IntType}, cel.StringType,
			cel.UnaryBinding(func(_ ref.Val) ref.Val {
				year := time.Now().Format("2006")
				return types.String(year)
			}),
		),
	),
	cel.Function("shortyear",
		cel.Overload("shortyear_string",
			[]*cel.Type{cel.IntType}, cel.StringType,
			cel.UnaryBinding(func(_ ref.Val) ref.Val {
				year := time.Now().Format("06")
				return types.String(year)
			}),
		),
	),
	cel.Function("month",
		cel.Overload("month_string",
			[]*cel.Type{cel.IntType}, cel.StringType,
			cel.UnaryBinding(func(_ ref.Val) ref.Val {
				month := time.Now().Format("01")
				return types.String(month)
			}),
		),
	),
	cel.Function("day",
		cel.Overload("day_string",
			[]*cel.Type{cel.IntType}, cel.StringType,
			cel.UnaryBinding(func(_ ref.Val) ref.Val {
				day := time.Now().Format("02")
				return types.String(day)
			}),
		),
	),
	cel.Function("timestamp_second",
		cel.Overload("timestamp_second_string",
			[]*cel.Type{cel.IntType}, cel.StringType,
			cel.UnaryBinding(func(_ ref.Val) ref.Val {
				timestamp := strconv.FormatInt(time.Now().Unix(), 10)
				return types.String(timestamp)
			}),
		),
	),
}

// reverseCheck 检查反向连接
func reverseCheck(r *proto.Reverse, timeout int64) bool {
	if len(config.ReverseCeyeApiKey) == 0 || len(r.Domain) == 0 {
		return false
	}

	time.Sleep(time.Second * time.Duration(timeout))

	sub := strings.Split(r.Domain, ".")[0]
	urlStr := fmt.Sprintf("http://api.ceye.io/v1/records?token=%s&type=dns&filter=%s", config.ReverseCeyeApiKey, sub)

	resp, err := network.ReverseGet(urlStr)
	if err != nil {
		return false
	}

	if !bytes.Contains(resp, []byte(`"data": []`)) && bytes.Contains(resp, []byte(`{"code": 200`)) { // api返回结果不为空
		return true
	}

	if bytes.Contains(resp, []byte(`<title>503`)) { // api返回结果不为空
		return false
	}

	return false
}

// jndiCheck 检查 JNDI 连接
func jndiCheck(reverse *proto.Reverse, timeout int64) bool {
	if len(config.ReverseJndi) == 0 && len(config.ReverseApiPort) == 0 {
		return false
	}

	time.Sleep(time.Second * time.Duration(timeout))

	urlStr := fmt.Sprintf("http://%s:%s/?api=%s", reverse.Url.Domain, config.ReverseApiPort, reverse.Url.Path[1:])

	resp, err := network.ReverseGet(urlStr)
	if err != nil {
		return false
	}

	if strings.Contains(string(resp), "yes") {

		return true
	}

	return false
}
