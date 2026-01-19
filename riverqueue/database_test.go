// SPDX-License-Identifier: ice License 1.0

package riverqueue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateConnectOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		addresses    []string
		currentIndex int
		expected     []int
	}{
		{
			name:         "single address",
			addresses:    []string{"master1"},
			currentIndex: 0,
			expected:     []int{0},
		},
		{
			name:         "two addresses from index 0",
			addresses:    []string{"master1", "master2"},
			currentIndex: 0,
			expected:     []int{1, 0},
		},
		{
			name:         "two addresses from index 1",
			addresses:    []string{"master1", "master2"},
			currentIndex: 1,
			expected:     []int{0, 1},
		},
		{
			name:         "three addresses from index 0",
			addresses:    []string{"master1", "master2", "master3"},
			currentIndex: 0,
			expected:     []int{1, 2, 0},
		},
		{
			name:         "three addresses from index 1",
			addresses:    []string{"master1", "master2", "master3"},
			currentIndex: 1,
			expected:     []int{2, 0, 1},
		},
		{
			name:         "three addresses from index 2",
			addresses:    []string{"master1", "master2", "master3"},
			currentIndex: 2,
			expected:     []int{0, 1, 2},
		},
		{
			name:         "four addresses from index 0",
			addresses:    []string{"master1", "master2", "master3", "master4"},
			currentIndex: 0,
			expected:     []int{1, 2, 3, 0},
		},
		{
			name:         "four addresses from index 1",
			addresses:    []string{"master1", "master2", "master3", "master4"},
			currentIndex: 1,
			expected:     []int{2, 3, 0, 1},
		},
		{
			name:         "four addresses from index 3",
			addresses:    []string{"master1", "master2", "master3", "master4"},
			currentIndex: 3,
			expected:     []int{0, 1, 2, 3},
		},
		{
			name:         "five addresses from index 2",
			addresses:    []string{"a", "b", "c", "d", "e"},
			currentIndex: 2,
			expected:     []int{3, 4, 0, 1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := calculateConnectOrder(tt.addresses, tt.currentIndex)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateConnectOrderProperties(t *testing.T) {
	t.Parallel()

	// Property-based tests to verify invariants.
	tests := []struct {
		name      string
		addresses int
		startIdx  int
	}{
		{name: "1 address", addresses: 1, startIdx: 0},
		{name: "2 addresses from 0", addresses: 2, startIdx: 0},
		{name: "2 addresses from 1", addresses: 2, startIdx: 1},
		{name: "3 addresses from 0", addresses: 3, startIdx: 0},
		{name: "3 addresses from 1", addresses: 3, startIdx: 1},
		{name: "3 addresses from 2", addresses: 3, startIdx: 2},
		{name: "10 addresses from 5", addresses: 10, startIdx: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			addresses := make([]string, tt.addresses)
			for i := range addresses {
				addresses[i] = "master"
			}

			result := calculateConnectOrder(addresses, tt.startIdx)

			// Invariant 1: Result length should equal number of addresses
			require.Len(t, result, tt.addresses)

			// Invariant 2: Result should contain all indices from 0 to len(addresses)-1
			seen := make(map[int]bool)
			for _, idx := range result {
				seen[idx] = true
			}
			for i := 0; i < tt.addresses; i++ {
				require.True(t, seen[i], "index %d should be present in result", i)
			}

			// Invariant 3: No duplicate indices
			require.Len(t, seen, tt.addresses)

			// Invariant 4: For cases with 2+ addresses, the first element should not be the current index
			if tt.addresses > 1 {
				require.NotEqual(t, tt.startIdx, result[0],
					"first element should be different from current index for 2+ addresses")
			}
		})
	}
}
