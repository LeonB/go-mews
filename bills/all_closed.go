package bills

import (
	"time"

	"github.com/tim-online/go-mews/accountingitems"
	"github.com/tim-online/go-mews/json"
)

const (
	endpointAllClosed = "bills/getAllClosed"
)

// List all products
func (s *Service) AllClosed(requestBody *AllRequest) (*AllResponse, error) {
	// @TODO: create wrapper?
	if err := s.Client.CheckTokens(); err != nil {
		return nil, err
	}

	apiURL, err := s.Client.GetApiURL(endpointAllClosed)
	if err != nil {
		return nil, err
	}

	responseBody := &AllResponse{}
	httpReq, err := s.Client.NewRequest(apiURL, requestBody)
	if err != nil {
		return nil, err
	}

	_, err = s.Client.Do(httpReq, responseBody)
	return responseBody, err
}

func (s *Service) NewAllClosedRequest() *AllRequest {
	return &AllRequest{}
}

type AllClosedRequest struct {
	json.BaseRequest
	StartUTC *time.Time `json:"StartUtc,omitempty"`
	EndUTC   *time.Time `json:"EndUtc,omitempty"`
}

type AllClosedResponse struct {
	Bills Bills `json:"Bills"` // The closed bills.
}

type Bills []Bill

type Bill struct {
	ID         string    `json:"Id"`         // Unique identifier of the bill.
	CustomerID string    `json:"CustomerId"` // Unique identifier of the Customer the bill is issued to.
	CompanyID  string    `json:"CompanyId"`  // Unique identifier of the Company the bill is issued to.
	CounterID  string    `json:"CounterId"`  // Unique identifier of the bill Counter.
	State      BillState `json:"State"`      // State of the bill.
	Type       BillType  `json:"Type"`       // Type of the bill.
	Number     string    `json:"Number"`     // Number of the bill.
	IssuedUTC  time.Time `json:"IssuedUtc"`  // Date and time of the bill issuance in UTC timezone in ISO 8601 format.
	TaxedUTC   json.Date `json:"TaxedUtc"`   // Taxation date of the bill in UTC timezone in ISO 8601 format.
	DueUTC     time.Time `json:"DueUtc"`     // Bill due date and time in UTC timezone in ISO 8601 format.
	Notes      string    `json:"Notes"`      // Additional notes.
	Revenue    Revenue   `json:"Revenue"`    // The revenue items on the bill.
	Payments   Payments  `json:"Payments"`   // The payments on the bill.
}

type BillType string

const (
	BillTypeReceipt BillType = "Receipt"
	BillTypeInvoice BillType = "Invoice"
)

type BillState string

const (
	BillStateOpen   BillState = "open"
	BillStateClosed BillState = "closed"
)

type Revenue []accountingitems.AccountingItem
type Payments []accountingitems.AccountingItem

type Amount struct {
	Currency   string    `json:"Currency"`   // ISO-4217 code of the Currency.
	NetValue   float64   `json:"NetValue"`   // Net value in case the item is taxed.
	GrossValue float64   `json:"GrossValue"` // Gross value including all taxes.
	TaxValues  TaxValues `json:"TaxValues"`  // The tax values applied.

	// Deprecated?
	Net     float64  `json:"Net"`     // Net value in case the item is taxed.
	Tax     float64  `json:"Tax"`     // Tax value in case the item is taxed.
	TaxRate *float64 `json:"TaxRate"` // Tax rate in case the item is taxed (e.g. 0.21).
	Value   float64  `json:"Value"`   // Amount in the currency (including tax if taxed).
}

type TaxValues []TaxValue

type TaxValue struct {
	Code  string  `json:"Code"`  // Code corresponding to tax type.
	Value float64 `json:"Value"` // Amount of tax applied.
}
