package claim

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
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

func TestResultOutputs_SetMetadata(t *testing.T) {
	metadataKey := "so-meta"
	metadataValue := "i am metadata"
	outputName := "test1"
	wantoutputs := OutputMetadata{
		outputName: map[string]string{
			metadataKey: metadataValue,
		},
	}

	t.Run("new value", func(t *testing.T) {
		outputs := OutputMetadata{}
		err := outputs.SetMetadata(outputName, metadataKey, metadataValue)
		require.NoError(t, err, "SetMetadata failed")
		assert.Equal(t, wantoutputs, outputs, "SetMetadata did not produce the expected structure")
	})

	t.Run("existing value", func(t *testing.T) {
		outputs := OutputMetadata{
			outputName: map[string]string{
				metadataKey: "old_value",
			},
		}
		err := outputs.SetMetadata(outputName, metadataKey, metadataValue)
		require.NoError(t, err, "SetMetadata failed")
		assert.Equal(t, wantoutputs, outputs, "SetMetadata did not produce the expected structure")
	})

	t.Run("existing invalid structure", func(t *testing.T) {
		outputs := OutputMetadata{
			outputName: map[string]interface{}{
				metadataKey: struct{}{},
			},
		}
		err := outputs.SetMetadata(outputName, metadataKey, metadataValue)
		require.EqualError(t, err, "cannot set the claim result's OutputMetadata[test1][so-meta] because it is not type map[string]string but map[string]interface {}")
	})
}

func TestResultOutputs_GetMetadata(t *testing.T) {
	metadataKey := "so-meta"
	outputName := "test1"
	t.Run("output has metadata", func(t *testing.T) {
		value := "that's so meta"

		outputs := OutputMetadata{}
		err := outputs.SetMetadata(outputName, metadataKey, value)
		require.NoError(t, err, "SetMetadata failed")

		gotValue, ok := outputs.GetMetadata(outputName, metadataKey)
		require.True(t, ok, "GetMetadata should find the value")
		assert.Equal(t, value, gotValue, "GetMetadata should return the value that we set")
	})

	t.Run("output not found", func(t *testing.T) {
		outputs := OutputMetadata{}
		gotValue, ok := outputs.GetMetadata(outputName, metadataKey)
		require.False(t, ok, "GetMetadata should report that it did not find the value")
		assert.Empty(t, gotValue, "GetMetadata should return an empty value when one isn't found")
	})

	t.Run("output has no metadata", func(t *testing.T) {
		outputs := OutputMetadata{
			outputName: map[string]string{
				"other": "stuff",
			},
		}

		gotValue, ok := outputs.GetMetadata(outputName, metadataKey)
		require.False(t, ok, "GetMetadata should report that it did not find the value")
		assert.Empty(t, gotValue, "GetMetadata should return an empty value when one isn't found")
	})

	t.Run("output has different structure", func(t *testing.T) {
		outputs := OutputMetadata{
			outputName: map[string]interface{}{
				"other": struct{}{},
			},
		}

		gotValue, ok := outputs.GetMetadata(outputName, metadataKey)
		require.False(t, ok, "GetMetadata should report that it did not find the value")
		assert.Empty(t, gotValue, "GetMetadata should return an empty value when one isn't found")
	})

}

func TestResultOutputs_SetContentDigest(t *testing.T) {
	testcases := []struct {
		value     string
		wantValue string
	}{
		{"abc123", "abc123"},
	}

	for _, tc := range testcases {
		t.Run(tc.wantValue, func(t *testing.T) {
			wantoutputs := OutputMetadata{
				"test1": map[string]string{
					OutputContentDigest: tc.wantValue,
				},
			}

			outputs := OutputMetadata{}
			err := outputs.SetContentDigest("test1", tc.value)
			require.NoError(t, err, "SetContentDigest failed")
			assert.Equal(t, wantoutputs, outputs, "SetContentDigest did not produce the expected structure")
		})
	}
}

func TestResultOutputs_GetContentDigest(t *testing.T) {
	testcases := []struct {
		value     string
		wantValue string
		wantOK    bool
	}{
		{value: "abc123", wantValue: "abc123", wantOK: true},
	}

	for _, tc := range testcases {
		t.Run("existing metadata", func(t *testing.T) {
			outputs := OutputMetadata{}
			err := outputs.SetContentDigest("test1", tc.value)
			require.NoError(t, err, "SetContentDigest failed")

			generatedByBundle, ok := outputs.GetContentDigest("test1")
			require.Equal(t, tc.wantOK, ok, "GetGeneratedByBundle did not return the expected ok value")
			assert.Equal(t, tc.wantValue, generatedByBundle, "GetGeneratedByBundle did not return the expected value")
		})
	}
}

func TestResultOutputs_SetGeneratedByBundle(t *testing.T) {
	testcases := []struct {
		value     bool
		wantValue string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tc := range testcases {
		t.Run(tc.wantValue, func(t *testing.T) {
			wantoutputs := OutputMetadata{
				"test1": map[string]string{
					OutputGeneratedByBundle: tc.wantValue,
				},
			}

			outputs := OutputMetadata{}
			err := outputs.SetGeneratedByBundle("test1", tc.value)
			require.NoError(t, err, "SetGeneratedByBundle failed")
			assert.Equal(t, wantoutputs, outputs, "SetGeneratedByBundle did not produce the expected structure")
		})
	}
}

func TestResultOutputs_GetGeneratedByBundle(t *testing.T) {
	testcases := []struct {
		value     string
		wantValue bool
		wantOK    bool
	}{
		{value: "true", wantValue: true, wantOK: true},
		{value: "false", wantValue: false, wantOK: true},
		{value: "invalid", wantValue: false, wantOK: false},
	}

	for _, tc := range testcases {
		t.Run("existing metadata", func(t *testing.T) {
			outputs := OutputMetadata{}
			err := outputs.SetMetadata("test1", OutputGeneratedByBundle, tc.value)
			require.NoError(t, err, "SetMetadata failed")

			generatedByBundle, ok := outputs.GetGeneratedByBundle("test1")
			require.Equal(t, tc.wantOK, ok, "GetGeneratedByBundle did not return the expected ok value")
			assert.Equal(t, tc.wantValue, generatedByBundle, "GetGeneratedByBundle did not return the expected value")
		})
	}
}

func TestResult_HasLogs(t *testing.T) {
	c, err := New("test", ActionInstall, bundle.Bundle{}, nil)
	require.NoError(t, err)
	r, _ := c.NewResult(StatusSucceeded)
	require.NoError(t, err)
	assert.False(t, r.HasLogs(), "expected HasLogs to return false")

	// Record that the result generated logs
	r.OutputMetadata.SetGeneratedByBundle(OutputInvocationImageLogs, true)
	assert.True(t, r.HasLogs(), "expected HasLogs to return true")
}
