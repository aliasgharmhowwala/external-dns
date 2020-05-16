/*
Copyright 2020 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
        "context"
        "fmt"
        "os"
        "net/http"
        "time"

        log "github.com/sirupsen/logrus"
        udnssdk "github.com/aliasgharmhowwala/ultradns-sdk-go"
        "sigs.k8s.io/external-dns/endpoint"
        "sigs.k8s.io/external-dns/plan"
)

const (
                ultradnsTTL    = 86400
                ultradnsCreate = "CREATE"
                ultradnsDelete = "DELETE"
                ultradnsUpdate = "UPDATE"
)

type UltraDNSProvider struct {
        client  udnssdk.Client

        domainFilter endpoint.DomainFilter
        DryRun       bool
        AccountName string
}

type UltraDNSChanges struct {
        Action string

        ResourceRecordSetULtraDNS udnssdk.RRSet
}
type UltraDNSZones struct {
        Zones []UltraDNSZone `json:"zones"`
}

type UltraDNSZone struct {

        Properties struct {
                Name string `json:"name"`
                AccountName string `json:"accountName`
                Type string `json:"type"`
                DnssecStatus string `json:"dnssecStatus"`
                Status string `json:"status"`
                Owner string `json:"owner"`
                ResourceRecordCount int `json:"resourceRecordCount"`
                LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
        } `json:"properties"`
}



// NewUltraDNSProvider initializes a new UltraDNS DNS based provider
func NewUltraDNSProvider(domainFilter endpoint.DomainFilter, dryRun bool) (*UltraDNSProvider, error) {
	log.Infof ("Under provider function")
        Username, ok := os.LookupEnv("ULTRADNS_USERNAME")
        if !ok {
                return nil, fmt.Errorf("no username found")
        }

        Password, ok := os.LookupEnv("ULTRADNS_PASSWORD")
        if !ok {
                return nil, fmt.Errorf("no password found")
        }

        BaseURL, ok := os.LookupEnv("ULTRADNS_BASEURL")
        if !ok {
                return nil, fmt.Errorf("no baseurl found")
        }
	AccountName, ok := os.LookupEnv("ULTRADNS_ACCOUNTNAME")
        if !ok {
                return nil, fmt.Errorf("Please provide valid accountname")
        }

        client, err := udnssdk.NewClient(Username, Password, BaseURL)
        if err != nil {

         return nil, fmt.Errorf("Connection cannot be established")
        }



        provider := &UltraDNSProvider{
                client:       *client,
                domainFilter: domainFilter,
                DryRun:       dryRun,
                AccountName:  AccountName,
        }

        
        return provider, nil
}


// Zones returns list of hosted zones
func (p *UltraDNSProvider) Zones(ctx context.Context) ([]udnssdk.Zone, error) {
        log.Infof ("Under Zones function")
	zonename := ""
        zoneKey := &udnssdk.ZoneKey{
                Zone: zonename,
                AccountName: p.AccountName,
        }
        zones, err := p.fetchZones(ctx,zoneKey)
        if err != nil {
              return nil, err
        }

        return zones, nil
}

func (p *UltraDNSProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
        log.Infof("Under Records function")
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

        log.Infof("zones : %v",zones)

	for _, zone := range zones {
		log.Infof("zones : %v",zone)
                rrsetKey := udnssdk.RRSetKey{
                        Zone: zone.Properties.Name
                        Type: string('')
                        Name: string('')
                }
		records, err := p.fetchRecords(ctx, zone.Properties.Name)
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			if supportedRecordType(r.RRType) {
				name := fmt.Sprintf("%s.%s", r.OwnerName, zone.Properties.Name)

				// root name is identified by the empty string and should be
				// translated to zone name for the endpoint entry.
				if r.OwnerName == "" {
					name = zone.Properties.Name
				}

				endPointTTL := endpoint.NewEndpointWithTTL(name, r.RRType, endpoint.TTL(r.TTL), r.RData)
				endpoints = append(endpoints, endPointTTL)
			}
		}
        }
        var endpoints []*endpoint.Endpoint
	return endpoints, nil
}

func (p *UltraDNSProvider) fetchRecords(ctx context.Context, domain string, k udnssdk.RRSetKey) ([]udnssdk.RRSet[], error) {
        log.Infof("Under fetchRecords function")

        // TODO: Sane Configuration for timeouts / retries
	maxerrs := 5
	waittime := 5 * time.Second

	rrsets := []RRSet{}
	errcnt := 0
        offset := 0
        limit  := 1000

	for {
		reqRrsets, ri, res, err := s.SelectWithOffsetWithLimit(k, offset, limit)
		if err != nil {
			if res != nil && res.StatusCode >= 500 {
				errcnt = errcnt + 1
				if errcnt < maxerrs {
					time.Sleep(waittime)
					continue
				}
			}
			return rrsets, err
		}

		log.Printf("ResultInfo: %+v\n", ri)
		for _, rrset := range reqRrsets {
			rrsets = append(rrsets, rrset)
		}
		if ri.ReturnedCount+ri.Offset >= ri.TotalCount {
			return rrsets, nil
		}
		offset = ri.ReturnedCount + ri.Offset
		continue
	}
}



func (p *UltraDNSProvider) fetchZones(ctx context.Context, zoneKey *udnssdk.ZoneKey) ([]udnssdk.Zone, error) {
        log.Infof("Under fetch zones function")
                                
        // Select will list the zone rrsets, paginating through all available results
        // TODO: Sane Configuration for timeouts / retries
        maxerrs := 5
        waittime := 5 * time.Second

        zones := []udnssdk.Zone{}
	
        errcnt := 0
        offset := 0
        limit := 1000

        for {
                reqZones, ri, res, err := p.client.Zone.SelectWithOffset(zoneKey, offset, limit)
                if err != nil {
                        if res != nil && res.StatusCode >= 500 {
                                errcnt = errcnt + 1
                                if errcnt < maxerrs {
                                        time.Sleep(waittime)
                                        continue      
                                }
                        }
                        return zones, err
                }

                log.Printf("ResultInfo: %+v\n", ri)
                for _, zone := range reqZones {
                        
			if p.domainFilter.IsConfigured() {
                                p.domainFilter.Match(zone.Properties.Name)
                                zones = append(zones, zone)
                        } else {
				 zones = append(zones, zone)
			}
                }
                if ri.ReturnedCount+ri.Offset >= ri.TotalCount {
                        return zones, nil
                }
                offset = ri.ReturnedCount + ri.Offset
                continue
        }
}

func (p *UltraDNSProvider) submitChanges(ctx context.Context, changes []*UltraDNSChanges) error {
        log.Infof("In submitChanges function")
        if len(changes) == 0 {
		log.Infof("All records are already up to date")
		return nil
	}

	zones, err := p.Zones(ctx)
	if err != nil {
		return err
	}

	zoneChanges := seperateChangeByZone(zones, changes)

	for zoneName, changes := range zoneChanges {
		for _, change := range changes {

			log.WithFields(log.Fields{
				"record": change.ResourceRecordSet.OwnerName,
				"type":   change.ResourceRecordSet.RRType,
				"ttl":    change.ResourceRecordSet.TTL,
				"action": change.Action,
				"zone":   zoneName,
			}).Info("Changing record.")

			switch change.Action {
                        case vultrCreate:
                                rrsetKey := udnssdk.RRSetKey{
                                        Zone: zoneName
                                        Type: change.ResourceRecordSet.RRType
                                        Name: change.ResourceRecordSet.OwnerName
                                }
				err = p.client.RRSets.Create(rrsetKey, change.ResourceRecordSet)
				if err != nil {
					return err
				}
			}
		}
	}

        return nil
}

func (p *UltraDNSProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
        log.Infof("In ApplyChanges function")

        combinedChanges := make([]*UltraDNSChanges, 0, len(changes.Create)+len(changes.UpdateNew)+len(changes.Delete))

        combinedChanges = append(combinedChanges, newUltraDNSChanges(ultradnsCreate, changes.Create)...)
        combinedChanges = append(combinedChanges, newUltraDNSChanges(ultradnsUpdate, changes.UpdateNew)...)
        combinedChanges = append(combinedChanges, newUltraDNSChanges(ultradnsDelete, changes.Delete)...)

        return p.submitChanges(ctx, combinedChanges)
}

func newUltraDNSChanges(action string, endpoints []*endpoint.Endpoint) []*UltraDNSChanges {
        log.Infof("In newUltraDNSChanges function action string '%s' ",action)
        changes := make([]*UltraDNSChanges, 0, len(endpoints))
        ttl := ultradnsTTL
        for _, e := range endpoints {

                if e.RecordTTL.IsConfigured() {
                        ttl = int(e.RecordTTL)
                }

                change := &UltraDNSChanges{
                        Action: action,
                        ResourceRecordSetULtraDNS: udnssdk.RRSet{
                                RRType: e.RecordType,
                                OwnerName: e.DNSName,
                                RData: e.Targets,
                                TTL:  ttl,
                        },
                }
                changes = append(changes, change)
        }
        return changes
}

func seperateChangeByZone(zones []udnssdk.Zone, changes []*UltraDNSChanges) map[string][]*UltraDNSChanges {
        log.Infof("In seperate changes by zone function")
        change := make(map[string][]*UltraDNSChanges)
	zoneNameID := zoneIDName{}

	for _, z := range zones {
		zoneNameID.Add(z.Properties.Name, z.Properties.Name)
		change[z.Properties.Name] = []*UltraDNSChanges{}
	}

	for _, c := range changes {
		zone, _ := zoneNameID.FindZone(c.ResourceRecordSet.OwnerName)
		if zone == "" {
			log.Debugf("Skipping record %s because no hosted zone matching record DNS Name was detected", c.ResourceRecordSet.OwnerName)
			continue
		}
		change[zone] = append(change[zone], c)

	}
        return change
}

func (p *UltraDNSProvider) getRecordID(ctx context.Context, zone string, record udnssdk.RRSet) (recordID int, err error) {
        log.Infof("In seperate changes by zone function")
        return 0, fmt.Errorf("no record was found")
}
