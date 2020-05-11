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
        "strconv"
		"strings"
		"net/http"
		"net/url"
		"time"
		
        log "github.com/sirupsen/logrus"
        "github.com/aliasgharmhowwala/ultradns-sdk-go/udnssdk"
        "sigs.k8s.io/external-dns/endpoint"
        "sigs.k8s.io/external-dns/plan"
)

const (
        ultradnsTTL    = 86400
)

type UltraDNSProvider struct {
        client  udnssdk.Client

        domainFilter endpoint.DomainFilter
        DryRun       bool
}

type UltraDNSChanges struct {
        Action string

        ResourceRecordSet udnssdk.RRSets
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

        client := udnssdk.NewClient(Username, Password, BaseURL)
        client.SetUserAgent(fmt.Sprintf("ExternalDNS/%s", client.UserAgent))

        provider := &UltraDNSProvider{
                client:       *client,
                domainFilter: domainFilter,
                DryRun:       dryRun,
        }

        return provider, nil
}


func (p *UltraDNSProvider) fetchZones() {
	p.client.get('/zones/','')
}

func (p *UltraDNSProvider) FindZone (string Zone)(*http.Response, error){
	resp,err := p.client.get('/zones/'+Zone )

	if err != nil {
		log.Errorf("Could not get the requested zone please check the string")
		return resp, err
	}

	return resp,err
}
// ApplyChanges applies a given set of changes in a given zone.
func (p *UltraDNSProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	// zoneNameIDMapper := zoneIDName{}
	// zones, err := p.fetchZones()

	// if err != nil {
	// 	log.Warnf("No zones to fetch endpoints from!")
	// 	return nil
	// }

	// for _, z := range zones.Zones {
	// 	zoneNameIDMapper[z.Zone] = z.Zone
	// }

	// _, cf := p.createRecords(zoneNameIDMapper, changes.Create)
	// if !p.dryRun {
	// 	if len(cf) > 0 {
	// 		log.Warnf("Not all desired endpoints could be created, retrying next iteration")
	// 		for _, f := range cf {
	// 			log.Warnf("Not created was DNSName: '%s' RecordType: '%s'", f.DNSName, f.RecordType)
	// 		}
	// 	}
	// }

	_,cf := p.createZone (endpoint.DNSName,changes.Create)

	if !p.dryRun {
		if len(cf) > 0 {
			log.Warnf("The desired endpoint could be created, retrying next iteration")
			for _, f := range cf {
				log.Warnf("Not created was DNSName: '%s' RecordType: '%s'", f.DNSName, f.RecordType)
			}
		}
	}

	// _, df := p.deleteRecords(zoneNameIDMapper, changes.Delete)
	// if !p.dryRun {
	// 	if len(df) > 0 {
	// 		log.Warnf("Not all endpoints that require deletion could be deleted, retrying next iteration")
	// 		for _, f := range df {
	// 			log.Warnf("Not deleted was DNSName: '%s' RecordType: '%s'", f.DNSName, f.RecordType)
	// 		}
	// 	}
	// }

	// _, uf := p.updateNewRecords(zoneNameIDMapper, changes.UpdateNew)
	// if !p.dryRun {
	// 	if len(uf) > 0 {
	// 		log.Warnf("Not all endpoints that require updating could be updated, retrying next iteration")
	// 		for _, f := range uf {
	// 			log.Warnf("Not updated was DNSName: '%s' RecordType: '%s'", f.DNSName, f.RecordType)
	// 		}
	// 	}
	// }

	// for _, uold := range changes.UpdateOld {
	// 	if !p.dryRun {
	// 		log.Debugf("UpdateOld (ignored) for DNSName: '%s' RecordType: '%s'", uold.DNSName, uold.RecordType)
	// 	}
	// }

	return nil
}

// func (p *UltraDNSProvider) fetchZones() (zones UltraDNSZones, err error) {
// 	log.Debugf("Trying to fetch zones from UltraDNS")
// 	resp, err := p.request("GET", "config-dns/v2/zones?showAll=true&types=primary%2Csecondary", nil)
// 	if err != nil {
// 		log.Errorf("Failed to fetch zones from Akamai")
// 		return zones, err
// 	}

// 	err = json.NewDecoder(resp.Body).Decode(&zones)
// 	if err != nil {
// 		log.Errorf("Could not decode json response from Akamai on zone request")
// 		return zones, err
// 	}
// 	defer resp.Body.Close()

// 	filteredZones := akamaiZones{}
// 	for _, zone := range zones.Zones {
// 		if !p.zoneIDFilter.Match(zone.ContractID) {
// 			log.Debugf("Skipping zone: '%s' with ZoneID: '%s', it does not match against ZoneID filters", zone.Zone, zone.ContractID)
// 			continue
// 		}
// 		filteredZones.Zones = append(filteredZones.Zones, akamaiZone{ContractID: zone.ContractID, Zone: zone.Zone})
// 		log.Debugf("Fetched zone: '%s' (ZoneID: %s)", zone.Zone, zone.ContractID)
// 	}
// 	lenFilteredZones := len(filteredZones.Zones)
// 	if lenFilteredZones == 0 {
// 		log.Warnf("No zones could be fetched")
// 	} else {
// 		log.Debugf("Fetched '%d' zones from Akamai", lenFilteredZones)
// 	}

// 	return filteredZones, nil
// }

func (p *UltraDNSProvider) createZone(string Zone, endpoints []*endpoint.Endpoint) (created []*endpoint.Endpoint, failed []*endpoint.Endpoint) {
	for _, endpoint := range endpoints {

		if !p.domainFilter.Match(endpoint.DNSName) {
			log.Debugf("Skipping creation at UltraDNS of endpoint DNSName: '%s' RecordType: '%s', it does not match against Domain filters", endpoint.DNSName, endpoint.RecordType)
			continue
		}
		zoneNameresponse, _ := FindZone(endpoint.DNSName)
		log.Debugf("The response which got generated "+zoneNameresponse)
		// if zoneName, _ := FindZone(endpoint.DNSName); zoneName == "" {
		// 	akamaiRecord := p.newAkamaiRecord(endpoint.DNSName, endpoint.RecordType, endpoint.Targets...)
		// 	body, _ := json.MarshalIndent(akamaiRecord, "", "  ")

		// 	log.Infof("Create new Endpoint at Akamai FastDNS - Zone: '%s', DNSName: '%s', RecordType: '%s', Targets: '%+v'", zoneName, endpoint.DNSName, endpoint.RecordType, endpoint.Targets)

		// 	if p.dryRun {
		// 		continue
		// 	}
		// 	_, err := p.request("POST", "config-dns/v2/zones/"+zoneName+"/names/"+endpoint.DNSName+"/types/"+endpoint.RecordType, bytes.NewReader(body))
		// 	if err != nil {
		// 		log.Errorf("Failed to create Akamai endpoint DNSName: '%s' RecordType: '%s' for zone: '%s'", endpoint.DNSName, endpoint.RecordType, zoneName)
		// 		failed = append(failed, endpoint)
		// 		continue
		// 	}
		// 	created = append(created, endpoint)
		// } else {
		// 	log.Warnf("No matching zone for endpoint addition DNSName: '%s' RecordType: '%s'", endpoint.DNSName, endpoint.RecordType)
		// 	failed = append(failed, endpoint)
		// }
	}
	return created, failed
}
