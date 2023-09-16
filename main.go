package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"nikc.org/departure-board/hsl"
)

type Departure struct {
	SortBy      int    `json:"-"`
	Departure   string `json:"-"`
	Destination string `json:"dst"`
	Due         string `json:"due"`
	Etd         string `json:"etd"`
	Station     string `json:"sta"`
	Service     string `json:"srv"`
}

var (
	ErrNoStations = errors.New("no stations")
	ErrNotoken    = errors.New("no token")

	HSLToken = ""

	optTimeout = flag.Int("timeout", 5, "timeout for calling the remote service")
	optRows    = flag.Int("num", 10, "number of results to fetch per station")
	optOffset  = flag.Int("offset", 0, "amount to offset current time in minutes")
	optWindow  = flag.Int("window", 0, "width of window to query in minutes")
	jsonOut    = flag.Bool("json", false, "json output")
	stations   []string
)

func init() {
	if token, ok := os.LookupEnv("HSL_TOKEN"); ok {
		HSLToken = token
	}
}

func main() {
	flag.Parse()

	stations = flag.Args()

	if err := mainWithErr(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func mainWithErr(out io.Writer) error {
	if len(stations) == 0 {
		return ErrNoStations
	} else if HSLToken == "" {
		return ErrNotoken
	}

	cli := hsl.New(HSLToken)
	departures := 10
	options := (&hsl.FetchOptions{}).WithRows(uint(departures))

	if optRows != nil {
		departures = *optRows
		options.WithRows(uint(departures))
	}
	if optOffset != nil {
		options.WithOffset(uint(*optOffset))
	}
	if optWindow != nil {
		options.WithWindow(uint(*optWindow))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*optTimeout)*time.Second)
	defer cancel()

	// currentHour := time.Now().Hour()
	results := []Departure{}

	for _, stationCode := range stations {
		page := hsl.DeparturePage{}
		err := cli.GetDepartures(ctx, stationCode, &page, options)
		if err != nil {
			return err
		}

		for _, s := range page.Departures {

			results = append(results,
				Departure{
					SortBy: s.Due.Hour()*60 + s.Due.Minute(),
					Departure: fmt.Sprintf("%-5s %3s %-32s %9s",
						fmt.Sprintf("%02d:%02d", s.Due.Hour(), s.Due.Minute()), s.Service, s.Destination, stringifyEtd(s.Etd, s.Due)),
					Due:         fmt.Sprintf("%02d:%02d", s.Due.Hour(), s.Due.Minute()),
					Etd:         stringifyEtd(s.Etd, s.Due),
					Destination: s.Destination,
					Service:     s.Service,
					Station:     stationCode,
				})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].SortBy < results[j].SortBy
	})

	if len(results) == 0 {
		if *jsonOut {
			return jsonOutput(out, []Departure{})
		} else {
			io.WriteString(out, "no departures\n")
		}
		return nil
	}

	if len(results) < departures {
		departures = len(results)
	}

	if *jsonOut {
		return jsonOutput(out, results[0:departures])
	}

	plainTextOutput(out, results[0:departures])

	return nil
}

func stringifyEtd(due *time.Time, compareTo *time.Time) string {
	if due.Compare(*compareTo) == 0 {
		return "On time"
	}

	return fmt.Sprintf("%02d:%02d", due.Hour(), due.Minute())
}

func plainTextOutput(out io.Writer, departures []Departure) {
	fmt.Fprintf(out, "%-5s %-3s %-32s %9s\n", "When", "Srv", "To", "Expected")
	for _, d := range departures {
		fmt.Fprintln(out, d.Departure)
	}
}

func jsonOutput(out io.Writer, departures []Departure) error {
	page := struct {
		Offset     int            `json:"offset"`
		Stations   map[string]int `json:"stations"`
		Departures []Departure    `json:"departures"`
	}{
		Offset:     *optOffset,
		Stations:   map[string]int{},
		Departures: departures,
	}

	for _, s := range stations {
		page.Stations[s] = 0
	}

	for _, d := range departures {
		page.Stations[d.Station]++
	}

	j, err := json.Marshal(page)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, string(j))

	return nil
}
