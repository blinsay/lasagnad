package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3Key(t *testing.T) {
	tcs := []struct {
		prefix   string
		name     string
		filetype string
		id       string
		expected string
	}{
		{
			prefix:   "lasagna",
			name:     "mork",
			filetype: "jpeg",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			expected: "lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.jpg",
		},
		{
			prefix:   "lasagna",
			name:     "mork",
			filetype: "gif",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			expected: "lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.gif",
		},
		{
			prefix:   "lasagna",
			name:     "mork",
			filetype: "png",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			expected: "lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.png",
		},
	}

	for _, tc := range tcs {
		id, err := imgidFromString(tc.id)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, s3key(tc.prefix, tc.name, tc.filetype, id))
	}
}

func TestS3URL(t *testing.T) {
	tcs := []struct {
		bucket   string
		prefix   string
		name     string
		filetype string
		id       string
		expected string
	}{
		{
			bucket:   "garf",
			prefix:   "lasagna",
			name:     "mork",
			filetype: "jpeg",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			expected: "https://garf.s3.amazonaws.com/lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.jpg",
		},
		{
			bucket:   "garf",
			prefix:   "lasagna",
			name:     "mork",
			filetype: "gif",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			expected: "https://garf.s3.amazonaws.com/lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.gif",
		},
		{
			bucket:   "garf",
			prefix:   "lasagna",
			name:     "mork",
			filetype: "png",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			expected: "https://garf.s3.amazonaws.com/lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.png",
		},
	}

	for _, tc := range tcs {
		id, err := imgidFromString(tc.id)
		require.NoError(t, err)

		assert.Equal(t,
			tc.expected,
			s3url(tc.bucket, tc.prefix, tc.name, tc.filetype, id).String())
	}
}

func TestIdAndFiletype(t *testing.T) {
	tcs := []struct {
		key      string
		id       string
		filetype string
		err      error
	}{
		{
			key:      "lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.png",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			filetype: "png",
		},
		{
			key:      "lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.jpg",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			filetype: "jpeg",
		},
		{
			key:      "lasagna/mork/7287194dfdb24cb741413ebb7f9b121d.gif",
			id:       "7287194dfdb24cb741413ebb7f9b121d",
			filetype: "gif",
		},
	}

	for _, tc := range tcs {
		expectedID, idErr := imgidFromString(tc.id)
		require.NoError(t, idErr, "test setup failed")

		id, filetype, err := idAndFiletype(tc.key)

		assert.Equal(t, expectedID, id, "%s: id not equal", tc.key)
		assert.Equal(t, tc.filetype, filetype, "%s: filetype not equal", tc.key)
		assert.Equal(t, tc.err, err, "%s: err not equal", tc.key)
	}
}
