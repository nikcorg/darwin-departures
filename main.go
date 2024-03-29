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
	"strconv"
	"strings"
	"time"

	"nikc.org/departure-board/nationalrail"
)

type Departure struct {
	SortBy      int    `json:"-"`
	Departure   string `json:"-"`
	Destination string `json:"dst"`
	Due         string `json:"due"`
	Etd         string `json:"etd"`
	Platform    string `json:"pla"`
	Station     string `json:"sta"`
}

var (
	ErrNoStations = errors.New("no stations")
	ErrNotoken    = errors.New("no token")

	DarwinToken = ""

	optTimeout = flag.Int("timeout", 5, "timeout for calling the remote service")
	optRows    = flag.Int("num", 10, "number of results to fetch per station")
	optOffset  = flag.Int("offset", 0, "amount to offset current time in minutes (-120 to 120)")
	optWindow  = flag.Int("window", 0, "width of window to query in minutes, (1 to 120)")
	jsonOut    = flag.Bool("json", false, "json output")
	stations   []string
)

func init() {
	if token, ok := os.LookupEnv("DARWIN_TOKEN"); ok {
		DarwinToken = token
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
	} else if DarwinToken == "" {
		return ErrNotoken
	}

	cli := nationalrail.New(DarwinToken)
	departures := 10
	options := &nationalrail.FetchOptions{Rows: departures}

	if optRows != nil {
		options.Rows, departures = *optRows, *optRows*len(stations)
	}
	if optOffset != nil {
		options.TimeOffset = *optOffset
	}
	if optWindow != nil {
		options.TimeWindow = *optWindow
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*optTimeout)*time.Second)
	defer cancel()

	currentHour := time.Now().Hour()
	results := []Departure{}

	for _, stationCode := range stations {
		r, err := cli.GetDeparturesBoard(ctx, stationCode, options)
		if err != nil {
			return err
		}

		res := r.Body.GetDepartureBoardResponse.GetStationBoardResult
		for _, s := range res.TrainServices.Service {
			hours, minutes, ok := strings.Cut(s.Std, ":")
			if !ok {
				return errors.New("error parsing departure time")
			}

			hh, _ := strconv.Atoi(hours)
			mm, _ := strconv.Atoi(minutes)

			// If we're past midday and the departure hour is less than 12 that means it's an after midnight 
			// time, because all departures are in the future. Add 24 to ensure the hour will slot in in the 
			// correct position when sorting. This only affects the sorting order, not the displayed time.
			if currentHour > 12 && hh < 12 {
				hh += 24
			}

			results = append(results,
				Departure{
					SortBy:      hh*60 + mm,
					Departure:   fmt.Sprintf("%s %s %-20s %3s %9s", s.Std, stationCode, s.Destination.Location.LocationName, s.Platform, s.Etd),
					Due:         s.Std,
					Platform:    s.Platform,
					Etd:         s.Etd,
					Destination: s.Destination.Location.LocationName,
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
			io.WriteString(out, "no departures")
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

func plainTextOutput(out io.Writer, departures []Departure) {
	fmt.Fprintf(out, "%-5s %s %-20s %3s %9s\n", "When", "Sta", "To", "Plt", "Expected")
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
