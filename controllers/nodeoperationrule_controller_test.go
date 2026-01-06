/*
Copyright 2024.

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
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	norv1beta1 "github.com/oracle/oci-cloud-controller-manager/api/node-cycling/v1beta1"
	providercfg "github.com/oracle/oci-cloud-controller-manager/pkg/cloudprovider/providers/oci/config"
	"github.com/oracle/oci-cloud-controller-manager/pkg/oci/client"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
)

func TestUpdateResultToList(t *testing.T) {
	testCases := []struct {
		name          string
		updatedResult norv1beta1.NodeOperationResult
		currentList   []norv1beta1.NodeOperationResult
		expected      []norv1beta1.NodeOperationResult
	}{
		{
			name:     "corner case: updateResult is nil and currentList is nil",
			expected: make([]norv1beta1.NodeOperationResult, 0),
		},
		{
			name:          "corner case: updateResult is empty and currentList is empty",
			updatedResult: norv1beta1.NodeOperationResult{},
			currentList:   make([]norv1beta1.NodeOperationResult, 0),
			expected:      make([]norv1beta1.NodeOperationResult, 0),
		},
		{
			name:          "updateResult is not nil and currentList is nil",
			updatedResult: norv1beta1.NodeOperationResult{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
			expected:      []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"}},
		},
		{
			name:        "updateResult is nil and currentList is not nil",
			currentList: []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"}},
			expected:    []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"}},
		},
		{
			name:          "updateResult is contained in the currentList, and currentList is updated",
			updatedResult: norv1beta1.NodeOperationResult{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest1"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
			expected: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
		},
		{
			name:          "updateResult is not contained in the currentList, it is added to currentList",
			updatedResult: norv1beta1.NodeOperationResult{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.0", WorkRequestId: "workrequest1"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
			expected: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.0", WorkRequestId: "workrequest1"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
			},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedList := updateResultToList(testCase.updatedResult, testCase.currentList)
			if !reflect.DeepEqual(testCase.expected, updatedList) {
				t.Errorf("expected: %+v, but actual updateResultToList => %+v", testCase.expected, updatedList)
			} else {
				t.Logf("expected: %+v, and actual updateResultToList => %+v", testCase.expected, updatedList)
			}
		})
	}
}

func TestAddToListIfNotExist(t *testing.T) {
	testCases := []struct {
		name         string
		newNames     []string
		currentNames []string
		expected     []string
	}{
		{
			name:     "corner case: newNames does not exist and currentNames does not exist",
			expected: []string{},
		},
		{
			name:         "corner case: newNames is nil and currentNames is nil",
			newNames:     nil,
			currentNames: nil,
			expected:     []string{},
		},
		{
			name:         "corner case: newNames is empty and currentNames is empty",
			newNames:     []string{},
			currentNames: []string{},
			expected:     []string{},
		},
		{
			name:         "corner case: newNames is nil and currentNames is non-nil",
			newNames:     nil,
			currentNames: []string{"1.1.0.1"},
			expected:     []string{"1.1.0.1"},
		},
		{
			name:         "corner case: newNames is empty and currentNames is non-nil",
			newNames:     []string{},
			currentNames: []string{"1.1.0.1"},
			expected:     []string{"1.1.0.1"},
		},
		{
			name:         "corner case: newNames is non-nil and currentNames is nil",
			newNames:     []string{"1.1.0.1"},
			currentNames: nil,
			expected:     []string{"1.1.0.1"},
		},
		{
			name:         "corner case: newNames is non-nil and currentNames is nil",
			newNames:     []string{"1.1.0.1"},
			currentNames: []string{},
			expected:     []string{"1.1.0.1"},
		},
		{
			name:         "newNames does not exist in currentNames",
			newNames:     []string{"1.1.0.1"},
			currentNames: []string{"1.1.0.2", "1.1.0.3"},
			expected:     []string{"1.1.0.2", "1.1.0.3", "1.1.0.1"},
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedList := addToListIfNotExist(testCase.newNames, testCase.currentNames)
			if !reflect.DeepEqual(testCase.expected, updatedList) {
				t.Errorf("expected: %+v, but actual addToListIfNotExist => %+v", testCase.expected, updatedList)
			} else {
				t.Logf("expected: %+v, and actual addToListIfNotExist => %+v", testCase.expected, updatedList)
			}
		})
	}
}

func TestRemoveResultFromList(t *testing.T) {
	testCases := []struct {
		name        string
		nodeName    string
		currentList []norv1beta1.NodeOperationResult
		expected    []norv1beta1.NodeOperationResult
	}{
		{
			name:     "corner case: nodeName is nil and currentList is nil",
			expected: []norv1beta1.NodeOperationResult{},
		},
		{
			name:        "corner case: nodeName is empty and currentList is empty",
			nodeName:    "",
			currentList: []norv1beta1.NodeOperationResult{},
			expected:    []norv1beta1.NodeOperationResult{},
		},
		{
			name: "corner case: result is empty and currentList is non-empty",
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
			expected: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
		},
		{
			name:     "corner case: result is non-empty and currentList is empty",
			nodeName: "1.1.0.1",
			expected: []norv1beta1.NodeOperationResult{},
		},
		{
			name:     "result is contained in partial currentList",
			nodeName: "1.1.0.1",
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
			expected: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
		},
		{
			name:     "result is contained in partial currentList",
			nodeName: "1.1.0.2",
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest1"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
			expected: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest1"},
			},
		},
		{
			name:     "result is contained in all currentList",
			nodeName: "1.1.0.1",
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.1", WorkRequestId: "workrequest2"},
			},
			expected: []norv1beta1.NodeOperationResult{},
		},
		{
			name:     "result is not contained in currentList",
			nodeName: "1.1.0.1",
			currentList: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.0", WorkRequestId: "workrequest2"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
			expected: []norv1beta1.NodeOperationResult{
				{NodeName: "1.1.0.0", WorkRequestId: "workrequest2"},
				{NodeName: "1.1.0.2", WorkRequestId: "workrequest3"},
			},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedList := removeResultFromList(testCase.nodeName, testCase.currentList)
			if !reflect.DeepEqual(testCase.expected, updatedList) {
				t.Errorf("expected: %+v, but actual removeResultFromList => %+v", testCase.expected, updatedList)
			} else {
				t.Logf("expected: %+v, and actual removeResultFromList => %+v", testCase.expected, updatedList)
			}
		})
	}
}

func TestRemoveNodeFromList(t *testing.T) {
	testCases := []struct {
		name     string
		nodeName string
		nodes    []string
		expected []string
	}{
		{
			name:     "corner case: nodeName is nil and nodes is nil",
			expected: []string{},
		},
		{
			name:     "corner case: nodeName is nil and nodes is non-nil",
			nodes:    []string{"1.1.0.0"},
			expected: []string{"1.1.0.0"},
		},
		{
			name:     "corner case: nodeName is non-nil and nodes is nil",
			nodeName: "1.1.0.0",
			expected: []string{},
		},
		{
			name:     "corner case: nodeName is empty and nodes is non-empty",
			nodeName: "",
			nodes:    []string{"1.1.0.0"},
			expected: []string{"1.1.0.0"},
		},
		{
			name:     "corner case: nodes is non empty and nodes is non-empty",
			nodeName: "1.1.0.0",
			nodes:    []string{},
			expected: []string{},
		},
		{
			name:     "nodes is contained in nodes and result will be empty",
			nodeName: "1.1.0.0",
			nodes:    []string{"1.1.0.0"},
			expected: []string{},
		},
		{
			name:     "nodes is contained in nodes",
			nodeName: "1.1.0.0",
			nodes:    []string{"1.1.0.1", "1.1.0.0", "1.1.0.2"},
			expected: []string{"1.1.0.1", "1.1.0.2"},
		},
		{
			name:     "nodes is not contained in nodes",
			nodeName: "1.1.0.0",
			nodes:    []string{"1.1.0.1"},
			expected: []string{"1.1.0.1"},
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedList := removeNodeFromList(testCase.nodeName, testCase.nodes)
			if !reflect.DeepEqual(testCase.expected, updatedList) {
				t.Errorf("expected: %+v, but actual removeNodeFromList => %+v", testCase.expected, updatedList)
			} else {
				t.Logf("expected: %+v, and actual removeNodeFromList => %+v", testCase.expected, updatedList)
			}
		})
	}
}

func TestMergeFilterLabels(t *testing.T) {
	testCases := []struct {
		name         string
		triggerLabel map[string]string
		customLabels map[string]string
		expected     labels.Selector
		errMsg       string
	}{
		{
			name:     "corner case: both labels are nil",
			expected: labels.Everything(),
		},
		{
			name:         "happy case: matchCustomLabels is empty",
			triggerLabel: map[string]string{okeReservedLabelKey: "true"},
			expected:     labels.SelectorFromSet(map[string]string{okeReservedLabelKey: "true"}),
		},
		{
			name:         "happy case: matchCustomLabels is non-empty",
			triggerLabel: map[string]string{okeReservedLabelKey: "true"},
			customLabels: map[string]string{"mydomain.com/deployment": "green", "ad": "1"},
			expected:     labels.SelectorFromSet(map[string]string{okeReservedLabelKey: "true", "mydomain.com/deployment": "green", "ad": "1"}),
		},
		{
			name:         "unhappy case: invalid value in triggerLabel - invalid character",
			triggerLabel: map[string]string{okeReservedLabelKey: "**aa"},
			customLabels: map[string]string{"deployment": "green", "ad": "1"},
			errMsg:       "values[0][oke.oraclecloud.com/node_operation]: Invalid value: \"**aa\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
		},
		{
			name:         "unhappy case: invalid value in triggerLabel - too long length",
			triggerLabel: map[string]string{okeReservedLabelKey: "iamlongerthan63charskjhfsadhfjdshfaldshfajdsahfksjhfkjshjkfhasjdkfhsdjafhjdskjsdfhdsfhaskj"},
			customLabels: map[string]string{"deployment": "green", "ad": "1"},
			errMsg:       "values[0][oke.oraclecloud.com/node_operation]: Invalid value: \"iamlongerthan63charskjhfsadhfjdshfaldshfajdsahfksjhfkjshjkfhasjdkfhsdjafhjdskjsdfhdsfhaskj\": must be no more than 63 characters",
		},
		{
			name:         "unhappy case: invalid key in customLabels - key cannot start with slash",
			triggerLabel: map[string]string{okeReservedLabelKey: "true"},
			customLabels: map[string]string{"/mydomain.com": "green", "ad": "1"},
			errMsg:       "key: Invalid value: \"/mydomain.com\": prefix part must be non-empty",
		},
		{
			name:         "unhappy case: invalid key in customLabels - key has invalid character",
			triggerLabel: map[string]string{okeReservedLabelKey: "true"},
			customLabels: map[string]string{"mydomain.com/%a": "green", "ad": "1"},
			errMsg:       "key: Invalid value: \"mydomain.com/%a\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
		},
		{
			name:         "unhappy case: invalid key in customLabels - key length is too long",
			triggerLabel: map[string]string{okeReservedLabelKey: "true"},
			customLabels: map[string]string{"iamlongerthan63charskjhfsadhfjdshfaldshfajdsahfksjhfkjshjkfhasjdkfhsdjafhjdskjsdfhdsfhaskja": "green", "ad": "1"},
			errMsg:       "key: Invalid value: \"iamlongerthan63charskjhfsadhfjdshfaldshfajdsahfksjhfkjshjkfhasjdkfhsdjafhjdskjsdfhdsfhaskja\": name part must be no more than 63 characters",
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			labelSelector, err := mergeFilterLabels(testCase.triggerLabel, testCase.customLabels)
			if (err == nil && len(testCase.errMsg) != 0) || (err != nil && len(testCase.errMsg) == 0) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.errMsg, err)
				t.FailNow()
			}
			if err != nil && !reflect.DeepEqual(testCase.errMsg, err.Error()) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.errMsg, err.Error())
				t.FailNow()
			}
			if !reflect.DeepEqual(testCase.expected, labelSelector) {
				t.Errorf("expected: %+v, but actual mergeFilterLabels => %+v", testCase.expected, labelSelector)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual mergeFilterLabels => %+v", testCase.expected, labelSelector)
			}
		})
	}
}

func TestValidateNOR(t *testing.T) {
	testCases := []struct {
		name     string
		nor      norv1beta1.NodeOperationRule
		expected labels.Selector
		errMsg   string
	}{
		{
			name: "happy case: valid nor cr",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchTriggerLabel: map[string]string{okeReservedLabelKey: "true"},
						MatchCustomLabels: map[string]string{"deploy": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			expected: labels.SelectorFromSet(map[string]string{okeReservedLabelKey: "true", "deploy": "green"}),
		},
		{
			name: "unhappy case: missing OKE required label",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchCustomLabels: map[string]string{"deploy": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			errMsg: "MatchOKELabel in NodeSelector in the Spec of Node Operation Rule is invalid",
		},
		{
			name: "unhappy case: OKE required label does not contain required key",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchTriggerLabel: map[string]string{"incorrectKey": ""},
						MatchCustomLabels: map[string]string{"deploy": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			errMsg: "MatchOKELabel in NodeSelector in the Spec of NodeOperationRule does not contain required key",
		},
		{
			name: "unhappy case: OKE required label format is invalid",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchTriggerLabel: map[string]string{okeReservedLabelKey: "*"},
						MatchCustomLabels: map[string]string{"deploy": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			errMsg: "values[0][oke.oraclecloud.com/node_operation]: Invalid value: \"*\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')",
		},
		{
			name: "unhappy case: Custom label format is invalid",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchTriggerLabel: map[string]string{okeReservedLabelKey: "true"},
						MatchCustomLabels: map[string]string{"*": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			errMsg: "key: Invalid value: \"*\": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')",
		},
		{
			name: "unhappy case: actions contain more than 1 actions",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume", "reboot"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchTriggerLabel: map[string]string{okeReservedLabelKey: "true"},
						MatchCustomLabels: map[string]string{"deploy": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			errMsg: "actions in the Spec of NodeOperationRule is invalid",
		},
		{
			name: "unhappy case: actions contain invalid action",
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"bvr", "rebot"},
					NodeSelector: norv1beta1.NodeSelector{
						MatchTriggerLabel: map[string]string{okeReservedLabelKey: "true"},
						MatchCustomLabels: map[string]string{"deploy": "green"},
					},
					MaxParallelism: 2,
					NodeEvictionSettings: norv1beta1.NodeEvictionSettings{
						EvictionGracePeriod:             10,
						IsForceActionAfterGraceDuration: true,
					},
				},
			},
			errMsg: "actions in the Spec of NodeOperationRule is invalid",
		},
	}

	norReconciler := &NodeOperationRuleReconciler{
		Recorder: record.NewBroadcaster().NewRecorder(scheme.Scheme, v1.EventSource{Component: "nor-controller"}),
		Config: &providercfg.Config{
			Auth: providercfg.AuthConfig{
				TenancyID: "mockTenancyId",
			},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			labelSelector, err := norReconciler.validateNOR(context.Background(), testCase.nor)
			if (err == nil && len(testCase.errMsg) != 0) || (err != nil && len(testCase.errMsg) == 0) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.errMsg, err)
				t.FailNow()
			}
			if err != nil && !reflect.DeepEqual(testCase.errMsg, err.Error()) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.errMsg, err.Error())
				t.FailNow()
			}
			if !reflect.DeepEqual(testCase.expected, labelSelector) {
				t.Errorf("expected: %+v, but actual validateNOR => %+v", testCase.expected, labelSelector)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual validateNOR => %+v", testCase.expected, labelSelector)
			}
		})
	}

}

func TestCalculateParallelism(t *testing.T) {
	testCases := []struct {
		name                string
		maxParallelism      int
		inProgressNodesSize int
		nodeCandidates      []*v1.Node
		expected            int
	}{
		{
			name:     "nodeCandidates is nil",
			expected: 0,
		},
		{
			name:     "nodeCandidates is empty",
			expected: 0,
		},
		{
			name: "nodeCandidates size is greater than maxParallelism",
			nodeCandidates: []*v1.Node{
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
			},
			maxParallelism:      1,
			inProgressNodesSize: 0,
			expected:            1,
		},
		{
			name: "nodeCandidates size is less than maxParallelism",
			nodeCandidates: []*v1.Node{
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			maxParallelism:      3,
			inProgressNodesSize: 0,
			expected:            1,
		},
		{
			name: "nodeCandidates size is equal to maxParallelism",
			nodeCandidates: []*v1.Node{
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
			},
			maxParallelism:      2,
			inProgressNodesSize: 0,
			expected:            2,
		},
		{
			name: "nodeCandidates size is equal to maxParallelism",
			nodeCandidates: []*v1.Node{
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			maxParallelism:      1,
			inProgressNodesSize: 1,
			expected:            0,
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parallelism := calculateParallelism(testCase.maxParallelism, testCase.inProgressNodesSize, testCase.nodeCandidates)
			if !reflect.DeepEqual(testCase.expected, parallelism) {
				t.Errorf("expected: %+v, but actual calculateParallelism => %+v", testCase.expected, parallelism)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual calculateParallelism => %+v", testCase.expected, parallelism)
			}
		})
	}
}

func TestGetAndSortCandidates(t *testing.T) {
	testCases := []struct {
		name            string
		nodesWithLabel  []*v1.Node
		inProgressNodes []norv1beta1.NodeOperationResult
		pendingNodes    []string
		retryableNodes  []norv1beta1.NodeOperationResult
		expected        []*v1.Node
		expectedErr     error
	}{
		{
			name:     "corner case: all inputs are nil",
			expected: []*v1.Node{},
		},
		{
			name:     "corner case: all inputs are empty",
			expected: []*v1.Node{},
		},
		{
			name:            "corner case: nodeCandidates is nil, and there is 1 in progress node and 1 pending node and 1 retryable node",
			inProgressNodes: []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.1", WorkRequestId: "workrequest1"}},
			pendingNodes:    []string{"1.1.0.1"},
			retryableNodes:  []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.2", WorkRequestId: "workrequest2"}},
			expected:        make([]*v1.Node, 0),
			expectedErr:     errors.New("there are no nodes with labels retrieved from Kube API server but there are still some nodes need operation"),
		},
		{
			name:            "corner case: nodeCandidates is empty, and there is 1 in progress node and 1 pending node and 1 retryable node",
			nodesWithLabel:  make([]*v1.Node, 0),
			inProgressNodes: []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.1", WorkRequestId: "workrequest1"}},
			pendingNodes:    []string{"1.1.0.1"},
			retryableNodes:  []norv1beta1.NodeOperationResult{{NodeName: "1.1.0.2", WorkRequestId: "workrequest2"}},
			expected:        make([]*v1.Node, 0),
			expectedErr:     errors.New("there are no nodes with labels retrieved from Kube API server but there are still some nodes need operation"),
		},
		{
			name: "nodeCandidates is not empty, in progress nodes, pending nodes and retryable nodes are nil",
			nodesWithLabel: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			expected: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
		},
		{
			name: "nodeCandidates is not empty, in progress nodes, pending nodes and retryable nodes are empty",
			nodesWithLabel: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			inProgressNodes: make([]norv1beta1.NodeOperationResult, 0),
			pendingNodes:    make([]string, 0),
			retryableNodes:  make([]norv1beta1.NodeOperationResult, 0),
			expected: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
		},
		{
			name: "nodesWithLabel are 5 nodes: node1 ~ node5, node1 is in progress, node2 is pending, node3 is backoff. expected candidates are node4, node5, node2 and node3",
			nodesWithLabel: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode3",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode4",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId4",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode5",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId5",
					},
					Status: v1.NodeStatus{},
				},
			},
			inProgressNodes: []norv1beta1.NodeOperationResult{
				{
					NodeName:      "fakeNode1",
					WorkRequestId: "fakeWR1",
				},
			},
			pendingNodes: []string{"fakeNode2"},
			retryableNodes: []norv1beta1.NodeOperationResult{
				{
					NodeName:      "fakeNode3",
					WorkRequestId: "fakeWR3",
				},
			},
			expected: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode4",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId4",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode5",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId5",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode3",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidates, err := getAndSortCandidates(context.Background(), "nor-test", testCase.nodesWithLabel, testCase.inProgressNodes, testCase.pendingNodes, testCase.retryableNodes)
			if (err == nil && testCase.expectedErr != nil) || (err != nil && testCase.expectedErr == nil) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.expectedErr, err)
				t.FailNow()
			}
			if err != nil && !reflect.DeepEqual(testCase.expectedErr, err) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.expectedErr, err)
				t.FailNow()
			}
			if !reflect.DeepEqual(testCase.expected, candidates) {
				t.Errorf("expected: %+v, but actual getAndSortCandidates => %+v", testCase.expected, candidates)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual getAndSortCandidates => %+v", testCase.expected, candidates)
			}
		})
	}
}

func TestConvertNodeOperationResultsToMap(t *testing.T) {
	testCases := []struct {
		name     string
		results  []norv1beta1.NodeOperationResult
		expected map[string]norv1beta1.NodeOperationResult
	}{
		{
			name:     "corner case: results is nil",
			expected: make(map[string]norv1beta1.NodeOperationResult),
		},
		{
			name:     "corner case: results is empty",
			expected: make(map[string]norv1beta1.NodeOperationResult),
		},
		{
			name: "results has 3 elements",
			results: []norv1beta1.NodeOperationResult{
				{
					NodeName:      "fakeNode1",
					WorkRequestId: "fakeWR1",
				}, {
					NodeName:      "fakeNode2",
					WorkRequestId: "fakeWR2",
				},
				{
					NodeName:      "fakeNode3",
					WorkRequestId: "fakeWR3",
				},
			},
			expected: map[string]norv1beta1.NodeOperationResult{
				"fakeNode1": {
					NodeName:      "fakeNode1",
					WorkRequestId: "fakeWR1",
				},
				"fakeNode2": {
					NodeName:      "fakeNode2",
					WorkRequestId: "fakeWR2",
				},
				"fakeNode3": {
					NodeName:      "fakeNode3",
					WorkRequestId: "fakeWR3",
				},
			},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resultsMap := convertNodeOperationResultsToMap(testCase.results)
			if !reflect.DeepEqual(testCase.expected, resultsMap) {
				t.Errorf("expected: %+v, but actual convertNodeOperationResultsToMap => %+v", testCase.expected, resultsMap)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual convertNodeOperationResultsToMap => %+v", testCase.expected, resultsMap)
			}
		})
	}
}

func TestConvertV1NodesToMap(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    []*v1.Node
		expected map[string]*v1.Node
	}{
		{
			name:     "corner case: nodes is nil",
			expected: make(map[string]*v1.Node),
		},
		{
			name:     "corner case: nodes is empty",
			expected: make(map[string]*v1.Node),
		},
		{
			name: "nodes has 3 nodes",
			nodes: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode3",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			expected: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				"fakeNode2": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				"fakeNode3": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode3",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nodesMap := convertV1NodesToMap(testCase.nodes)
			if !reflect.DeepEqual(testCase.expected, nodesMap) {
				t.Errorf("expected: %+v, but actual convertV1NodesToMap => %+v", testCase.expected, nodesMap)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual convertV1NodesToMap => %+v", testCase.expected, nodesMap)
			}
		})
	}

}

func TestTriggerNodeAction(t *testing.T) {
	testCases := []struct {
		name            string
		node            *v1.Node
		nor             norv1beta1.NodeOperationRule
		expected        string
		expectedFailure NodeOperationFailure
	}{
		{
			name: "corner case: actions is nil",
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "fakeNode1",
				},
				Spec: v1.NodeSpec{
					ProviderID: "fakeInstanceId1",
				},
				Status: v1.NodeStatus{},
			},
			nor:      norv1beta1.NodeOperationRule{},
			expected: "",
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode1",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "",
				errorMsg:   errActionsFormat.Error(),
				trackingId: "",
			},
		},
		{
			name: "corner case: actions is empty",
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "fakeNode1",
				},
				Spec: v1.NodeSpec{
					ProviderID: "fakeInstanceId1",
				},
				Status: v1.NodeStatus{},
			},
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: make([]norv1beta1.Action, 0),
				},
			},
			expected: "",
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode1",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "",
				errorMsg:   errActionsFormat.Error(),
				trackingId: "",
			},
		},
		{
			name: "corner case: actions is invalid",
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "fakeNode1",
				},
				Spec: v1.NodeSpec{
					ProviderID: "fakeInstanceId1",
				},
				Status: v1.NodeStatus{},
			},
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"terminate"},
				},
			},
			expected: "",
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode1",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "",
				errorMsg:   errActionsFormat.Error(),
				trackingId: "",
			},
		},
		{
			name: "trigger bvr",
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "fakeNode1",
				},
				Spec: v1.NodeSpec{
					ProviderID: "fakeInstanceId1",
				},
				Status: v1.NodeStatus{},
			},
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"replaceBootVolume"},
				},
			},
			expected:        "fakeInstanceId1fakeWR",
			expectedFailure: NodeOperationFailure{},
		},
		{
			name: "trigger reboot",
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "fakeNode2",
				},
				Spec: v1.NodeSpec{
					ProviderID: "fakeInstanceId2",
				},
				Status: v1.NodeStatus{},
			},
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"reboot"},
				},
			},
			expected:        "fakeInstanceId2fakeWR",
			expectedFailure: NodeOperationFailure{},
		},
		{
			name: "unhappy case: trigger reboot and return failure",
			node: &v1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "failedTooManyRequestsNode1",
				},
				Spec: v1.NodeSpec{
					ProviderID: "failedTooManyRequestsInstance1",
				},
				Status: v1.NodeStatus{},
			},
			nor: norv1beta1.NodeOperationRule{
				Spec: norv1beta1.NodeOperationRuleSpec{
					Actions: []norv1beta1.Action{"reboot"},
				},
			},
			expected: "",
			expectedFailure: NodeOperationFailure{
				nodeName:   "failedTooManyRequestsNode1",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "TooManyRequests",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
	}

	norReconciler := &NodeOperationRuleReconciler{
		OCIClient: OciClientMock{},
		Config: &providercfg.Config{
			ClusterID: "fakeClusterId",
		},
		BvrRateLimiter:    rate.NewLimiter(rate.Every(time.Minute/time.Duration(10)), 10),
		RebootRateLimiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(10)), 10),
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workRequestId, failure := norReconciler.triggerNodeAction(context.Background(), testCase.node, testCase.nor)
			if !reflect.DeepEqual(testCase.expected, workRequestId) || !reflect.DeepEqual(testCase.expectedFailure, failure) {
				t.Errorf("expected: %+v, but actual triggerNodeAction => %+v", testCase.expected, workRequestId)
				t.Errorf("expected: %+v, but actual triggerNodeAction => %+v", testCase.expectedFailure, failure)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual triggerNodeAction => %+v", testCase.expected, workRequestId)
				t.Logf("expected: %+v, and actual triggerNodeAction => %+v", testCase.expectedFailure, failure)
			}
		})
	}

}

func TestConvertApiErrorToNodeOperationFailureType(t *testing.T) {
	testCases := []struct {
		name            string
		nodeName        string
		err             error
		workRequestId   string
		expectedFailure NodeOperationFailure
	}{
		{
			name:          "error is Bvr rate limited",
			nodeName:      "fakeNode1",
			err:           errors.New("rate limited for operation replace boot volume on cluster"),
			workRequestId: "",
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode1",
				errorType:  NodeOperationFailureTypeRateLimited,
				errorCode:  "",
				errorMsg:   rateLimitedErrorMsg,
				trackingId: "",
			},
		},
		{
			name:          "error is reboot rate limited",
			nodeName:      "fakeNode2",
			err:           errors.New("rate limited for operation reboot on cluster"),
			workRequestId: "",
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRateLimited,
				errorCode:  "",
				errorMsg:   rateLimitedErrorMsg,
				trackingId: "",
			},
		},
		{
			name:     "error is 400 BadRequest",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusBadRequest,
				Code:         "BadRequest",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "BadRequest",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "feature flag is not enabled",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusBadRequest,
				Code:         "BadRequest",
				Message:      "Node Action is not enabled",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeNonRetryable,
				errorCode:  "BadRequest",
				errorMsg:   "Node Action is not enabled",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 400 invalid parameter",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusBadRequest,
				Code:         "InvalidParameter",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "InvalidParameter",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 401 NotAuthenticated",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusUnauthorized,
				Code:         "NotAuthenticated",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "NotAuthenticated",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 404 NotAuthorizedOrNotFound",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusNotFound,
				Code:         "NotAuthorizedOrNotFound",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "NotAuthorizedOrNotFound",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 409 conflict",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusConflict,
				Code:         "Conflict",
				Message:      "Node Action is already registered.",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "Conflict",
				errorMsg:   "Node Action is already registered.",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 409 IncorrectState",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusNotFound,
				Code:         "IncorrectState",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "IncorrectState",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 412 NoEtagMatch",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusPreconditionFailed,
				Code:         "NoEtagMatch",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "NoEtagMatch",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 429 TooManyRequests",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusTooManyRequests,
				Code:         "TooManyRequests",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "TooManyRequests",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:     "error is 500 InternalServerError without work request id",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode:   http.StatusInternalServerError,
				Code:         "InternalServerError",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "InternalServerError",
				errorMsg:   "fake error msg",
				trackingId: "aa/bb/cc",
			},
		},
		{
			name:          "error is 500 InternalServerError with work request id",
			nodeName:      "fakeNode2",
			workRequestId: "fakeWR",
			err: mockServiceError{
				StatusCode:   http.StatusInternalServerError,
				Code:         "InternalServerError",
				Message:      "fake error msg",
				OpcRequestID: "aa/bb/cc",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "InternalServerError",
				errorMsg:   "fake error msg",
				trackingId: "fakeWR",
			},
		},
		{
			name:     "error is uncategorized",
			nodeName: "fakeNode2",
			err: mockServiceError{
				StatusCode: http.StatusServiceUnavailable,
				Code:       "StatusServiceUnavailable",
				Message:    "fake error msg",
			},
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "StatusServiceUnavailable",
				errorMsg:   "uncategorizedError: fake error msg",
				trackingId: "",
			},
		},
		{
			name:     "error is not a service error",
			nodeName: "fakeNode2",
			err:      errors.New("unknownError"),
			expectedFailure: NodeOperationFailure{
				nodeName:   "fakeNode2",
				errorType:  NodeOperationFailureTypeRetryable,
				errorCode:  "",
				errorMsg:   "unknownError",
				trackingId: "",
			},
		},
	}
	norReconciler := &NodeOperationRuleReconciler{
		OCIClient: OciClientMock{},
		Config: &providercfg.Config{
			ClusterID: "fakeClusterId",
		},
		Recorder: mockRecorder(),
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testNor := mockNor(testCase.nodeName)
			failure := norReconciler.convertApiErrorToNodeOperationFailureType(context.Background(), testNor, testCase.nodeName, testCase.err, testCase.workRequestId)
			if !reflect.DeepEqual(testCase.expectedFailure, failure) {
				t.Errorf("expected: %+v, but actual convertApiErrorToNodeOperationFailureType => %+v", testCase.expectedFailure, failure)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual convertApiErrorToNodeOperationFailureType => %+v", testCase.expectedFailure, failure)
			}
		})
	}
}

func mockNor(name string) norv1beta1.NodeOperationRule {
	return norv1beta1.NodeOperationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func mockRecorder() record.EventRecorder {
	broadcaster := record.NewBroadcaster()
	return broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "NodeOperationRule"})
}

func TestGetNodeNames(t *testing.T) {
	testCases := []struct {
		name     string
		nodes    []*v1.Node
		expected []string
	}{
		{
			name: "several nodes",
			nodes: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode3",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			expected: []string{"fakeNode1", "fakeNode2", "fakeNode3"},
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			names := getNodeNames(testCase.nodes)
			if !reflect.DeepEqual(testCase.expected, names) {
				t.Errorf("expected: %+v, but actual getNodeNames => %+v", testCase.expected, names)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual getNodeNames => %+v", testCase.expected, names)
			}
		})
	}
}

func TestGetWorkRequest(t *testing.T) {
	timeString := "2024-09-19 14:00:00"
	layout := "2006-01-02 15:04:05"
	parsedTime, _ := time.Parse(layout, timeString)
	norName := "fakeNor"
	testCases := []struct {
		name          string
		workRequestId string
		nodeName      string
		expected      NodeOperationResultUpdate
	}{
		{
			name:          "work request is accepted",
			nodeName:      "fakeNode",
			workRequestId: "accepted",
			expected: NodeOperationResultUpdate{
				NodeName:      "fakeNode",
				WorkRequestId: "accepted",
				update:        UpdateTypeInProgress,
			},
		},
		{
			name:          "work request is in progress",
			nodeName:      "fakeNode",
			workRequestId: "inprogress",
			expected: NodeOperationResultUpdate{
				NodeName:      "fakeNode",
				WorkRequestId: "inprogress",
				update:        UpdateTypeInProgress,
			},
		},
		{
			name:          "work request is succeeded",
			nodeName:      "fakeNode",
			workRequestId: "succeeded",
			expected: NodeOperationResultUpdate{
				NodeName:      "fakeNode",
				WorkRequestId: "succeeded",
				update:        UpdateTypeSucceeded,
				timeFinished:  metav1.NewTime(parsedTime),
			},
		},
		{
			name:          "work request is canceling",
			nodeName:      "fakeNode",
			workRequestId: "canceling",
			expected: NodeOperationResultUpdate{
				NodeName:      "fakeNode",
				WorkRequestId: "canceling",
				update:        UpdateTypeCanceled,
			},
		},
		{
			name:          "work request is canceled",
			nodeName:      "fakeNode",
			workRequestId: "canceled",
			expected: NodeOperationResultUpdate{
				NodeName:      "fakeNode",
				WorkRequestId: "canceled",
				update:        UpdateTypeCanceled,
			},
		},
		{
			name:          "work request is uncategorized",
			nodeName:      "fakeNode",
			workRequestId: "uncategorized",
			expected: NodeOperationResultUpdate{
				NodeName:      "fakeNode",
				WorkRequestId: "uncategorized",
				update:        UpdateTypeRetryable,
			},
		},
	}

	norReconciler := &NodeOperationRuleReconciler{
		OCIClient: OciClientMock{},
		Config: &providercfg.Config{
			ClusterID: "fakeClusterId",
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		testNor := mockNor(norName)
		t.Run(testCase.name, func(t *testing.T) {
			update := norReconciler.getWorkRequest(context.Background(), testNor, testCase.workRequestId, testCase.nodeName)
			if !reflect.DeepEqual(testCase.expected, update) {
				t.Errorf("expected: %+v, but actual getWorkRequest => %+v", testCase.expected, update)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual getWorkRequest => %+v", testCase.expected, update)
			}
		})
	}
}

func TestIsSubset(t *testing.T) {
	testCases := []struct {
		name           string
		nodesInStatus  map[string]norv1beta1.NodeOperationResult
		nodesWithLabel map[string]*v1.Node
		isSubset       bool
		outliers       []string
	}{
		{
			name: "nodesInStatus is nil",
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: true,
			outliers: make([]string, 0),
		},
		{
			name:          "nodesInStatus is empty",
			nodesInStatus: make(map[string]norv1beta1.NodeOperationResult),
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: true,
			outliers: make([]string, 0),
		},
		{
			name: "nodesWithLabel is nil",
			nodesInStatus: map[string]norv1beta1.NodeOperationResult{
				"fakeNode1": {
					NodeName:      "fakeNode1",
					WorkRequestId: "fakeWR1",
				},
			},
			isSubset: false,
			outliers: []string{"fakeNode1"},
		},
		{
			name: "nodesWithLabel is empty",
			nodesInStatus: map[string]norv1beta1.NodeOperationResult{
				"fakeNode1": {
					NodeName:      "fakeNode1",
					WorkRequestId: "fakeWR1",
				},
			},
			nodesWithLabel: make(map[string]*v1.Node, 0),
			isSubset:       false,
			outliers:       []string{"fakeNode1"},
		},
		{
			name: "partial nodes in nodesWithLabel",
			nodesInStatus: map[string]norv1beta1.NodeOperationResult{
				"fakeNode1": {
					NodeName:      "fakeNode1",
					WorkRequestId: "fakeWR1",
				},
				"fakeNode2": {
					NodeName:      "fakeNode2",
					WorkRequestId: "fakeWR2",
				},
			},
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode2": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				"fakeNode3": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode3",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: false,
			outliers: []string{"fakeNode1"},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result1, result2 := isSubset(testCase.nodesInStatus, testCase.nodesWithLabel)
			if !reflect.DeepEqual(testCase.isSubset, result1) || !reflect.DeepEqual(testCase.outliers, result2) {
				t.Errorf("expected: %+v, but actual isSubset => %+v", testCase.isSubset, result1)
				t.Errorf("expected: %+v, but actual isSubset => %+v", testCase.outliers, result2)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual isSubset => %+v", testCase.isSubset, result1)
				t.Logf("expected: %+v, and actual isSubset => %+v", testCase.outliers, result2)
			}
		})
	}
}

func TestIsNodeNameSubset(t *testing.T) {
	testCases := []struct {
		name           string
		nodeNames      []string
		nodesWithLabel map[string]*v1.Node
		isSubset       bool
		outliers       []string
	}{
		{
			name: "nodeNames is nil",
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: true,
			outliers: make([]string, 0),
		},
		{
			name:      "nodeNames is empty",
			nodeNames: make([]string, 0),
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: true,
			outliers: make([]string, 0),
		},
		{
			name:      "nodesWithLabel is nil",
			nodeNames: []string{"fakeNode1", "fakeNode2"},
			isSubset:  false,
			outliers:  []string{"fakeNode1", "fakeNode2"},
		},
		{
			name:           "nodesWithLabel is empty",
			nodeNames:      []string{"fakeNode1", "fakeNode2"},
			nodesWithLabel: make(map[string]*v1.Node, 0),
			isSubset:       false,
			outliers:       []string{"fakeNode1", "fakeNode2"},
		},
		{
			name:      "all node names are contained in nodesWithLabel",
			nodeNames: []string{"fakeNode1", "fakeNode2"},
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				"fakeNode2": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: true,
			outliers: make([]string, 0),
		},
		{
			name:      "partial node names are contained in nodesWithLabel",
			nodeNames: []string{"fakeNode1", "fakeNode3"},
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				"fakeNode2": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: false,
			outliers: []string{"fakeNode3"},
		},
		{
			name:      "all node names are not contained in nodesWithLabel",
			nodeNames: []string{"fakeNode3", "fakeNode4"},
			nodesWithLabel: map[string]*v1.Node{
				"fakeNode1": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode1",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				"fakeNode2": {
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "fakeNode2",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
			},
			isSubset: false,
			outliers: []string{"fakeNode3", "fakeNode4"},
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result1, result2 := isNodeNameSubset(testCase.nodeNames, testCase.nodesWithLabel)
			if !reflect.DeepEqual(testCase.isSubset, result1) || !reflect.DeepEqual(testCase.outliers, result2) {
				t.Errorf("expected: %+v, but actual isNodeNameSubset => %+v", testCase.isSubset, result1)
				t.Errorf("expected: %+v, but actual isNodeNameSubset => %+v", testCase.outliers, result2)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual isNodeNameSubset => %+v", testCase.isSubset, result1)
				t.Logf("expected: %+v, and actual isNodeNameSubset => %+v", testCase.outliers, result2)
			}
		})
	}
}

func TestCancelOperations(t *testing.T) {
	timeString := "2024-09-19 14:00:00"
	layout := "2006-01-02 15:04:05"
	parsedTime, _ := time.Parse(layout, timeString)
	successTime := metav1.NewTime(parsedTime)
	testCases := []struct {
		name                  string
		nor                   norv1beta1.NodeOperationRule
		updateNorStatus       norv1beta1.NodeOperationRuleStatus
		expectedUpdatedStatus norv1beta1.NodeOperationRuleStatus
		expectedErr           error
	}{
		{
			name: "case 1: cancel in-progress node without failure, cancel pending nodes, cancel backOff nodes",
			nor: norv1beta1.NodeOperationRule{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: norv1beta1.NodeOperationRuleStatus{
					InProgressNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node1",
							WorkRequestId: "accepted1",
						},
						{
							NodeName:      "node2",
							WorkRequestId: "accepted2",
						},
					},
					PendingNodes: []string{"node3", "node4"},
					BackOffNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node5",
							WorkRequestId: "fakeOpcId5",
						},
						{
							NodeName:      "node6",
							WorkRequestId: "fakeOpcId6",
						},
					},
					CanceledNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node7",
							WorkRequestId: "fakeworkrequest7",
						},
						{
							NodeName:      "node8",
							WorkRequestId: "fakeOpcId8",
						},
					},
					SucceededNodes: []norv1beta1.NodeOperationSuccess{
						{
							NodeName:         "node9",
							SuccessTimestamp: successTime,
						},
					},
				},
			},
			updateNorStatus: norv1beta1.NodeOperationRuleStatus{
				InProgressNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node1",
						WorkRequestId: "accepted1",
					},
					{
						NodeName:      "node2",
						WorkRequestId: "accepted2",
					},
				},
				PendingNodes: []string{"node3", "node4"},
				BackOffNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node5",
						WorkRequestId: "fakeOpcId5",
					},
					{
						NodeName:      "node6",
						WorkRequestId: "fakeOpcId6",
					},
				},
				CanceledNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node7",
						WorkRequestId: "fakeworkrequest7",
					},
					{
						NodeName:      "node8",
						WorkRequestId: "fakeOpcId8",
					},
				},
				SucceededNodes: []norv1beta1.NodeOperationSuccess{
					{
						NodeName:         "node9",
						SuccessTimestamp: successTime,
					},
				},
			},
			expectedUpdatedStatus: norv1beta1.NodeOperationRuleStatus{
				InProgressNodes: []norv1beta1.NodeOperationResult{},
				PendingNodes:    []string{},
				BackOffNodes:    []norv1beta1.NodeOperationResult{},
				CanceledNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node7",
						WorkRequestId: "fakeworkrequest7",
					},
					{
						NodeName:      "node8",
						WorkRequestId: "fakeOpcId8",
					},
					{
						NodeName:      "node3",
						WorkRequestId: "",
					},
					{
						NodeName:      "node4",
						WorkRequestId: "",
					},
					{
						NodeName:      "node5",
						WorkRequestId: "",
					},
					{
						NodeName:      "node6",
						WorkRequestId: "",
					},
					{
						NodeName:      "node1",
						WorkRequestId: "accepted1",
					},
					{
						NodeName:      "node2",
						WorkRequestId: "accepted2",
					},
				},
				SucceededNodes: []norv1beta1.NodeOperationSuccess{
					{
						NodeName:         "node9",
						SuccessTimestamp: successTime,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "case 2: cancel in-progress node with failure, cancel pending nodes, cancel backOff nodes",
			nor: norv1beta1.NodeOperationRule{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: norv1beta1.NodeOperationRuleStatus{
					InProgressNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node1",
							WorkRequestId: "accepted1",
						},
						{
							NodeName:      "node2",
							WorkRequestId: "failed2",
						},
						{
							NodeName:      "node10",
							WorkRequestId: "accepted10",
						},
					},
					PendingNodes: []string{"node3", "node4"},
					BackOffNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node5",
							WorkRequestId: "fakeOpcId5",
						},
						{
							NodeName:      "node6",
							WorkRequestId: "fakeOpcId6",
						},
					},
					CanceledNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node7",
							WorkRequestId: "fakeworkrequest7",
						},
						{
							NodeName:      "node8",
							WorkRequestId: "fakeOpcId8",
						},
					},
					SucceededNodes: []norv1beta1.NodeOperationSuccess{
						{
							NodeName:         "node9",
							SuccessTimestamp: successTime,
						},
					},
				},
			},
			updateNorStatus: norv1beta1.NodeOperationRuleStatus{
				InProgressNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node1",
						WorkRequestId: "accepted1",
					},
					{
						NodeName:      "node2",
						WorkRequestId: "failed2",
					},
					{
						NodeName:      "node10",
						WorkRequestId: "accepted10",
					},
				},
				PendingNodes: []string{"node3", "node4"},
				BackOffNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node5",
						WorkRequestId: "fakeOpcId5",
					},
					{
						NodeName:      "node6",
						WorkRequestId: "fakeOpcId6",
					},
				},
				CanceledNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node7",
						WorkRequestId: "fakeworkrequest7",
					},
					{
						NodeName:      "node8",
						WorkRequestId: "fakeOpcId8",
					},
				},
				SucceededNodes: []norv1beta1.NodeOperationSuccess{
					{
						NodeName:         "node9",
						SuccessTimestamp: successTime,
					},
				},
			},
			expectedUpdatedStatus: norv1beta1.NodeOperationRuleStatus{
				InProgressNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node2",
						WorkRequestId: "failed2",
					},
					{
						NodeName:      "node10",
						WorkRequestId: "accepted10",
					},
				},
				PendingNodes: []string{},
				BackOffNodes: []norv1beta1.NodeOperationResult{},
				CanceledNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node7",
						WorkRequestId: "fakeworkrequest7",
					},
					{
						NodeName:      "node8",
						WorkRequestId: "fakeOpcId8",
					},
					{
						NodeName:      "node3",
						WorkRequestId: "",
					},
					{
						NodeName:      "node4",
						WorkRequestId: "",
					},
					{
						NodeName:      "node5",
						WorkRequestId: "",
					},
					{
						NodeName:      "node6",
						WorkRequestId: "",
					},
					{
						NodeName:      "node1",
						WorkRequestId: "accepted1",
					},
				},
				SucceededNodes: []norv1beta1.NodeOperationSuccess{
					{
						NodeName:         "node9",
						SuccessTimestamp: successTime,
					},
				},
			},
			expectedErr: errors.New("fail to cancel work request"),
		},
		{
			name: "case 3: nor status has not initialized and nor status is different from updateNorStatus",
			nor: norv1beta1.NodeOperationRule{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: norv1beta1.NodeOperationRuleStatus{
					PendingNodes: []string{"node3", "node4"},
					CanceledNodes: []norv1beta1.NodeOperationResult{
						{
							NodeName:      "node7",
							WorkRequestId: "fakeworkrequest7",
						},
						{
							NodeName:      "node8",
							WorkRequestId: "fakeOpcId8",
						},
					},
					SucceededNodes: []norv1beta1.NodeOperationSuccess{
						{
							NodeName:         "node9",
							SuccessTimestamp: successTime,
						},
					},
				},
			},
			updateNorStatus: norv1beta1.NodeOperationRuleStatus{
				InProgressNodes: []norv1beta1.NodeOperationResult{},
				PendingNodes:    []string{"node3", "node4"},
				BackOffNodes:    []norv1beta1.NodeOperationResult{},
				CanceledNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node7",
						WorkRequestId: "fakeworkrequest7",
					},
					{
						NodeName:      "node8",
						WorkRequestId: "fakeOpcId8",
					},
				},
				SucceededNodes: []norv1beta1.NodeOperationSuccess{
					{
						NodeName:         "node9",
						SuccessTimestamp: successTime,
					},
				},
			},
			expectedUpdatedStatus: norv1beta1.NodeOperationRuleStatus{
				InProgressNodes: []norv1beta1.NodeOperationResult{},
				PendingNodes:    []string{},
				BackOffNodes:    []norv1beta1.NodeOperationResult{},
				CanceledNodes: []norv1beta1.NodeOperationResult{
					{
						NodeName:      "node7",
						WorkRequestId: "fakeworkrequest7",
					},
					{
						NodeName:      "node8",
						WorkRequestId: "fakeOpcId8",
					},
					{
						NodeName:      "node3",
						WorkRequestId: "",
					},
					{
						NodeName:      "node4",
						WorkRequestId: "",
					},
				},
				SucceededNodes: []norv1beta1.NodeOperationSuccess{
					{
						NodeName:         "node9",
						SuccessTimestamp: successTime,
					},
				},
			},
			expectedErr: nil,
		},
	}

	norReconciler := &NodeOperationRuleReconciler{
		OCIClient: OciClientMock{},
		Config: &providercfg.Config{
			ClusterID: "fakeClusterId",
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			updatedStatus, err := norReconciler.cancelOperations(context.Background(), testCase.nor, testCase.updateNorStatus)
			if (err == nil && testCase.expectedErr != nil) || (err != nil && testCase.expectedErr == nil) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.expectedErr, err)
				t.FailNow()
			}
			if err != nil && !reflect.DeepEqual(testCase.expectedErr, err) {
				t.Errorf("expected err:\n%+v\nbut got err:\n%+v", testCase.expectedErr, err)
				t.FailNow()
			}
			if !reflect.DeepEqual(testCase.expectedUpdatedStatus, updatedStatus) {
				t.Errorf("expected: %+v, but actual cancelOperations => %+v", testCase.expectedUpdatedStatus, updatedStatus)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual cancelOperations => %+v", testCase.expectedUpdatedStatus, updatedStatus)
			}
		})
	}
}

func TestInitializeListInStatus(t *testing.T) {
	testCases := []struct {
		name       string
		resultList []norv1beta1.NodeOperationResult
		expected   []norv1beta1.NodeOperationResult
	}{
		{
			name:     "case 1: resultList is nil",
			expected: make([]norv1beta1.NodeOperationResult, 0),
		},
		{
			name:       "case 2: resultList is empty",
			resultList: []norv1beta1.NodeOperationResult{},
			expected:   make([]norv1beta1.NodeOperationResult, 0),
		},
		{
			name:       "case 3: resultList is not empty and expected value should be same as resultList",
			resultList: []norv1beta1.NodeOperationResult{{NodeName: "fakeNode1", WorkRequestId: "fakeWorkRequest1"}},
			expected:   []norv1beta1.NodeOperationResult{{NodeName: "fakeNode1", WorkRequestId: "fakeWorkRequest1"}},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := initializeListInStatus(testCase.resultList)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual initializeListInStatus => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual initializeListInStatus => %+v", testCase.expected, actual)
			}
		})
	}
}

func TestInitializeSuccessListInStatus(t *testing.T) {
	timeString := "2024-09-19 14:00:00"
	layout := "2006-01-02 15:04:05"
	parsedTime, _ := time.Parse(layout, timeString)
	successTime := metav1.NewTime(parsedTime)

	testCases := []struct {
		name       string
		resultList []norv1beta1.NodeOperationSuccess
		expected   []norv1beta1.NodeOperationSuccess
	}{
		{
			name:     "case 1: resultList is nil",
			expected: make([]norv1beta1.NodeOperationSuccess, 0),
		},
		{
			name:       "case 2: resultList is empty",
			resultList: []norv1beta1.NodeOperationSuccess{},
			expected:   make([]norv1beta1.NodeOperationSuccess, 0),
		},
		{
			name:       "case 3: resultList is not empty and expected value should be same as resultList",
			resultList: []norv1beta1.NodeOperationSuccess{{NodeName: "fakeNode1", SuccessTimestamp: successTime}},
			expected:   []norv1beta1.NodeOperationSuccess{{NodeName: "fakeNode1", SuccessTimestamp: successTime}},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := initializeSuccessListInStatus(testCase.resultList)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual initializeSuccessListInStatus => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual initializeSuccessListInStatus => %+v", testCase.expected, actual)
			}
		})
	}
}

func TestInitializePendingList(t *testing.T) {
	testCases := []struct {
		name       string
		resultList []string
		expected   []string
	}{
		{
			name:     "case 1: resultList is nil",
			expected: make([]string, 0),
		},
		{
			name:       "case 2: resultList is empty",
			resultList: []string{},
			expected:   make([]string, 0),
		},
		{
			name:       "case 3: resultList is not empty and expected value should be same as resultList",
			resultList: []string{"fakeNode1"},
			expected:   []string{"fakeNode1"},
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := initializePendingList(testCase.resultList)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual initializePendingList => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual initializePendingList => %+v", testCase.expected, actual)
			}
		})
	}

}

func TestIsNorStale(t *testing.T) {
	testCases := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expected       bool
	}{
		{
			name:     "initialization",
			expected: false,
		},
		{
			name:           "nil stored version",
			currentVersion: "228744453",
			expected:       false,
		},
		{
			name:           "empty stored version",
			currentVersion: "228744453",
			latestVersion:  "",
			expected:       false,
		},
		{
			name:          "nil current version",
			latestVersion: "228744453",
			expected:      true,
		},
		{
			name:           "empty current version",
			currentVersion: "",
			latestVersion:  "228744453",
			expected:       true,
		},
		{
			name:           "NOR is update-to-date",
			currentVersion: "228744453",
			latestVersion:  "228744453",
			expected:       false,
		},
		{
			name:           "NOR is stale",
			currentVersion: "228717558",
			latestVersion:  "228744446",
			expected:       true,
		},
		{
			name:           "Nor is update-to-date with more digits",
			currentVersion: "103893",
			latestVersion:  "92455",
			expected:       false,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := isNorStale(testCase.currentVersion, testCase.latestVersion)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual isNorStale => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual isNorStale => %+v", testCase.expected, actual)
			}
		})
	}
}

func TestGetNorNameNodeName(t *testing.T) {
	testCases := []struct {
		name     string
		norName  string
		nodeName string
		expected string
	}{
		{
			name:     "both are nil",
			expected: "",
		},
		{
			name:     "both are empty",
			norName:  "",
			nodeName: "",
			expected: "",
		},
		{
			name:     "both are non-empty strings",
			norName:  "nor-test",
			nodeName: "10.10.10.8",
			expected: "nor-test10.10.10.8",
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := getNorNameNodeName(testCase.norName, testCase.nodeName)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual getNorNameNodeName => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual getNorNameNodeName => %+v", testCase.expected, actual)
			}
		})
	}
}

func TestCalculateLatency(t *testing.T) {
	testCases := []struct {
		name     string
		timer    sync.Map
		norName  string
		nodeName string
		expected float64
	}{
		{
			name:     "start timestamp does not exist",
			timer:    sync.Map{},
			norName:  "nor-test",
			nodeName: "10.10.1.1",
			expected: 0,
		},
		{
			name:     "star timestamp exist",
			timer:    getTimerMap(),
			norName:  "nor-test",
			nodeName: "10.10.1.1",
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := calculateLatency(testCase.timer, testCase.norName, testCase.nodeName)
			_, found := testCase.timer.Load(getNorNameNodeName(testCase.norName, testCase.nodeName))
			if testCase.norName == "start timestamp does not exist" {
				if !reflect.DeepEqual(testCase.expected, actual) && !found {
					t.Errorf("expected: %+v, but actual calculateLatency => %+v", testCase.expected, actual)
					t.FailNow()
				} else {
					t.Logf("expected: %+v, and actual calculateLatency => %+v", testCase.expected, actual)
				}
			} else if testCase.name == "star timestamp exist" {
				if actual <= 0 && !found {
					t.Errorf("expected: %+v, but actual calculateLatency => %+v", testCase.expected, actual)
					t.FailNow()
				} else {
					t.Logf("now: %+v, actual calculateLatency => %+v", time.Now(), actual)
				}
			}

		})
	}
}

func TestSortSucceededNodesAndGarbageCollect(t *testing.T) {
	now := metav1.Now()
	additionalSuccedednodeItem := norv1beta1.NodeOperationSuccess{
		NodeName:         "latestNode",
		SuccessTimestamp: metav1.NewTime(now.Time.Add(time.Duration(3000) * time.Second)),
	}
	testCases := []struct {
		name                          string
		succededNodesBeforeGC         []norv1beta1.NodeOperationSuccess
		succededNodesAfterGC          []norv1beta1.NodeOperationSuccess
		numberOfNodesGarbageCollected int
		err                           error
	}{
		{
			name:                          "corner case: succededNodesBeforeGC and succededNodesAfterGC is nil",
			succededNodesAfterGC:          nil,
			numberOfNodesGarbageCollected: 0,
		},
		{
			name:                          "corner case: succededNodesBeforeGC is empty and succededNodesAfterGC is empty",
			succededNodesBeforeGC:         make([]norv1beta1.NodeOperationSuccess, 0),
			succededNodesAfterGC:          make([]norv1beta1.NodeOperationSuccess, 0),
			numberOfNodesGarbageCollected: 0,
		},
		{
			name:                          "SucceededNodes less than 1000",
			succededNodesBeforeGC:         generateSucceededNodes(0, 10, now),
			succededNodesAfterGC:          generateSucceededNodes(0, 10, now),
			numberOfNodesGarbageCollected: 0,
		},
		{
			name:                          "SucceededNodes count more than 1000",
			succededNodesBeforeGC:         generateSucceededNodes(0, 1100, now),
			succededNodesAfterGC:          generateSucceededNodes(100, 1100, now),
			numberOfNodesGarbageCollected: 100,
		},
		{
			name:                          "SucceededNodes count more than 1000, and also needs to be sorted",
			succededNodesBeforeGC:         append([]norv1beta1.NodeOperationSuccess{additionalSuccedednodeItem}, generateSucceededNodes(0, 1000, now)...),
			succededNodesAfterGC:          append(generateSucceededNodes(1, 1000, now), additionalSuccedednodeItem),
			numberOfNodesGarbageCollected: 1,
		},
	}

	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			SucceededNodes, numberOfNodesGarbageCollected := sortSucceededNodesAndGarbageCollect(testCase.succededNodesBeforeGC)
			if !reflect.DeepEqual(testCase.succededNodesAfterGC, SucceededNodes) {
				t.Errorf("expected: %+v, but actual mergeFilterLabels => %+v", testCase.succededNodesAfterGC, SucceededNodes)
			} else {
				t.Logf("expected: %+v, and actual mergeFilterLabels => %+v", testCase.succededNodesAfterGC, SucceededNodes)
			}
			if numberOfNodesGarbageCollected != testCase.numberOfNodesGarbageCollected {
				t.Errorf("expected number of garbageCollection: %d, but actual number of garbageCollection => %d", testCase.numberOfNodesGarbageCollected, numberOfNodesGarbageCollected)
			}
		})
	}
}

func generateSucceededNodes(rangeStart int, rangeEnd int, now metav1.Time) []norv1beta1.NodeOperationSuccess {
	succeededNodes := make([]norv1beta1.NodeOperationSuccess, 0)
	dummyNodeIp := "1.0.1."
	for rangeStart < rangeEnd {
		succeededNodes = append(succeededNodes,
			norv1beta1.NodeOperationSuccess{
				NodeName:         fmt.Sprintf("%s%d", dummyNodeIp, rangeStart),
				SuccessTimestamp: metav1.NewTime(now.Time.Add(time.Duration(rangeStart) * time.Second)),
			})
		rangeStart++
	}
	return succeededNodes
}

func TestRemoveLatency(t *testing.T) {
	testCases := []struct {
		name     string
		timer    sync.Map
		norName  string
		nodeName string
	}{
		{
			name:     "map is nil",
			norName:  "nor-test",
			nodeName: "10.10.1.1",
		},
		{
			name: "every thing is nil",
		},
		{
			name:     "start timestamp does not exist",
			timer:    sync.Map{},
			norName:  "nor-test",
			nodeName: "10.10.1.1",
		},
		{
			name:     "star timestamp exist",
			timer:    getTimerMap(),
			norName:  "nor-test",
			nodeName: "10.10.1.1",
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			removeLatency(testCase.timer, testCase.norName, testCase.nodeName)
			_, found := testCase.timer.Load(getNorNameNodeName(testCase.norName, testCase.nodeName))
			if found {
				t.Errorf("expected: not found, but actual removeLatency => found")
				t.FailNow()
			} else {
				t.Logf("expected: not found, and actual removeLatency is expected")
			}
		})
	}
}

func getTimerMap() sync.Map {
	var timer sync.Map
	ts := time.Now()
	timer.LoadOrStore("nor-test10.10.1.1", ts)
	return timer
}

func TestGetLatestResourceVersion(t *testing.T) {
	testCases := []struct {
		name       string
		versionMap map[types.NamespacedName]string
		key        types.NamespacedName
		version    string
		expected   string
	}{
		{
			name:     "all are nil",
			expected: "",
		},
		{
			name:     "all are empty",
			key:      types.NamespacedName{},
			version:  "",
			expected: "",
		},
		{
			name:       "map is empty",
			versionMap: make(map[types.NamespacedName]string),
			key: types.NamespacedName{
				Name:      "nor-test",
				Namespace: "",
			},
			version:  "12345",
			expected: "12345",
		},
		{
			name: "map has version",
			versionMap: map[types.NamespacedName]string{types.NamespacedName{
				Name:      "nor-test",
				Namespace: "",
			}: "12344"},
			key: types.NamespacedName{
				Name:      "nor-test",
				Namespace: "",
			},
			version:  "12345",
			expected: "12344",
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			globalResourceVersionMap = testCase.versionMap
			actual := getLatestResourceVersion(testCase.key, testCase.version)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual getLatestResourceVersion => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual getLatestResourceVersion => %+v", testCase.expected, actual)
			}
		})
	}
}

func TestInvalidParameterErrorNeedsAlarm(t *testing.T) {
	testCases := []struct {
		name     string
		errorMsg string
		expected bool
	}{
		{
			name:     "msg is nil",
			expected: false,
		},
		{
			name:     "msg is empty",
			errorMsg: "",
			expected: false,
		},
		{
			name:     "invalid nodeid",
			errorMsg: "Invalid nodeId",
			expected: true,
		},
		{
			name:     "invalid node Id",
			errorMsg: "Invalid ocid.instance.xxx: Invalid node Id",
			expected: true,
		},
		{
			name:     "invalid cluster id",
			errorMsg: "Invalid clusterId",
			expected: true,
		},
		{
			name:     "not is not a member of the cluster",
			errorMsg: "The node does not belong to the specified cluster",
			expected: true,
		},
		{
			name:     "do not need alarm",
			errorMsg: "Not a valid node",
			expected: false,
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := invalidParameterErrorNeedsAlarm(testCase.errorMsg)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual invalidParameterErrorNeedsAlarm => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual invalidParameterErrorNeedsAlarm => %+v", testCase.expected, actual)
			}
		})
	}
}

func TestAddNodeToPendingNodesDueToMaxParallelism(t *testing.T) {
	testCases := []struct {
		name                string
		parallelism         int
		candidates          []*v1.Node
		currentPendingNodes []string
		expected            []string
	}{
		{
			name: "candidates is nil and currentPendingNodes is nil so updated PendingNodes should keep same",
		},
		{
			name:                "candidates is nil and currentPendingNodes is nil so updated PendingNodes should keep same",
			currentPendingNodes: []string{"10.0.10.122"},
			expected:            []string{"10.0.10.122"},
		},
		{
			name:        "[candidate size is greater than parallelism] 1 candidate: 10.0.10.122 needs to be triggered and parallelism is as 0, so 10.0.10.122 needs to be added to pendingNodes",
			parallelism: 0,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: make([]string, 0),
			expected:            []string{"10.0.10.121"},
		},
		{
			name:        "[candidate size is greater than parallelism] candidates have 2 nodes need to be triggered, parallelism is 1, 1 candidate needs to be added to pendingNodes",
			parallelism: 1,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: make([]string, 0),
			expected:            []string{"10.0.10.122"},
		},
		{
			name:        "[candidate size is greater than parallelism] candidates have 3 nodes need to be triggered, parallelism is 2, 1 candidate needs to be added to pendingNodes",
			parallelism: 2,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: make([]string, 0),
			expected:            []string{"10.0.10.123"},
		},
		{
			name:        "[candidate size is greater than parallelism] candidates have 5 nodes need to be triggered, parallelism is 1, 4 candidate needs to be added to pendingNodes",
			parallelism: 1,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.124",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId4",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.125",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId5",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: make([]string, 0),
			expected:            []string{"10.0.10.122", "10.0.10.123", "10.0.10.124", "10.0.10.125"},
		},
		{
			name:        "[candidate size is greater than parallelism] candidates have 5 nodes need to be triggered, parallelism is 3, 2 candidate needs to be added to pendingNodes",
			parallelism: 3,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.124",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId4",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.125",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId5",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: make([]string, 0),
			expected:            []string{"10.0.10.124", "10.0.10.125"},
		},
		{
			name:        "[candidate size is greater than parallelism] candidates have 5 nodes need to be triggered, parallelism is 3, 2 candidate needs to be added to pendingNodes and current pendingNodes have nodes",
			parallelism: 3,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.124",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId4",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.125",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId5",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: []string{"10.0.10.124", "10.0.10.126"},
			expected:            []string{"10.0.10.124", "10.0.10.126", "10.0.10.125"},
		},
		{
			name:        "[candidate size is equal to parallelism] candidates have 3 nodes need to be triggered, parallelism is 3, no candidate needs to be added to pendingNodes",
			parallelism: 3,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: []string{},
			expected:            []string{},
		},
		{
			name:        "[candidate size is equal to parallelism] candidates have 3 nodes need to be triggered, parallelism is 3, no candidate needs to be added to pendingNodes and current pendingNodes have nodes",
			parallelism: 3,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: []string{"10.0.10.126"},
			expected:            []string{"10.0.10.126"},
		},
		{
			name:        "[candidate size is less than parallelism] candidates have 3 nodes need to be triggered, parallelism is 3, no candidate needs to be added to pendingNodes",
			parallelism: 4,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: []string{},
			expected:            []string{},
		}, {
			name:        "[candidate size is less than parallelism] candidates have 3 nodes need to be triggered, parallelism is 3, no candidate needs to be added to pendingNodes and current pendingNodes have nodes",
			parallelism: 3,
			candidates: []*v1.Node{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.121",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId1",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.122",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId2",
					},
					Status: v1.NodeStatus{},
				},
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "10.0.10.123",
					},
					Spec: v1.NodeSpec{
						ProviderID: "fakeInstanceId3",
					},
					Status: v1.NodeStatus{},
				},
			},
			currentPendingNodes: []string{"10.0.10.121"},
			expected:            []string{"10.0.10.121"},
		},
	}
	t.Parallel()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := addNodesToPendingNodesDueToParallelism(testCase.parallelism, testCase.candidates, testCase.currentPendingNodes)
			if !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("expected: %+v, but actual addNodesToPendingNodesDueToParallelism => %+v", testCase.expected, actual)
				t.FailNow()
			} else {
				t.Logf("expected: %+v, and actual addNodesToPendingNodesDueToParallelism => %+v", testCase.expected, actual)
			}
		})
	}
}

type OciClientMock struct {
}

func (c OciClientMock) Lustre() client.LustreInterface {
	return nil
}

func (c OciClientMock) LoadBalancer(logger *zap.SugaredLogger, s string, config *client.OCIClientConfig) client.GenericLoadBalancerInterface {
	return nil
}

func (c OciClientMock) NewWorkloadIdentityClient(logger *zap.SugaredLogger, lbType string, ociClientConfig *client.OCIClientConfig) client.Interface {
	return nil
}

func (c OciClientMock) Compute() client.ComputeInterface {
	return nil
}

func (c OciClientMock) Networking(config *client.OCIClientConfig) client.NetworkingInterface {
	return nil
}

func (c OciClientMock) BlockStorage() client.BlockStorageInterface {
	return nil
}

func (c OciClientMock) FSS(config *client.OCIClientConfig) client.FileStorageInterface {
	return nil
}

func (c OciClientMock) Identity(config *client.OCIClientConfig) client.IdentityInterface {
	return nil
}

func (c OciClientMock) ContainerEngine() client.ContainerEngineInterface {
	return &ContainerEngineMock{}
}

func (c OciClientMock) CertManager() client.CertificateManagerInterface {
	return &MockCertificateManagerClient{}
}

type ContainerEngineMock struct {
}

func (c *ContainerEngineMock) GetVirtualNode(ctx context.Context, virtualNodeId, virtualNodePoolId string) (*containerengine.VirtualNode, error) {
	return nil, nil
}

func (c *ContainerEngineMock) ReplaceBootVolumeClusterNode(ctx context.Context, nodeId string, clusterId string, nor norv1beta1.NodeOperationRule) (string, error) {
	return nodeId + "fakeWR", nil
}

func (c *ContainerEngineMock) RebootClusterNode(ctx context.Context, nodeId string, clusterId string, nor norv1beta1.NodeOperationRule) (string, error) {
	if strings.Contains(nodeId, "TooManyRequests") {
		return "", mockServiceError{
			StatusCode:   http.StatusTooManyRequests,
			Code:         "TooManyRequests",
			Message:      "fake error msg",
			OpcRequestID: "aa/bb/cc",
		}
	}
	return nodeId + "fakeWR", nil
}

func (c *ContainerEngineMock) GetWorkRequest(ctx context.Context, workRequestId string) (*containerengine.WorkRequest, error) {
	if workRequestId == "accepted" {
		return &containerengine.WorkRequest{
			Id:     &workRequestId,
			Status: containerengine.WorkRequestStatusAccepted,
		}, nil
	} else if workRequestId == "inprogress" {
		return &containerengine.WorkRequest{
			Id:     &workRequestId,
			Status: containerengine.WorkRequestStatusInProgress,
		}, nil
	} else if workRequestId == "failed" {
		return &containerengine.WorkRequest{
			Id:     &workRequestId,
			Status: containerengine.WorkRequestStatusFailed,
		}, nil
	} else if workRequestId == "succeeded" {
		timeString := "2024-09-19 14:00:00"
		layout := "2006-01-02 15:04:05"
		parsedTime, _ := time.Parse(layout, timeString)
		return &containerengine.WorkRequest{
			Id:           &workRequestId,
			Status:       containerengine.WorkRequestStatusSucceeded,
			TimeFinished: &common.SDKTime{Time: parsedTime},
		}, nil
	} else if workRequestId == "canceling" {
		return &containerengine.WorkRequest{
			Id:     &workRequestId,
			Status: containerengine.WorkRequestStatusCanceling,
		}, nil
	} else if workRequestId == "canceled" {
		return &containerengine.WorkRequest{
			Id:     &workRequestId,
			Status: containerengine.WorkRequestStatusCanceled,
		}, nil
	}
	return &containerengine.WorkRequest{
		Id:     &workRequestId,
		Status: "unknown",
	}, nil
}

func (c *ContainerEngineMock) DeleteWorkRequest(ctx context.Context, workRequestId string) (string, error) {
	if strings.Contains(workRequestId, "accepted") {
		return workRequestId, nil
	}
	if strings.Contains(workRequestId, "failed") {
		return workRequestId, errors.New("fail to cancel work request")
	}
	return workRequestId, nil
}

type mockServiceError struct {
	StatusCode   int
	Code         string
	Message      string
	OpcRequestID string
}

func (m mockServiceError) GetHTTPStatusCode() int {
	return m.StatusCode
}

func (m mockServiceError) GetMessage() string {
	return m.Message
}

func (m mockServiceError) GetCode() string {
	return m.Code
}

func (m mockServiceError) GetOpcRequestID() string {
	return m.OpcRequestID
}
func (m mockServiceError) Error() string {
	return m.Message
}
