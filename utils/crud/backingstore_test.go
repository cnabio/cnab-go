package crud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackingStore_Read(t *testing.T) {
	testcases := []struct {
		name      string
		autoclose bool
	}{
		{name: "Default AutoClose Connections", autoclose: true},
		{name: "Self Managed Connections", autoclose: false},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMockStore()
			s.data[TestItemType] = map[string][]byte{"key1": []byte("value1")}
			bs := NewBackingStore(s)
			bs.AutoClose = tc.autoclose

			val, err := bs.Read(TestItemType, "key1")
			require.NoError(t, err, "expected Read to succeed")
			assert.Equal(t, "value1", string(val), "Read returned the wrong data")

			connectCount, err := s.GetConnectCount()
			require.NoError(t, err, "GetConnectCount failed")
			assert.Equal(t, 1, connectCount, "Connect should have been called once")

			closeCount, err := s.GetCloseCount()
			require.NoError(t, err, "GetCloseCount failed")
			if tc.autoclose {
				assert.Equal(t, 1, closeCount, "Close should have been automatically called")
			} else {
				assert.Equal(t, 0, closeCount, "Close should not be automatically called")
			}
		})
	}
}

func TestBackingStore_Store(t *testing.T) {
	testcases := []struct {
		name      string
		autoclose bool
	}{
		{name: "Default AutoClose Connections", autoclose: true},
		{name: "Self Managed Connections", autoclose: false},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMockStore()
			bs := NewBackingStore(s)
			bs.AutoClose = tc.autoclose

			err := bs.Save(TestItemType, "key1", []byte("value1"))
			require.NoError(t, err, "expected Store to succeed")

			connectCount, err := s.GetConnectCount()
			require.NoError(t, err, "GetConnectCount failed")
			assert.Equal(t, 1, connectCount, "Connect should have been called once")

			closeCount, err := s.GetCloseCount()
			require.NoError(t, err, "GetCloseCount failed")
			if tc.autoclose {
				assert.Equal(t, 1, closeCount, "Close should have been automatically called")
			} else {
				assert.Equal(t, 0, closeCount, "Close should not be automatically called")
			}

			val, err := bs.Read(TestItemType, "key1")
			require.NoError(t, err, "expected Read to succeed")
			assert.Equal(t, "value1", string(val), "stored value did not survive the round trip")

			connectCount, err = s.GetConnectCount()
			require.NoError(t, err, "GetConnectCount failed")
			if tc.autoclose {
				assert.Equal(t, 2, connectCount, "Connect should be called again after the connection is closed")
			} else {
				assert.Equal(t, 1, connectCount, "Connect should only be called once when the connection remains open")
			}

			closeCount, err = s.GetCloseCount()
			require.NoError(t, err, "GetCloseCount failed")
			if tc.autoclose {
				assert.Equal(t, 2, closeCount, "Close is called automatically for every method")
			} else {
				assert.Equal(t, 0, closeCount, "Close should not be automatically called")
			}
		})
	}
}

func TestBackingStore_List(t *testing.T) {
	testcases := []struct {
		name      string
		autoclose bool
	}{
		{name: "Default AutoClose Connections", autoclose: true},
		{name: "Self Managed Connections", autoclose: false},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMockStore()
			s.data[TestItemType] = map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			}
			bs := NewBackingStore(s)
			bs.AutoClose = tc.autoclose

			results, err := bs.List(TestItemType)
			require.NoError(t, err, "expected List to succeed")
			require.Contains(t, results, "key1")
			require.Contains(t, results, "key2")

			connectCount, err := s.GetConnectCount()
			require.NoError(t, err, "GetConnectCount failed")
			assert.Equal(t, 1, connectCount, "Connect should have been called once")

			closeCount, err := s.GetCloseCount()
			require.NoError(t, err, "GetCloseCount failed")
			if tc.autoclose {
				assert.Equal(t, 1, closeCount, "Close should have been automatically called")
			} else {
				assert.Equal(t, 0, closeCount, "Close should not be automatically called")
			}
		})
	}
}

func TestBackingStore_Delete(t *testing.T) {
	testcases := []struct {
		name      string
		autoclose bool
	}{
		{name: "Default AutoClose Connections", autoclose: true},
		{name: "Self Managed Connections", autoclose: false},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMockStore()
			s.data[TestItemType] = map[string][]byte{"key1": []byte("value1")}
			bs := NewBackingStore(s)
			bs.AutoClose = tc.autoclose

			err := bs.Delete(TestItemType, "key1")
			require.NoError(t, err, "expected Delete to succeed")

			connectCount, err := s.GetConnectCount()
			require.NoError(t, err, "GetConnectCount failed")
			assert.Equal(t, 1, connectCount, "Connect should have been called once")

			closeCount, err := s.GetCloseCount()
			require.NoError(t, err, "GetCloseCount failed")
			if tc.autoclose {
				assert.Equal(t, 1, closeCount, "Close should have been automatically called")
			} else {
				assert.Equal(t, 0, closeCount, "Close should not be automatically called")
			}

			val, _ := bs.Read(TestItemType, "key1")
			assert.Empty(t, val, "Delete should have removed the entry")
		})
	}
}

func TestBackingStore_ReadAll(t *testing.T) {
	testcases := []struct {
		name      string
		autoclose bool
	}{
		{name: "Default AutoClose Connections", autoclose: true},
		{name: "Self Managed Connections", autoclose: false},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMockStore()
			s.data[TestItemType] = map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			}
			bs := NewBackingStore(s)
			bs.AutoClose = tc.autoclose

			results, err := bs.ReadAll(TestItemType)
			require.NoError(t, err, "expected ReadAll to succeed")
			assert.Contains(t, results, []byte("value1"))
			assert.Contains(t, results, []byte("value2"))

			connectCount, err := s.GetConnectCount()
			require.NoError(t, err, "GetConnectCount failed")
			assert.Equal(t, 1, connectCount, "Connect should have been called once")

			closeCount, err := s.GetCloseCount()
			require.NoError(t, err, "GetCloseCountFailed")
			if tc.autoclose {
				assert.Equal(t, 1, closeCount, "Close should have been automatically called")
			} else {
				assert.Equal(t, 0, closeCount, "Close should not be automatically called")
			}
		})
	}
}
