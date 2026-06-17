package invoicexpress_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	invoicexpress "github.com/voluzi/go-invoicexpress"
)

func ExampleNewClient() {
	client := invoicexpress.NewClient("my-account", "my-api-key",
		invoicexpress.WithRateLimit(10, 5), // pace to 10 req/s, burst 5
	)
	_ = client
}

// ExampleInvoicesService_CreateAndFinalize shows the common path: issue a
// legally-finalized invoice-receipt in one call.
func ExampleInvoicesService_CreateAndFinalize() {
	client := invoicexpress.NewClient("my-account", "my-api-key")
	ctx := context.Background()

	inv, err := client.Invoices.CreateAndFinalize(ctx, invoicexpress.DocumentTypeInvoiceReceipt,
		&invoicexpress.InvoiceCreateRequest{
			Date:   invoicexpress.NewDate(time.Now()),
			Client: invoicexpress.ClientRef{Name: "ACME, Lda", FiscalID: "999999990"}, // public Consumidor Final placeholder
			Items: []invoicexpress.ItemRef{{
				Name:      "Plano Pro",
				UnitPrice: invoicexpress.NewDecimal("29.00"),
				Quantity:  invoicexpress.NewDecimal("1"),
				Tax:       &invoicexpress.TaxRef{Name: "IVA23"},
			}},
		})
	if err != nil {
		// handle error
		return
	}
	fmt.Println(inv.SequenceNumber)
}

// ExampleAsAPIError shows inspecting a typed API error.
func ExampleAsAPIError() {
	client := invoicexpress.NewClient("my-account", "my-api-key")
	_, err := client.Invoices.Get(context.Background(), invoicexpress.DocumentTypeInvoice, 999)
	if invoicexpress.IsNotFound(err) {
		fmt.Println("invoice not found")
		return
	}
	var apiErr *invoicexpress.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("API error %d: %v\n", apiErr.StatusCode, apiErr.Errors)
	}
}
