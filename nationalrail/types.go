package nationalrail

import "encoding/xml"

type GetDepartureBoardResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetDepartureBoardResponse struct {
			GetStationBoardResult struct {
				GeneratedAt  string `xml:"generatedAt"`
				LocationName string `xml:"locationName"`
				Crs          string `xml:"crs"`
				NrccMessages struct {
					Message string `xml:"message"`
				} `xml:"nrccMessages"`
				BusServices struct {
					Service []TService `xml:"service"`
				} `xml:"busServices"`
				TrainServices struct {
					Service []TService `xml:"service"`
				} `xml:"trainServices"`
			} `xml:"GetStationBoardResult"`
		} `xml:"GetDepartureBoardResponse"`
	} `xml:"Body"`
}

type TService struct {
	Std          string `xml:"std"`
	Etd          string `xml:"etd"`
	Platform     string `xml:"platform"`
	Operator     string `xml:"operator"`
	OperatorCode string `xml:"operatorCode"`
	ServiceType  string `xml:"serviceType"`
	Length       string `xml:"length"`
	ServiceID    string `xml:"serviceID"`
	Origin       struct {
		Location struct {
			LocationName string `xml:"locationName"`
			Crs          string `xml:"crs"`
		} `xml:"location"`
	} `xml:"origin"`
	Destination struct {
		Location struct {
			LocationName string `xml:"locationName"`
			Crs          string `xml:"crs"`
		} `xml:"location"`
	} `xml:"destination"`
}
