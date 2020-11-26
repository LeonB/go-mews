package bills

import (
	gojson "encoding/json"
	"errors"
	"fmt"

	"github.com/tim-online/go-mews/json"
	"github.com/tim-online/go-mews/omitempty"
)

const (
	endpointGetPDF = "bills/getPDF"
)

// List all products
func (s *Service) GetPDF(requestBody *GetPDFRequest) (*GetPDFResponse, error) {
	// @TODO: create wrapper?
	if err := s.Client.CheckTokens(); err != nil {
		return nil, err
	}

	apiURL, err := s.Client.GetApiURL(endpointGetPDF)
	if err != nil {
		return nil, err
	}

	responseBody := &GetPDFResponse{}
	httpReq, err := s.Client.NewRequest(apiURL, requestBody)
	if err != nil {
		return nil, err
	}

	_, err = s.Client.Do(httpReq, responseBody)
	return responseBody, err
}

func (s *Service) NewGetPDFRequest() *GetPDFRequest {
	return &GetPDFRequest{}
}

type GetPDFRequest struct {
	json.BaseRequest
	// Unique identifier of the Bill to be printed.
	BillID string `json:"BillId"`
	// Unique identifier of the Bill print event returned by previous invocation.
	BillPrintEventID string `json:"BillPrintEventId,omitempty"`
}

func (r GetPDFRequest) MarshalJSON() ([]byte, error) {
	return omitempty.MarshalJSON(r)
}

type GetPDFResponse struct {
	BillID string        `json:"BillId"`
	Result BillPDFResult `json:"Result"`
}

type BillPDFResult struct {
	Discriminator  BillPDFResultDiscriminator `json:"Discriminator"`
	BillPDFFile    BillPDFFile
	BillPrintEvent BillPrintEvent
}

func (r *BillPDFResult) UnmarshalJSON(data []byte) error {
	type Alias BillPDFResult
	st := struct {
		Alias
		Value gojson.RawMessage `json:"value"`
	}{}

	err := gojson.Unmarshal(data, &st)
	if err != nil {
		return err
	}

	if st.Discriminator == BillPDFFileDiscriminator {
		err := gojson.Unmarshal(st.Value, &st.BillPDFFile)
		if err != nil {
			return err
		}
		*r = BillPDFResult(st.Alias)
		return nil
	} else if st.Discriminator == BillPrintEventDiscriminator {
		err := gojson.Unmarshal(st.Value, &st.BillPrintEvent)
		if err != nil {
			return err
		}
		*r = BillPDFResult(st.Alias)
		return nil
	}

	return errors.New(fmt.Sprintf("Unknown discriminator: %s", st.Discriminator))
}

type BillPDFResultDiscriminator string

var (
	BillPDFFileDiscriminator    BillPDFResultDiscriminator = "BillPdfFile"
	BillPrintEventDiscriminator BillPDFResultDiscriminator = "BillPrintEvent"
)

type BillPDFFile struct {
	Base64Data []byte `json:"Base64Data"`
}

type BillPrintEvent struct {
	BillPrintEventID string `json:"BillPrintEventId"`
}
