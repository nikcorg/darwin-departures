package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

	optRows   = flag.Int("num", 10, "number of results to fetch per station")
	optOffset = flag.Int("offset", 0, "amount to offset current time in minutes (-120 to 120)")
	optWindow = flag.Int("window", 0, "width of window to query in minutes, (1 to 120)")
	jsonOut   = flag.Bool("json", false, "json output")
	stations  []string
)

func main() {
	flag.Parse()

	stations = flag.Args()

	if err := mainWithErr(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func mainWithErr() error {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	currentHour := 23
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

			// Adjust hour for sorting
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
			fmt.Println("[]")
		} else {
			fmt.Println("no departures")
		}
		return nil
	}

	if len(results) < departures {
		departures = len(results)
	}

	if *jsonOut {
		return jsonOutput(results[0:departures])
	}

	plainTextOutput(results[0:departures])

	return nil
}

func plainTextOutput(departures []Departure) {
	fmt.Printf("%-5s %s %-20s %3s %9s\n", "When", "Sta", "To", "Plt", "Expected")
	for _, d := range departures {
		fmt.Println(d.Departure)
	}
}

func jsonOutput(departures []Departure) error {
	page := struct {
		Offset     int            `json:"offset"`
		Stations   map[string]int `json:"stations"`
		Departures []Departure    `json:"departures"`
		UpdatedAt  time.Time      `json:"updatedAt"`
	}{
		Offset:     *optOffset,
		Stations:   map[string]int{},
		Departures: departures,
		UpdatedAt:  time.Now().UTC(),
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

	fmt.Println(string(j))

	return nil
}
