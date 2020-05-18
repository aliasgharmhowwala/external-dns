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
        _ "net/http"
        "time"
	_ "strings"

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

        ResourceRecordSetUltraDNS udnssdk.RRSet
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
	var endpoints []*endpoint.Endpoint
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

        log.Infof("zones : %v",zones)

	for _, zone := range zones {
		log.Infof("zones : %v",zone)
		rrsetType  := ""
		ownerName := ""
                rrsetKey := udnssdk.RRSetKey{
                        Zone: zone.Properties.Name,
                        Type: rrsetType,
                        Name: ownerName,
                }
                
                if zone.Properties.ResourceRecordCount !=0 {
		        records, err := p.fetchRecords(ctx, zone.Properties.Name, rrsetKey)
                        if err != nil {
                                return nil, err
                        }

                        for _, r := range records {
                                        log.Infof("owner name %s",r.OwnerName)
                                        name := fmt.Sprintf("%s%s", r.OwnerName, zone.Properties.Name)

                                        // root name is identified by the empty string and should be
                                        // translated to zone name for the endpoint entry.
                                        if r.OwnerName == "" {
                                                name = zone.Properties.Name
                                        }

                                        endPointTTL := endpoint.NewEndpointWithTTL(name, r.RRType, endpoint.TTL(r.TTL),r.RData...)
                                        log.Infof("endpoint with TTL %v",endPointTTL)
                                        endpoints = append(endpoints, endPointTTL)
                        }
                }

       }
	log.Infof("endpoints %v",endpoints)
	return endpoints, nil
}

func (p *UltraDNSProvider) fetchRecords(ctx context.Context, domain string, k udnssdk.RRSetKey) ([]udnssdk.RRSet , error) {
        log.Infof("Under fetchRecords function")

        // TODO: Sane Configuration for timeouts / retries
	maxerrs := 5
	waittime := 5 * time.Second

	rrsets := []udnssdk.RRSet{}
	errcnt := 0
        offset := 0
        limit  := 1000

	for {
		reqRrsets, ri, res, err := p.client.RRSets.SelectWithOffsetWithLimit(k, offset, limit)
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
                                if p.domainFilter.Match(zone.Properties.Name) {
					zones = append(zones, zone)
				}
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
	log.Infof("Change Printin submitChanges by zone %+v\n%+v ",changes,zones)
	zoneChanges := seperateChangeByZone(zones, changes)

	for zoneName, changes := range zoneChanges {
		for _, change := range changes {

			log.WithFields(log.Fields{
				"record": change.ResourceRecordSetUltraDNS.OwnerName,
				"type":   change.ResourceRecordSetUltraDNS.RRType,
				"ttl":    change.ResourceRecordSetUltraDNS.TTL,
				"action": change.Action,
				"zone":   zoneName,
			}).Info("Changing record.")

			switch change.Action {
                        case ultradnsCreate:
                                rrsetKey := udnssdk.RRSetKey{
                                        Zone: zoneName,
                                        Type: change.ResourceRecordSetUltraDNS.RRType,
                                        Name: change.ResourceRecordSetUltraDNS.OwnerName,
                                }
				res,err := p.client.RRSets.Create(rrsetKey, change.ResourceRecordSetUltraDNS)
				_ = res
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
        changes.Create = []*endpoint.Endpoint{
                {DNSName: "kubernetes.ultradns.provider.test.com.", RecordType: "A", Targets: endpoint.Targets{"192.168.0.104"}},
                {DNSName: "ttl.kubernetes.ultradns.provider.test.com.", RecordType: "TXT", Targets: endpoint.Targets{"192.168.0.104"}, RecordTTL: 100},
        }

        combinedChanges := make([]*UltraDNSChanges, 0, len(changes.Create)+len(changes.UpdateNew)+len(changes.Delete))
        log.Infof("value of changes %v,%v,%v",changes.Create,changes.UpdateNew,changes.Delete)
        combinedChanges = append(combinedChanges, newUltraDNSChanges(ultradnsCreate, changes.Create)...)
        combinedChanges = append(combinedChanges, newUltraDNSChanges(ultradnsUpdate, changes.UpdateNew)...)
        combinedChanges = append(combinedChanges, newUltraDNSChanges(ultradnsDelete, changes.Delete)...)

        return p.submitChanges(ctx, combinedChanges)
}

func newUltraDNSChanges(action string, endpoints []*endpoint.Endpoint) []*UltraDNSChanges {
        log.Infof("In newUltraDNSChanges function action string '%s' ",action)
        changes := make([]*UltraDNSChanges, 0, len(endpoints))
        ttl := ultradnsTTL
	log.Infof("Changes %v ",endpoints)
        for _, e := range endpoints {

                if e.RecordTTL.IsConfigured() {
                        ttl = int(e.RecordTTL)
                }

                change := &UltraDNSChanges{
                        Action: action,
                        ResourceRecordSetUltraDNS: udnssdk.RRSet{
                                RRType: e.RecordType,
                                OwnerName: e.DNSName,
                                RData: e.Targets,
                                TTL:  ttl,
                        },
                }
		log.Infof("Changes %v ",change)
                changes = append(changes, change)
        }
        return changes
}

func seperateChangeByZone(zones []udnssdk.Zone, changes []*UltraDNSChanges) map[string][]*UltraDNSChanges {
        log.Infof("In seperate changes by zone function")
        change := make(map[string][]*UltraDNSChanges)
	zoneNameID := zoneIDName{}
	log.Infof("zones Printin seperate changes by zones %+v ",zones)
	for _, z := range zones {
		zoneNameID.Add(z.Properties.Name, z.Properties.Name)
		change[z.Properties.Name] = []*UltraDNSChanges{}
	}
        log.Infof("Change Printin seperate changes by zone %+v ",change)

	for _, c := range changes {
		log.Infof("owner Name: %s",c.ResourceRecordSetUltraDNS.OwnerName)
		zone, _ := zoneNameID.FindZone(c.ResourceRecordSetUltraDNS.OwnerName)
		if zone == "" {
			log.Infof("Skipping record %s because no hosted zone matching record DNS Name was detected", c.ResourceRecordSetUltraDNS.OwnerName)
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
