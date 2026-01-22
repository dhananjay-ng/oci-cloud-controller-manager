/*
Copyright 2022.

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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"reflect"
	"testing"

	"github.com/oracle/oci-cloud-controller-manager/api/v1beta1"
	"github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
	"github.com/oracle/oci-cloud-controller-manager/pkg/util"
	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	errors2 "github.com/pkg/errors"
	"go.uber.org/zap"

	npnv1beta1 "github.com/oracle/oci-cloud-controller-manager/api/v1beta1"
)

func TestComputeAveragesByReturnCode(t *testing.T) {
	testCases := []struct {
		name     string
		metrics  []ErrorMetric
		expected map[string]float64
	}{
		{
			name:     "base case",
			metrics:  nil,
			expected: map[string]float64{},
		},
		{
			name:     "base case 2",
			metrics:  endToEndLatencySlice{}.ErrorMetric(),
			expected: map[string]float64{},
		},
		{
			name: "base case e2e time",
			metrics: endToEndLatencySlice{
				endToEndLatency{timeTaken: 5.0},
				endToEndLatency{timeTaken: 10.0},
			}.ErrorMetric(),
			expected: map[string]float64{
				util.Success: 7.5,
			},
		},
		{
			name: "base case vnic attachment time",
			metrics: VnicAttachmentResponseSlice{
				VnicAttachmentResponse{timeTaken: 8.5},
				VnicAttachmentResponse{timeTaken: 6.5},
			}.ErrorMetric(),
			expected: map[string]float64{
				util.Success: 7.5,
			},
		},
		{
			name: "base case ip application time",
			metrics: IPAllocationSlice{
				IPAllocation{timeTaken: 5.0},
			}.ErrorMetric(),
			expected: map[string]float64{
				util.Success: 5.0,
			},
		},
		{
			name: "ip application failures",
			metrics: IPAllocationSlice{
				IPAllocation{timeTaken: 1.0, err: errors.New("http status code: 500")},
				IPAllocation{timeTaken: 2.0, err: errors.New("http status code: 500")},
				IPAllocation{timeTaken: 3.0, err: errors.New("http status code: 429")},
				IPAllocation{timeTaken: 4.0, err: errors.New("http status code: 401")},
				IPAllocation{timeTaken: 5.0},
				IPAllocation{timeTaken: 6.0},
			}.ErrorMetric(),
			expected: map[string]float64{
				util.Err5XX:  1.5,
				util.Err429:  3.0,
				util.Err4XX:  4.0,
				util.Success: 5.5,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			averages := computeAveragesByReturnCode(tc.metrics)
			if !reflect.DeepEqual(averages, tc.expected) {
				t.Errorf("expected metrics:\n%+v\nbut got:\n%+v", tc.expected, averages)
			}
		})
	}
}

var (
	trueVal          = true
	falseVal         = false
	testAddress1     = "1.1.1.1"
	testAddress2     = "2.2.2.2"
	testIPv6Address1 = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	testIPv6Address2 = "2001:0db8:85a3:0000:0000:8a2e:0370:1fde"
)

func TestFilterPrimaryIp(t *testing.T) {
	testCases := []struct {
		name     string
		ips      *vnicSecondaryAddresses
		expected *vnicSecondaryAddresses
	}{
		{
			name: "base case",
			ips:  &vnicSecondaryAddresses{},
			expected: &vnicSecondaryAddresses{
				V6: []core.Ipv6{},
				V4: []core.PrivateIp{},
			},
		},
		{
			name: "filter primary ip",
			ips: &vnicSecondaryAddresses{
				V4: []core.PrivateIp{
					{
						IsPrimary: &trueVal,
						IpAddress: &testAddress1,
					},
				},
			},
			expected: &vnicSecondaryAddresses{
				V6: []core.Ipv6{},
				V4: []core.PrivateIp{},
			},
		},
		{
			name: "primary and secondary ip",
			ips: &vnicSecondaryAddresses{
				V4: []core.PrivateIp{
					{IsPrimary: &trueVal, IpAddress: &testAddress1},
					{IsPrimary: &falseVal, IpAddress: &testAddress2},
				},
				V6: []core.Ipv6{
					{IpAddress: &testIPv6Address1},
					{IpAddress: &testIPv6Address2},
				},
			},
			expected: &vnicSecondaryAddresses{
				V4: []core.PrivateIp{
					{IsPrimary: &falseVal, IpAddress: &testAddress2},
				},
				V6: []core.Ipv6{
					{IpAddress: &testIPv6Address1},
					{IpAddress: &testIPv6Address2},
				},
			},
		},
		{
			name: "only secondary ipv6",
			ips: &vnicSecondaryAddresses{
				V6: []core.Ipv6{
					{IpAddress: &testIPv6Address2},
				},
			},
			expected: &vnicSecondaryAddresses{
				V4: []core.PrivateIp{},
				V6: []core.Ipv6{
					{IpAddress: &testIPv6Address2},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := filterPrimaryIp(tc.ips)
			if !reflect.DeepEqual(filtered, tc.expected) {
				t.Errorf("expected ips:\n%+v\nbut got:\n%+v", tc.expected, filtered)
			}
		})
	}
}

func TestGetHostIpAddress(t *testing.T) {
	testCases := []struct {
		name       string
		ipFamilies []string
		npn        v1beta1.NativePodNetwork
		vnics      map[string]*vnicSecondaryAddresses
		expected   map[string]*vnicSecondaryAddresses
	}{
		{
			name:       "single vnic",
			ipFamilies: []string{IPv4, IPv6},
			npn:        v1beta1.NativePodNetwork{},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
				},
			},
			expected: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6:       []core.Ipv6{{IpAddress: &testIPv6Address2}},
					V4:       []core.PrivateIp{{IpAddress: &testAddress2}},
					hostIpv4: &testAddress1,
					hostIpv6: &testIPv6Address1,
				},
			},
		},
		{
			name:       "multiple vnic",
			ipFamilies: []string{IPv4, IPv6},
			npn:        v1beta1.NativePodNetwork{},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
				},
				"vnic-2": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
				},
			},
			expected: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1},
					},
					hostIpv4: &testAddress1,
					hostIpv6: &testIPv6Address1,
				},
				"vnic-2": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress2},
					},
					hostIpv4: &testAddress1,
					hostIpv6: &testIPv6Address1,
				},
			},
		},
		{
			name:       "vnic with ip cidr address",
			ipFamilies: []string{IPv4, IPv6},
			npn: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								DisplayName: "foobar",
							},
						},
					},
				},
			},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2, CidrPrefixLength: common.Int(124)},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1, CidrPrefixLength: common.Int(32)},
						{IpAddress: &testAddress2, CidrPrefixLength: common.Int(30)},
					},
				},
				"vnic-2": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1, CidrPrefixLength: common.Int(120)},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1, CidrPrefixLength: common.Int(32)},
						{IpAddress: &testAddress2, CidrPrefixLength: common.Int(30)},
					},
				},
			},
			expected: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address2, CidrPrefixLength: common.Int(124)},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress2, CidrPrefixLength: common.Int(30)},
					},
					hostIpv4: &testAddress1,
					hostIpv6: &testIPv6Address1,
				},
				"vnic-2": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1, CidrPrefixLength: common.Int(120)},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress2, CidrPrefixLength: common.Int(30)},
					},
					hostIpv4: &testAddress1,
					hostIpv6: &testIPv6Address2,
				},
			},
		},
	}
	fmt.Println("hey bar..")
	//npn := npnv1beta1.NativePodNetwork{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := assignHostIpAddressForVnic(tc.vnics, tc.ipFamilies, &tc.npn)
			if !reflect.DeepEqual(filtered, tc.expected) {
				t.Errorf("expected ips:\n%+v\nbut got:\n%+v", tc.expected, filtered)
			}
		})
	}
}

func TestValidateMaxPodCountWithSecondaryIPCount(t *testing.T) {
	testCases := []struct {
		name        string
		ipFamilies  []string
		maxPodCount int
		vnics       map[string]*vnicSecondaryAddresses
		expectedErr error
	}{
		{
			name:       "single vnic",
			ipFamilies: []string{IPv4, IPv6},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
				},
			},
			maxPodCount: 2,
			expectedErr: nil,
		},
		{
			name:       "multiple vnic",
			ipFamilies: []string{IPv4, IPv6},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
				},
				"vnic-2": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
				},
			},
			maxPodCount: 34,
			expectedErr: nil,
		},
		{
			name:       "single vnic IPv4",
			ipFamilies: []string{IPv4},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
				},
			},
			maxPodCount: 3,
			expectedErr: errors2.Errorf("Allocated IPv4 count != maxPodCount (3 != 2)"),
		},
		{
			name:       "single vnic IPv6",
			ipFamilies: []string{IPv6},
			vnics: map[string]*vnicSecondaryAddresses{
				"vnic": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
				},
			},
			maxPodCount: 3,
			expectedErr: errors2.Errorf("Allocated IPv6 count != maxPodCount (3 != 2)"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMaxPodCountWithSecondaryIPCount(tc.vnics, tc.maxPodCount, tc.ipFamilies)

			if err != nil {
				if !reflect.DeepEqual(err.Error(), tc.expectedErr.Error()) {
					t.Errorf("expected err:\n%+v\nbut got:\n%+v", tc.expectedErr, err)
				}
			}
		})
	}
}

func TestTotalAllocatedSecondaryIpsForInstance(t *testing.T) {
	testCases := []struct {
		name     string
		ips      map[string]*vnicSecondaryAddresses
		expected IpAddressCountByVersion
	}{
		{
			name:     "base case",
			ips:      map[string]*vnicSecondaryAddresses{},
			expected: IpAddressCountByVersion{V4: 0, V6: 0},
		},
		{
			name: "one vnic, two ips",
			ips: map[string]*vnicSecondaryAddresses{
				"one": {V4: []core.PrivateIp{
					{IpAddress: &testAddress1},
					{IpAddress: &testAddress2},
				}},
			},
			expected: IpAddressCountByVersion{V4: 2, V6: 0},
		},
		{
			name: "two vnic, 1/2 ips ",
			ips: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					}},
				"two": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress2},
					}},
			},
			expected: IpAddressCountByVersion{V4: 3, V6: 0},
		},
		{
			name: "three vnic, 1/2 IPv4  1/2 IPv6 ips ",
			ips: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
				},
				"two": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress2},
					},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
					},
				},
			},
			expected: IpAddressCountByVersion{V4: 3, V6: 3},
		},
		{
			name: "three vnic 1/2 IPv6 ips ",
			ips: map[string]*vnicSecondaryAddresses{
				"one": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
				},
				"two": {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
					},
				},
			},
			expected: IpAddressCountByVersion{V4: 0, V6: 3},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allocated := totalAllocatedSecondaryIpsForInstance(tc.ips)
			if !reflect.DeepEqual(allocated, tc.expected) {
				t.Errorf("expected ip count:\n%+v\nbut got:\n%+v", tc.expected, allocated)
			}
		})
	}
}

func TestGetAdditionalSecondaryIPsNeededPerVNIC(t *testing.T) {
	testCases := []struct {
		name                  string
		existingIpsByVnic     map[string]*vnicSecondaryAddresses
		ipFamilies            []string
		allocatedSecondaryIps IpAddressCountByVersion
		npnCR                 v1beta1.NativePodNetwork
		gvaNics               []GvaNics
		expected              []VnicIPAllocations
		err                   error
	}{
		{
			name:              "base case",
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{},
			ipFamilies:        []string{IPv4},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(0),
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected:              []VnicIPAllocations{},
			err:                   nil,
		},
		{
			name:       "one vnic with one additional IP required",
			ipFamilies: []string{IPv4},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(2),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
					}},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 1, V6: 0},
			expected: []VnicIPAllocations{
				{
					"one",
					[]string{IPv4},
					IpAddressCountByVersion{V4: 2, V6: 0},
				},
			},
			err: nil,
		},
		{
			name:       "one vnic with space for required IPs",
			ipFamilies: []string{IPv4},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(31),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 18, V6: 0},
			expected: []VnicIPAllocations{
				{
					"one",
					[]string{IPv4},
					IpAddressCountByVersion{V4: 14, V6: 0},
				},
			},
			err: nil,
		},
		{
			name:       "one vnic without space for required IPs",
			ipFamilies: []string{IPv4},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(31),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1},
					},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 19, V6: 0},
			expected:              nil,
			err:                   errors.New("failed to allocate required number of IPs with existing VNICs"),
		},
		{
			name:       "one vnic for required IPv4 and IPv6",
			ipFamilies: []string{IPv4, IPv6},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(31),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
					},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 18, V6: 18},
			expected: []VnicIPAllocations{
				{
					"one",
					[]string{IPv4, IPv6},
					IpAddressCountByVersion{V4: 14, V6: 14},
				},
			},
			err: nil,
		},
		{
			name:       "two vnic for required IPv4 and IPv6",
			ipFamilies: []string{IPv4, IPv6},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(62),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
					},
				},
				"two": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
					},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 48, V6: 48},
			expected: []VnicIPAllocations{
				{
					"one",
					[]string{IPv4},
					IpAddressCountByVersion{V4: 2, V6: 2},
				},
				{
					"two",
					[]string{IPv4, IPv6},
					IpAddressCountByVersion{V4: 14, V6: 14},
				},
			},
			err: nil,
		},
		{
			name:       "two vnic for required IPv4 and IPv6",
			ipFamilies: []string{IPv4, IPv6},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(34),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
				"two": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{
					"one",
					[]string{IPv4, IPv6},
					IpAddressCountByVersion{V4: 32, V6: 32},
				},
				{
					"two",
					[]string{IPv4, IPv6},
					IpAddressCountByVersion{V4: 4, V6: 4},
				},
			},
			err: nil,
		},
		{
			name:       "one vnic",
			ipFamilies: []string{IPv4},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(31),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected:              []VnicIPAllocations{{"one", []string{IPv4}, IpAddressCountByVersion{V4: 32, V6: 0}}},
			err:                   nil,
		},
		{
			name:       "two vnic for required IPv4 and IPv6",
			ipFamilies: []string{IPv4, IPv6},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(64),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
				"two": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
				"three": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"one", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 32, V6: 32}},
				{"two", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 32, V6: 32}},
				{"three", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 3, V6: 3}},
			},
			err: nil,
		},
		{
			name:       "single vnic for required IPv4 and IPv6",
			ipFamilies: []string{IPv4, IPv6},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
				"two": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"one", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 32, V6: 32}},
				{"two", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 2, V6: 2}},
			},
			err: nil,
		},
		{
			name:       "Secondary vnics present on spec",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("green"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"green"},
							IpCount:              31,
							IpFamilies:           []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								DisplayName: "",
								FreeformTags: map[string]string{
									"applicationResource": "green",
									"ip-count":            "31",
								},
							},
						},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"green": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1}, {IpAddress: &testIPv6Address1},
					},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"green", []string{IPv4, IPv6}, IpAddressCountByVersion{
					V4: 14,
					V6: 14,
				}},
			},
			err: nil,
		},
		{
			name:       "Secondary vnic present with IPv4 and IPv6",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("app-a"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-a"},
							IpCount:              31,
							IpFamilies:           []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-a"},
								IpCount:              31,
							},
						},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"app-a": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"app-a", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 32, V6: 32}},
			},
			err: nil,
		},
		{
			name:       "Secondary vnic with 14 ip count",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("app-a"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-a"},
							IpCount:              14,
							IpFamilies:           []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(31),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-a"},
								IpCount:              14,
							},
						},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"app-a": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"app-a", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 15, V6: 15}},
			},
			err: nil,
		},
		{
			name:       "2 Secondary vnic with IPv4 and IPv6 and different ip count",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("app-a"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-a"},
							IpCount:              20,
							IpFamilies:           []string{IPv4, IPv6},
						},
					},
				},
				{
					vnicId: common.String("app-b"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-b"},
							IpCount:              15,
							IpFamilies:           []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-a"},
								IpCount:              20,
							},
						},
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-b"},
								IpCount:              15,
							},
						},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"app-a": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
				"app-b": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"app-a", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 21, V6: 21}},
				{"app-b", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 16, V6: 16}},
			},
			err: nil,
		},
		{
			name:       "3 Secondary vnic with IPv4 partially allocated ips",
			ipFamilies: []string{IPv4},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("app-a"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-a"},
							IpCount:              20,
							IpFamilies:           []string{IPv4},
						},
					},
				},
				{
					vnicId: common.String("app-b"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-b"},
							IpCount:              15,
							IpFamilies:           []string{IPv4},
						},
					},
				},
				{
					vnicId: common.String("app-c"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							ApplicationResources: []string{"app-c"},
							IpCount:              9,
							IpFamilies:           []string{IPv4},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-a"},
								IpCount:              20,
							},
						},
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-b"},
								IpCount:              15,
							},
						},
						{
							CreateVnicDetails: v1beta1.CreateVnicDetails{
								ApplicationResources: []string{"app-c"},
								IpCount:              9,
							},
						},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"app-a": {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
						{IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1}, {IpAddress: &testAddress1},
					},
					V6: []core.Ipv6{},
				},
				"app-b": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
				"app-c": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"app-a", []string{IPv4}, IpAddressCountByVersion{V4: 1, V6: 0}},
				{"app-b", []string{IPv4}, IpAddressCountByVersion{V4: 16, V6: 0}},
				{"app-c", []string{IPv4}, IpAddressCountByVersion{V4: 10, V6: 0}},
			},
			err: nil,
		},
		{
			name:       "GVA with CIDR IPv4 and IPv6 expansion",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("app-cidr"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							IpCount:    16,
							IpFamilies: []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{CreateVnicDetails: v1beta1.CreateVnicDetails{IpCount: 16}},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"app-cidr": {
					V4: []core.PrivateIp{
						{IpAddress: common.String("10.0.0.0"), CidrPrefixLength: common.Int(30)},
					},
					V6: []core.Ipv6{
						{IpAddress: common.String("2001:db8::"), CidrPrefixLength: common.Int(124)},
					},
				},
			},
			expected: []VnicIPAllocations{
				{"app-cidr", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 13, V6: 1}},
			},
			err: nil,
		},
		{
			name:       "GVA with IP CIDR Address and IP Address",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("app-cidr"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							IpCount:    16,
							IpFamilies: []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(32),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{CreateVnicDetails: v1beta1.CreateVnicDetails{IpCount: 16}},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"app-cidr": {
					V4: []core.PrivateIp{
						{IpAddress: common.String("10.0.0.0"), CidrPrefixLength: common.Int(30)},
						{IpAddress: common.String("10.0.0.0"), CidrPrefixLength: common.Int(32)},
					},
					V6: []core.Ipv6{
						{IpAddress: common.String("2001:db8::"), CidrPrefixLength: common.Int(124)},
						{IpAddress: common.String("2001:db8::")},
					},
				},
			},
			expected: []VnicIPAllocations{
				{"app-cidr", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 12, V6: 0}},
			},
			err: nil,
		},
		{
			name:       "GVA CIDR IPv6 trimmed to ipCount",
			ipFamilies: []string{IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("cidr6"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							IpCount:    2,
							IpFamilies: []string{IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(2),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{CreateVnicDetails: v1beta1.CreateVnicDetails{IpCount: 2}},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"cidr6": {
					V6: []core.Ipv6{
						{IpAddress: common.String("2001:db8::"), CidrPrefixLength: common.Int(124)},
					},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"cidr6", []string{IPv6}, IpAddressCountByVersion{V4: 0, V6: 1}},
			},
			err: nil,
		},
		{
			name:       "GVA fresh node",
			ipFamilies: []string{IPv4, IPv6},
			gvaNics: []GvaNics{
				{
					vnicId: common.String("fresh-node"),
					SecondaryVnicSpec: &v1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							IpCount:    32,
							IpFamilies: []string{IPv4, IPv6},
						},
					},
				},
			},
			npnCR: v1beta1.NativePodNetwork{
				Spec: v1beta1.NativePodNetworkSpec{
					MaxPodCount: common.Int(2),
					SecondaryVnics: []v1beta1.SecondaryVnic{
						{CreateVnicDetails: v1beta1.CreateVnicDetails{IpCount: 32}},
					},
				},
			},
			existingIpsByVnic: map[string]*vnicSecondaryAddresses{
				"fresh-node": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			allocatedSecondaryIps: IpAddressCountByVersion{V4: 0, V6: 0},
			expected: []VnicIPAllocations{
				{"fresh-node", []string{IPv4, IPv6}, IpAddressCountByVersion{V4: 33, V6: 33}},
			},
			err: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allocation, err := getAdditionalSecondaryIPsNeededPerVNIC(tc.existingIpsByVnic, &tc.npnCR, tc.allocatedSecondaryIps, tc.ipFamilies, tc.gvaNics, logr.FromContextOrDiscard(context.Background()))
			if (err == nil && tc.err != nil) || err != nil && tc.err == nil {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", tc.err, err)
				t.FailNow()
			}
			if err != nil && err.Error() != tc.err.Error() {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", tc.expected, allocation)
			}

			gotSumV4, gotSumV6 := 0, 0
			expectedSumV4, expectedSumV6 := 0, 0
			for _, v := range allocation {
				gotSumV4 += v.ips.V4
				gotSumV6 += v.ips.V6
			}
			for _, v := range tc.expected {
				expectedSumV4 += v.ips.V4
				expectedSumV6 += v.ips.V6
			}

			t.Logf("expected ip allocation:\n%+v\n got ip allocation:\n%+v", tc.expected, allocation)
			if gotSumV4 != expectedSumV4 || gotSumV6 != expectedSumV6 {
				t.Errorf("expected ip allocation:\n%+v\nbut got:\n%+v", tc.expected, allocation)
				t.FailNow()
			}
		})
	}
}

func TestGetIpFamilies(t *testing.T) {
	ipVersion4 := "IPv4"
	ipVersion6 := "IPv6"
	testCases := []struct {
		name     string
		npn      v1beta1.NativePodNetwork
		ipFamily []string
		ctx      context.Context
		expected []string
	}{
		{
			name:     "nil case",
			npn:      v1beta1.NativePodNetwork{Spec: v1beta1.NativePodNetworkSpec{IPFamilies: nil}},
			expected: []string{},
		},
		{
			name:     "Dual stack IPv4 preferred",
			npn:      v1beta1.NativePodNetwork{Spec: v1beta1.NativePodNetworkSpec{IPFamilies: []*string{&ipVersion4, &ipVersion6}}},
			expected: []string{IPv4, IPv6},
		},
		{
			name:     "Dual stack IPv6 preferred",
			npn:      v1beta1.NativePodNetwork{Spec: v1beta1.NativePodNetworkSpec{IPFamilies: []*string{&ipVersion6, &ipVersion4}}},
			expected: []string{IPv6, IPv4},
		},
		{
			name:     "Single stack IPv4",
			npn:      v1beta1.NativePodNetwork{Spec: v1beta1.NativePodNetworkSpec{IPFamilies: []*string{&ipVersion4}}},
			expected: []string{IPv4},
		},
		{
			name:     "Single stack IPv6",
			npn:      v1beta1.NativePodNetwork{Spec: v1beta1.NativePodNetworkSpec{IPFamilies: []*string{&ipVersion6}}},
			expected: []string{IPv6},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ipFamilies, _ := getIpFamilies(tc.ctx, tc.npn)
			if !reflect.DeepEqual(ipFamilies, tc.expected) {
				t.Errorf("expected ips:\n%+v\nbut got:\n%+v", tc.expected, ipFamilies)
			}
		})
	}
}

var (
	one             = "one"
	mac1            = "11.bb.cc.dd.ee.66"
	routerIP1       = "192.168.1.1"
	ipv6routerIP    = "2001:db8:1234:1a00::"
	cidr1           = "10.0.0.0/64"
	ipv6cidr        = "2001:0db8:/32"
	testIPaddressv4 = "10.0.0.1"
	testIPaddressv6 = "2603:c020:1a:9905:2375:cb62:d7a0:5240"
	ipCIdrAddressV4 = core.PrivateIp{IpAddress: &testIPaddressv4, CidrPrefixLength: common.Int(31)}
	ipCIdrAddressV6 = core.Ipv6{IpAddress: &testIPaddressv6, CidrPrefixLength: common.Int(124)}
	hostAddressIpv4 = "1.0.0.0"
	hostAddressIpv6 = "2001:db8:1234:1a01::"
	subnetVnic1     = SubnetVnic{
		Vnic:   &core.Vnic{Id: &one, MacAddress: &mac1},
		Subnet: &core.Subnet{VirtualRouterIp: &routerIP1, CidrBlock: &cidr1, Ipv6CidrBlock: &ipv6cidr, Ipv6VirtualRouterIp: &ipv6routerIP},
	}
	subnetVnic2 = SubnetVnic{
		Vnic:   &core.Vnic{Id: &one, MacAddress: &mac1},
		Subnet: &core.Subnet{VirtualRouterIp: &routerIP1, CidrBlock: &cidr1, Ipv6CidrBlock: nil, Ipv6CidrBlocks: []string{ipv6cidr}, Ipv6VirtualRouterIp: &ipv6routerIP},
	}
	npnVnic1 = v1beta1.VNICAddress{
		VNICID:      &one,
		MACAddress:  &mac1,
		RouterIP:    &routerIP1,
		HostAddress: &hostAddressIpv4,
		Addresses:   []*string{&testAddress1, &testAddress2},
		SubnetCidr:  &cidr1,
	}
	singleStackIPv4 = v1beta1.VNICAddress{
		VNICID:      &one,
		MACAddress:  &mac1,
		RouterIP:    &routerIP1,
		Addresses:   []*string{&testAddress1, &testAddress2},
		HostAddress: &hostAddressIpv4,
		HostAddresses: []v1beta1.HostAddress{
			{V4: &hostAddressIpv4},
		},
		PodAddresses: []v1beta1.PodAddress{
			{V4: &testAddress1},
			{V4: &testAddress2},
		},
		RouterIPs: []v1beta1.RouterIP{
			{V4: &routerIP1},
		},
		SubnetCidr:  &cidr1,
		SubnetCidrs: []v1beta1.SubnetCidr{{V4: &cidr1}},
	}
	singleStackIPv6 = v1beta1.VNICAddress{
		VNICID:     &one,
		MACAddress: &mac1,
		HostAddresses: []v1beta1.HostAddress{
			{
				V4: nil,
				V6: &hostAddressIpv6,
			},
		},
		PodAddresses: []v1beta1.PodAddress{
			{V6: &testIPv6Address1},
			{V6: &testIPv6Address2},
		},
		RouterIPs: []v1beta1.RouterIP{{
			V6: &ipv6routerIP,
			V4: nil,
		}},
		SubnetCidrs: []v1beta1.SubnetCidr{{
			V4: nil,
			V6: &ipv6cidr,
		}},
		RouterIP:    nil,
		SubnetCidr:  nil,
		HostAddress: nil,
	}
	dualStack = v1beta1.VNICAddress{
		VNICID:      &one,
		MACAddress:  &mac1,
		RouterIP:    &routerIP1,
		Addresses:   []*string{&testAddress1},
		HostAddress: &hostAddressIpv4,
		HostAddresses: []v1beta1.HostAddress{
			{
				V4: &hostAddressIpv4,
				V6: &hostAddressIpv6,
			},
		},
		PodAddresses: []v1beta1.PodAddress{{
			V4: &testAddress1,
			V6: &testIPv6Address1,
		}},
		RouterIPs: []v1beta1.RouterIP{{
			V4: &routerIP1,
			V6: &ipv6routerIP,
		}},
		SubnetCidr: &cidr1,
		SubnetCidrs: []v1beta1.SubnetCidr{{
			V4: &cidr1,
			V6: &ipv6cidr,
		}},
	}
	gvaNics = v1beta1.VNICAddress{
		VNICID:     &one,
		MACAddress: &mac1,
		RouterIP:   &routerIP1,
		Addresses: []*string{
			common.String("10.0.0.0"),
			common.String("10.0.0.1"),
		},
		HostAddress: &hostAddressIpv4,
		HostAddresses: []v1beta1.HostAddress{
			{
				V4: &hostAddressIpv4,
				V6: &hostAddressIpv6,
			},
		},
		NicConfiguration: &v1beta1.NicConfiguration{
			IpCount:    common.Int(2),
			IpFamilies: []string{IPv4, IPv6},
		},
		PodAddresses: []v1beta1.PodAddress{
			{
				V4: common.String("10.0.0.0"),
				V6: common.String("2603:c020:1a:9905:2375:cb62:d7a0:5240"),
			},
			{
				V4: common.String("10.0.0.1"),
				V6: common.String("2603:c020:1a:9905:2375:cb62:d7a0:5241"),
			},
		},
		RouterIPs: []v1beta1.RouterIP{{
			V4: &routerIP1,
			V6: &ipv6routerIP,
		}},
		SubnetCidr: &cidr1,
		SubnetCidrs: []v1beta1.SubnetCidr{{
			V4: &cidr1,
			V6: &ipv6cidr,
		}},
	}
)

func TestConvertCoreVNICtoNPNStatus(t *testing.T) {
	testCases := []struct {
		name                       string
		existingSecondaryVNICs     []SubnetVnic
		existingSecondaryIpsByVNIC map[string]*vnicSecondaryAddresses
		ipFamilies                 []string
		expected                   []v1beta1.VNICAddress
		gvaNics                    []GvaNics
	}{
		{
			name:                   "base case",
			existingSecondaryVNICs: []SubnetVnic{},
			ipFamilies:             []string{},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				"vnic1": {
					V4: []core.PrivateIp{},
					V6: []core.Ipv6{},
				},
			},
			expected: []v1beta1.VNICAddress{},
		},
		{
			name:                   "backward compatibility",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic1},
			ipFamilies:             []string{},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				one: {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
					hostIpv4: &hostAddressIpv4,
				},
			},
			expected: []v1beta1.VNICAddress{npnVnic1},
		},
		{
			name:                   "Dual stack",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic1},
			ipFamilies:             []string{IPv4, IPv6},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				one: {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
					},
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
					},
					hostIpv4: &hostAddressIpv4,
					hostIpv6: &hostAddressIpv6,
				},
			},
			expected: []v1beta1.VNICAddress{dualStack},
		},
		{
			name:                   "Single stack IPv4",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic1},
			ipFamilies:             []string{IPv4},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				one: {
					V4: []core.PrivateIp{
						{IpAddress: &testAddress1},
						{IpAddress: &testAddress2},
					},
					hostIpv4: &hostAddressIpv4,
				},
			},
			expected: []v1beta1.VNICAddress{singleStackIPv4},
		},
		{
			name:                   "Single stack IPv6",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic1},
			ipFamilies:             []string{IPv6},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				one: {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					hostIpv6: &hostAddressIpv6,
				},
			},
			expected: []v1beta1.VNICAddress{singleStackIPv6},
		},
		{
			name:                   "Single stack IPv6 ULA prefix CIDR",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic2},
			ipFamilies:             []string{IPv6},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				one: {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					hostIpv6: &hostAddressIpv6,
				},
			},
			expected: []v1beta1.VNICAddress{singleStackIPv6},
		},
		{
			name:                   "Single stack IPv6 ULA prefix CIDR",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic2},
			ipFamilies:             []string{IPv6},
			gvaNics:                []GvaNics{},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				one: {
					V6: []core.Ipv6{
						{IpAddress: &testIPv6Address1},
						{IpAddress: &testIPv6Address2},
					},
					hostIpv6: &hostAddressIpv6,
				},
			},
			expected: []v1beta1.VNICAddress{singleStackIPv6},
		},
		{
			name:                   "ip cidr address",
			existingSecondaryVNICs: []SubnetVnic{subnetVnic1},
			ipFamilies:             []string{IPv4, IPv6},
			existingSecondaryIpsByVNIC: map[string]*vnicSecondaryAddresses{
				"one": {
					V4: []core.PrivateIp{
						ipCIdrAddressV4,
					},
					V6: []core.Ipv6{
						ipCIdrAddressV6,
					},
					hostIpv4: &hostAddressIpv4,
					hostIpv6: &hostAddressIpv6,
				},
			},
			expected: []v1beta1.VNICAddress{gvaNics},
			gvaNics: []GvaNics{
				{
					vnicId: &one,
					SecondaryVnicSpec: &npnv1beta1.SecondaryVnic{
						CreateVnicDetails: v1beta1.CreateVnicDetails{
							IpCount:      2,
							AssignIpv6Ip: true,
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vnics := convertCoreVNICtoNPNStatus(tc.existingSecondaryVNICs, tc.existingSecondaryIpsByVNIC, tc.ipFamilies, tc.gvaNics)
			if !reflect.DeepEqual(vnics, tc.expected) {
				t.Errorf("expected npnVNIC to be:\n%+v\nbut got:\n%+v", tc.expected, vnics)
			}
		})
	}
}

type MockOCIClient struct {
}

func (c MockOCIClient) Lustre() client.LustreInterface {
	return nil
}

func (c MockOCIClient) LoadBalancer(*zap.SugaredLogger, string, *client.OCIClientConfig) client.GenericLoadBalancerInterface {
	return nil
}

func (c MockOCIClient) BlockStorage() client.BlockStorageInterface {
	return nil
}

func (c MockOCIClient) FSS(ociClientConfig *client.OCIClientConfig) client.FileStorageInterface {
	return nil
}

func (c MockOCIClient) Identity(ociClientConfig *client.OCIClientConfig) client.IdentityInterface {
	return nil
}

func (c MockOCIClient) ContainerEngine() client.ContainerEngineInterface {
	return nil
}

func (MockOCIClient) CertManager() client.CertificateManagerInterface {
	return MockCertificateManagerClient{}
}

type MockCertificateManagerClient struct{}

func (m MockCertificateManagerClient) GetValidCertificate(ctx context.Context, id string) (*certificatesmanagement.Certificate, error) {
	//TODO implement me
	return nil, nil
}

func (MockOCIClient) NewWorkloadIdentityClient(logger *zap.SugaredLogger, lbType string, ociClientConfig *client.OCIClientConfig) client.Interface {
	return MockOCIClient{}
}

// MockVirtualNetworkClient mocks VirtualNetwork client implementation
type MockVirtualNetworkClient struct {
}

func (c *MockVirtualNetworkClient) GetIpv6(ctx context.Context, id string) (*core.Ipv6, error) {
	return &core.Ipv6{}, nil
}

func (c *MockVirtualNetworkClient) CreateNetworkSecurityGroup(ctx context.Context, compartmentId, vcnId, displayName, lbId string) (*core.NetworkSecurityGroup, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) UpdateNetworkSecurityGroup(ctx context.Context, id, etag string, freeformTags map[string]string) (*core.NetworkSecurityGroup, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) GetNetworkSecurityGroup(ctx context.Context, id string) (*core.NetworkSecurityGroup, *string, error) {
	return nil, nil, nil
}

func (c *MockVirtualNetworkClient) ListNetworkSecurityGroups(ctx context.Context, displayName, compartmentId, vcnId string) ([]core.NetworkSecurityGroup, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) DeleteNetworkSecurityGroup(ctx context.Context, id, etag string) (*string, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) AddNetworkSecurityGroupSecurityRules(ctx context.Context, id string, details core.AddNetworkSecurityGroupSecurityRulesDetails) (*core.AddNetworkSecurityGroupSecurityRulesResponse, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) RemoveNetworkSecurityGroupSecurityRules(ctx context.Context, id string, details core.RemoveNetworkSecurityGroupSecurityRulesDetails) (*core.RemoveNetworkSecurityGroupSecurityRulesResponse, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) ListNetworkSecurityGroupSecurityRules(ctx context.Context, id string, direction core.ListNetworkSecurityGroupSecurityRulesDirectionEnum) ([]core.SecurityRule, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) UpdateNetworkSecurityGroupSecurityRules(ctx context.Context, id string, details core.UpdateNetworkSecurityGroupSecurityRulesDetails) (*core.UpdateNetworkSecurityGroupSecurityRulesResponse, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) GetSubnet(ctx context.Context, id string) (*core.Subnet, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) GetSubnetFromCacheByIP(ip client.IpAddresses) (*core.Subnet, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) IsRegionalSubnet(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func (c *MockVirtualNetworkClient) GetVcn(ctx context.Context, id string) (*core.Vcn, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) GetSecurityList(ctx context.Context, id string) (core.GetSecurityListResponse, error) {
	return core.GetSecurityListResponse{}, nil
}

func (c *MockVirtualNetworkClient) UpdateSecurityList(ctx context.Context, id string, etag string, ingressRules []core.IngressSecurityRule, egressRules []core.EgressSecurityRule) (core.UpdateSecurityListResponse, error) {
	return core.UpdateSecurityListResponse{}, nil
}

func (c *MockVirtualNetworkClient) ListPrivateIps(ctx context.Context, vnicId string) ([]core.PrivateIp, error) {
	if &vnicId == nil {
		return nil, errors.New("vnic id is nil")
	}
	if vnicId == "err" {
		return nil, errors.New("failed to list ipv4")
	}
	return privateIps[vnicId], nil
}

func (c *MockVirtualNetworkClient) GetPrivateIp(ctx context.Context, id string) (*core.PrivateIp, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) CreatePrivateIp(ctx context.Context, vnicID string) (*core.PrivateIp, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) GetPublicIpByIpAddress(ctx context.Context, id string) (*core.PublicIp, error) {
	return nil, nil
}

func (c *MockVirtualNetworkClient) ListIpv6s(ctx context.Context, vnicId string) ([]core.Ipv6, error) {
	if &vnicId == nil {
		return nil, errors.New("vnic id is nil")
	}
	if vnicId == "err" {
		return nil, errors.New("failed to list ipv6")
	}
	return ipv6s[vnicId], nil
}

func (c *MockVirtualNetworkClient) CreateIpv6(ctx context.Context, vnicID string) (*core.Ipv6, error) {
	return nil, nil
}

// MockComputeClient mocks Compute client implementation
type MockComputeClient struct{}

func (c *MockComputeClient) GetPrimaryVNICFromCacheByInstance(instanceID string) *core.Vnic {
	return nil
}

func (c *MockComputeClient) GetInstance(ctx context.Context, id string) (*core.Instance, error) {
	return nil, nil
}

func (c *MockComputeClient) GetInstanceByNodeName(ctx context.Context, compartmentID, vcnID, nodeName string) (*core.Instance, error) {
	return nil, nil
}

func (c *MockComputeClient) GetPrimaryVNICForInstance(ctx context.Context, compartmentID, instanceID string) (*core.Vnic, error) {
	return nil, nil
}

func (c *MockComputeClient) AttachVnic(ctx context.Context, opts client.AttachVnicOptions) (response core.VnicAttachment, err error) {
	return core.VnicAttachment{}, nil
}

func (c *MockComputeClient) FindVolumeAttachment(ctx context.Context, compartmentID, volumeID string, instanceID *string) (core.VolumeAttachment, error) {
	return nil, nil
}

func (c *MockComputeClient) AttachVolume(ctx context.Context, instanceID, volumeID string, isSharable bool) (core.VolumeAttachment, error) {
	return nil, nil
}

func (c *MockComputeClient) AttachParavirtualizedVolume(ctx context.Context, instanceID, volumeID string, isPvEncryptionInTransitEnabled bool, isSharable bool) (core.VolumeAttachment, error) {
	return nil, nil
}

func (c *MockComputeClient) WaitForVolumeAttached(ctx context.Context, attachmentID string) (core.VolumeAttachment, error) {
	return nil, nil
}

func (c *MockComputeClient) DetachVolume(ctx context.Context, id string) error {
	return nil
}

func (c *MockComputeClient) WaitForVolumeDetached(ctx context.Context, attachmentID string) error {
	return nil
}

func (c *MockComputeClient) ListVolumeAttachments(ctx context.Context, compartmentID, volumeID string) ([]core.VolumeAttachment, error) {
	return nil, nil
}

func (c *MockComputeClient) WaitForUHPVolumeLoggedOut(ctx context.Context, attachmentID string) error {
	return nil
}

func (MockOCIClient) Compute() client.ComputeInterface {
	return &MockComputeClient{}
}

func (MockOCIClient) Networking(ociClientConfig *client.OCIClientConfig) client.NetworkingInterface {
	return &MockVirtualNetworkClient{}
}

func (c *MockVirtualNetworkClient) GetVNIC(ctx context.Context, id string) (*core.Vnic, error) {
	vnicCounter++
	if vnics[id].LifecycleState == core.VnicLifecycleStateProvisioning && vnicCounter%3 == 0 {
		copy := vnics[id]
		copy.LifecycleState = core.VnicLifecycleStateAvailable
		return copy, nil // Available
	}
	return vnics[id], nil
}

func (c *MockComputeClient) ListVnicAttachments(ctx context.Context, compartmentID, instanceID string) ([]core.VnicAttachment, error) {
	return attachedVnicsList[compartmentID], nil
}

func (c *MockComputeClient) GetVnicAttachment(ctx context.Context, vnicAttachmentId *string) (response *core.VnicAttachment, err error) {
	attachmentCounter++
	resp := vnicAttachments[*vnicAttachmentId]
	if *resp.Id == "attachmentid5" {
		resp.LifecycleState = core.VnicAttachmentLifecycleStateDetached // Detached
	}
	if attachmentCounter%3 == 0 {
		resp.LifecycleState = core.VnicAttachmentLifecycleStateAttached // Attached
	}
	return &resp, nil
}

var (
	vnicAttachments = map[string]core.VnicAttachment{
		"attachmentid1": {
			Id:             common.String("attachmentid1"),
			VnicId:         common.String("vnic1"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid2": {
			Id:             common.String("attachmentid2"),
			VnicId:         common.String("vnic2"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid3": {
			Id:             common.String("attachmentid3"),
			VnicId:         common.String("vnic3"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid4": {
			Id:             common.String("attachmentid4"),
			VnicId:         common.String("vnic4"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid5": {
			Id:             common.String("attachmentid5"),
			VnicId:         common.String("vnic5"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid6": {
			Id:             common.String("attachmentid6"),
			VnicId:         common.String("vnic6"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid7": {
			Id:             common.String("attachmentid7"),
			VnicId:         common.String("vnic7"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttached,
		},
		"attachmentid8": {
			Id:             common.String("attachmentid8"),
			VnicId:         common.String("vnic8"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttaching,
		},
		"attachmentid9": {
			Id:             common.String("attachmentid9"),
			VnicId:         common.String("vnic9"),
			LifecycleState: core.VnicAttachmentLifecycleStateDetached,
		},
		"attachmentid10": {
			Id:             common.String("attachmentid10"),
			VnicId:         common.String("vnic10"),
			LifecycleState: core.VnicAttachmentLifecycleStateDetached,
		},
		"attachmentid11": {
			Id:             common.String("attachmentid11"),
			VnicId:         common.String("vnic11"),
			LifecycleState: core.VnicAttachmentLifecycleStateDetached,
		},
		"attachmentid12": {
			Id:             common.String("attachmentid12"),
			VnicId:         common.String("vnic12"),
			LifecycleState: core.VnicAttachmentLifecycleStateAttaching,
		},
	}
	False  = false
	Subnet = "test-subnet"
	vnics  = map[string]*core.Vnic{
		"vnic1": {
			Id:             common.String("vnic1"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic2": {
			Id:             common.String("vnic2"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic3": {
			Id:             common.String("vnic3"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic4": {
			Id:             common.String("vnic4"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic5": {
			Id:             common.String("vnic5"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic6": {
			Id:             common.String("vnic6"),
			LifecycleState: core.VnicLifecycleStateProvisioning,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic7": {
			Id:             common.String("vnic7"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic8": {
			Id:             common.String("vnic8"),
			LifecycleState: core.VnicLifecycleStateAvailable,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic9": {
			Id:             common.String("vnic9"),
			LifecycleState: core.VnicLifecycleStateProvisioning,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic10": {
			Id:             common.String("vnic10"),
			LifecycleState: core.VnicLifecycleStateTerminating,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic11": {
			Id:             common.String("vnic11"),
			LifecycleState: core.VnicLifecycleStateTerminated,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
		"vnic12": {
			Id:             nil,
			LifecycleState: core.VnicLifecycleStateTerminated,
			IsPrimary:      &False,
			SubnetId:       &Subnet,
		},
	}

	privateIps = map[string][]core.PrivateIp{
		"vnic1": {
			{
				IsPrimary: &falseVal,
				IpAddress: &testAddress1,
			},
			{
				IsPrimary: &falseVal,
				IpAddress: &testAddress1,
			},
			{
				IsPrimary: &falseVal,
				IpAddress: &testAddress1,
			},
			{
				IsPrimary: &falseVal,
				IpAddress: &testAddress1,
			},
			{
				IsPrimary: &falseVal,
				IpAddress: &testAddress1,
			},
		},
	}

	ipv6s = map[string][]core.Ipv6{
		"vnic1": {
			{IpAddress: &testIPv6Address1},
			{IpAddress: &testIPv6Address2},
			{IpAddress: &testIPv6Address1},
			{IpAddress: &testIPv6Address2},
			{IpAddress: &testIPv6Address1},
		},
	}

	attachedVnicsList = map[string][]core.VnicAttachment{
		"vnics attached": {
			{
				Id:             common.String("attachmentid1"),
				VnicId:         common.String("vnic1"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
			{
				Id:             common.String("attachmentid2"),
				VnicId:         common.String("vnic2"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
			{
				Id:             common.String("attachmentid3"),
				VnicId:         common.String("vnic3"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
			{
				Id:             common.String("attachmentid4"),
				VnicId:         common.String("vnic4"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
		},
		"single vnic not attached": {
			{
				Id:             common.String("attachmentid6"),
				VnicId:         common.String("vnic6"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
			{
				Id:             common.String("attachmentid7"),
				VnicId:         common.String("vnic7"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
			{
				Id:             common.String("attachmentid8"),
				VnicId:         common.String("vnic8"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttaching,
			},
		},
		"vnic in detaching or detached after a while": {
			{
				Id:             common.String("attachmentid1"),
				VnicId:         common.String("vnic1"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
			{
				Id:             common.String("attachmentid5"),
				VnicId:         common.String("vnic5"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttaching,
			},
		},
		"vnic not available": {
			{
				Id:             common.String("attachmentid11"),
				VnicId:         common.String("vnic11"),
				LifecycleState: core.VnicAttachmentLifecycleStateDetached,
			},
			{
				Id:             common.String("attachmentid12"),
				VnicId:         common.String("vnic12"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttaching,
			},
		},
		"vnic becomes available eventually": {
			{
				Id:             common.String("attachmentid6"),
				VnicId:         common.String("vnic6"),
				LifecycleState: core.VnicAttachmentLifecycleStateAttached,
			},
		},
	}
	attachmentCounter = 1
	vnicCounter       = 1
)

func TestValidateVnicAttachmentsAreInAttachedState(t *testing.T) {
	testCases := []struct {
		name              string
		in                string
		compartmentid     string
		output            bool
		requiredVnicCount int
		err               error
		counter           int
	}{
		{
			name:              "all vnics attached",
			in:                "instanceid",
			compartmentid:     "vnics attached",
			output:            true,
			requiredVnicCount: 4,
			err:               nil,
		},
		{
			name:              "one vnic stuck in attaching",
			in:                "instanceid",
			compartmentid:     "single vnic not attached",
			output:            true,
			requiredVnicCount: 3,
			err:               nil,
		},
		{
			name:              "vnics in other lifecycle states",
			in:                "instanceid",
			compartmentid:     "vnic in detaching or detached after a while",
			output:            false,
			requiredVnicCount: 2,
			err:               errors.New("vnic attachment is in detaching/detached state"),
		},
		{
			name:              "not enough vnic attached",
			in:                "instanceid",
			compartmentid:     "vnic not available",
			output:            false,
			requiredVnicCount: 2,
			err:               errNotEnoughVnicsAttached,
		},
		{
			name:              "vnic becomes available eventually",
			in:                "instanceid",
			compartmentid:     "vnic becomes available eventually",
			output:            true,
			requiredVnicCount: 1,
			err:               nil,
		},
	}

	npn := &NativePodNetworkReconciler{
		OCIClient: MockOCIClient{},
	}

	t.Parallel()
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, existingSecondaryIpsbyVNIC, _ := npn.getPrimaryAndSecondaryVNICs(context.Background(), tt.compartmentid, tt.in)
			result, err := npn.validateVnicAttachmentsAreInAttachedState(context.Background(), tt.in, tt.requiredVnicCount, existingSecondaryIpsbyVNIC)
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("validateVnicAttachmentsAreInAttachedState(%s) got error %s, expected %s", tt.in, err, tt.err)
			}
			if !reflect.DeepEqual(result, tt.output) {
				t.Errorf("validateVnicAttachmentsAreInAttachedState(%s) => %t, want %t", tt.in, result, tt.output)
			}
		})
	}
}

func TestGetSecondaryIpsByVNICs(t *testing.T) {
	testCases := []struct {
		name                  string
		nodeIpFamilies        []string
		existingSecondaryVnic []SubnetVnic
		output                map[string]*vnicSecondaryAddresses
		err                   error
	}{
		{
			name:           "List call IPv4 and IPv6",
			nodeIpFamilies: []string{IPv4, IPv6},
			existingSecondaryVnic: []SubnetVnic{{Vnic: &core.Vnic{
				Id: common.String("vnic1"),
			}}},
			output: map[string]*vnicSecondaryAddresses{
				"vnic1": {
					V6:         ipv6s["vnic1"],
					V4:         privateIps["vnic1"],
					hostIpv6:   nil,
					hostIpv4:   nil,
					ipFamilies: []string{IPv4, IPv6},
				},
			},
			err: nil,
		},
		{
			name:           "List call IPv4 only",
			nodeIpFamilies: []string{IPv4},
			existingSecondaryVnic: []SubnetVnic{{Vnic: &core.Vnic{
				Id: common.String("vnic1"),
			}}},
			output: map[string]*vnicSecondaryAddresses{
				"vnic1": {
					V6:         []core.Ipv6{},
					V4:         privateIps["vnic1"],
					hostIpv6:   nil,
					hostIpv4:   nil,
					ipFamilies: []string{IPv4},
				},
			},
			err: nil,
		},
		{
			name:           "List call IPv6 only",
			nodeIpFamilies: []string{IPv6},
			existingSecondaryVnic: []SubnetVnic{{Vnic: &core.Vnic{
				Id: common.String("vnic1"),
			}}},
			output: map[string]*vnicSecondaryAddresses{
				"vnic1": {
					V6:         ipv6s["vnic1"],
					V4:         []core.PrivateIp{},
					hostIpv6:   nil,
					hostIpv4:   nil,
					ipFamilies: []string{IPv6},
				},
			},
			err: nil,
		},
	}

	npn := &NativePodNetworkReconciler{
		OCIClient: MockOCIClient{},
	}

	t.Parallel()
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ipsByVNICs, err := npn.getSecondaryIpsByVNICs(context.Background(), tt.existingSecondaryVnic, tt.nodeIpFamilies)
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("got error %s, expected %s", err, tt.err)
			}
			if !reflect.DeepEqual(ipsByVNICs, tt.output) {
				t.Errorf("getSecondaryIpsByVNICs=> %+v, want %+v", ipsByVNICs, tt.output)
			}
		})
	}
}

func TestExpandIpAddressesInRange(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		prefix  int
		want    []string
		wantErr bool
	}{
		{
			name:   "ipv4 /32 single",
			ip:     "10.0.0.1",
			prefix: 32,
			want:   []string{"10.0.0.1"},
		},
		{
			name:   "ipv4 /30 range",
			ip:     "10.0.0.1",
			prefix: 30,
			want:   []string{"10.0.0.0", "10.0.0.1", "10.0.0.2", "10.0.0.3"},
		},
		{
			name:    "ipv4 invalid prefix",
			ip:      "10.0.0.1",
			prefix:  33,
			wantErr: true,
		},
		{
			name:   "ipv6 /128 single",
			ip:     "2001:db8::1",
			prefix: 128,
			want:   []string{"2001:db8::1"},
		},
		{
			name:   "ipv6 /126 range",
			ip:     "2001:db8::1",
			prefix: 126,
			want:   []string{"2001:db8::", "2001:db8::1", "2001:db8::2", "2001:db8::3"},
		},
		{
			name:    "invalid ip format",
			ip:      "not-an-ip",
			prefix:  24,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandIpAddressesInRange(tt.ip, tt.prefix)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("want %#v, got %#v", tt.want, got)
			}
		})
	}
}

// New tests for getBlockSizeForCidrPrefix
func TestGetBlockSizeForCidrPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix int
		family string
		want   int
	}{
		// IPv4 cases
		{name: "IPv4 /32 => 1", prefix: 32, family: IPv4, want: 1},
		{name: "IPv4 /30 => 4", prefix: 30, family: IPv4, want: 4},
		{name: "IPv4 /24 => 256", prefix: 24, family: IPv4, want: 256},
		{name: "IPv4 /16 => 65536", prefix: 16, family: IPv4, want: 65536},
		{name: "IPv4 /2 => 1073741824", prefix: 2, family: IPv4, want: 1073741824},
		{name: "IPv4 invalid /33 => 0", prefix: 33, family: IPv4, want: 0},

		// IPv6 cases
		{name: "IPv6 /128 => 1", prefix: 128, family: IPv6, want: 1},
		{name: "IPv6 /127 => 2", prefix: 127, family: IPv6, want: 2},
		{name: "IPv6 /124 => 16", prefix: 124, family: IPv6, want: 16},
		{name: "IPv6 /120 => 256", prefix: 120, family: IPv6, want: 256},
		{name: "IPv6 invalid /129 => 0", prefix: 129, family: IPv6, want: 0},

		// Unknown family should default to IPv4 semantics
		{name: "Unknown family defaults to IPv4 semantics", prefix: 24, family: "unknown", want: 256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBlockSizeForCidrPrefix(tt.prefix, tt.family)
			if got != tt.want {
				t.Fatalf("getBlockSizeForCidrPrefix(%d, %q) = %d, want %d", tt.prefix, tt.family, got, tt.want)
			}
		})
	}
}
