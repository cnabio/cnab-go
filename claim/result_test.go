package claim

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResults_Sort(t *testing.T) {
	r := Results{
		{ID: "2"},
		{ID: "1"},
		{ID: "3"},
	}

	sort.Sort(r)

	assert.Equal(t, "1", r[0].ID, "Results did not sort 1 to the first slot")
	assert.Equal(t, "2", r[1].ID, "Results did not sort 2 to the second slot")
	assert.Equal(t, "3", r[2].ID, "Results did not sort 3 to the third slot")
}

func TestResult_Validate(t *testing.T) {
	t.Run("if result has a valid status, validation should pass", func(t *testing.T) {
		validStatuses := []string{
			StatusCanceled,
			StatusRunning,
			StatusFailed,
			StatusPending,
			StatusSucceeded,
			StatusUnknown,
		}
		for _, status := range validStatuses {
			t.Run(status+" status", func(t *testing.T) {
				r, err := exampleClaim.NewResult(status)
				require.NoError(t, err, "NewResult failed")

				err = r.Validate()
				assert.NoError(t, err, "%s is a valid claim status", status)
			})
		}
	})

	t.Run("if result has empty Id, validation should fail", func(t *testing.T) {
		r, err := exampleClaim.NewResult(StatusSucceeded)
		require.NoError(t, err, "NewResult failed")

		r.ID = ""
		err = r.Validate()
		assert.EqualError(t, err, "the result id must be set")
	})

	t.Run("if result has empty claimID, validation should fail", func(t *testing.T) {
		r, err := exampleClaim.NewResult(StatusSucceeded)
		require.NoError(t, err, "NewResult failed")

		r.ClaimID = ""
		err = r.Validate()
		assert.EqualError(t, err, "the claimID must be set")
	})

	t.Run("if result has invalid status, validation should fail", func(t *testing.T) {
		r, err := exampleClaim.NewResult("invalidStatus")
		require.NoError(t, err, "NewResult failed")

		err = r.Validate()
		assert.EqualError(t, err, "invalid status: invalidStatus")
	})
}

func TestResultOutputs_SetContentDigest(t *testing.T) {
	contentDigest := "sha256:abc123"
	wantoutputs := OutputMetadata{
		"test1": map[string]string{
			OutputContentDigest: contentDigest,
		},
	}

	t.Run("new value", func(t *testing.T) {
		outputs := OutputMetadata{}
		err := outputs.SetContentDigest("test1", contentDigest)
		require.NoError(t, err, "SetContentDigest failed")
		assert.Equal(t, wantoutputs, outputs, "SetContentDigest did not produce the expected structure")
	})

	t.Run("existing value", func(t *testing.T) {
		outputs := OutputMetadata{
			"test1": map[string]string{
				OutputContentDigest: "old_value",
			},
		}
		err := outputs.SetContentDigest("test1", contentDigest)
		require.NoError(t, err, "SetContentDigest failed")
		assert.Equal(t, wantoutputs, outputs, "SetContentDigest did not produce the expected structure")
	})

	t.Run("existing invalid structure", func(t *testing.T) {
		outputs := OutputMetadata{
			"test1": map[string]interface{}{
				OutputContentDigest: struct{}{},
			},
		}
		err := outputs.SetContentDigest("test1", contentDigest)
		require.EqualError(t, err, "cannot set the claim result's Outputs[test1][contentDigest] because it is not type map[string]string but map[string]interface {}")
	})
}

func TestResultOutputs_GetContentDigest(t *testing.T) {
	t.Run("output has digest", func(t *testing.T) {
		contentDigest := "sha256:abc123"

		outputs := OutputMetadata{}
		err := outputs.SetContentDigest("test1", contentDigest)
		require.NoError(t, err, "SetContentDigest failed")

		gotContentDigest, ok := outputs.GetContentDigest("test1")
		require.True(t, ok, "GetContentDigest should find the digest")
		assert.Equal(t, contentDigest, gotContentDigest, "GetContentDigest should return the digest that we set")
	})

	t.Run("output not found", func(t *testing.T) {
		outputs := OutputMetadata{}
		gotContentDigest, ok := outputs.GetContentDigest("test1")
		require.False(t, ok, "GetContentDigest should report that it did not find the contentDigest")
		assert.Empty(t, gotContentDigest, "GetContentDigest should return an empty digest when one isn't found")
	})

	t.Run("output has no digest", func(t *testing.T) {
		outputs := OutputMetadata{
			"test1": map[string]string{
				"other": "stuff",
			},
		}

		gotContentDigest, ok := outputs.GetContentDigest("test1")
		require.False(t, ok, "GetContentDigest should report that it did not find the contentDigest")
		assert.Empty(t, gotContentDigest, "GetContentDigest should return an empty digest when one isn't found")
	})

	t.Run("output has different structure", func(t *testing.T) {
		outputs := OutputMetadata{
			"test1": map[string]interface{}{
				"other": struct{}{},
			},
		}

		gotContentDigest, ok := outputs.GetContentDigest("test1")
		require.False(t, ok, "GetContentDigest should report that it did not find the contentDigest")
		assert.Empty(t, gotContentDigest, "GetContentDigest should return an empty digest when one isn't found")
	})

}
