// Package main demonstrates basic usage of the go-invoicexpress library.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/voluzi/go-invoicexpress"
)

func main() {
	// Create a client with your InvoiceXpress account name and API key.
	client := invoicexpress.NewClient("my-account", "my-api-key")

	ctx := context.Background()

	// --- Clients ---

	// Create a client.
	customer, err := client.Clients.Create(ctx, &invoicexpress.ClientCreateRequest{
		Name:     "Acme Corp",
		Code:     "ACME",
		Email:    "billing@acme.com",
		Country:  "Portugal",
		FiscalID: "508000000",
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	fmt.Printf("Created client: %s (ID: %d)\n", customer.Name, customer.ID)

	// --- Items ---

	// Create an item.
	item, err := client.Items.Create(ctx, &invoicexpress.ItemCreateRequest{
		Name:        "Consulting",
		Description: "Software consulting services",
		UnitPrice:   invoicexpress.NewDecimal("100.00"),
		Unit:        "hour",
		Tax:         &invoicexpress.TaxRef{Name: "IVA23"},
	})
	if err != nil {
		log.Fatalf("create item: %v", err)
	}
	fmt.Printf("Created item: %s (ID: %d)\n", item.Name, item.ID)

	// --- Invoices ---

	// Create an invoice.
	now := time.Now()
	inv, err := client.Invoices.Create(ctx, invoicexpress.DocumentTypeInvoice, &invoicexpress.InvoiceCreateRequest{
		Date:    invoicexpress.NewDate(now),
		DueDate: invoicexpress.NewDate(now.AddDate(0, 0, 30)),
		Client: invoicexpress.ClientRef{
			Name: "Acme Corp",
			Code: "ACME",
		},
		Items: []invoicexpress.ItemRef{
			{
				Name:      "Consulting",
				UnitPrice: invoicexpress.NewDecimal("100.00"),
				Quantity:  invoicexpress.NewDecimal("8"),
				Unit:      "hour",
				Tax:       &invoicexpress.TaxRef{Name: "IVA23"},
			},
		},
	})
	if err != nil {
		log.Fatalf("create invoice: %v", err)
	}
	fmt.Printf("Created invoice: ID=%d, Total=%s\n", inv.ID, inv.Total)

	// Finalize the invoice.
	inv, err = client.Invoices.ChangeState(ctx, invoicexpress.DocumentTypeInvoice, inv.ID, invoicexpress.StateFinalized, "")
	if err != nil {
		log.Fatalf("finalize invoice: %v", err)
	}
	fmt.Printf("Finalized invoice: %s\n", inv.SequenceNumber)

	// Send the invoice by email.
	err = client.Invoices.SendByEmail(ctx, invoicexpress.DocumentTypeInvoice, inv.ID, &invoicexpress.EmailRequest{
		Client:  invoicexpress.EmailClientRef{Email: "billing@acme.com"},
		Subject: fmt.Sprintf("Invoice %s", inv.SequenceNumber),
		Body:    "Please find your invoice attached.",
	})
	if err != nil {
		log.Fatalf("send invoice by email: %v", err)
	}
	fmt.Println("Invoice sent by email.")

	// Generate PDF (waits until ready).
	pdfURL, err := client.Invoices.GeneratePDF(ctx, inv.ID, 2*time.Second)
	if err != nil {
		log.Fatalf("generate PDF: %v", err)
	}
	fmt.Printf("PDF URL: %s\n", pdfURL)

	// --- Taxes ---

	taxes, err := client.Taxes.List(ctx)
	if err != nil {
		log.Fatalf("list taxes: %v", err)
	}
	fmt.Printf("Found %d taxes\n", len(taxes))

	// --- Sequences ---

	sequences, err := client.Sequences.List(ctx)
	if err != nil {
		log.Fatalf("list sequences: %v", err)
	}
	fmt.Printf("Found %d sequences\n", len(sequences))

	// --- SAF-T Export ---

	year, month, _ := now.Date()
	saft, err := client.SAFT.Export(ctx, int(month), year, 3*time.Second)
	if err != nil {
		log.Fatalf("export SAF-T: %v", err)
	}
	fmt.Printf("SAF-T XML: %s\n", saft.XMLURL)
}
