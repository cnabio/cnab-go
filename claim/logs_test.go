package claim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/utils/crud"
)

func addTestRunWithLogs(t *testing.T, cp Provider, logContent string) Claim {
	b := bundle.Bundle{}
	c, err := New("test", ActionInstall, b, nil)
	require.NoError(t, err)
	require.NoError(t, cp.SaveClaim(c))

	rStart, err := c.NewResult(StatusRunning)
	require.NoError(t, err)
	require.NoError(t, cp.SaveResult(rStart))

	rStop, err := c.NewResult(StatusSucceeded)
	require.NoError(t, err)
	rStop.OutputMetadata.SetGeneratedByBundle(OutputInvocationImageLogs, false)
	require.NoError(t, cp.SaveResult(rStop))

	rRando, err := c.NewResult(StatusUnknown) // Add extra result at the end just to make sure we search for the logs correctly
	require.NoError(t, err)
	require.NoError(t, cp.SaveResult(rRando))

	o := NewOutput(c, rStop, OutputInvocationImageLogs, []byte(logContent))
	require.NoError(t, cp.SaveOutput(o))

	return c
}

func TestGetLogs(t *testing.T) {
	const logContent = "some mighty fine logs"

	backingStore := crud.NewMockStore()
	cp := NewClaimStore(crud.NewBackingStore(backingStore), nil, nil)
	c := addTestRunWithLogs(t, cp, logContent)

	logs, ok, err := GetLogs(cp, c.ID)
	require.NoError(t, err)
	assert.True(t, ok, "expected to find logs")
	assert.Equal(t, logContent, logs, "wrong logs found")
}

func TestGetLogs_NotFound(t *testing.T) {
	backingStore := crud.NewMockStore()
	cp := NewClaimStore(crud.NewBackingStore(backingStore), nil, nil)

	b := bundle.Bundle{}
	c, err := New("test", ActionInstall, b, nil)
	require.NoError(t, err)
	require.NoError(t, cp.SaveClaim(c))

	rStart, err := c.NewResult(StatusRunning)
	require.NoError(t, err)
	require.NoError(t, cp.SaveResult(rStart))

	logs, ok, err := GetLogs(cp, c.ID)
	require.NoError(t, err)
	assert.False(t, ok, "expected to not find logs")
	assert.Empty(t, logs, "expected an empty log result")
}

func TestGetLastLogs(t *testing.T) {
	backingStore := crud.NewMockStore()
	cp := NewClaimStore(crud.NewBackingStore(backingStore), nil, nil)
	addTestRunWithLogs(t, cp, "first run")
	c := addTestRunWithLogs(t, cp, "second run")

	logs, ok, err := GetLastLogs(cp, c.Installation)
	require.NoError(t, err)
	assert.True(t, ok, "expected to find logs")
	assert.Equal(t, "second run", logs, "wrong logs found")
}
