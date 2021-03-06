package claim

import (
	"strings"

	"github.com/cnabio/cnab-go/utils/crud"
)

// GetLastLogs returns the logs from the last time the installation was executed.
func GetLastLogs(p Provider, installation string) (string, bool, error) {
	claim, err := p.ReadLastClaim(installation)
	if err != nil {
		return "", false, err
	}
	return GetLogs(p, claim.ID)
}

// GetLogs returns the logs generated by a claim.
func GetLogs(p Provider, claimID string) (string, bool, error) {
	results, err := p.ReadAllResults(claimID)
	if err != nil {
		return "", false, err
	}
	for i := len(results) - 1; i >= 0; i-- { // Go through results in descending order
		result := results[i]
		if result.HasLogs() {
			logsOutput, err := p.ReadOutput(Claim{ID: result.ClaimID}, result, OutputInvocationImageLogs)
			if err != nil {
				// Gracefully handle the logs not being persisted anymore
				if strings.Contains(err.Error(), crud.ErrRecordDoesNotExist.Error()) {
					return "", false, nil
				}
				return "", false, err
			}

			return string(logsOutput.Value), true, nil
		}
	}

	return "", false, nil
}
