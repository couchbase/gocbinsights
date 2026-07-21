package cbinsights_test

import (
	"testing"

	cbinsights "github.com/couchbase/gocbinsights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidScheme(t *testing.T) {
	_, err := cbinsights.NewCluster("couchbase://localhost", cbinsights.NewBasicAuthCredential("username", "password"), DefaultOptions())

	assert.ErrorIs(t, err, cbinsights.ErrInvalidArgument)
}

func TestNoScheme(t *testing.T) {
	_, err := cbinsights.NewCluster("//localhost", cbinsights.NewBasicAuthCredential("username", "password"), DefaultOptions())

	assert.ErrorIs(t, err, cbinsights.ErrInvalidArgument)
}

// TestSetCredential_SameType verifies that SetCredential accepts a credential of the same
// type as the one originally provided to NewCluster.
func TestSetCredential_SameType(t *testing.T) {
	cluster, err := cbinsights.NewCluster("http://localhost", cbinsights.NewBasicAuthCredential("user", "pass"), DefaultOptions())
	require.NoError(t, err)

	defer cluster.Close()

	err = cluster.SetCredential(cbinsights.NewBasicAuthCredential("newuser", "newpass"))
	assert.NoError(t, err)
}

// TestSetCredential_DifferentType verifies that SetCredential rejects a credential of a
// different type than the one originally provided to NewCluster.
func TestSetCredential_DifferentType(t *testing.T) {
	cluster, err := cbinsights.NewCluster("https://localhost", cbinsights.NewBasicAuthCredential("user", "pass"), DefaultOptions())
	require.NoError(t, err)

	defer cluster.Close()

	err = cluster.SetCredential(cbinsights.NewJWTCredential("token"))
	assert.ErrorIs(t, err, cbinsights.ErrInvalidArgument)
}

// TestSetCredential_Nil verifies that SetCredential rejects a nil credential.
func TestSetCredential_Nil(t *testing.T) {
	cluster, err := cbinsights.NewCluster("http://localhost", cbinsights.NewBasicAuthCredential("user", "pass"), DefaultOptions())
	require.NoError(t, err)

	defer cluster.Close()

	err = cluster.SetCredential(nil)
	assert.ErrorIs(t, err, cbinsights.ErrInvalidArgument)
}
