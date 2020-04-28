package claim

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Installation is a non-storage construct representing an installation of a
// bundle.
type Installation struct {
	Name string
	Claims
}

// GetInstallationTimestamp searches the claims associated with the installation
// for the first claim for Install and returns its timestamp.
func (i Installation) GetInstallationTimestamp() (time.Time, error) {
	if len(i.Claims) == 0 {
		return time.Time{}, fmt.Errorf("the installation %s has no claims", i.Name)
	}

	sort.Sort(i.Claims)
	for _, c := range i.Claims {
		if c.Action == ActionInstall {
			return c.Created, nil
		}
	}

	return time.Time{}, fmt.Errorf("the installation %s has never been installed", i.Name)
}

// GetLastClaim returns the most recent (last) claim associated with the
// installation.
func (i Installation) GetLastClaim() (Claim, error) {
	if len(i.Claims) == 0 {
		return Claim{}, fmt.Errorf("the installation %s has no claims", i.Name)
	}

	sort.Sort(i.Claims)
	lastClaim := i.Claims[len(i.Claims)-1]
	return lastClaim, nil
}

// GetLastResult returns the most recent (last) result associated with the
// installation.
func (i Installation) GetLastResult() (Result, error) {
	lastClaim, err := i.GetLastClaim()
	if err != nil {
		return Result{}, err
	}

	if len(lastClaim.Results) == 0 {
		return Result{}, errors.New("the last claim has no results")
	}

	sort.Sort(lastClaim.Results)
	lastResult := lastClaim.Results[len(lastClaim.Results)-1]
	return lastResult, nil
}

// GetLastStatus returns the status from the most recent (last) result
// associated with the installation, or "unknown" if it cannot be determined.
func (i Installation) GetLastStatus() string {
	lastResult, err := i.GetLastResult()
	if err != nil {
		return StatusUnknown
	}

	return lastResult.Status
}

// GetLastOutputs returns the most recent (last) value of each output associated
// with the installation, sorted by the output name.
func (i Installation) GetLastOutputs(p Provider) (*Outputs, error) {
	var results Results

	claims, err := p.ReadAllClaims(i.Name)
	if err != nil {
		return nil, err
	}
	i.Claims = claims

	// TODO: (carolynvs) retrieve data concurrently
	for _, c := range i.Claims {
		resultIds, err := p.ListResults(c.ID)
		if err != nil {
			return nil, err
		}
		for _, resultID := range resultIds {
			results = append(results, Result{
				ID:      resultID,
				ClaimID: c.ID,
				Claim:   c,
			})
		}
	}

	// Determine the result that contains the final output value for each output
	// outputName -> resultID
	sort.Sort(results)
	lastOutputs := map[string]Result{}
	for _, result := range results {
		outputNames, err := p.ListOutputs(result.ID)
		if err != nil {
			return nil, err
		}
		for _, outputName := range outputNames {
			lastOutputs[outputName] = result
		}
	}

	outputs := NewOutputs()
	for outputName, result := range lastOutputs {
		output, err := p.ReadOutput(result.Claim, result, outputName)
		if err != nil {
			return nil, err
		}

		outputs.Append(output)
	}

	sort.Sort(outputs)
	return outputs, nil
}
