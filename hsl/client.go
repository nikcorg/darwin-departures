package hsl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"
)

type DeparturePage struct {
	Stop struct {
		Name string
		Code string
	}
	Departures []Departure
}

type Departure struct {
	Service     string
	Departure   string
	Destination string
	Due         *time.Time
	Etd         *time.Time
	Type        string
	Station     string
}

type Client struct {
	token    string
	endpoint string
}

func New(t string) *Client {
	return &Client{t, hslQueryEndpoint}
}

type FetchOptions struct {
	Rows       uint
	TimeOffset uint
	TimeWindow uint

	startTime time.Time
}

func (fo *FetchOptions) WithRows(r uint) *FetchOptions {
	if r > 0 {
		fo.Rows = r
	}
	return fo
}

func (fo *FetchOptions) WithWindow(o uint) *FetchOptions {
	if o > 0 {
		fo.TimeWindow = o
	}
	return fo
}

func (fo *FetchOptions) WithOffset(o uint) *FetchOptions {
	if o > 0 {
		fo.TimeOffset = o
		fo.startTime = time.Now().Add(time.Minute * time.Duration(o))
	}

	return fo
}

func (c *Client) GetDepartures(ctx context.Context, sta string, page *DeparturePage, opts *FetchOptions) error {
	resp, err := c.sendQuery(ctx, sta, opts)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var gresp graphResponse[stopQueryResponse]
	if err := json.Unmarshal(body, &gresp); err != nil {
		return err
	}

	if err := gresp.Error(); err != nil {
		return err
	}

	stop := gresp.Data.Stop

	page.Stop.Name = stop.Name
	page.Stop.Code = stop.Code

	for _, s := range stop.StopTimes {
		d := Departure{}

		sat := arrivalToTime(s.ScheduledArrival)
		rta := arrivalToTime(s.RealtimeArrival)

		d.Service = s.Trip.Route.ShortName
		d.Destination = s.Headsign
		d.Station = stop.Name
		d.Type = fmt.Sprintf("%s", shortVehicleMode(stop.VehicleMode))
		d.Due = sat
		d.Etd = rta

		page.Departures = append(page.Departures, d)
	}

	return nil
}

func arrivalToTime(t uint64) *time.Time {
	t0 := time.Now().UTC().Round(time.Hour * 24).Add(time.Duration(t) * time.Second).Round(time.Second)

	return &t0
}

func shortVehicleMode(m string) string {
	switch m {
	case "TRAM":
		return "T"
	case "METRO":
		return "M"
	case "BUS":
		return "B"
	}

	return "?"
}

type graphRequest[T any] struct {
	Query     string `json:"query"`
	Variables T      `json:"variables,omitempty"`
}

type graphResponse[T any] struct {
	Data   T            `json:"data"`
	Errors []graphError `json:"errors"`
}

func (r *graphResponse[T]) Error() error {
	if len(r.Errors) > 0 {
		return errors.New("graph response includes errors")
	}

	return nil
}

type graphError struct {
	Message string
}

func (c *Client) sendQuery(ctx context.Context, sta string, opts *FetchOptions) (*http.Response, error) {
	query, err := getDeparturesQuery(sta, opts)
	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(json))
	if err != nil {
		return nil, err
	}

	req.Header.Add(hslAuthHeader, c.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type requestOpts struct {
	Stop      string `json:"stop"`
	TimeRange uint   `json:"timeRange"`
	StartTime uint64 `json:"startTime"`
	Num       uint   `json:"num"`
}

func getDeparturesQuery(stop string, opts *FetchOptions) (*graphRequest[requestOpts], error) {
	ropts := requestOpts{Stop: stop, Num: opts.Rows}

	if opts.TimeWindow > 0 {
		ropts.TimeRange = opts.TimeWindow
	}

	if opts.TimeOffset > 0 {
		ropts.StartTime = uint64(opts.startTime.Unix())
	}

	buf := &bytes.Buffer{}
	if err := gqlReq.Execute(buf, ropts); err != nil {
		return nil, err
	}

	body := graphRequest[requestOpts]{}
	body.Query = buf.String()
	body.Variables = ropts

	return &body, nil
}

type stopQueryResponse struct {
	Stop stop `json:"stop"`
}

type stop struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	Code        string `json:"code"`
	VehicleMode string `json:"vehicleMode"`
	VehicleType int    `json:"vehicleType"`

	StopTimes []stopTime `json:"stoptimesWithoutPatterns"`
}

type stopTime struct {
	Trip             trip   `json:"trip"`
	ScheduledArrival uint64 `json:"scheduledArrival"`
	RealtimeArrival  uint64 `json:"realtimeArrival"`
	ArrivalDelay     int64  `json:"arrivalDelay"`
	RealtimeState    string `json:"realtimeState"`
	Headsign         string `json:"headsign"`
	PickupType       string `json:"pickupType"`
}

type trip struct {
	Route     route  `json:"route"`
	ServiceID string `json:"serviceId"`
}

type route struct {
	ShortName string `json:"shortName"`
}

// HSL:1220409
const hslAuthHeader = "digitransit-subscription-key"
const hslQueryEndpoint = "https://api.digitransit.fi/routing/v1/routers/hsl/index/graphql"
const hslQuery = `query (
	$stop: String!
	$num: Int
	{{if gt .StartTime 0}}$startTime: Long{{end}}
	{{if gt .TimeRange 0}}$timeRange: Int{{end}}
) {
  stop(id: $stop) {
    name
    desc
		code
		vehicleMode
		vehicleType

    stoptimesWithoutPatterns(
			numberOfDepartures: $num
			{{if gt .StartTime 0}}startTime: $startTime{{end}}
			{{if gt .TimeRange 0}}timeRange: $timeRange{{end}}
		) {
      trip {
        route {
          shortName
        }
        serviceId
      }
      scheduledArrival
      realtimeArrival
      arrivalDelay
      realtimeState
      headsign
      pickupType
    }
  }
}`

var gqlReq = template.Must(template.New("query").Parse(hslQuery))
