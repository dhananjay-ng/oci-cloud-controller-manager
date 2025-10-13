package client

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/pkg/errors"
)

// CertificateManagerInterface create an interface to manage certificate related operations
type CertificateManagerInterface interface {
	GetValidCertificate(ctx context.Context, id string) (*certificatesmanagement.Certificate, error)
}

// GetValidCertificate reads the certificate id provided using oci cert service and performs minor validation
// This checks if the certificate is active and expired
// if the certificate is not valid return appropriate error message
func (c *client) GetValidCertificate(ctx context.Context, id string) (*certificatesmanagement.Certificate, error) {
	if !c.rateLimiter.Reader.TryAccept() {
		return nil, RateLimitError(false, "GetValidCertificate")
	}

	req := certificatesmanagement.GetCertificateRequest{CertificateId: common.String(id)}
	if &c.certificatesManagementClient == nil {
		return nil, fmt.Errorf("certificates management client not initialized")
	}
	resp, err := c.certificatesManagementClient.GetCertificate(ctx, req)
	c.logger.Debugf("Retrieved Certificate response %+v with reqeust %+v", resp, req)
	incRequestCounter(err, getVerb, certificateResource)
	if err != nil {
		c.logger.Errorf("Error retrieving Certificate %+v: %s", id, errors.WithStack(err))
		return nil, errors.New(fmt.Sprintf("unexpected error in reading certificate: %s", err.Error()))
	}
	if &resp == nil || &resp.Certificate == nil {
		return nil, errors.New(fmt.Sprintf("unexpected nil response from GetValidCertificate"))
	}
	if resp.Certificate.LifecycleState != certificatesmanagement.CertificateLifecycleStateActive ||
		(resp.Certificate.TimeOfDeletion != nil && resp.Certificate.TimeOfDeletion.Before(time.Now())) {
		return nil, errors.New(fmt.Sprintf("No valid certificate found with id %s", id))
	}
	return &resp.Certificate, nil
}
