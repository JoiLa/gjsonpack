package gjsonpack

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const tokenTrue int64 = -1

const tokenFalse int64 = -2

const tokenNull int64 = -3

const tokenEmptyString int64 = -4

const tokenUndefined int64 = -5

type dictionaryString []string
type dictionaryIntegers []string
type dictionaryFloat []float64

func (d dictionaryString) Len() int64 {
	return int64(len(d))
}
func (d dictionaryIntegers) Len() int64 {
	return int64(len(d))
}
func (d dictionaryFloat) Len() int64 {
	return int64(len(d))
}

type dictionary struct {
	Strings  dictionaryString
	Integers dictionaryIntegers
	Floats   dictionaryFloat
}

// 语法树数据结构体
type ast interface{}

// 语法树信息
type astInfo struct {
	Type  string
	Index int64
}

// Pack 主要对MAP、Struct进行压缩
func Pack(json interface{}) (string, error) {
	defer func() (string, error) {
		return "", errors.New("pack failed")
	}()
	var dictionaryObj dictionary
	dictionaryObj.Strings = make(dictionaryString, 0)
	dictionaryObj.Integers = make(dictionaryIntegers, 0)
	dictionaryObj.Floats = make(dictionaryFloat, 0)
	astTree, astTreeErr := recursiveAstBuilder(json, &dictionaryObj)
	if astTreeErr != nil {
		return "", astTreeErr
	}
	// A set of shorthands proxies for the length of the dictionaries
	var stringLength = dictionaryObj.Strings.Len()
	var integerLength = dictionaryObj.Integers.Len()
	var floatLength = dictionaryObj.Floats.Len()
	var packed = strings.Join(dictionaryObj.Strings, "|")
	packed += "^" + strings.Join(dictionaryObj.Integers, "|")
	packed += "^" + strings.Join(_arrayFloatToArrayString(dictionaryObj.Floats), "|")
	// And add the structure
	recursiveGeneratePackedStr, recursiveGeneratePackedErr := recursiveParser(astTree, stringLength, integerLength, floatLength)
	if recursiveGeneratePackedErr != nil {
		return "", recursiveGeneratePackedErr
	}
	packed += "^" + recursiveGeneratePackedStr
	return packed, nil
}

// Unpack 解压 packed 参数中的数据
func Unpack(packed string, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &json.InvalidUnmarshalError{Type: reflect.TypeOf(v)}
	}
	// A raw buffer
	var rawBuffers = strings.Split(packed, "^")
	dictionarySlice := make([]interface{}, 0)
	var buffer string
	// Add the strings values
	buffer = rawBuffers[0]
	if buffer != "" {
		bufferSlice := strings.Split(buffer, "|")
		bufferSliceLen := len(bufferSlice)
		for i := 0; i < bufferSliceLen; i++ {
			dictionarySlice = append(dictionarySlice, _decodeStr(bufferSlice[i]))
		}
	}
	// Add the integers values
	buffer = rawBuffers[1]
	if buffer != "" {
		bufferSlice := strings.Split(buffer, "|")
		bufferSliceLen := len(bufferSlice)
		for i := 0; i < bufferSliceLen; i++ {
			to10Hex, to10HexErr := _baseString36To10(bufferSlice[i])
			if to10HexErr != nil {
				return to10HexErr
			}
			dictionarySlice = append(dictionarySlice, to10Hex)
		}
	}
	// Add the floats values
	buffer = rawBuffers[2]
	if buffer != "" {
		bufferSlice := strings.Split(buffer, "|")
		bufferSliceLen := len(bufferSlice)
		for i := 0; i < bufferSliceLen; i++ {
			to10Hex, to10HexErr := strconv.ParseFloat(bufferSlice[i], 10)
			if to10HexErr != nil {
				return to10HexErr
			}
			dictionarySlice = append(dictionarySlice, to10Hex)
		}
	}
	// Tokenizer the structure
	buffer = rawBuffers[3]
	tokenSlice := make([]interface{}, 0)
	if buffer != "" {
		var number36 = ""
		bufferLen := int64(len(buffer))
		for i := int64(0); i < bufferLen; i++ {
			var symbol = _substr(buffer, i, 1)
			if symbol == "|" || symbol == "$" || symbol == "@" || symbol == "]" {
				if number36 != "" {
					to10Hex, to10HexErr := _baseString36To10(number36)
					if to10HexErr != nil {
						return to10HexErr
					}
					tokenSlice = append(tokenSlice, to10Hex)
					number36 = ""
				}
				if symbol != "|" {
					tokenSlice = append(tokenSlice, symbol)
				}
			} else {
				number36 += symbol
			}
		}
	}
	// A shorthand proxy for tokenSlice.length
	var tokenSliceLen = int64(len(tokenSlice))
	// The index of the next token to read
	var tokensIndex = int64(0)
	// A shorthand proxy for dictionarySlice.length
	var dictionarySliceLen = int64(len(dictionarySlice))
	// 递归解析
	unPackerParser, unPackerParserErr := recursiveUnPackerParser(dictionarySlice, tokenSlice, dictionarySliceLen, tokenSliceLen, &tokensIndex)
	if unPackerParserErr != nil {
		return unPackerParserErr
	}
	jsonBytes, jsonMarshalErr := json.Marshal(unPackerParser)
	if jsonMarshalErr != nil {
		return jsonMarshalErr
	}
	jsonUnmarshalErr := json.Unmarshal(jsonBytes, v)
	if jsonUnmarshalErr != nil {
		return jsonUnmarshalErr
	}
	return nil
}

// recursiveUnPackerParser 递归解析
func recursiveUnPackerParser(dictionarySlice, tokenSlice []interface{}, dictionarySliceLen, tokenSliceLen int64, tokenSliceIndex *int64) (interface{}, error) {
	defer func() (interface{}, error) {
		return nil, fmt.Errorf("%s", "unpacked error")
	}()
	// Maybe '$' (object) or '@' (array)
	refSymbol := reflect.ValueOf(tokenSlice[*tokenSliceIndex])
	refSymbolKind := refSymbol.Kind()
	if refSymbolKind != reflect.String {
		return nil, fmt.Errorf("Bad token %s isn't a type! ", refSymbolKind.String())
	}
	*tokenSliceIndex++
	var symbol = refSymbol.Interface().(string)
	switch symbol {
	case "@":
		// Parse an array
		var node = make([]interface{}, 0)
		for ; *tokenSliceIndex < tokenSliceLen; *tokenSliceIndex++ {
			var value = tokenSlice[*tokenSliceIndex]
			if value == "]" {
				return node, nil
			}
			if value == "@" || value == "$" {
				var recursiveUnPackerValue, recursiveUnPackerValueErr = recursiveUnPackerParser(dictionarySlice, tokenSlice, dictionarySliceLen, tokenSliceLen, tokenSliceIndex)
				if recursiveUnPackerValueErr != nil {
					return nil, recursiveUnPackerValueErr
				}
				node = append(node, recursiveUnPackerValue)
			} else {
				switch value {
				case tokenTrue:
					node = append(node, true)
					break
				case tokenFalse:
					node = append(node, false)
					break
				case tokenNull, tokenUndefined:
					node = append(node, nil)
					break
				case tokenEmptyString:
					node = append(node, "")
					break
				default:
					fetchIndex, fetchIndexExists := value.(int64)
					if !fetchIndexExists || fetchIndex > dictionarySliceLen {
						return nil, fmt.Errorf("Bad dictionary %v isn't a efficient range! ", value)
					}
					node = append(node, dictionarySlice[fetchIndex])
					break
				}
			}
		}
		return node, nil
	case "$":
		// Parse a object
		var node = make(map[string]interface{}, 0)
		for ; *tokenSliceIndex < tokenSliceLen; *tokenSliceIndex++ {
			var nodeKey = tokenSlice[*tokenSliceIndex]
			if nodeKey == "]" {
				return node, nil
			}
			var nodeKeyString string
			if nodeKey == tokenEmptyString {
				nodeKeyString = ""
			} else {
				fetchIndex, fetchIndexExists := nodeKey.(int64)
				if !fetchIndexExists || fetchIndex > dictionarySliceLen {
					return nil, fmt.Errorf("Bad dictionary %v isn't a efficient range! ", nodeKey)
				}
				nodeKeyString = dictionarySlice[fetchIndex].(string)
			}

			*tokenSliceIndex++
			var value = tokenSlice[*tokenSliceIndex]
			if value == "@" || value == "$" {
				var recursiveUnPackerValue, recursiveUnPackerValueErr = recursiveUnPackerParser(dictionarySlice, tokenSlice, dictionarySliceLen, tokenSliceLen, tokenSliceIndex)
				if recursiveUnPackerValueErr != nil {
					return nil, recursiveUnPackerValueErr
				}
				node[nodeKeyString] = recursiveUnPackerValue
			} else {
				switch value {
				case tokenTrue:
					node[nodeKeyString] = true
					break
				case tokenFalse:
					node[nodeKeyString] = false
					break
				case tokenNull, tokenUndefined:
					node[nodeKeyString] = nil
					break
				case tokenEmptyString:
					node[nodeKeyString] = ""
					break
				default:
					fetchIndex, fetchIndexExists := value.(int64)
					if !fetchIndexExists || fetchIndex > dictionarySliceLen {
						return nil, fmt.Errorf("Bad dictionary %v isn't a efficient range! ", value)
					}
					node[nodeKeyString] = dictionarySlice[fetchIndex]
					break
				}
			}
		}
		return node, nil
	}
	return nil, fmt.Errorf("Bad token %s isn't a type! ", refSymbolKind.String())
}

// recursiveParser 递归语法树解析器
func recursiveParser(item interface{}, stringLength, integerLength, floatLength int64) (string, error) {
	refItem := reflect.ValueOf(item)
	refItemKind := refItem.Kind()
	// If the item is Array, then is a object of
	// type [object Object] or [object Array]
	if refItemKind == reflect.Slice || refItemKind == reflect.Array {
		// The packed resulting
		itemSliceLen := refItem.Len()
		var packed string
		if itemSliceLen > 0 {
			packed = refItem.Index(0).Interface().(string)
		}
		if itemSliceLen > 1 {
			for i := 1; i < itemSliceLen; i++ {
				recursiveGeneratePackedStr, recursiveGeneratePackedErr := recursiveParser(refItem.Index(i).Interface(), stringLength, integerLength, floatLength)
				if recursiveGeneratePackedErr != nil {
					return "", recursiveGeneratePackedErr
				}
				packed += recursiveGeneratePackedStr + "|"
			}
		}
		if _substr(packed, int64(len(packed)-1), 1) == "|" {
			packed = _substr(packed, 0, int64(len(packed))-1)
		}
		packed += "]"
		return packed, nil
	}
	if refItemKind != reflect.Struct {
		return "", errors.New("The item is alien! ")
	}
	currentAstInfo, currentAstInfoExists := refItem.Interface().(astInfo)
	if !currentAstInfoExists {
		return "", errors.New("The item is alien! ")
	}
	// A shorthand proxies
	switch currentAstInfo.Type {
	case "strings":
		// Just return the base 36 of index
		return _baseInt10To36(currentAstInfo.Index), nil
	case "integers":
		// Return a base 36 of index plus stringLength offset
		return _baseInt10To36(stringLength + currentAstInfo.Index), nil
	case "floats":
		// Return a base 36 of index plus stringLength and integerLength offset
		return _baseInt10To36(stringLength + integerLength + currentAstInfo.Index), nil
	case "boolean", "null", "undefined", "empty":
		return strconv.FormatInt(currentAstInfo.Index, 10), nil
	}
	return "", errors.New("The item is alien! ")
}

// recursiveAstBuilder 递归语法树生成
func recursiveAstBuilder(item interface{}, dictionaryObj *dictionary) (ast, error) {
	refItem := reflect.ValueOf(item)
	refItemKind := refItem.Kind()
	if refItemKind == reflect.Invalid {
		// The item is null
		return astInfo{Type: "null", Index: tokenNull}, nil
	}
	switch refItemKind {
	case reflect.Slice, reflect.Array:
		// The item is Array Object
		astArray := make([]interface{}, 0)
		astArray = append(astArray, "@")
		itemSliceLen := refItem.Len()
		for i := 0; i < itemSliceLen; i++ {
			builderArrayAst, builderArrayAstErr := recursiveAstBuilder(refItem.Index(i).Interface(), dictionaryObj)
			if builderArrayAstErr != nil {
				return nil, builderArrayAstErr
			}
			astArray = append(astArray, builderArrayAst)
		}
		return astArray, nil
	case reflect.Map:
		astMap := make([]interface{}, 0)
		astMap = append(astMap, "$")
		for _, refNodeKey := range refItem.MapKeys() {
			builderMapKeyAst, builderMapKeyAstErr := recursiveAstBuilder(refNodeKey.Interface(), dictionaryObj)
			if builderMapKeyAstErr != nil {
				return nil, builderMapKeyAstErr
			}
			astMap = append(astMap, builderMapKeyAst)
			builderMapValueAst, builderMapValueAstErr := recursiveAstBuilder(refItem.MapIndex(refNodeKey).Interface(), dictionaryObj)
			if builderMapValueAstErr != nil {
				return nil, builderMapValueAstErr
			}
			astMap = append(astMap, builderMapValueAst)
		}
		return astMap, nil
	case reflect.Struct:
		// The item is Object
		astStruct := make([]interface{}, 0)
		astStruct = append(astStruct, "$")
		refItemType := refItem.Type()
		fieldLen := refItemType.NumField()
		for i := 0; i < fieldLen; i++ {
			objField := refItemType.Field(i)
			var structName string
			structJsonTagName := objField.Tag.Get("json")
			if structJsonTagName == "-" {
				continue
			}
			if structJsonTagName == "" {
				structJsonTagName = objField.Name
			}
			structName = structJsonTagName

			builderStructNameAst, builderStructNameAstErr := recursiveAstBuilder(structName, dictionaryObj)
			if builderStructNameAstErr != nil {
				return nil, builderStructNameAstErr
			}
			astStruct = append(astStruct, builderStructNameAst)
			builderStructValueAst, builderStructValueAstErr := recursiveAstBuilder(refItem.Field(i).Interface(), dictionaryObj)
			if builderStructValueAstErr != nil {
				return nil, builderStructValueAstErr
			}
			astStruct = append(astStruct, builderStructValueAst)
		}
		return astStruct, nil
	case reflect.String:
		// The item is String
		itemString := refItem.String()
		if len(itemString) <= 0 {
			// The item empty string
			return astInfo{Type: "empty", Index: tokenEmptyString}, nil
		}
		// The index of that word in the dictionary
		index := _indexOf(dictionaryObj.Strings, itemString)
		// If not, add to the dictionary and actualize the index
		if index == -1 {
			dictionaryObj.Strings = append(dictionaryObj.Strings, _encodeStr(itemString))
			index = dictionaryObj.Strings.Len() - 1
		}
		return astInfo{Type: "strings", Index: index}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// The item is integer
		return assertionIntegers(refItem.Int(), dictionaryObj), nil
	case reflect.Float32, reflect.Float64:
		// The item is float
		itemFloat := refItem.Float()
		// check number is integer
		if math.Mod(itemFloat, 1) == 0 {
			// The item is integer
			return assertionIntegers(int64(itemFloat), dictionaryObj), nil
		}
		// The item is float
		return assertionFloat(itemFloat, dictionaryObj), nil
	case reflect.Interface, reflect.Ptr:
		t1 := refItem.Elem().Interface()
		return recursiveAstBuilder(t1, dictionaryObj)
	case reflect.Bool:
		// The item is boolean
		var index int64
		if refItem.Bool() {
			index = tokenTrue
		} else {
			index = tokenFalse
		}
		return astInfo{Type: "boolean", Index: index}, nil
	}
	return nil, errors.New("Unexpected argument of type " + refItemKind.String())
}

// assertionIntegers 断言整数
func assertionIntegers(number int64, dictionaryObj *dictionary) astInfo {
	// The index of that number in the dictionary
	index := _indexOf(dictionaryObj.Integers, number)
	if index == -1 {
		// If not, add to the dictionary and actualize the index
		dictionaryObj.Integers = append(dictionaryObj.Integers, _baseInt10To36(number))
		index = int64(len(dictionaryObj.Integers) - 1)
	}
	return astInfo{Type: "integers", Index: index}
}

// assertionFloat 断言浮点数
func assertionFloat(number float64, dictionaryObj *dictionary) astInfo {
	// The index of that number in the dictionary
	index := _indexOf(dictionaryObj.Floats, number)
	if index == -1 {
		// If not, add to the dictionary and actualize the index
		dictionaryObj.Floats = append(dictionaryObj.Floats, number)
		index = int64(len(dictionaryObj.Floats) - 1)
	}
	return astInfo{Type: "floats", Index: index}
}

// _indexOf 寻找位置
func _indexOf(arr interface{}, value interface{}) int64 {
	switch arr.(type) {
	case dictionaryString:
		arrV := arr.(dictionaryString)
		arrVLen := int64(len(arrV))
		if arrVLen <= 0 {
			return -1
		}
		for i := int64(0); i < arrVLen; i++ {
			if arrV[i] == value {
				return i
			}
		}
		break
	case dictionaryIntegers:
		arrV := arr.(dictionaryIntegers)
		arrVLen := int64(len(arrV))
		if arrVLen <= 0 {
			return -1
		}
		for i := int64(0); i < arrVLen; i++ {
			if arrV[i] == value {
				return i
			}
		}
		break
	case dictionaryFloat:
		arrV := arr.(dictionaryFloat)
		arrVLen := int64(len(arrV))
		if arrVLen <= 0 {
			return -1
		}
		for i := int64(0); i < arrVLen; i++ {
			if arrV[i] == value {
				return i
			}
		}
		break
	default:
		return -1
	}
	return -1
}

// _encodeStr 字符串编码
func _encodeStr(str string) string {
	if str == "" {
		return str
	}
	regRule := `[\+ \|\^\%]`
	if ok, _ := regexp.MatchString(regRule, str); !ok {
		return str
	}
	reg, err := regexp.Compile(regRule)
	if err != nil {
		return str
	}
	return reg.ReplaceAllStringFunc(str, func(s string) string {
		switch s {
		case " ":
			return "+"
		case "+":
			return "%2B"
		case "|":
			return "%7C"
		case "^":
			return "%5E"
		case "%":
			return "%25"
		}
		return ""
	})
}

// _decodeStr 字符串解码
func _decodeStr(str string) string {
	if str == "" {
		return str
	}
	regRule := `\+|%2B|%7C|%5E|%25`
	if ok, _ := regexp.MatchString(regRule, str); !ok {
		return str
	}
	reg, err := regexp.Compile(regRule)
	if err != nil {
		return str
	}
	return reg.ReplaceAllStringFunc(str, func(s string) string {
		switch s {
		case "+":
			return " "
		case "%2B":
			return "+"
		case "%7C":
			return "|"
		case "%5E":
			return "^"
		case "%25":
			return "%"
		}
		return ""
	})
}

// _baseInt10To36 10进制转36进制
func _baseInt10To36(number int64) string {
	return strings.ToUpper(strconv.FormatInt(number, 36))
}

// _baseInt10To36 36进制转10进制
func _baseString36To10(number string) (int64, error) {
	return strconv.ParseInt(strings.ToLower(number), 36, 64)
}

// _arrayFloatToArrayString 数组类型转换
func _arrayFloatToArrayString(arr []float64) []string {
	arrLen := len(arr)
	arrString := make([]string, len(arr))
	if arrLen <= 0 {
		return arrString
	}
	for i := 0; i < arrLen; i++ {
		arrString[i] = _formatFloat(arr[i], 16)
	}
	return arrString
}

// _formatFloat 主要逻辑就是先乘，trunc之后再除回去，就达到了保留N位小数的效果
func _formatFloat(num float64, decimal int) string {
	// 默认乘1
	d := float64(1)
	if decimal > 0 {
		// 10的N次方
		d = math.Pow10(decimal)
	}
	// math.trunc作用就是返回浮点数的整数部分
	// 再除回去，小数点后无效的0也就不存在了
	return strconv.FormatFloat(math.Trunc(num*d)/d, 'f', -1, 64)
}

// Substr _substr()
func _substr(str string, start int64, length int64) string {
	if start < 0 || length < -1 {
		return str
	}
	switch {
	case length == -1:
		return str[start:]
	case length == 0:
		return ""
	}
	end := start + length
	if end > int64(len(str)) {
		end = int64(len(str))
	}
	return str[start:end]
}
