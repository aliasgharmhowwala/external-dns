/*
Copyright 2017 The Kubernetes Authors.

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
	"net/http"
	"os"
	"reflect"
	_ "strings"
	"testing"
	"encoding/json"
	"log"
	"fmt"

	udnssdk "github.com/aliasgharmhowwala/ultradns-sdk-go"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

type mockUltraDNSZone struct {
	client *udnssdk.Client
}

func (m *mockUltraDNSZone) Create(domain, InstanceIP string) error {
	return nil
}

func (m *mockUltraDNSZone) Delete(ctx context.Context, domain string) error {
	return nil
}

func (m *mockUltraDNSZone) SelectWithOffset(k udnssdk.ZoneKey, offset int, limit int) (zones []udnssdk.Zone, ResultInfo string, resp *http.Response, err error) {
	zones = []udnssdk.Zone{}
	zone := udnssdk.Zone{}
	zoneJson := `{
		   Properties { 
				Name:                 "test-ultradns-provider.com.",
				AccountName:          "teamrest",
				Type:                 "PRIMARY",
				DnssecStatus:         "UNSIGNED",
				Status:               "ACTIVE",
				Owner:                "teamrest",
				ResourceRecordCount:  7,
				LastModifiedDateTime: "",
			      },
		}`
	if err := json.Unmarshal([]byte(zoneJson), &zone); err != nil {
        log.Fatal(err)
    	}
	
	zones = append(zones,zone)
	return zones, "", nil, nil
}

type mockUltraDNSRecord struct {
	client *udnssdk.Client
}

func (m *mockUltraDNSRecord) Create(k udnssdk.RRSetKey, rrset udnssdk.RRSet) error {
	return nil
}

func (m *mockUltraDNSRecord) Delete(k udnssdk.RRSetKey) error {
	return nil
}

func (m *mockUltraDNSRecord) SelectWithOffsetWithLimit(k udnssdk.RRSetKey, offset int, limit int) (rrsets []udnssdk.RRSet, ResultInfo string, resp *http.Response, err error) {
	return []udnssdk.RRSet{{
		OwnerName: "test-ultradns-provider.com.",
		RRType:    endpoint.RecordTypeA,
		RData:     []string{"1.1.1.1"},
		TTL:       86400,
	}}, "", nil, nil
}

func (m *mockUltraDNSRecord) Update(k udnssdk.RRSetKey, rrset udnssdk.RRSet) error {
	return nil
}

func TestNewUltraDNSProvider(t *testing.T) {
	_ = os.Setenv("ULTRADNS_USERNAME", "")
	_ = os.Setenv("ULTRADNS_PASSWORD", "")
	_ = os.Setenv("ULTRADNS_BASEURL", "")
	_ = os.Setenv("ULTRADNS_ACCOUNTNAME", "")
	_, err := NewUltraDNSProvider(endpoint.NewDomainFilter([]string{"test-ultradns-provider.com"}), true)
	if err != nil {
		t.Errorf("failed : %s", err)
	}

	_ = os.Unsetenv("ULTRADNS_PASSWORD")
	_ = os.Unsetenv("ULTRADNS_USERNAME")
	_ = os.Unsetenv("ULTRADNS_BASEURL")
	_ = os.Unsetenv("ULTRADNS_ACCOUNTNAME")
	_, err = NewUltraDNSProvider(endpoint.NewDomainFilter([]string{"test-ultradns-provider.com"}), true)
	if err == nil {
		t.Errorf("expected to fail")
	}
}

func TestUltraDNSProvider_Zones(t *testing.T) {
	mocked := mockUltraDNSZone{}
	provider := &UltraDNSProvider{
		client: udnssdk.Client{
			Zone: mocked.client.Zone,
		},
	}
	
	expected, _, _, err := provider.client.Zone.SelectWithOffset(&udnssdk.ZoneKey{}, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	zones, err := provider.Zones(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, zones) {
		t.Fatal(err)
	}
}

func TestUltraDNSProvider_Records(t *testing.T) {
	mocked := mockUltraDNSRecord{nil}
	mockedDomain := mockUltraDNSZone{nil}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
			RRSets: mocked.client.RRSets,
			Zone:   mockedDomain.client.Zone,
		},
	}
	rrsetKey := udnssdk.RRSetKey{}
	expected, _, _, err := provider.client.RRSets.SelectWithOffsetWithLimit(rrsetKey, 0, 1000)
	records, err := provider.Records(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range records {
		assert.Equal(t, fmt.Sprintf("%s.", v.DNSName), expected[0].OwnerName)
		assert.Equal(t, v.RecordType, expected[0].RRType)
		assert.Equal(t, int(v.RecordTTL), expected[0].TTL)
	}

}

func TestUltraDNSProvider_ApplyChanges(t *testing.T) {
	changes := &plan.Changes{}
	mocked := mockUltraDNSRecord{nil}
	mockedDomain := mockUltraDNSZone{nil}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
                        RRSets: mocked.client.RRSets,
                        Zone:   mockedDomain.client.Zone,

		},
	}

	changes.Create = []*endpoint.Endpoint{
		{DNSName: "test-ultradns-provider.com.", Targets: endpoint.Targets{"1.1.1.1"}},
		{DNSName: "ttl.test-ultradns-provider.com.", Targets: endpoint.Targets{"1.1.1.1"}, RecordTTL: 100},
	}

	changes.UpdateNew = []*endpoint.Endpoint{{DNSName: "test-ultradns-provider.com.", Targets: endpoint.Targets{"1.1.2.2"}, RecordType: "A", RecordTTL: 100}}
	changes.Delete = []*endpoint.Endpoint{{DNSName: "test-ultradns-provider.com.", Targets: endpoint.Targets{"1.1.2.2"}, RecordType: "A"}}
	err := provider.ApplyChanges(context.Background(), changes)
	if err != nil {
		t.Errorf("should not fail, %s", err)
	}
}

func TestUltraDNSProvider_getSpecificRecord(t *testing.T) {
	mocked := mockUltraDNSRecord{nil}
	mockedDomain := mockUltraDNSZone{nil}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
                        RRSets: mocked.client.RRSets,
                        Zone:   mockedDomain.client.Zone,

		},
	}

	recordSetKey := udnssdk.RRSetKey{
		Zone: "test-ultradns-provider.com.",
		Type: "A",
		Name: "teamrest",
	}
	err := provider.getSpecificRecord(context.Background(), recordSetKey)
	if err != nil {
		t.Fatal(err)
	}

}
