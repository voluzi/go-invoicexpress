// Package invoicexpress provides a Go client for the InvoiceXpress API v2.
//
// InvoiceXpress is a Portuguese invoicing platform. This library covers
// all major resources: invoices, estimates, guides, clients, items,
// sequences, taxes, SAF-T exports, and accounts.
//
// # Authentication
//
// Authentication is done via an API key passed as a query parameter on
// every request. Create a client with your account name and API key:
//
//	client := invoicexpress.NewClient("account-name", "your-api-key")
package invoicexpress
