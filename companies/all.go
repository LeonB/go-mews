package companies

import (
	"time"

	"github.com/tim-online/go-mews/configuration"
	"github.com/tim-online/go-mews/json"
)

const (
	endpointAll = "companies/getAll"
)

// List all products
func (s *Service) All(requestBody *AllRequest) (*AllResponse, error) {
	// @TODO: create wrapper?
	if err := s.Client.CheckTokens(); err != nil {
		return nil, err
	}

	apiURL, err := s.Client.GetApiURL(endpointAll)
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

func (s *Service) NewAllRequest() *AllRequest {
	return &AllRequest{}
}

type AllRequest struct {
	json.BaseRequest
}

type AllResponse struct {
	Companies []Company `json:"companies"`
}

type Company struct {
	ID                          string                `json:"Id"`                          // Unique identifier of the company.
	Name                        string                `json:"Name"`                        // Name of the company.
	Number                      int                   `json:"Number"`                      // Unique number of the company.
	IsActive                    bool                  `json:"IsActive"`                    // Whether the company is still active.
	Identifier                  string                `json:"Identifier"`                  // Identifier of the company (e.g. legal identifier).
	TaxIdentifier               string                `json:"TaxIdentifier"`               // Tax identification number of the company.
	AdditionalTaxIdentifier     string                `json:"AdditionalTaxIdentifier"`     // Additional tax identifer of the company.
	ElectronicInvoiceIdentifier string                `json:"ElectronicInvoiceIdentifier"` // Electronic invoice identifer of the company.
	AccountingCode              string                `json:"AccountingCode"`              // Accounting code of the company.
	BillingCode                 string                `json:"BillingCode"`                 // Billing code of the company.
	Address                     configuration.Address `json:"Address"`                     // Address of the company (if it is non-empty, otherwise null).
	InvoiceDueInterval          string                `json:"InvoiceDueInterval"`
	CreatedUtc                  time.Time             `json:"CreatedUtc"`
	UpdatedUtc                  time.Time             `json:"UpdatedUtc"`
	Iata                        interface{}           `json:"Iata"`
	Telephone                   string                `json:"Telephone"`
	ContactPerson               string                `json:"ContactPerson"`
	Contact                     string                `json:"Contact"`
	Notes                       string                `json:"Notes"`
	TaxIdentificationNumber     string                `json:"TaxIdentificationNumber"`
}
