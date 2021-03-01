package action

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-go/claim"
	"github.com/cnabio/cnab-go/driver"
	"github.com/cnabio/cnab-go/valuesource"
)

// stateful is there just to make callers of opFromClaims more readable
const stateful = false

// Well known constants define the Well Known CNAB actions to be taken
const (
	ActionDryRun = "io.cnab.dry-run"
	ActionHelp   = "io.cnab.help"
	ActionLog    = "io.cnab.log"
	ActionStatus = "io.cnab.status"
)

// Action executes a bundle operation and helps save the results.
type Action struct {
	Claims         claim.Provider
	Driver         driver.Driver
	SaveAllOutputs bool
	SaveOutputs    []string
	SaveLogs       bool
}

// New creates an Action.
// - Driver to execute the operation with.
// - Claim Provider for persisting the claim data. If not set, calls to Save* will error.
func New(d driver.Driver, cp claim.Provider) Action {
	return Action{
		Claims: cp,
		Driver: d,
	}
}

// Run executes the action and records the status in a claim result. The
// caller is responsible for persisting the claim records and outputs using the
// SaveOperationResult function. An error is only returned when the operation could not
// be executed, otherwise any error is returned in the OperationResult.
func (a Action) Run(c claim.Claim, creds valuesource.Set, opCfgs ...OperationConfigFunc) (driver.OperationResult, claim.Result, error) {
	if a.Driver == nil {
		return driver.OperationResult{}, claim.Result{}, errors.New("the action driver is not set")
	}

	err := c.Validate()
	if err != nil {
		return driver.OperationResult{}, claim.Result{}, err
	}

	invocImage, err := a.selectInvocationImage(c)
	if err != nil {
		return driver.OperationResult{}, claim.Result{}, err
	}

	op, err := opFromClaim(stateful, c, invocImage, creds)
	if err != nil {
		return driver.OperationResult{}, claim.Result{}, err
	}

	err = OperationConfigs(opCfgs).ApplyConfig(op)
	if err != nil {
		return driver.OperationResult{}, claim.Result{}, err
	}

	logFile, err := a.captureLogs(op)
	if err != nil {
		return driver.OperationResult{}, claim.Result{}, err
	}

	var opErr *multierror.Error
	opResult, err := a.Driver.Run(op)
	if err != nil {
		opErr = multierror.Append(opErr, err)
	}

	err = a.saveLogs(logFile, opResult)
	if err != nil {
		opErr = multierror.Append(opErr, err)
	}

	err = opResult.SetDefaultOutputValues(*op)
	if err != nil {
		opErr = multierror.Append(opErr, err)
	}

	cr, err := buildClaimResult(c, opResult, opErr)
	if err != nil {
		opErr = multierror.Append(opErr, err)
	}

	// These are any errors from running the operation or processing the result,
	// We don't return it as an error because at this point the bundle has been
	// executed and we are returning results that should be persisted. We don't
	// want someone checking if an error occurred then ignoring the other return
	// values.
	opResult.Error = opErr.ErrorOrNil()

	return opResult, cr, nil
}

// captureLogs to a temporary file when action.SaveLogs is set.
func (a Action) captureLogs(op *driver.Operation) (*os.File, error) {
	if !a.SaveLogs {
		return nil, nil
	}

	logFile, err := ioutil.TempFile("", "cnab-logs")
	if err != nil {
		return nil, errors.Wrapf(err, "error creating temp log file")
	}

	op.Out = io.MultiWriter(op.Out, logFile)
	op.Err = io.MultiWriter(op.Err, logFile)
	return logFile, nil
}

// saveLogs as an output when action.SaveLogs is set.
func (a Action) saveLogs(logFile *os.File, opResult driver.OperationResult) error {
	if logFile == nil {
		return nil
	}

	defer func() {
		logFile.Close()
		os.Remove(logFile.Name())
	}()

	_, logOutputNameInUse := opResult.Outputs[claim.OutputInvocationImageLogs]
	if logOutputNameInUse {
		// The bundle is using our reserved log output name, so skip saving the logs
	}

	// Read from the beginning of the file
	_, err := logFile.Seek(0, io.SeekStart)
	if err != nil {
		return errors.Wrapf(err, "error seeking the log file")
	}

	logsB, err := ioutil.ReadAll(logFile)
	if err != nil {
		return errors.Wrapf(err, "error reading log file %s", logFile.Name())
	}
	if opResult.Outputs == nil {
		opResult.Outputs = make(map[string]string)
	}
	opResult.Outputs[claim.OutputInvocationImageLogs] = string(logsB)

	return nil
}

// SaveInitialClaim with the specified status. If not used, the caller is
// responsible for persisting the claim.
func (a Action) SaveInitialClaim(c claim.Claim, status string) error {
	if a.Claims == nil {
		return errors.New("the action claims provider is not set")
	}

	err := a.saveClaimWithStatus(c, status)
	return errors.Wrap(err, "could not save the pending action's status, the bundle was not executed")
}

// SaveOperationResult saves the ClaimResult and Outputs. The caller is
// responsible for having already persisted the claim itself, for example using
// SaveInitialClaim.
func (a Action) SaveOperationResult(opResult driver.OperationResult, c claim.Claim, r claim.Result) error {
	if a.Claims == nil {
		return errors.New("the action claims provider is not set")
	}

	// Keep accumulating errors from any error returned from the operation
	// We must save the claim even when the op failed, but we want to report
	// ALL errors back.
	var bigerr *multierror.Error
	bigerr = multierror.Append(bigerr, opResult.Error)

	err := a.Claims.SaveResult(r)
	if err != nil {
		bigerr = multierror.Append(bigerr, err)
	}

	for outputName, outputValue := range opResult.Outputs {
		if !a.shouldSaveOutput(outputName) {
			continue
		}

		output := claim.NewOutput(c, r, outputName, []byte(outputValue))
		err = a.Claims.SaveOutput(output)
		if err != nil {
			bigerr = multierror.Append(bigerr, err)
		}
	}

	return bigerr.ErrorOrNil()
}

func (a Action) shouldSaveOutput(name string) bool {
	if a.SaveAllOutputs {
		return true
	}
	for _, output := range a.SaveOutputs {
		if name == output {
			return true
		}
	}
	return false
}

// saveClaimWithStatus saves a claim and a result with the specified status.
func (a Action) saveClaimWithStatus(c claim.Claim, status string) error {
	r, err := c.NewResult(status)
	if err != nil {
		return err
	}

	err = r.Validate()
	if err != nil {
		return err
	}

	err = a.Claims.SaveClaim(c)
	if err != nil {
		return err
	}

	return a.Claims.SaveResult(r)
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

// allowedTypes takes an output Schema and returns a map of the allowed types (to true)
// or an error (if reading the allowed types from the schema failed).
func allowedTypes(outputSchema definition.Schema) (map[string]bool, error) {
	var outputTypes []string
	mapOutputTypes := map[string]bool{}

	// Get list of one or more allowed types for this output
	outputType, ok, err1 := outputSchema.GetType()
	if !ok { // there are multiple types
		var err2 error
		outputTypes, ok, err2 = outputSchema.GetTypes()
		if !ok {
			return mapOutputTypes, fmt.Errorf("Getting a single type errored with %q and getting multiple types errored with %q", err1, err2)
		}
	} else {
		outputTypes = []string{outputType}
	}

	// Turn allowed outputs into map for easier membership checking
	for _, v := range outputTypes {
		mapOutputTypes[v] = true
	}

	// All integers make acceptable numbers, and our helper function provides the most specific type.
	if mapOutputTypes["number"] {
		mapOutputTypes["integer"] = true
	}

	return mapOutputTypes, nil
}

// keys takes a map and returns the keys joined into a comma-separate string.
func keys(stringMap map[string]bool) string {
	var keys []string
	for k := range stringMap {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

// isTypeOK uses the content and allowedTypes arguments to make sure the content of an output file matches one of the allowed types.
// The other arguments (name and allowedTypesList) are used when assembling error messages.
func isTypeOk(name, content string, allowedTypes map[string]bool) error {
	if !allowedTypes["string"] { // String output types are always passed through as the escape hatch for non-JSON bundle outputs.
		var value interface{}
		if err := json.Unmarshal([]byte(content), &value); err != nil {
			return fmt.Errorf("failed to parse %q: %s", name, err)
		}

		v, err := golangTypeToJSONType(value)
		if err != nil {
			return fmt.Errorf("%q is not a known JSON type. Expected %q to be one of: %s", name, v, keys(allowedTypes))
		}
		if !allowedTypes[v] {
			return fmt.Errorf("%q is not any of the expected types (%s) because it is %q", name, keys(allowedTypes), v)
		}
	}
	return nil
}

// buildClaimResult from the result of executing a bundle operation.
// A result is _always_ returned, even when an error is returned.
func buildClaimResult(c claim.Claim, opResult driver.OperationResult, opErr *multierror.Error) (result claim.Result, err error) {
	if accErr := opErr.ErrorOrNil(); accErr != nil {
		result, err = c.NewResult(claim.StatusFailed)
		if err == nil {
			result.Message = accErr.Error()
		}
	} else {
		result, err = c.NewResult(claim.StatusSucceeded)
	}

	if err != nil {
		return claim.Result{}, err
	}

	err = setOutputsOnClaimResult(c, &result, opResult)

	return result, err
}

// setOutputsOnClaimResult updates the result with the name and metadata of each output generated by
// the operation.
// Metadata:
// - contentDigest: string
// - generatedByBundle: boolean
func setOutputsOnClaimResult(c claim.Claim, result *claim.Result, opResult driver.OperationResult) error {
	var outputErrors []error

	for outputName, outputValue := range opResult.Outputs {
		outputDef, isDefined := c.Bundle.Outputs[outputName]
		result.OutputMetadata.SetGeneratedByBundle(outputName, isDefined)
		if isDefined {
			err := validateOutputType(c.Bundle, outputName, outputDef, outputValue)
			if err != nil {
				outputErrors = append(outputErrors, err)
			}
		}

		if outputValue != "" {
			result.OutputMetadata.SetContentDigest(outputName, buildOutputContentDigest(outputValue))
		}
	}

	if len(outputErrors) > 0 {
		return fmt.Errorf("error: %s", outputErrors)
	}

	return nil
}

// validateOutputType checks that the type of the output matches the output's defined type.
func validateOutputType(bundle bundle.Bundle, outputName string, outputDef bundle.Output, outputValue string) error {
	name := outputDef.Definition
	if name == "" {
		return fmt.Errorf("invalid bundle: no definition set for output %q", outputName)
	}

	outputSchema := bundle.Definitions[name]
	if outputSchema == nil {
		return fmt.Errorf("invalid bundle: output %q references definition %q, which was not found", outputName, name)
	}
	outputTypes, err := allowedTypes(*outputSchema)
	if err != nil {
		return err
	}

	if outputValue != "" {
		err := isTypeOk(outputName, outputValue, outputTypes)
		if err != nil {
			return err
		}
	}
	return nil
}

// buildOutputContentDigest generates the contentDigest metadata string for an output
// Example: sha256:6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090
func buildOutputContentDigest(outputValue string) string {
	digestB := sha256.Sum256([]byte(outputValue))
	digest := hex.EncodeToString(digestB[:])
	return fmt.Sprintf("sha256:%s", digest)
}

func (a Action) selectInvocationImage(c claim.Claim) (bundle.InvocationImage, error) {
	if len(c.Bundle.InvocationImages) == 0 {
		return bundle.InvocationImage{}, errors.New("no invocationImages are defined in the bundle")
	}

	for _, ii := range c.Bundle.InvocationImages {
		if a.Driver.Handles(ii.ImageType) {
			return ii, nil
		}
	}

	return bundle.InvocationImage{}, errors.New("driver is not compatible with any of the invocation images in the bundle")
}

func getImageMap(b bundle.Bundle) ([]byte, error) {
	imgs := b.Images
	if imgs == nil {
		imgs = make(map[string]bundle.Image)
	}
	return json.Marshal(imgs)
}

func opFromClaim(stateless bool, c claim.Claim, ii bundle.InvocationImage, creds valuesource.Set) (*driver.Operation, error) {
	env, files, err := expandCredentials(c.Bundle, creds, stateless, c.Action)
	if err != nil {
		return nil, err
	}

	// Quick verification that no params were passed that are not actual legit params.
	for key := range c.Parameters {
		if _, ok := c.Bundle.Parameters[key]; !ok {
			return nil, fmt.Errorf("undefined parameter %q", key)
		}
	}

	if err := injectParameters(c, env, files); err != nil {
		return nil, err
	}

	bundleBytes, err := json.Marshal(c.Bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bundle contents: %s", err)
	}
	files["/cnab/bundle.json"] = string(bundleBytes)

	imgMap, err := getImageMap(c.Bundle)
	if err != nil {
		return nil, fmt.Errorf("unable to generate image map: %s", err)
	}
	files["/cnab/app/image-map.json"] = string(imgMap)

	claimBytes, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claim: %s", err)
	}
	files["/cnab/claim.json"] = string(claimBytes)

	env["CNAB_ACTION"] = c.Action
	env["CNAB_BUNDLE_NAME"] = c.Bundle.Name
	env["CNAB_BUNDLE_VERSION"] = c.Bundle.Version
	env["CNAB_CLAIMS_VERSION"] = string(c.SchemaVersion)
	env["CNAB_INSTALLATION_NAME"] = c.Installation
	env["CNAB_REVISION"] = c.Revision

	return &driver.Operation{
		Action:       c.Action,
		Installation: c.Installation,
		Parameters:   c.Parameters,
		Image:        ii,
		Revision:     c.Revision,
		Environment:  env,
		Files:        files,
		Outputs:      getOutputsGeneratedByAction(c.Action, c.Bundle),
		Bundle:       &c.Bundle,
	}, nil
}

// getOutputsGeneratedByAction returns a map of output paths to the name of the output, filtered by the specified action.
func getOutputsGeneratedByAction(action string, b bundle.Bundle) map[string]string {
	outputs := make(map[string]string, len(b.Outputs))
	for outputName, outputDef := range b.Outputs {
		if !outputDef.AppliesTo(action) {
			continue
		}

		outputs[outputDef.Path] = outputName
	}

	return outputs
}

func injectParameters(c claim.Claim, env, files map[string]string) error {
	for k, param := range c.Bundle.Parameters {
		rawval, ok := c.Parameters[k]
		if !ok {
			if param.Required && param.AppliesTo(c.Action) {
				return fmt.Errorf("missing required parameter %q for action %q", k, c.Action)
			}
			continue
		}

		contents, err := json.Marshal(rawval)
		if err != nil {
			return err
		}

		// In order to preserve the exact string value the user provided
		// we don't marshal string parameters
		value := string(contents)
		if value[0] == '"' {
			value, ok = rawval.(string)
			if !ok {
				return fmt.Errorf("failed to parse parameter %q as string", k)
			}
		}

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

// expandCredentials expands the given set into env vars and paths per the spec in the bundle.
//
// This matches the credentials required by the bundle to the credentials present
// in the Set, and then expands them per the definition in the Bundle.
func expandCredentials(b bundle.Bundle, set valuesource.Set, stateless bool, action string) (env, files map[string]string, err error) {
	env, files = map[string]string{}, map[string]string{}
	for name, val := range b.Credentials {
		src, ok := set[name]
		if !ok {
			if stateless || !val.Required || !val.AppliesTo(action) {
				continue
			}
			err = fmt.Errorf("credential %q is missing from the user-supplied credentials", name)
			return
		}
		if val.EnvironmentVariable != "" {
			env[val.EnvironmentVariable] = src
		}
		if val.Path != "" {
			files[val.Path] = src
		}
	}
	return
}
