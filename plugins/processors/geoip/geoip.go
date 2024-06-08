package geoip

import (
	"fmt"
	"net"

	"github.com/IncSW/geoip2"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

const sampleConfig = `
  ## db_path is the location of the MaxMind GeoIP2 City database
  db_path = "/var/lib/GeoIP/GeoLite2-City.mmdb"

  [[processors.geoip.lookup]
	# get the ip from the field "source_ip" and put the lookup results in the respective destination fields (if specified)
	field = "source_ip"
	dest_country = "source_country"
	dest_city = "source_city"
	dest_lat = "source_lat"
	dest_lon = "source_lon"
  `

type lookupEntry struct {
	Field       string `toml:"field"`
	DestCountry string `toml:"dest_country"`
	DestCity    string `toml:"dest_city"`
	DestLat     string `toml:"dest_lat"`
	DestLon     string `toml:"dest_lon"`
	Asn         string `toml:"asn"`
	AsnOrg      string `toml:"asn_org"`
}

type GeoIP struct {
	DBPath  string          `toml:"db_path"`
	DBType  string          `toml:"db_type"`
	Lookups []lookupEntry   `toml:"lookup"`
	Log     telegraf.Logger `toml:"-"`
}

var cityReader *geoip2.CityReader
var countryReader *geoip2.CountryReader
var asnReader *geoip2.ASNReader

func (g *GeoIP) SampleConfig() string {
	return sampleConfig
}

func (g *GeoIP) Description() string {
	return "GeoIP looks up the country code, city name and latitude/longitude for IP addresses in the MaxMind GeoIP database"
}

func (g *GeoIP) Apply(metrics ...telegraf.Metric) []telegraf.Metric {
	for _, point := range metrics {
		for _, lookup := range g.Lookups {
			if lookup.Field != "" {
				if value, ok := point.GetField(lookup.Field); ok {
					if g.DBType == "city" || g.DBType == "" {
						record, err := cityReader.Lookup(net.ParseIP(value.(string)))
						if err != nil {
							if err.Error() != "not found" {
								g.Log.Errorf("GeoIP lookup error: %v", err)
							}
							continue
						}
						if len(lookup.DestCountry) > 0 {
							point.AddField(lookup.DestCountry, record.Country.ISOCode)
						}
						if len(lookup.DestCity) > 0 {
							point.AddField(lookup.DestCity, record.City.Names["en"])
						}
						if len(lookup.DestLat) > 0 {
							point.AddField(lookup.DestLat, record.Location.Latitude)
						}
						if len(lookup.DestLon) > 0 {
							point.AddField(lookup.DestLon, record.Location.Longitude)
						}
					} else if g.DBType == "asn" {
						record, err := asnReader.Lookup(net.ParseIP(value.(string)))
						if err != nil {
							if err.Error() != "not found" {
								g.Log.Errorf("GeoIP lookup error: %v", err)
							}
							continue
						}
						if len(lookup.Asn) > 0 {
							point.AddField(lookup.Asn, record.AutonomousSystemNumber)
						}
						if len(lookup.AsnOrg) > 0 {
							point.AddField(lookup.AsnOrg, record.AutonomousSystemOrganization)
						}
					} else if g.DBType == "country" {
						record, err := countryReader.Lookup(net.ParseIP(value.(string)))
						if err != nil {
							if err.Error() != "not found" {
								g.Log.Errorf("GeoIP lookup error: %v", err)
							}
							continue
						}
						if len(lookup.DestCountry) > 0 {
							point.AddField(lookup.DestCountry, record.Country.ISOCode)
						}
					} else {
						g.Log.Errorf("Invalid GeoIP database type specified: %s", g.DBType)
					}
				}
			}
		}
	}
	return metrics
}

func (g *GeoIP) Init() error {
	if g.DBType == "city" || g.DBType == "" {
		r, err := geoip2.NewCityReaderFromFile(g.DBPath)
		if err != nil {
			return fmt.Errorf("Error opening GeoIP database: %v", err)
		} else {
			cityReader = r
		}
	} else if g.DBType == "country" {
		r, err := geoip2.NewCountryReaderFromFile(g.DBPath)
		if err != nil {
			return fmt.Errorf("Error opening GeoIP database: %v", err)
		} else {
			countryReader = r
		}
	} else if g.DBType == "asn" {
		r, err := geoip2.NewASNReaderFromFile(g.DBPath)
		if err != nil {
			return fmt.Errorf("Error opening GeoIP database: %v", err)
		} else {
			asnReader = r
		}
	} else {
		return fmt.Errorf("Invalid GeoIP database type specified: %s", g.DBType)
	}
	return nil
}

func init() {
	processors.Add("geoip", func() telegraf.Processor {
		return &GeoIP{
			DBPath: "/var/lib/GeoIP/GeoLite2-Country.mmdb",
		}
	})
}
