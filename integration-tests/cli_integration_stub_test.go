//go:build !integration

package integration_tests_test

import "testing"

func TestIntegrationBuildTagRequired(t *testing.T) {
	t.Skip("integration tests require -tags=integration")
}
