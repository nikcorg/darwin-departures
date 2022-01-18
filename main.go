package main

import (
	"context"
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
	SortBy    int
	Departure string
}

var (
	ErrNoStations = errors.New("no stations")
	ErrNotoken    = errors.New("no token")

	DarwinToken = ""

	optRows   = flag.Int("num", 10, "number of results to fetch (maximum)")
	optOffset = flag.Int("offset", 0, "amount to offset current time in minutes (-120 to 120)")
	optWindow = flag.Int("window", 0, "width of window to query in minutes, (1 to 120)")
)

func main() {
	flag.Parse()

	if err := mainWithErr(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func mainWithErr(stations []string) error {
	if len(stations) == 0 {
		return ErrNoStations
	} else if DarwinToken == "" {
		return ErrNotoken
	}

	cli := nationalrail.New(DarwinToken)
	departures := 10
	options := &nationalrail.FetchOptions{Rows: departures}

	if optRows != nil {
		options.Rows, departures = *optRows, *optRows
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
				Departure{hh*60 + mm, fmt.Sprintf("%s %s %-20s %3s %9s", s.Std, stationCode, s.Destination.Location.LocationName, s.Platform, s.Etd)},
			)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].SortBy < results[j].SortBy
	})

	if len(results) == 0 {
		fmt.Println("no departures")
		return nil
	}

	if len(results) < departures {
		departures = len(results)
	}

	fmt.Printf("%-5s %s %-20s %3s %9s\n", "When", "Sta", "To", "Plt", "Expected")
	for _, d := range results[0:departures] {
		fmt.Println(d.Departure)
	}

	return nil
}
