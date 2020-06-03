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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	_ "strings"
	"testing"

	udnssdk "github.com/aliasgharmhowwala/ultradns-sdk-go"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

type mockUltraDNSZone struct {
	client *udnssdk.Client
}

func (m *mockUltraDNSZone) SelectWithOffsetWithLimit(k *udnssdk.ZoneKey, offset int, limit int) (zones []udnssdk.Zone, ResultInfo udnssdk.ResultInfo, resp *http.Response, err error) {
	zones = []udnssdk.Zone{}
	zone := udnssdk.Zone{}
	zoneJson := `
                        {
                           "properties": {
                              "name":"test-ultradns-provider.com.",
                              "accountName":"teamrest",
                              "type":"PRIMARY",
                              "dnssecStatus":"UNSIGNED",
                              "status":"ACTIVE",
                              "owner":"teamrest",
                              "resourceRecordCount":7,
                              "lastModifiedDateTime":""
                           }
                        }`
	if err := json.Unmarshal([]byte(zoneJson), &zone); err != nil {
		log.Fatal(err)
	}

	zones = append(zones, zone)
	return zones, udnssdk.ResultInfo{}, nil, nil
}

type mockUltraDNSRecord struct {
	client *udnssdk.Client
}

func (m *mockUltraDNSRecord) Create(k udnssdk.RRSetKey, rrset udnssdk.RRSet) (*http.Response, error) {
	return nil, nil
}

func (m *mockUltraDNSRecord) Select(k udnssdk.RRSetKey) ([]udnssdk.RRSet, error) {
	return []udnssdk.RRSet{{
		OwnerName: "test-ultradns-provider.com.",
		RRType:    endpoint.RecordTypeA,
		RData:     []string{"1.1.1.1"},
		TTL:       86400,
	}}, nil

}

func (m *mockUltraDNSRecord) SelectWithOffset(k udnssdk.RRSetKey, offset int) ([]udnssdk.RRSet, udnssdk.ResultInfo, *http.Response, error) {
	return nil, udnssdk.ResultInfo{}, nil, nil
}

func (m *mockUltraDNSRecord) Update(udnssdk.RRSetKey, udnssdk.RRSet) (*http.Response, error) {
	return nil, nil
}

func (m *mockUltraDNSRecord) Delete(k udnssdk.RRSetKey) (*http.Response, error) {
	return nil, nil
}

func (m *mockUltraDNSRecord) SelectWithOffsetWithLimit(k udnssdk.RRSetKey, offset int, limit int) (rrsets []udnssdk.RRSet, ResultInfo udnssdk.ResultInfo, resp *http.Response, err error) {
	return []udnssdk.RRSet{{
		OwnerName: "test-ultradns-provider.com.",
		RRType:    endpoint.RecordTypeA,
		RData:     []string{"1.1.1.1"},
		TTL:       86400,
	}}, udnssdk.ResultInfo{}, nil, nil
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
			Zone: &mocked,
		},
	}

	zoneKey := &udnssdk.ZoneKey{
		Zone:        "",
		AccountName: "teamrest",
	}

	expected, _, _, err := provider.client.Zone.SelectWithOffsetWithLimit(zoneKey, 0, 1000)
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
	mocked := mockUltraDNSRecord{}
	mockedDomain := mockUltraDNSZone{}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
			RRSets: &mocked,
			Zone:   &mockedDomain,
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
			RRSets: &mocked,
			Zone:   &mockedDomain,
		},
	}

	changes.Create = []*endpoint.Endpoint{
		{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.1.1"}, RecordType: "A"},
		{DNSName: "ttl.test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.1.1"}, RecordType: "A", RecordTTL: 100},
	}
	changes.Create = []*endpoint.Endpoint{{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.1.2"}, RecordType: "A"}}
	changes.UpdateNew = []*endpoint.Endpoint{{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.2.2"}, RecordType: "A", RecordTTL: 100}}
	changes.UpdateNew = []*endpoint.Endpoint{{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.2.2", "1.1.2.3", "1.1.2.4"}, RecordType: "A", RecordTTL: 100}}
	changes.Delete = []*endpoint.Endpoint{{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.2.2", "1.1.2.3", "1.1.2.4"}, RecordType: "A", RecordTTL: 100}}
	changes.Delete = []*endpoint.Endpoint{{DNSName: "ttl.test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.1.1"}, RecordType: "A", RecordTTL: 100}}
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
			RRSets: &mocked,
			Zone:   &mockedDomain,
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

//Fail case scenario testing where CNAME and TXT Record name are same
func TestUltraDNSProvider_ApplyChangesCNAME(t *testing.T) {
	changes := &plan.Changes{}
	mocked := mockUltraDNSRecord{nil}
	mockedDomain := mockUltraDNSZone{nil}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
			RRSets: &mocked,
			Zone:   &mockedDomain,
		},
	}

	changes.Create = []*endpoint.Endpoint{
		{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.1.1"}, RecordType: "CNAME"},
		{DNSName: "test-ultradns-provider.com", Targets: endpoint.Targets{"1.1.1.1"}, RecordType: "TXT"},
	}

	err := provider.ApplyChanges(context.Background(), changes)
	if err == nil {
		t.Errorf("expected to fail")
	}

}

// This will work if you would set the environment variables such as "ULTRADNS_INTEGRATION" and zone should be avaialble "kubernetes-ultradns-provider-test.com"
func TestUltraDNSProvider_ApplyChanges_Integration(t *testing.T) {

	_, ok := os.LookupEnv("ULTRADNS_INTEGRATION")
	if !ok {
		log.Printf("Skipping test")

	} else {

		provider, err := NewUltraDNSProvider(endpoint.NewDomainFilter([]string{"kubernetes-ultradns-provider-test.com"}), false)
		changes := &plan.Changes{}
		changes.Create = []*endpoint.Endpoint{
			{DNSName: "kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"1.1.1.1"}, RecordType: "A"},
			{DNSName: "ttl.kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"2001:0db8:85a3:0000:0000:8a2e:0370:7334"}, RecordType: "AAAA", RecordTTL: 100},
		}

		err = provider.ApplyChanges(context.Background(), changes)
		if err != nil {
			t.Errorf("should not fail, %s", err)
		}

		changes = &plan.Changes{}
		changes.UpdateNew = []*endpoint.Endpoint{
			{DNSName: "kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"1.1.2.2"}, RecordType: "A", RecordTTL: 100},
			{DNSName: "ttl.kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"2001:0db8:85a3:0000:0000:8a2e:0370:7335"}, RecordType: "AAAA", RecordTTL: 100}}
		err = provider.ApplyChanges(context.Background(), changes)
		if err != nil {
			t.Errorf("should not fail, %s", err)
		}

		changes = &plan.Changes{}
		changes.Delete = []*endpoint.Endpoint{
			{DNSName: "ttl.kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"2001:0db8:85a3:0000:0000:8a2e:0370:7335"}, RecordType: "AAAA", RecordTTL: 100},
			{DNSName: "kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"1.1.2.2"}, RecordType: "A", RecordTTL: 100}}

		err = provider.ApplyChanges(context.Background(), changes)
		if err != nil {
			t.Errorf("should not fail, %s", err)
		}

	}

}

// This will work if you would set the environment variables such as "ULTRADNS_INTEGRATION" and zone should be avaialble "kubernetes-ultradns-provider-test.com" for multiple target
func TestUltraDNSProvider_ApplyChanges_MultipleTarget_integeration(t *testing.T) {
	_, ok := os.LookupEnv("ULTRADNS_INTEGRATION")
	if !ok {
		log.Printf("Skipping test")

	} else {

		provider, err := NewUltraDNSProvider(endpoint.NewDomainFilter([]string{"kubernetes-ultradns-provider-test.com"}), false)
		changes := &plan.Changes{}
		changes.Create = []*endpoint.Endpoint{
			{DNSName: "kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"1.1.1.1", "1.1.2.2"}, RecordType: "A"}}

		err = provider.ApplyChanges(context.Background(), changes)
		if err != nil {
			t.Errorf("should not fail, %s", err)
		}

		changes = &plan.Changes{}
		changes.UpdateNew = []*endpoint.Endpoint{{DNSName: "kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"1.1.2.2", "192.168.0.24"}, RecordType: "A", RecordTTL: 100}}

		err = provider.ApplyChanges(context.Background(), changes)
		if err != nil {
			t.Errorf("should not fail, %s", err)
		}
		changes = &plan.Changes{}
		changes.Delete = []*endpoint.Endpoint{{DNSName: "kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"1.1.2.2", "192.168.0.24"}, RecordType: "A"}}

		err = provider.ApplyChanges(context.Background(), changes)
		if err != nil {
			t.Errorf("should not fail, %s", err)
		}
	}
}

func TestUltraDNSProvider_newSBPoolObjectCreation(t *testing.T) {
	mocked := mockUltraDNSRecord{nil}
	mockedDomain := mockUltraDNSZone{nil}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
			RRSets: &mocked,
			Zone:   &mockedDomain,
		},
	}
	changes = &plan.Changes{}
	changes.UpdateNew = []*endpoint.Endpoint{{DNSName: "kubernetes-ultradns-provider-test.com.", Targets: endpoint.Targets{"1.1.2.2", "192.168.0.24"}, RecordType: "A", RecordTTL: 100}}
	changesList := provider.newUltraDNSChanges("UPDATE", changes)
	for _, _ = range changesList[0].ResourceRecordSetUltraDNS.RData {

		rrdataInfo := udnssdk.SBRDataInfo{
			RunProbes: true,
			Priority:  1,
			State:     "NORMAL",
			Threshold: 1,
			Weight:    nil,
		}
		sbpoolRDataList = append(sbpoolRDataList, rrdataInfo)
	}
	sbPoolObject := udnssdk.SBPoolProfile{
		Context:     udnssdk.SBPoolSchema,
		Order:       "ROUND_ROBIN",
		Description: "kubernetes-ultradns-provider-test.com.",
		MaxActive:   2,
		MaxServed:   2,
		RDataInfo:   sbpoolRDataList,
		RunProbes:   true,
		ActOnProbes: true,
	}

	actualSBPoolObject, _ := newSBPoolObjectCreation(context.Background(), changesList[0])
	assert.Equal(t, sbPoolObject, actualSBPoolObject)

}

//Testcase to check fail scenario for multiple AAAA targets
func TestUltraDNSProvider_MultipleTargetAAAA(t *testing.T) {
	mocked := mockUltraDNSRecord{nil}
	mockedDomain := mockUltraDNSZone{nil}

	provider := &UltraDNSProvider{
		client: udnssdk.Client{
			RRSets: &mocked,
			Zone:   &mockedDomain,
		},
	}

	changes := &plan.Changes{}
	changes.Create = []*endpoint.Endpoint{
		{DNSName: "ttl.kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:0db8:85a3:0000:0000:8a2e:0370:7335"}, RecordType: "AAAA", RecordTTL: 100},
	}
	err = provider.ApplyChanges(context.Background(), changes)
	if err == nil {
		t.Errorf("We wanted it to fail since multiple AAAA targets are not allowed")
	}
}

func TestUltraDNSProvider_MultipleTargetCNAME(t *testing.T) {
	_, ok := os.LookupEnv("ULTRADNS_INTEGRATION")
	if !ok {
		log.Printf("Skipping test")

	} else {
		provider, err := NewUltraDNSProvider(endpoint.NewDomainFilter([]string{"kubernetes-ultradns-provider-test.com"}), false)
		changes := &plan.Changes{}

		changes.Create = []*endpoint.Endpoint{
			{DNSName: "ttl.kubernetes-ultradns-provider-test.com", Targets: endpoint.Targets{"nginx.loadbalancer.com.","nginx1.loadbalancer.com."}, RecordType: "CNAME", RecordTTL: 100},
		}
		err = provider.ApplyChanges(context.Background(), changes)
		if err == nil {
			t.Errorf("We wanted it to fail since multiple CNAME targets are not allowed")
		}
	}
	if err == nil {
		t.Errorf("We wanted it to fail since multiple CNAME targets are not allowed")
	}
}