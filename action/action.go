package action

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
)

// stateful is there just to make callers of opFromClaims more readable
const stateful = false

// Action describes one of the primary actions that can be executed in CNAB.
//
// The actions are:
// - install
// - upgrade
// - uninstall
// - downgrade
// - status
type Action interface {
	// Run an action, and record the status in the given claim
	Run(*claim.Claim, credentials.Set, io.Writer) error
}

func golangTypeToJSONType(value interface{}) (string, error) {
	switch v := value.(type) {
	case nil:
		return "null", nil
	case bool:
		return "boolean", nil
	case float64:
		// All numeric values are parsed by JSON into float64s. When a value could be an integer, it could also be a number, so give the more specific answer.
		if math.Trunc(v) == v {
			return "integer", nil
		}
		return "number", nil
	case string:
		return "string", nil
	case map[string]interface{}:
		return "object", nil
	case []interface{}:
		return "array", nil
	default:
		return fmt.Sprintf("%T", value), fmt.Errorf("unsupported type: %T", value)
	}
}

func setOutputsOnClaim(claim *claim.Claim, outputs map[string]string) error {
	outputErrors := []string{}
	claim.Outputs = map[string]interface{}{}

	if claim.Bundle.Outputs != nil {
		for outputName, v := range claim.Bundle.Outputs.Fields {
			name := claim.Bundle.Outputs.Fields[outputName].Definition
			outputSchema := claim.Bundle.Definitions[name]

			var outputTypes []string
			outputType, ok, _ := outputSchema.GetType()
			if !ok { // there are multiple types
				var err error
				outputTypes, ok, err = outputSchema.GetTypes()
				if !ok {
					panic(err)
				}
			} else {
				outputTypes = []string{outputType}
			}

			mapOutputTypes := map[string]bool{}
			for _, thing := range outputTypes {
				mapOutputTypes[thing] = true
			}

			content := outputs[v.Path]
			if content != "" {
				var value interface{}

				if !mapOutputTypes["string"] { // String output types are always passed through as the escape hatch for non-JSON bundle outputs.
					err := json.Unmarshal([]byte(content), &value)
					if err != nil {
						outputErrors = append(outputErrors, fmt.Sprintf("failed to parse %q: %s", outputName, err))
					} else {
						v, err := golangTypeToJSONType(value)
						if err != nil {
							outputErrors = append(outputErrors, fmt.Sprintf("%q is not a known JSON type it is %q; expected one of: %s", outputName, v, strings.Join(outputTypes, ", ")))
						}
						switch {
						case mapOutputTypes[v]:
							break
						case v == "integer" && mapOutputTypes["number"]: // All integers make acceptable numbers, and our helper function provides the most specific type.
							break
						default:
							outputErrors = append(outputErrors, fmt.Sprintf("%q is not any of the expected types (%s) because it is %q", outputName, strings.Join(outputTypes, ", "), v))
						}
					}
				}
				claim.Outputs[outputName] = outputs[v.Path]
			}
		}
	}

	if len(outputErrors) > 0 {
		return fmt.Errorf("error: %s", outputErrors)
	}

	return nil
}

func selectInvocationImage(d driver.Driver, c *claim.Claim) (bundle.InvocationImage, error) {
	if len(c.Bundle.InvocationImages) == 0 {
		return bundle.InvocationImage{}, errors.New("no invocationImages are defined in the bundle")
	}

	for _, ii := range c.Bundle.InvocationImages {
		if d.Handles(ii.ImageType) {
			if c.RelocationMap != nil {
				if img, ok := c.RelocationMap[ii.Image]; ok {
					ii.Image = img
				}
			}
			return ii, nil
		}
	}

	return bundle.InvocationImage{}, errors.New("driver is not compatible with any of the invocation images in the bundle")
}

func getImageMap(b *bundle.Bundle) ([]byte, error) {
	imgs := b.Images
	if imgs == nil {
		imgs = make(map[string]bundle.Image)
	}
	return json.Marshal(imgs)
}

func appliesToAction(action string, parameter bundle.ParameterDefinition) bool {
	if len(parameter.ApplyTo) == 0 {
		return true
	}
	for _, act := range parameter.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
}

func opFromClaim(action string, stateless bool, c *claim.Claim, ii bundle.InvocationImage, creds credentials.Set, w io.Writer) (*driver.Operation, error) {
	env, files, err := creds.Expand(c.Bundle, stateless)
	if err != nil {
		return nil, err
	}

	// Quick verification that no params were passed that are not actual legit params.
	for key := range c.Parameters {
		if _, ok := c.Bundle.Parameters.Fields[key]; !ok {
			return nil, fmt.Errorf("undefined parameter %q", key)
		}
	}

	if c.Bundle.Parameters != nil {
		if err := injectParameters(action, c, env, files); err != nil {
			return nil, err
		}
	}

	imgMap, err := getImageMap(c.Bundle)
	if err != nil {
		return nil, fmt.Errorf("unable to generate image map: %s", err)
	}
	files["/cnab/app/image-map.json"] = string(imgMap)

	env["CNAB_INSTALLATION_NAME"] = c.Name
	env["CNAB_ACTION"] = action
	env["CNAB_BUNDLE_NAME"] = c.Bundle.Name
	env["CNAB_BUNDLE_VERSION"] = c.Bundle.Version

	var outputs []string
	if c.Bundle.Outputs != nil {
		for k := range c.Bundle.Outputs.Fields {
			outputs = append(outputs, c.Bundle.Outputs.Fields[k].Path)
		}
	}

	return &driver.Operation{
		Action:       action,
		Installation: c.Name,
		Parameters:   c.Parameters,
		Image:        ii.Image,
		ImageType:    ii.ImageType,
		Revision:     c.Revision,
		Environment:  env,
		Files:        files,
		Outputs:      outputs,
		Out:          w,
	}, nil
}

func injectParameters(action string, c *claim.Claim, env, files map[string]string) error {
	requiredMap := map[string]struct{}{}
	for _, key := range c.Bundle.Parameters.Required {
		requiredMap[key] = struct{}{}
	}
	for k, param := range c.Bundle.Parameters.Fields {
		rawval, ok := c.Parameters[k]
		if !ok {
			_, required := requiredMap[k]
			if required && appliesToAction(action, param) {
				return fmt.Errorf("missing required parameter %q for action %q", k, action)
			}
			continue
		}
		value := fmt.Sprintf("%v", rawval)
		if param.Destination == nil {
			// env is a CNAB_P_
			env[fmt.Sprintf("CNAB_P_%s", strings.ToUpper(k))] = value
			continue
		}
		if param.Destination.Path != "" {
			files[param.Destination.Path] = value
		}
		if param.Destination.EnvironmentVariable != "" {
			env[param.Destination.EnvironmentVariable] = value
		}
	}
	return nil
}
