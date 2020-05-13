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

        client, err := udnssdk.NewClient(Username, Password, BaseURL)
        if err != nil {

         return nil, fmt.Errorf("Connection cannot be established")
        }


        provider := &UltraDNSProvider{
                client:       *client,
                domainFilter: domainFilter,
                DryRun:       dryRun,
        }

        return provider, nil
}


// Zones returns list of hosted zones
func (p *UltraDNSProvider) Zones(ctx context.Context) ([]http.Request, error) {
        log.Infof ("Under Zones function")
        zones, err := p.fetchZones(ctx)
        if err != nil {
              return nil, err
        }

        return zones, nil
}

func (p *UltraDNSProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
        log.Infof("Under Records function")


        var endpoints []*endpoint.Endpoint
        return endpoints, nil
}

func (p *UltraDNSProvider) fetchRecords(ctx context.Context, domain string) ([]http.Request, error) {
        log.Infof("Under fetchRecords function")
        var req []http.Request

        //if err != nil {
        //      return nil, err
        //}

        return req, nil
}

func (p *UltraDNSProvider) fetchZones(ctx context.Context) ([]http.Request, error) {

        log.Infof("Under fetch zones function")

        var req []http.Request
        return req, nil
}

func (p *UltraDNSProvider) submitChanges(ctx context.Context, changes []*UltraDNSChanges) error {
        log.Infof("In submitChanges function")
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

func seperateChangeByZone(zones string, changes []*UltraDNSChanges) map[string][]*UltraDNSChanges {
        log.Infof("In seperate changes by zone function")
        change := make(map[string][]*UltraDNSChanges)
        return change
}

func (p *UltraDNSProvider) getRecordID(ctx context.Context, zone string, record udnssdk.RRSet) (recordID int, err error) {
        log.Infof("In seperate changes by zone function")
        return 0, fmt.Errorf("no record was found")
}
