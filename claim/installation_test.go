package claim

import (
	"testing"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"

	"github.com/cnabio/cnab-go/utils/crud"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallation_GetInstallationTimestamp(t *testing.T) {
	const installationName = "test"
	upgrade, err := New(installationName, ActionUpgrade, exampleBundle, nil)
	require.NoError(t, err)
	install1, err := New(installationName, ActionInstall, exampleBundle, nil)
	require.NoError(t, err)
	install2, err := New(installationName, ActionInstall, exampleBundle, nil)
	require.NoError(t, err)

	t.Run("has claims", func(t *testing.T) {
		i := Installation{
			Name:   installationName,
			Claims: Claims{upgrade, install1, install2},
		}

		installTime, err := i.GetInstallationTimestamp()
		require.NoError(t, err, "GetInstallationTimestamp failed")
		assert.Equal(t, install1.Created, installTime, "invalid installation time")
	})
	t.Run("no claims", func(t *testing.T) {
		i := Installation{Name: installationName}

		_, err := i.GetInstallationTimestamp()
		require.EqualError(t, err, "the installation test has no claims")
	})
}

func TestInstallation_GetLastClaim(t *testing.T) {
	upgrade := Claim{
		ID:     "2",
		Action: ActionUpgrade,
		Results: Results{
			{
				ID:     "1",
				Status: StatusRunning,
			},
		},
	}
	install := Claim{
		ID:     "1",
		Action: ActionInstall,
		Results: Results{
			{
				ID:     "1",
				Status: StatusSucceeded,
			},
		},
	}

	t.Run("claim exists", func(t *testing.T) {
		i := Installation{
			Name: "wordpress",
			Claims: Claims{
				upgrade,
				install,
			},
		}

		c, err := i.GetLastClaim()

		require.NoError(t, err, "GetLastClaim failed")
		assert.Equal(t, upgrade, c, "GetLastClaim did not return the expected claim")
	})

	t.Run("no claims", func(t *testing.T) {
		i := Installation{
			Name: "wordpress",
		}

		c, err := i.GetLastClaim()

		require.EqualError(t, err, "the installation wordpress has no claims")
		assert.Equal(t, Claim{}, c, "should return an empty claim when one cannot be found")
	})

}

func TestInstallation_GetLastResult(t *testing.T) {
	failed := Result{
		ID:     "2",
		Status: StatusFailed,
	}
	upgrade := Claim{
		ID:     "2",
		Action: ActionUpgrade,
		Results: Results{
			failed,
			{
				ID:     "1",
				Status: StatusRunning,
			},
		},
	}
	install := Claim{
		ID:     "1",
		Action: ActionInstall,
		Results: Results{
			{
				ID:     "1",
				Status: StatusSucceeded,
			},
		},
	}

	t.Run("result exists", func(t *testing.T) {
		i := Installation{
			Name: "wordpress",
			Claims: Claims{
				upgrade,
				install,
			},
		}

		r, err := i.GetLastResult()

		require.NoError(t, err, "GetLastResult failed")
		assert.Equal(t, failed, r, "GetLastResult did not return the expected result")
		assert.Equal(t, StatusFailed, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})

	t.Run("no claims", func(t *testing.T) {
		i := Installation{
			Name: "wordpress",
		}

		r, err := i.GetLastResult()

		require.EqualError(t, err, "the installation wordpress has no claims")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})

	t.Run("no results", func(t *testing.T) {
		i := Installation{
			Name: "wordpress",
			Claims: Claims{
				Claim{
					ID: "1",
				},
			},
		}

		r, err := i.GetLastResult()

		require.EqualError(t, err, "the last claim has no results")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})
}

func TestInstallation_GetLastOutputs(t *testing.T) {
	cp := NewClaimStore(crud.NewMockStore(), nil, nil)

	b := bundle.Bundle{
		Name: "mybun",
		Definitions: map[string]*definition.Schema{
			"output1": {
				Type: "string",
			},
			"output2": {
				Type: "string",
			},
		},
		Outputs: map[string]bundle.Output{
			"output1": {
				Definition: "output1",
			},
			"output2": {
				Definition: "output2",
				ApplyTo:    []string{"upgrade"},
			},
		},
	}

	// Generate claim data and outputs from install and upgrade
	const installationName = "test"
	installClaim, err := New(installationName, ActionInstall, b, nil)
	require.NoError(t, err)
	err = cp.SaveClaim(installClaim)
	require.NoError(t, err)
	installResult, err := installClaim.NewResult(StatusSucceeded)
	require.NoError(t, err)
	err = cp.SaveResult(installResult)
	require.NoError(t, err)
	installOutput1 := Output{
		Claim:  installClaim,
		Result: installResult,
		Name:   "output1",
		Value:  []byte("install output1"),
	}
	err = cp.SaveOutput(installOutput1)
	require.NoError(t, err)

	upgradeClaim, err := installClaim.NewClaim(ActionUpgrade, installClaim.Bundle, nil)
	require.NoError(t, err)
	err = cp.SaveClaim(upgradeClaim)
	require.NoError(t, err)
	upgradeResult, err := upgradeClaim.NewResult(StatusSucceeded)
	require.NoError(t, err)
	err = cp.SaveResult(upgradeResult)
	require.NoError(t, err)
	upgradeOutput1 := Output{
		Claim:  upgradeClaim,
		Result: upgradeResult,
		Name:   "output1",
		Value:  []byte("upgrade output1"),
	}
	err = cp.SaveOutput(upgradeOutput1)
	require.NoError(t, err)
	upgradeOutput2 := Output{
		Claim:  upgradeClaim,
		Result: upgradeResult,
		Name:   "output2",
		Value:  []byte("upgrade output2"),
	}
	err = cp.SaveOutput(upgradeOutput2)
	require.NoError(t, err)

	i := Installation{Name: installationName}
	outputs, err := i.GetLastOutputs(cp)

	require.NoError(t, err, "GetLastOutputs failed")
	require.NotNil(t, outputs, "did not get any outputs")
	assert.Equal(t, 2, outputs.Len(), "wrong number of outputs identified")

	gotOutput1, hasOutput1 := outputs.GetByName("output1")
	assert.True(t, hasOutput1, "should have found output1")
	assert.Equal(t, "upgrade output1", string(gotOutput1.Value), "did not find the most recent value for output1")

	gotOutput2, hasOutput2 := outputs.GetByName("output2")
	assert.True(t, hasOutput2, "should have found output2")
	assert.Equal(t, "upgrade output2", string(gotOutput2.Value), "did not find the most recent value for output2")
}
