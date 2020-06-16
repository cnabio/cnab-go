package claim

import (
	"sort"
	"testing"

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
		i := NewInstallation(installationName, Claims{upgrade, install1, install2})

		installTime, err := i.GetInstallationTimestamp()
		require.NoError(t, err, "GetInstallationTimestamp failed")
		assert.Equal(t, install1.Created, installTime, "invalid installation time")
	})
	t.Run("no claims", func(t *testing.T) {
		i := NewInstallation(installationName, nil)

		_, err := i.GetInstallationTimestamp()
		require.EqualError(t, err, "the installation test has no claims")
	})
}

func TestInstallation_GetLastClaim(t *testing.T) {
	upgrade := Claim{
		ID:     "2",
		Action: ActionUpgrade,
		results: &Results{
			{
				ID:     "1",
				Status: StatusRunning,
			},
		},
	}
	install := Claim{
		ID:     "1",
		Action: ActionInstall,
		results: &Results{
			{
				ID:     "1",
				Status: StatusSucceeded,
			},
		},
	}

	t.Run("claim exists", func(t *testing.T) {
		i := NewInstallation("wordpress", Claims{upgrade, install})

		c, err := i.GetLastClaim()

		require.NoError(t, err, "GetLastClaim failed")
		assert.Equal(t, upgrade, c, "GetLastClaim did not return the expected claim")
	})

	t.Run("no claims", func(t *testing.T) {
		i := NewInstallation("wordpress", nil)

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
		results: &Results{
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
		results: &Results{
			{
				ID:     "1",
				Status: StatusSucceeded,
			},
		},
	}

	t.Run("result exists", func(t *testing.T) {
		i := NewInstallation("wordpress", Claims{upgrade, install})

		r, err := i.GetLastResult()

		require.NoError(t, err, "GetLastResult failed")
		assert.Equal(t, failed, r, "GetLastResult did not return the expected result")
		assert.Equal(t, StatusFailed, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})

	t.Run("no claims", func(t *testing.T) {
		i := NewInstallation("wordpress", nil)

		r, err := i.GetLastResult()

		require.EqualError(t, err, "the installation wordpress has no claims")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})

	t.Run("no results", func(t *testing.T) {
		i := NewInstallation("wordpress", Claims{Claim{ID: "1", results: &Results{}}})

		r, err := i.GetLastResult()

		require.EqualError(t, err, "the last claim has no results")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})

	t.Run("no results loaded", func(t *testing.T) {
		i := NewInstallation("wordpress", Claims{Claim{ID: "1"}})

		r, err := i.GetLastResult()

		require.EqualError(t, err, "the last claim does not have any results loaded")
		assert.Equal(t, Result{}, r, "should return an empty result when one cannot be found")
		assert.Equal(t, StatusUnknown, i.GetLastStatus(), "GetLastStatus did not return the expected value")
	})
}

func TestInstallationByName_Sort(t *testing.T) {
	installations := InstallationByName{
		{Name: "c"},
		{Name: "a"},
		{Name: "b"},
	}

	sort.Sort(installations)

	assert.Equal(t, "a", installations[0].Name)
	assert.Equal(t, "b", installations[1].Name)
	assert.Equal(t, "c", installations[2].Name)
}

func TestInstallationByModified_Sort(t *testing.T) {
	cid1 := MustNewULID()
	cid2 := MustNewULID()
	cid3 := MustNewULID()
	cid4 := MustNewULID()

	installations := InstallationByModified{
		{Name: "c", Claims: []Claim{{ID: cid4}, {ID: cid2}}}, // require a sort for this to end up last (cid4 is the "oldest" timestamp)
		{Name: "a", Claims: []Claim{{ID: cid1}}},
		{Name: "b", Claims: []Claim{{ID: cid3}}},
	}

	installations.SortClaims()
	sort.Sort(installations)

	assert.Equal(t, "a", installations[0].Name)
	assert.Equal(t, "b", installations[1].Name)
	assert.Equal(t, "c", installations[2].Name)
}
