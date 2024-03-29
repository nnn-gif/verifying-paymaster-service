package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ququzone/verifying-paymaster-service/api"
	"github.com/ququzone/verifying-paymaster-service/errors"
	"github.com/ququzone/verifying-paymaster-service/logger"
	"github.com/ququzone/verifying-paymaster-service/models"
)

func jsonrpcError(c *gin.Context, code int, message string, data any, id *float64) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"result":  nil,
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"data":    data,
		},
		"id": id,
	})
}

func Process(service interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" {
			jsonrpcError(c, -32700, "Parse error", "POST method excepted", nil)
			return
		}

		if nil == c.Request.Body {
			jsonrpcError(c, -32700, "Parse error", "No POST data", nil)
			return
		}

		key := c.Param("key")
		if key == "" {
			jsonrpcError(c, -32700, "Key error", "No key", nil)
			return
		}
		apiKey := &models.ApiKeys{}
		apiKey, err := apiKey.FindByKey(service.(*api.Signer).Container.GetRepository(), key)
		if nil != err {
			logger.S().Errorf("Query api error: %v", err)
			jsonrpcError(c, -32700, "Database error", "Query apikey error", nil)
			return
		}
		if apiKey == nil || !apiKey.Enable {
			jsonrpcError(c, -32700, "Key error", "Apikey error", nil)
			return
		}

		// reading POST data
		body, err := io.ReadAll(c.Request.Body)
		if nil != err {
			jsonrpcError(c, -32700, "Parse error", "Error while reading request body", nil)
			return
		}

		// try to decode JSON
		data := make(map[string]interface{})
		err = json.Unmarshal(body, &data)
		if nil != err {
			jsonrpcError(c, -32700, "Parse error", "Error parsing json request", nil)
			return
		}

		id, ok := data["id"].(float64)
		if !ok {
			jsonrpcError(c, -32600, "Invalid Request", "No or invalid 'id' in request", nil)
			return
		}

		if data["jsonrpc"] != "2.0" {
			jsonrpcError(c, -32600, "Invalid Request", "Version of jsonrpc is not 2.0", &id)
			return
		}

		method, ok := data["method"].(string)
		if !ok {
			jsonrpcError(c, -32600, "Invalid Request", "No or invalid 'method' in request", &id)
			return
		}

		params, ok := data["params"].([]interface{})
		if !ok {
			jsonrpcError(c, -32602, "Invalid params", "No or invalid 'params' in request", &id)
			return
		}

		call := reflect.ValueOf(service).MethodByName(cases.Title(language.Und, cases.NoLower).String(method))
		if !call.IsValid() {
			jsonrpcError(c, -32601, "Method not found", "Method not found", &id)
			return
		}

		// validating and converting params
		// if call.Type().NumIn() != len(params) {
		// 	jsonrpcError(c, -32602, "Invalid params", "Invalid number of params", &id)
		// 	return
		// }

		args := make([]reflect.Value, len(params))
		for i, arg := range params {

			switch call.Type().In(i).Kind() {
			case reflect.Float32:
				val, ok := arg.(float32)
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Float64:
				val, ok := arg.(float64)
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Int:
				val, ok := arg.(int)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = int(fval)
					}
				}

				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Int8:
				val, ok := arg.(int8)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = int8(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Int16:
				val, ok := arg.(int16)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = int16(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Int32:
				val, ok := arg.(int32)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = int32(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Int64:
				val, ok := arg.(int64)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = int64(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Interface:
				args[i] = reflect.ValueOf(arg)

			case reflect.Map:
				val, ok := arg.(map[string]any)
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Slice:
				val, ok := arg.([]interface{})
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.String:
				val, _ := arg.(string)
				// if !ok {
				// 	// jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
				// 	// return
				// }
				args[i] = reflect.ValueOf(val)

			case reflect.Uint:
				val, ok := arg.(uint)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = uint(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Uint8:
				val, ok := arg.(uint8)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = uint8(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Uint16:
				val, ok := arg.(uint16)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = uint16(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Uint32:
				val, ok := arg.(uint32)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = uint32(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			case reflect.Uint64:
				val, ok := arg.(uint64)
				if !ok {
					var fval float64
					fval, ok = arg.(float64)
					if ok {
						val = uint64(fval)
					}
				}
				if !ok {
					jsonrpcError(c, -32602, "Invalid params", fmt.Sprintf("Param [%d] can't be converted to %v", i, call.Type().In(i).String()), &id)
					return
				}
				args[i] = reflect.ValueOf(val)

			default:
				if !ok {
					jsonrpcError(c, -32603, "Internal error", "Invalid method defination", &id)
					return
				}
			}

		}

		c.Set("json-rpc-request", data)
		result := call.Call(args)

		if err, ok := result[len(result)-1].Interface().(error); ok && err != nil {
			rpcErr, ok := err.(*errors.RPCError)

			if ok {
				jsonrpcError(c, rpcErr.Code(), rpcErr.Error(), rpcErr.Data(), &id)
			} else {
				jsonrpcError(c, -32601, err.Error(), err.Error(), &id)
			}
		} else if len(result) > 0 {
			c.JSON(http.StatusOK, map[string]interface{}{
				"result":  result[0].Interface(),
				"jsonrpc": "2.0",
				"id":      id,
			})
		} else {
			c.JSON(http.StatusOK, map[string]interface{}{
				"result":  nil,
				"jsonrpc": "2.0",
				"id":      id,
			})
		}
	}
}
