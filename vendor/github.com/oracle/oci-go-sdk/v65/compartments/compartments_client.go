// Copyright (c) 2016, 2018, 2024, Oracle and/or its affiliates.  All rights reserved.
// This software is dual-licensed to you under the Universal Permissive License (UPL) 1.0 as shown at https://oss.oracle.com/licenses/upl or Apache License 2.0 as shown at http://www.apache.org/licenses/LICENSE-2.0. You may choose either license.
// Code generated. DO NOT EDIT.

// Compartments Service API
//
// Use Compartments Service API to manage compartments.
//

package compartments

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"net/http"

	"regexp"
)

// CompartmentsClient a client for Compartments
type CompartmentsClient struct {
	common.BaseClient
	config                   *common.ConfigurationProvider
	requiredParamsInEndpoint map[string][]common.TemplateParamForPerRealmEndpoint
}

// NewCompartmentsClientWithConfigurationProvider Creates a new default Compartments client with the given configuration provider.
// the configuration provider will be used for the default signer as well as reading the region
func NewCompartmentsClientWithConfigurationProvider(configProvider common.ConfigurationProvider) (client CompartmentsClient, err error) {
	if enabled := common.CheckForEnabledServices("compartments"); !enabled {
		return client, fmt.Errorf("the Developer Tool configuration disabled this service, this behavior is controlled by OciSdkEnabledServicesMap variables. Please check if your local developer-tool-configuration.json file configured the service you're targeting or contact the cloud provider on the availability of this service")
	}
	provider, err := auth.GetGenericConfigurationProvider(configProvider)
	if err != nil {
		return client, err
	}
	baseClient, e := common.NewClientWithConfig(provider)
	if e != nil {
		return client, e
	}
	return newCompartmentsClientFromBaseClient(baseClient, provider)
}

// NewCompartmentsClientWithOboToken Creates a new default Compartments client with the given configuration provider.
// The obotoken will be added to default headers and signed; the configuration provider will be used for the signer
//
//	as well as reading the region
func NewCompartmentsClientWithOboToken(configProvider common.ConfigurationProvider, oboToken string) (client CompartmentsClient, err error) {
	baseClient, err := common.NewClientWithOboToken(configProvider, oboToken)
	if err != nil {
		return client, err
	}

	return newCompartmentsClientFromBaseClient(baseClient, configProvider)
}

func newCompartmentsClientFromBaseClient(baseClient common.BaseClient, configProvider common.ConfigurationProvider) (client CompartmentsClient, err error) {
	// Compartments service default circuit breaker is enabled
	baseClient.Configuration.CircuitBreaker = common.NewCircuitBreaker(common.DefaultCircuitBreakerSettingWithServiceName("Compartments"))
	common.ConfigCircuitBreakerFromEnvVar(&baseClient)
	common.ConfigCircuitBreakerFromGlobalVar(&baseClient)

	client = CompartmentsClient{BaseClient: baseClient}
	client.BasePath = "20160918"
	err = client.setConfigurationProvider(configProvider)
	return
}

// SetRegion overrides the region of this client.
func (client *CompartmentsClient) SetRegion(region string) {
	client.Host, _ = common.StringToRegion(region).EndpointForTemplateDottedRegion("compartments", client.getEndpointTemplatePerRealm(region), "compartments")
	client.parseEndpointTemplatePerRealm()
}

// SetConfigurationProvider sets the configuration provider including the region, returns an error if is not valid
func (client *CompartmentsClient) setConfigurationProvider(configProvider common.ConfigurationProvider) error {
	if ok, err := common.IsConfigurationProviderValid(configProvider); !ok {
		return err
	}

	// Error has been checked already
	region, _ := configProvider.Region()
	client.SetRegion(region)
	if client.Host == "" {
		return fmt.Errorf("invalid region or Host. Endpoint cannot be constructed without endpointServiceName or serviceEndpointTemplate for a dotted region")
	}
	client.config = &configProvider
	return nil
}

// ConfigurationProvider the ConfigurationProvider used in this client, or null if none set
func (client *CompartmentsClient) ConfigurationProvider() *common.ConfigurationProvider {
	return client.config
}

// EnableDualStackEndpoints Determines whether dual stack endpoint should be used or not.
// Default value is false
func (client *CompartmentsClient) EnableDualStackEndpoints(enableDualStack bool) {
	client.BaseClient.EnableDualStackEndpoints(enableDualStack)
}

// getEndpointTemplatePerRealm returns the endpoint template for the given region, if not found, returns the default endpoint template
func (client *CompartmentsClient) getEndpointTemplatePerRealm(region string) string {
	if client.IsOciRealmSpecificServiceEndpointTemplateEnabled() {
		realm, _ := common.StringToRegion(region).RealmID()
		templatePerRealmDict := map[string]string{
			"oc1": "https://{dualStack?ds.:}compartments.{region}.oci.{secondLevelDomain}",
		}
		if template, ok := templatePerRealmDict[realm]; ok {
			return template
		}
	}
	return "https://compartments.{region}.oci.{secondLevelDomain}"
}

// parseEndpointTemplatePerRealm parses the endpoint template per realm from the service endpoint template
// This function will build a map of template params to their values, this map is used when building the API endpoint
func (client *CompartmentsClient) parseEndpointTemplatePerRealm() {
	client.requiredParamsInEndpoint = make(map[string][]common.TemplateParamForPerRealmEndpoint)
	templateRegex := regexp.MustCompile(`{.*?}`)
	templateSubRegex := regexp.MustCompile(`{(.+)\+Dot}`)
	templates := templateRegex.FindAllString(client.Host, -1)
	for _, template := range templates {
		templateParam := templateSubRegex.FindStringSubmatch(template)
		if len(templateParam) > 1 {
			client.requiredParamsInEndpoint[templateParam[1]] = append(client.requiredParamsInEndpoint[templateParam[1]], common.TemplateParamForPerRealmEndpoint{
				Template:    templateParam[0],
				EndsWithDot: true,
			})
		} else {
			templateParam := template[1 : len(template)-1]
			client.requiredParamsInEndpoint[templateParam] = append(client.requiredParamsInEndpoint[templateParam], common.TemplateParamForPerRealmEndpoint{
				Template:    template,
				EndsWithDot: false,
			})
		}
	}
}

// SetCustomClientConfiguration sets client with retry and other custom configurations
func (client *CompartmentsClient) SetCustomClientConfiguration(config common.CustomClientConfiguration) {
	client.Configuration = config
	client.refreshRegion()
}

// refreshRegion will refresh the region of this client, this function will be called after setting the CustomClientConfiguration
func (client *CompartmentsClient) refreshRegion() {
	configProvider := *client.config
	region, _ := configProvider.Region()
	client.SetRegion(region)
}

// BulkDeleteResources Deletes multiple resources in the compartment. All resources must be in the same compartment. You must have the appropriate
// permissions to delete the resources in the request. This API can only be invoked from the tenancy's
// home region (https://docs.cloud.oracle.com/Content/Identity/regions/managingregions.htm#Home). This operation creates a
// WorkRequest. Use the GetWorkRequest
// API to monitor the status of the bulk action.
// A default retry strategy applies to this operation BulkDeleteResources()
func (client CompartmentsClient) BulkDeleteResources(ctx context.Context, request BulkDeleteResourcesRequest) (response BulkDeleteResourcesResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}

	if !(request.OpcRetryToken != nil && *request.OpcRetryToken != "") {
		request.OpcRetryToken = common.String(common.RetryToken())
	}

	ociResponse, err = common.Retry(ctx, request, client.bulkDeleteResources, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = BulkDeleteResourcesResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = BulkDeleteResourcesResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(BulkDeleteResourcesResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into BulkDeleteResourcesResponse")
	}
	return
}

// bulkDeleteResources implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) bulkDeleteResources(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/compartments/{compartmentId}/actions/bulkDeleteResources", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(BulkDeleteResourcesRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response BulkDeleteResourcesResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "BulkDeleteResources", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// BulkEditResource Moves multiple resources from one compartment to another. All resources must be in the same compartment.
// This API can only be invoked from the tenancy's home region (https://docs.cloud.oracle.com/Content/Identity/regions/managingregions.htm#Home).
// To move resources, you must have the appropriate permissions to move the resource in both the source and target
// compartments. This operation creates a WorkRequest.
// Use the GetWorkRequest API to monitor the status of the bulk action.
// A default retry strategy applies to this operation BulkEditResource()
func (client CompartmentsClient) BulkEditResource(ctx context.Context, request BulkEditResourceRequest) (response BulkEditResourceResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}

	if !(request.OpcRetryToken != nil && *request.OpcRetryToken != "") {
		request.OpcRetryToken = common.String(common.RetryToken())
	}

	ociResponse, err = common.Retry(ctx, request, client.bulkEditResource, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = BulkEditResourceResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = BulkEditResourceResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(BulkEditResourceResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into BulkEditResourceResponse")
	}
	return
}

// bulkEditResource implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) bulkEditResource(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/compartments/{compartmentId}/actions/bulkMoveResources", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(BulkEditResourceRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response BulkEditResourceResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "BulkEditResource", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// CreateCompartment Creates a new compartment in the specified compartment.
// Specify the parent compartment's OCID as the compartment ID in the request object. Remember that the tenancy
// is simply the root compartment. For information about OCIDs, see
// Resource Identifiers (https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm).
// You must also specify a *name* for the compartment, which must be unique across all compartments in
// your tenancy. You can use this name or the OCID when writing policies that apply
// to the compartment. For more information about policies, see
// How Policies Work (https://docs.cloud.oracle.com/Content/Identity/policieshow/how-policies-work.htm).
// You must also specify a *description* for the compartment (although it can be an empty string). It does
// not have to be unique, and you can change it anytime with
// UpdateCompartment.
// After you send your request, the new object's `lifecycleState` will temporarily be CREATING. Before using the
// object, first make sure its `lifecycleState` has changed to ACTIVE.
// A default retry strategy applies to this operation CreateCompartment()
func (client CompartmentsClient) CreateCompartment(ctx context.Context, request CreateCompartmentRequest) (response CreateCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}

	if !(request.OpcRetryToken != nil && *request.OpcRetryToken != "") {
		request.OpcRetryToken = common.String(common.RetryToken())
	}

	ociResponse, err = common.Retry(ctx, request, client.createCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = CreateCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = CreateCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(CreateCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into CreateCompartmentResponse")
	}
	return
}

// createCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) createCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/compartments", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(CreateCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response CreateCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "CreateCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// CreateManagedCompartment Creates a new managed compartment in your tenancy.
// **Important:** Managed Compartments can only be created by whitelisted service callers.
// You must specify tenancy's OCID as the value for the compartment ID (remember that the tenancy is simply the root compartment).
// You also must specify compartment name, description, service name and service entitlement id in the request object.
// After you send your request, the new object's `lifecycleState` will temporarily be CREATING. Before using the
// object, first make sure its `lifecycleState` has changed to ACTIVE.
// A default retry strategy applies to this operation CreateManagedCompartment()
func (client CompartmentsClient) CreateManagedCompartment(ctx context.Context, request CreateManagedCompartmentRequest) (response CreateManagedCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}

	if !(request.OpcRetryToken != nil && *request.OpcRetryToken != "") {
		request.OpcRetryToken = common.String(common.RetryToken())
	}

	ociResponse, err = common.Retry(ctx, request, client.createManagedCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = CreateManagedCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = CreateManagedCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(CreateManagedCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into CreateManagedCompartmentResponse")
	}
	return
}

// createManagedCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) createManagedCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/managedCompartments", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(CreateManagedCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response CreateManagedCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "CreateManagedCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// DeleteCompartment Deletes the specified compartment. The compartment must be empty.
// A default retry strategy applies to this operation DeleteCompartment()
func (client CompartmentsClient) DeleteCompartment(ctx context.Context, request DeleteCompartmentRequest) (response DeleteCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.deleteCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = DeleteCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = DeleteCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(DeleteCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into DeleteCompartmentResponse")
	}
	return
}

// deleteCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) deleteCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodDelete, "/compartments/{compartmentId}", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(DeleteCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response DeleteCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "DeleteCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// GetCompartment Gets the specified compartment's information.
// This operation does not return a list of all the resources inside the compartment. There is no single
// API operation that does that. Compartments can contain multiple types of resources (instances, block
// storage volumes, etc.). To find out what's in a compartment, you must call the "List" operation for
// each resource type and specify the compartment's OCID as a query parameter in the request. For example,
// call the ListInstances operation in the Cloud Compute
// Service or the ListVolumes operation in Cloud Block Storage.
// A default retry strategy applies to this operation GetCompartment()
func (client CompartmentsClient) GetCompartment(ctx context.Context, request GetCompartmentRequest) (response GetCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.getCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = GetCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = GetCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(GetCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into GetCompartmentResponse")
	}
	return
}

// getCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) getCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/compartments/{compartmentId}", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(GetCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response GetCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "GetCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// GetTenancy Get the specified tenancy's information.
// A default retry strategy applies to this operation GetTenancy()
func (client CompartmentsClient) GetTenancy(ctx context.Context, request GetTenancyRequest) (response GetTenancyResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.getTenancy, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = GetTenancyResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = GetTenancyResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(GetTenancyResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into GetTenancyResponse")
	}
	return
}

// getTenancy implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) getTenancy(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/tenancies/{tenancyId}", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(GetTenancyRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response GetTenancyResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "GetTenancy", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// GetWorkRequest Gets details on a specified work request. The workRequestID is returned in the opc-work-request-id header
// for any asynchronous operation in the compartment service.
// A default retry strategy applies to this operation GetWorkRequest()
func (client CompartmentsClient) GetWorkRequest(ctx context.Context, request GetWorkRequestRequest) (response GetWorkRequestResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.getWorkRequest, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = GetWorkRequestResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = GetWorkRequestResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(GetWorkRequestResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into GetWorkRequestResponse")
	}
	return
}

// getWorkRequest implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) getWorkRequest(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/workRequests/{workRequestId}", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(GetWorkRequestRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response GetWorkRequestResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "GetWorkRequest", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListAvailabilityDomains Lists the availability domains in your tenancy. Specify the OCID of either the tenancy or another
// of your compartments as the value for the compartment ID (remember that the tenancy is simply the root compartment).
// See Where to Get the Tenancy's OCID and User's OCID (https://docs.cloud.oracle.com/Content/API/Concepts/apisigningkey.htm#five).
// Note that the order of the results returned can change if availability domains are added or removed; therefore, do not
// create a dependency on the list order.
// A default retry strategy applies to this operation ListAvailabilityDomains()
func (client CompartmentsClient) ListAvailabilityDomains(ctx context.Context, request ListAvailabilityDomainsRequest) (response ListAvailabilityDomainsResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listAvailabilityDomains, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListAvailabilityDomainsResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListAvailabilityDomainsResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListAvailabilityDomainsResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListAvailabilityDomainsResponse")
	}
	return
}

// listAvailabilityDomains implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listAvailabilityDomains(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/availabilityDomains", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListAvailabilityDomainsRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListAvailabilityDomainsResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListAvailabilityDomains", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListBulkActionResourceTypes Lists the resource-types supported by compartment bulk actions. Use this API to help you provide the correct
// resource-type information to the BulkDeleteResources
// and BulkMoveResources operations. The returned list of
// resource-types provides the appropriate resource-type names to use with the bulk action operations along with
// the type of identifying information you'll need to provide for each resource-type. Most resource-types just
// require an OCID (https://docs.cloud.oracle.com/Content/General/Concepts/identifiers.htm) to identify a specific resource, but some resource-types,
// such as buckets, require you to provide other identifying information.
// A default retry strategy applies to this operation ListBulkActionResourceTypes()
func (client CompartmentsClient) ListBulkActionResourceTypes(ctx context.Context, request ListBulkActionResourceTypesRequest) (response ListBulkActionResourceTypesResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listBulkActionResourceTypes, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListBulkActionResourceTypesResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListBulkActionResourceTypesResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListBulkActionResourceTypesResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListBulkActionResourceTypesResponse")
	}
	return
}

// listBulkActionResourceTypes implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listBulkActionResourceTypes(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/compartments/bulkActionResourceTypes", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListBulkActionResourceTypesRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListBulkActionResourceTypesResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListBulkActionResourceTypes", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListCompartments Lists the compartments in a specified compartment. The members of the list
// returned depends on the values set for several parameters.
// With the exception of the tenancy (root compartment), the ListCompartments operation
// returns only the first-level child compartments in the parent compartment specified in
// `compartmentId`. The list does not include any subcompartments of the child
// compartments (grandchildren).
// The parameter `accessLevel` specifies whether to return only those compartments for which the
// requestor has INSPECT permissions on at least one resource directly
// or indirectly (the resource can be in a subcompartment).
// The parameter `compartmentIdInSubtree` applies only when you perform ListCompartments on the
// tenancy (root compartment). When set to true, the entire hierarchy of compartments can be returned.
// To get a full list of all compartments and subcompartments in the tenancy (root compartment),
// set the parameter `compartmentIdInSubtree` to true and `accessLevel` to ANY.
// See Where to Get the Tenancy's OCID and User's OCID (https://docs.cloud.oracle.com/Content/API/Concepts/apisigningkey.htm#five).
// A default retry strategy applies to this operation ListCompartments()
func (client CompartmentsClient) ListCompartments(ctx context.Context, request ListCompartmentsRequest) (response ListCompartmentsResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listCompartments, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListCompartmentsResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListCompartmentsResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListCompartmentsResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListCompartmentsResponse")
	}
	return
}

// listCompartments implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listCompartments(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/compartments", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListCompartmentsRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListCompartmentsResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListCompartments", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListFaultDomains Lists the Fault Domains in your tenancy. Specify the OCID of either the tenancy or another
// of your compartments as the value for the compartment ID (remember that the tenancy is simply the root compartment).
// See Where to Get the Tenancy's OCID and User's OCID (https://docs.cloud.oracle.com/Content/API/Concepts/apisigningkey.htm#five).
// A default retry strategy applies to this operation ListFaultDomains()
func (client CompartmentsClient) ListFaultDomains(ctx context.Context, request ListFaultDomainsRequest) (response ListFaultDomainsResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listFaultDomains, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListFaultDomainsResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListFaultDomainsResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListFaultDomainsResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListFaultDomainsResponse")
	}
	return
}

// listFaultDomains implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listFaultDomains(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/faultDomains", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListFaultDomainsRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListFaultDomainsResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListFaultDomains", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListManagedCompartments List the managed compartments under the specific tenancy and specific service name. You must specify tenancy's OCID
// and service name as the value
// A default retry strategy applies to this operation ListManagedCompartments()
func (client CompartmentsClient) ListManagedCompartments(ctx context.Context, request ListManagedCompartmentsRequest) (response ListManagedCompartmentsResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listManagedCompartments, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListManagedCompartmentsResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListManagedCompartmentsResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListManagedCompartmentsResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListManagedCompartmentsResponse")
	}
	return
}

// listManagedCompartments implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listManagedCompartments(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/managedCompartments", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListManagedCompartmentsRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListManagedCompartmentsResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListManagedCompartments", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListTenancies Get the specified tenancy's information with BareMetal service entitlement.
// A default retry strategy applies to this operation ListTenancies()
func (client CompartmentsClient) ListTenancies(ctx context.Context, request ListTenanciesRequest) (response ListTenanciesResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listTenancies, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListTenanciesResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListTenanciesResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListTenanciesResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListTenanciesResponse")
	}
	return
}

// listTenancies implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listTenancies(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/tenancies", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListTenanciesRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListTenanciesResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListTenancies", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListTenancyCompartmentTree List compartment summary by tenant Id
// A default retry strategy applies to this operation ListTenancyCompartmentTree()
func (client CompartmentsClient) ListTenancyCompartmentTree(ctx context.Context, request ListTenancyCompartmentTreeRequest) (response ListTenancyCompartmentTreeResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listTenancyCompartmentTree, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListTenancyCompartmentTreeResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListTenancyCompartmentTreeResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListTenancyCompartmentTreeResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListTenancyCompartmentTreeResponse")
	}
	return
}

// listTenancyCompartmentTree implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listTenancyCompartmentTree(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/tenancies/{tenancyId}/compartmentTree", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListTenancyCompartmentTreeRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListTenancyCompartmentTreeResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListTenancyCompartmentTree", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// ListWorkRequests Lists the work requests in compartment.
// A default retry strategy applies to this operation ListWorkRequests()
func (client CompartmentsClient) ListWorkRequests(ctx context.Context, request ListWorkRequestsRequest) (response ListWorkRequestsResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.listWorkRequests, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = ListWorkRequestsResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = ListWorkRequestsResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(ListWorkRequestsResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into ListWorkRequestsResponse")
	}
	return
}

// listWorkRequests implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) listWorkRequests(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodGet, "/workRequests", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(ListWorkRequestsRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response ListWorkRequestsResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "ListWorkRequests", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// MoveCompartment Move the compartment to a different parent compartment in the same tenancy. When you move a
// compartment, all its contents (subcompartments and resources) are moved with it. Note that
// the `CompartmentId` that you specify in the path is the compartment that you want to move.
// **IMPORTANT**: After you move a compartment to a new parent compartment, the access policies of
// the new parent take effect and the policies of the previous parent no longer apply. Ensure that you
// are aware of the implications for the compartment contents before you move it. For more
// information, see Moving a Compartment (https://docs.cloud.oracle.com/Content/Identity/compartments/managingcompartments.htm#MoveCompartment).
// A default retry strategy applies to this operation MoveCompartment()
func (client CompartmentsClient) MoveCompartment(ctx context.Context, request MoveCompartmentRequest) (response MoveCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}

	if !(request.OpcRetryToken != nil && *request.OpcRetryToken != "") {
		request.OpcRetryToken = common.String(common.RetryToken())
	}

	ociResponse, err = common.Retry(ctx, request, client.moveCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = MoveCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = MoveCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(MoveCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into MoveCompartmentResponse")
	}
	return
}

// moveCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) moveCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/compartments/{compartmentId}/actions/moveCompartment", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(MoveCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response MoveCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "MoveCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// RecoverCompartment Recover the compartment from DELETED state to ACTIVE state.
// A default retry strategy applies to this operation RecoverCompartment()
func (client CompartmentsClient) RecoverCompartment(ctx context.Context, request RecoverCompartmentRequest) (response RecoverCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.recoverCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = RecoverCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = RecoverCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(RecoverCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into RecoverCompartmentResponse")
	}
	return
}

// recoverCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) recoverCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/compartments/{compartmentId}/actions/recoverCompartment", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(RecoverCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response RecoverCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "RecoverCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// SetGovernanceFromChild Set the governing tenancy of child tenancy to be the parent (request originated from child)
// A default retry strategy applies to this operation SetGovernanceFromChild()
func (client CompartmentsClient) SetGovernanceFromChild(ctx context.Context, request SetGovernanceFromChildRequest) (response SetGovernanceFromChildResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.setGovernanceFromChild, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = SetGovernanceFromChildResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = SetGovernanceFromChildResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(SetGovernanceFromChildResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into SetGovernanceFromChildResponse")
	}
	return
}

// setGovernanceFromChild implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) setGovernanceFromChild(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/tenancies/{childTenancyId}/actions/setGovernanceFromChild", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(SetGovernanceFromChildRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response SetGovernanceFromChildResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "SetGovernanceFromChild", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// SetGovernanceFromParent Set the governing tenancy of child tenancy to be the parent (request originated from parent)
// A default retry strategy applies to this operation SetGovernanceFromParent()
func (client CompartmentsClient) SetGovernanceFromParent(ctx context.Context, request SetGovernanceFromParentRequest) (response SetGovernanceFromParentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.setGovernanceFromParent, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = SetGovernanceFromParentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = SetGovernanceFromParentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(SetGovernanceFromParentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into SetGovernanceFromParentResponse")
	}
	return
}

// setGovernanceFromParent implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) setGovernanceFromParent(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/tenancies/{childTenancyId}/actions/setGovernanceFromParent", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(SetGovernanceFromParentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response SetGovernanceFromParentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "SetGovernanceFromParent", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// UnsetGovernanceFromChild Remove the governing tenancy of child tenancy (request originated from child)
// A default retry strategy applies to this operation UnsetGovernanceFromChild()
func (client CompartmentsClient) UnsetGovernanceFromChild(ctx context.Context, request UnsetGovernanceFromChildRequest) (response UnsetGovernanceFromChildResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.unsetGovernanceFromChild, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = UnsetGovernanceFromChildResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = UnsetGovernanceFromChildResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(UnsetGovernanceFromChildResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into UnsetGovernanceFromChildResponse")
	}
	return
}

// unsetGovernanceFromChild implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) unsetGovernanceFromChild(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/tenancies/{childTenancyId}/actions/unsetGovernanceFromChild", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(UnsetGovernanceFromChildRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response UnsetGovernanceFromChildResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "UnsetGovernanceFromChild", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// UnsetGovernanceFromParent Remove the governing tenancy of child tenancy (request originated from parent)
// A default retry strategy applies to this operation UnsetGovernanceFromParent()
func (client CompartmentsClient) UnsetGovernanceFromParent(ctx context.Context, request UnsetGovernanceFromParentRequest) (response UnsetGovernanceFromParentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.unsetGovernanceFromParent, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = UnsetGovernanceFromParentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = UnsetGovernanceFromParentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(UnsetGovernanceFromParentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into UnsetGovernanceFromParentResponse")
	}
	return
}

// unsetGovernanceFromParent implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) unsetGovernanceFromParent(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPost, "/tenancies/{childTenancyId}/actions/unsetGovernanceFromParent", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(UnsetGovernanceFromParentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response UnsetGovernanceFromParentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "UnsetGovernanceFromParent", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}

// UpdateCompartment Updates the specified compartment's description or name. You can't update the root compartment.
// A default retry strategy applies to this operation UpdateCompartment()
func (client CompartmentsClient) UpdateCompartment(ctx context.Context, request UpdateCompartmentRequest) (response UpdateCompartmentResponse, err error) {
	var ociResponse common.OCIResponse
	policy := common.DefaultRetryPolicy()
	if client.RetryPolicy() != nil {
		policy = *client.RetryPolicy()
	}
	if request.RetryPolicy() != nil {
		policy = *request.RetryPolicy()
	}
	ociResponse, err = common.Retry(ctx, request, client.updateCompartment, policy)
	if err != nil {
		if ociResponse != nil {
			if httpResponse := ociResponse.HTTPResponse(); httpResponse != nil {
				opcRequestId := httpResponse.Header.Get("opc-request-id")
				response = UpdateCompartmentResponse{RawResponse: httpResponse, OpcRequestId: &opcRequestId}
			} else {
				response = UpdateCompartmentResponse{}
			}
		}
		return
	}
	if convertedResponse, ok := ociResponse.(UpdateCompartmentResponse); ok {
		response = convertedResponse
	} else {
		err = fmt.Errorf("failed to convert OCIResponse into UpdateCompartmentResponse")
	}
	return
}

// updateCompartment implements the OCIOperation interface (enables retrying operations)
func (client CompartmentsClient) updateCompartment(ctx context.Context, request common.OCIRequest, binaryReqBody *common.OCIReadSeekCloser, extraHeaders map[string]string) (common.OCIResponse, error) {

	httpRequest, err := request.HTTPRequest(http.MethodPut, "/compartments/{compartmentId}", binaryReqBody, extraHeaders)
	if err != nil {
		return nil, err
	}

	host := client.Host
	request.(UpdateCompartmentRequest).ReplaceMandatoryParamInPath(&client.BaseClient, client.requiredParamsInEndpoint)
	common.UpdateEndpointTemplateForOptions(&client.BaseClient)
	common.SetMissingTemplateParams(&client.BaseClient)
	defer func() {
		client.Host = host
	}()

	var response UpdateCompartmentResponse
	var httpResponse *http.Response
	httpResponse, err = client.Call(ctx, &httpRequest)
	defer common.CloseBodyIfValid(httpResponse)
	response.RawResponse = httpResponse
	if err != nil {
		apiReferenceLink := ""
		err = common.PostProcessServiceError(err, "Compartments", "UpdateCompartment", apiReferenceLink)
		return response, err
	}

	err = common.UnmarshalResponse(httpResponse, &response)
	return response, err
}
