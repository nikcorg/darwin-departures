package nationalrail

import (
	"bytes"
	"context"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
)

const endpoint = "https://lite.realtime.nationalrail.co.uk/OpenLDBWS/ldb11.asmx"

type Client struct {
	token string
}

func New(token string) *Client {
	return &Client{token}
}

type FetchOptions struct {
	Rows       int
	TimeOffset int
	TimeWindow int
}

func (fo *FetchOptions) WithDefaults() *FetchOptions {
	def := *fo

	if fo.Rows != 0 {
		def.Rows = fo.Rows
	}

	return &def
}

func (c *Client) GetDeparturesBoard(ctx context.Context, stationCode string, options *FetchOptions) (*GetDepartureBoardResponse, error) {
	var (
		err     error
		resp    *http.Response
		payload = strings.NewReader(getDeparturesBoardRequest(c.token, stationCode, options))
		request *http.Request
		db      = GetDepartureBoardResponse{}
	)

	if request, err = http.NewRequestWithContext(ctx, "POST", endpoint, payload); err != nil {
		return nil, err
	} else {
		request.Header.Add("Content-Type", "text/xml")
	}

	if resp, err = http.DefaultClient.Do(request); err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = xml.Unmarshal(body, &db); err != nil {
		return nil, err
	}

	return &db, nil
}

var (
	requestTemplate = template.Must(template.New("DeparturesBoardRequst").Parse(
		`<soap:Envelope
	xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
	xmlns:typ="http://thalesgroup.com/RTTI/2013-11-28/Token/types"
	xmlns:ldb="http://thalesgroup.com/RTTI/2017-10-01/ldb/">
	<soap:Header>
		<typ:AccessToken>
			<typ:TokenValue>{{.Token}}</typ:TokenValue>
		</typ:AccessToken>
	</soap:Header>
	<soap:Body>
		<ldb:GetDepartureBoardRequest>
			<ldb:numRows>{{.Options.Rows}}</ldb:numRows>
			<ldb:crs>{{.Station}}</ldb:crs>
			{{if ne 0 .Options.TimeOffset}}
				<ldb:timeOffset>{{.Options.TimeOffset}}</ldb:timeOffset>
			{{end}}
			{{if ne 0 .Options.TimeWindow}}
				<ldb:timeWindow>{{.Options.TimeWindow}}</ldb:timeWindow>
			{{end}}
		</ldb:GetDepartureBoardRequest>
	</soap:Body>
</soap:Envelope>
`))
)

type departuresRequestProps struct {
	Token   string
	Station string
	Options *FetchOptions
}

func getDeparturesBoardRequest(token string, stationCode string, options *FetchOptions) string {
	buf, data := &bytes.Buffer{}, departuresRequestProps{token, stationCode, options.WithDefaults()}

	if err := requestTemplate.Execute(buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}
