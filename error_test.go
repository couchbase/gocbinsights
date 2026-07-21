package cbinsights

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryErrorAsInsightsError(t *testing.T) {
	err := newQueryError(nil, "select *", "endpoint", 200, 23, "message", 0)

	var insightsError *InsightsError

	require.ErrorAs(t, err, &insightsError)
}

func TestQueryErrorIsErrQuery(t *testing.T) {
	err := newQueryError(nil, "select *", "endpoint", 200, 23, "message", 0)

	require.ErrorIs(t, err, ErrQuery)
}

func TestQueryErrorAsQueryError(t *testing.T) {
	err := newQueryError(nil, "select *", "endpoint", 200, 23, "message", 0)

	var queryError QueryError

	require.ErrorAs(t, err, &queryError)

	assert.Equal(t, 23, queryError.Code())
	assert.Equal(t, "message", queryError.Message())
}
