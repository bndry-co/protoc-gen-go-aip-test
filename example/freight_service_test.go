package example

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	examplefreightv1 "github.com/einride/protoc-gen-go-aip-test/proto/gen/einride/example/freight/v1"
	examplefreightv1connect "github.com/einride/protoc-gen-go-aip-test/proto/gen/einride/example/freight/v1/examplefreightv1connect"
)

func Test_FreightService(t *testing.T) {
	t.Parallel()
	t.Skip("this is just an example, the service is not implemented.")
	// setup client before test
	client := examplefreightv1connect.UnimplementedFreightServiceHandler{}
	// setup test suite
	suite := examplefreightv1connect.FreightServiceTestSuite{
		T:      t,
		Client: client,
	}

	// counter to keep track of unique IDs.
	var idCounter uint64

	// run tests for each resource in the service
	ctx := context.Background()
	suite.TestShipper(ctx, examplefreightv1connect.FreightServiceShipperTestSuiteConfig{
		// Create should return a resource which is valid to create, i.e.
		// all required fields set.
		Create: func() *examplefreightv1.Shipper {
			return &examplefreightv1.Shipper{
				DisplayName:    "Example shipper",
				BillingAccount: "billingAccounts/12345",
			}
		},
		// IDGenerator should return a valid and unique ID to use in the Create call.
		// If non-nil, this function will be called to set the ID on all Create calls.
		// If the ID field is required, tests will fail if this is nil.
		IDGenerator: func() string {
			id := atomic.AddUint64(&idCounter, 1)
			return fmt.Sprintf("valid-id-%d", id)
		},
		// Update should return a resource which is valid to update, i.e.
		// all required fields set.
		Update: func() *examplefreightv1.Shipper {
			return &examplefreightv1.Shipper{
				DisplayName:    "Updated example shipper",
				BillingAccount: "billingAccounts/54321",
			}
		},
	})
}

func Test_FreightService_AlternativeSetup(t *testing.T) {
	// Even though no implementation exists, the tests will pass but be skipped.
	examplefreightv1connect.TestServices(t, &aipTests{})
}

type aipTests struct{}

var _ examplefreightv1connect.FreightServiceTestSuiteConfigProvider = &aipTests{}

func (a aipTests) FreightServiceShipper(_ *testing.T) *examplefreightv1connect.FreightServiceShipperTestSuiteConfig {
	// Returns nil to indicate that it's not ready to be tested.
	return nil
}

func (a aipTests) FreightServiceSite(_ *testing.T) *examplefreightv1connect.FreightServiceSiteTestSuiteConfig {
	// Returns nil to indicate that it's not ready to be tested.
	return nil
}
