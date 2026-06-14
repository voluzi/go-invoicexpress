// Command draft-invoice creates a single DRAFT invoice against a real
// InvoiceXpress account, to smoke-test the library.
//
// A draft has no fiscal value and can be deleted from the InvoiceXpress UI —
// this program never finalizes, emails, or generates a PDF.
//
// Usage:
//
//	export INVOICEXPRESS_ACCOUNT=my-account
//	export INVOICEXPRESS_API_KEY=xxxxxxxx
//	go run ./examples/draft-invoice
//
// Override any field with flags, e.g.:
//
//	go run ./examples/draft-invoice -client "ACME, Lda" -nif 500000000 -price 29.00 -tax IVA23
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	ix "github.com/voluzi/go-invoicexpress"
)

func main() {
	account := flag.String("account", os.Getenv("INVOICEXPRESS_ACCOUNT"), "InvoiceXpress account name (the subdomain)")
	apiKey := flag.String("api-key", os.Getenv("INVOICEXPRESS_API_KEY"), "InvoiceXpress API key")
	clientName := flag.String("client", "Cliente de Teste", "client name")
	nif := flag.String("nif", "", "client fiscal ID / NIF (optional)")
	itemName := flag.String("item", "Serviço de teste", "item name")
	price := flag.String("price", "10.00", "unit price, as a decimal string")
	qty := flag.String("qty", "1", "quantity, as a decimal string")
	taxName := flag.String("tax", "IVA23", "tax name exactly as configured in your account")
	docType := flag.String("type", "invoices", "document type: invoices, invoice_receipts, simplified_invoices, ...")
	finalize := flag.Bool("finalize", false, "FINALIZE the document (legally binding; cannot be deleted, only cancelled)")
	flag.Parse()

	if *account == "" || *apiKey == "" {
		log.Fatal("set INVOICEXPRESS_ACCOUNT and INVOICEXPRESS_API_KEY (or pass -account / -api-key)")
	}

	client := ix.NewClient(*account, *apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Guard against InvoiceXpress's silent fallbacks (it does not error on a
	// bad NIF or unknown tax — it degrades to "Consumidor Final" / the default
	// tax). These are warnings, not hard stops, so you can still observe it.
	if *nif != "" && !ix.ValidPortugueseNIF(*nif) {
		fmt.Printf("⚠️  %q is not a valid Portuguese NIF — InvoiceXpress will likely issue to \"Consumidor Final\".\n", *nif)
	}
	if _, err := client.Taxes.FindByName(ctx, *taxName); errors.Is(err, ix.ErrTaxNotFound) {
		fmt.Printf("⚠️  tax %q not found in your account — InvoiceXpress will apply the default tax.\n", *taxName)
	}

	req := &ix.InvoiceCreateRequest{
		Date:   ix.NewDate(time.Now()),
		Client: ix.ClientRef{Name: *clientName, FiscalID: *nif},
		Items: []ix.ItemRef{{
			Name:      *itemName,
			UnitPrice: ix.NewDecimal(*price),
			Quantity:  ix.NewDecimal(*qty),
			Tax:       &ix.TaxRef{Name: *taxName},
		}},
	}

	var inv *ix.Invoice
	var err error
	if *finalize {
		inv, err = client.Invoices.CreateAndFinalize(ctx, ix.DocumentType(*docType), req)
	} else {
		inv, err = client.Invoices.Create(ctx, ix.DocumentType(*docType), req)
	}
	if err != nil {
		if apiErr, ok := ix.AsAPIError(err); ok {
			log.Fatalf("InvoiceXpress returned %d %s\n  messages: %v\n  raw body: %s",
				apiErr.StatusCode, apiErr.Status, apiErr.Errors, apiErr.Body)
		}
		log.Fatalf("create invoice: %v", err)
	}

	if *finalize {
		fmt.Println("✅ FINALIZED — legally binding; cannot be deleted, only cancelled.")
	} else {
		fmt.Println("✅ Draft created — NOT finalized, safe to delete in InvoiceXpress.")
	}
	fmt.Printf("  ID:        %d\n", inv.ID)
	fmt.Printf("  Status:    %s\n", inv.Status)
	if inv.SequenceNumber != "" {
		fmt.Printf("  Number:    %s\n", inv.SequenceNumber)
	}
	fmt.Printf("  Total:     %s %s\n", inv.Total, inv.Currency)
	if inv.Permalink != "" {
		fmt.Printf("  Permalink: %s\n", inv.Permalink)
	}
}
