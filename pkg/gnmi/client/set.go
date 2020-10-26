package gnmiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/openconfig/gnmi/proto/gnmi"
	"gopkg.in/yaml.v2"
)

var vTypes = []string{"json", "json_ietf", "string", "int", "uint", "bool", "decimal", "float", "bytes", "ascii"}

// SetCmdInput type holds set command input
type SetCmdInput struct {
	Deletes  []string
	Updates  []string
	Replaces []string

	UpdatePaths  []string
	ReplacePaths []string

	UpdateFiles  []string
	ReplaceFiles []string

	UpdateValues  []string
	ReplaceValues []string
}

// CreateSetRequest
func (g *GnmiClient) CreateSetRequest(setInput *SetCmdInput) (*gnmi.SetRequest, error) {
	gnmiPrefix, err := CreatePrefix("", g.Target)
	if err != nil {
		return nil, fmt.Errorf("prefix parse error: %v", err)
	}

	err = validateSetInput(setInput)
	if err != nil {
		return nil, err
	}

	//
	useUpdateFiles := len(setInput.UpdateFiles) > 0 && len(setInput.UpdateValues) == 0
	useReplaceFiles := len(setInput.ReplaceFiles) > 0 && len(setInput.ReplaceValues) == 0
	req := &gnmi.SetRequest{
		Prefix:  gnmiPrefix,
		Delete:  make([]*gnmi.Path, 0, len(setInput.Deletes)),
		Replace: make([]*gnmi.Update, 0),
		Update:  make([]*gnmi.Update, 0),
	}
	for _, p := range setInput.Deletes {
		gnmiPath, err := ParsePath(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		req.Delete = append(req.Delete, gnmiPath)
	}
	delimiter := ":::"
	for _, u := range setInput.Updates {
		singleUpdate := strings.Split(u, delimiter)
		if len(singleUpdate) < 3 {
			return nil, fmt.Errorf("invalid inline update format: %s", setInput.Updates)
		}
		gnmiPath, err := ParsePath(strings.TrimSpace(singleUpdate[0]))
		if err != nil {
			return nil, err
		}
		value := new(gnmi.TypedValue)
		err = setValue(value, singleUpdate[1], singleUpdate[2])
		if err != nil {
			return nil, err
		}
		req.Update = append(req.Update, &gnmi.Update{
			Path: gnmiPath,
			Val:  value,
		})
	}
	for _, r := range setInput.Replaces {
		singleReplace := strings.Split(r, delimiter)
		if len(singleReplace) < 3 {
			return nil, fmt.Errorf("invalid inline replace format: %s", setInput.Replaces)
		}
		gnmiPath, err := ParsePath(strings.TrimSpace(singleReplace[0]))
		if err != nil {
			return nil, err
		}
		value := new(gnmi.TypedValue)
		err = setValue(value, singleReplace[1], singleReplace[2])
		if err != nil {
			return nil, err
		}
		req.Replace = append(req.Replace, &gnmi.Update{
			Path: gnmiPath,
			Val:  value,
		})
	}
	for i, p := range setInput.UpdatePaths {
		gnmiPath, err := ParsePath(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		value := new(gnmi.TypedValue)
		if useUpdateFiles {
			var updateData []byte
			updateData, err = readFile(setInput.UpdateFiles[i])
			if err != nil {
				//logger.Printf("error reading data from file '%s': %v", setInput.updateFiles[i], err)
				return nil, err
			}
			switch strings.ToUpper(g.Encoding) {
			case "JSON":
				value.Value = &gnmi.TypedValue_JsonVal{
					JsonVal: bytes.Trim(updateData, " \r\n\t"),
				}
			case "JSON_IETF":
				value.Value = &gnmi.TypedValue_JsonIetfVal{
					JsonIetfVal: bytes.Trim(updateData, " \r\n\t"),
				}
			default:
				return nil, fmt.Errorf("encoding: %s not supported together with file values", g.Encoding)
			}
		} else {
			err = setValue(value, strings.ToLower(g.Encoding), setInput.UpdateValues[i])
			if err != nil {
				return nil, err
			}
		}
		req.Update = append(req.Update, &gnmi.Update{
			Path: gnmiPath,
			Val:  value,
		})
	}
	for i, p := range setInput.ReplacePaths {
		gnmiPath, err := ParsePath(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		value := new(gnmi.TypedValue)
		if useReplaceFiles {
			var replaceData []byte
			replaceData, err = readFile(setInput.ReplaceFiles[i])
			if err != nil {
				//logger.Printf("error reading data from file '%s': %v", setInput.replaceFiles[i], err)
				return nil, err
			}
			switch strings.ToUpper(g.Encoding) {
			case "JSON":
				value.Value = &gnmi.TypedValue_JsonVal{
					JsonVal: bytes.Trim(replaceData, " \r\n\t"),
				}
			case "JSON_IETF":
				value.Value = &gnmi.TypedValue_JsonIetfVal{
					JsonIetfVal: bytes.Trim(replaceData, " \r\n\t"),
				}
			default:
				return nil, fmt.Errorf("encoding: %s not supported together with file values", g.Encoding)
			}
		} else {
			err = setValue(value, "json", setInput.ReplaceValues[i])
			if err != nil {
				return nil, err
			}
		}
		req.Replace = append(req.Replace, &gnmi.Update{
			Path: gnmiPath,
			Val:  value,
		})
	}
	return req, nil
}

// readFile reads a json or yaml file. the the file is .yaml, converts it to json and returns []byte and an error
func readFile(name string) ([]byte, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	switch filepath.Ext(name) {
	case ".json":
		return data, err
	case ".yaml", ".yml":
		var out interface{}
		err = yaml.Unmarshal(data, &out)
		if err != nil {
			return nil, err
		}
		newStruct := convert(out)
		newData, err := json.Marshal(newStruct)
		if err != nil {
			return nil, err
		}
		return newData, nil
	default:
		return nil, fmt.Errorf("unsupported file format %s", filepath.Ext(name))
	}
}

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		nm := map[string]interface{}{}
		for k, v := range x {
			nm[k.(string)] = convert(v)
		}
		return nm
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}

func setValue(value *gnmi.TypedValue, typ, val string) error {
	var err error
	switch typ {
	case "json":
		buff := new(bytes.Buffer)
		err = json.NewEncoder(buff).Encode(strings.TrimRight(strings.TrimLeft(val, "["), "]"))
		if err != nil {
			return err
		}
		value.Value = &gnmi.TypedValue_JsonVal{
			JsonVal: bytes.Trim(buff.Bytes(), " \r\n\t"),
		}
	case "json_ietf":
		buff := new(bytes.Buffer)
		err = json.NewEncoder(buff).Encode(strings.TrimRight(strings.TrimLeft(val, "["), "]"))
		if err != nil {
			return err
		}
		value.Value = &gnmi.TypedValue_JsonIetfVal{
			JsonIetfVal: bytes.Trim(buff.Bytes(), " \r\n\t"),
		}
	case "ascii":
		value.Value = &gnmi.TypedValue_AsciiVal{
			AsciiVal: val,
		}
	case "bool":
		bval, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		value.Value = &gnmi.TypedValue_BoolVal{
			BoolVal: bval,
		}
	case "bytes":
		value.Value = &gnmi.TypedValue_BytesVal{
			BytesVal: []byte(val),
		}
	case "decimal":
		dVal := &gnmi.Decimal64{}
		value.Value = &gnmi.TypedValue_DecimalVal{
			DecimalVal: dVal,
		}
		return fmt.Errorf("decimal type not implemented")
	case "float":
		f, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		value.Value = &gnmi.TypedValue_FloatVal{
			FloatVal: float32(f),
		}
	case "int":
		k, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		value.Value = &gnmi.TypedValue_IntVal{
			IntVal: k,
		}
	case "uint":
		u, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		value.Value = &gnmi.TypedValue_UintVal{
			UintVal: u,
		}
	case "string":
		value.Value = &gnmi.TypedValue_StringVal{
			StringVal: val,
		}
	default:
		return fmt.Errorf("unknown type '%s', must be one of: %v", typ, vTypes)
	}
	return nil
}

func validateSetInput(setInput *SetCmdInput) error {
	if (len(setInput.Deletes)+len(setInput.Updates)+len(setInput.Replaces)) == 0 && (len(setInput.UpdatePaths)+len(setInput.ReplacePaths)) == 0 {
		return errors.New("no paths provided")
	}
	if len(setInput.UpdateFiles) > 0 && len(setInput.UpdateValues) > 0 {
		return errors.New("set update from file and value are not supported in the same command")
	}
	if len(setInput.ReplaceFiles) > 0 && len(setInput.ReplaceValues) > 0 {
		return errors.New("set replace from file and value are not supported in the same command")
	}
	if len(setInput.UpdatePaths) != len(setInput.UpdateValues) && len(setInput.UpdatePaths) != len(setInput.UpdateFiles) {
		return errors.New("missing update value/file or path")
	}
	if len(setInput.ReplacePaths) != len(setInput.ReplaceValues) && len(setInput.ReplacePaths) != len(setInput.ReplaceFiles) {
		return errors.New("missing replace value/file or path")
	}
	return nil
}
