package framework

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/common"
)

func (f *Framework) GetOrCreateCertificate() string {
	// Use the authority Id provided in parameter,
	// if the authority id is not present scan the compartment to find the authority
	// Use the authority to create a certificate
	if f.CertAuthorityOCID == "" {
		f.GetOrCreateAuthority()
	}
	if len(f.CertAuthorityOCID) <= 0 {
		Logf("No certificate authority found, Could not create authority id. Could not continue with certificate creation")
		return ""
	}
	// try to find a certificate that can be reused for E2E test
	listRequest := certificatesmanagement.ListCertificatesRequest{
		CompartmentId:  common.String(f.Compartment1),
		LifecycleState: certificatesmanagement.ListCertificatesLifecycleStateActive,
	}
	if ls, err := f.certificateClient.ListCertificates(context.Background(), listRequest); err != nil {
		Logf(fmt.Sprintf("no exisitng certificate found due to error %s", err))
	} else if len(ls.Items) > 0 {
		return *(ls.Items[0].Id)
	}
	// create a new certificate if not present
	Logf(fmt.Sprintf("no exisitng certificate found continue to create a new certificate"))
	validity := certificatesmanagement.Validity{TimeOfValidityNotBefore: &common.SDKTime{Time: time.Now().UTC()},
		TimeOfValidityNotAfter: &common.SDKTime{Time: time.Now().UTC().Add(time.Duration(90) * 24 * time.Hour)}}
	req := certificatesmanagement.CreateCertificateRequest{
		CreateCertificateDetails: certificatesmanagement.CreateCertificateDetails{
			Name:          common.String("test-certificate" + UniqueID()),
			CompartmentId: common.String(f.Compartment1),
			CertificateConfig: &certificatesmanagement.CreateCertificateIssuedByInternalCaConfigDetails{
				IssuerCertificateAuthorityId: common.String(f.CertAuthorityOCID),
				Validity:                     &validity,
				Subject: &certificatesmanagement.CertificateSubject{
					CommonName: common.String("Test Certificate"),
				},
				CertificateProfileType: certificatesmanagement.CertificateProfileTypeTlsClient,
			},
			Description:      common.String("nil"),
			CertificateRules: nil,
		},
	}
	if certificate, err := f.certificateClient.CreateCertificate(context.Background(), req); err != nil {
		Logf(fmt.Sprintf("Unable to create certificate due to error %s with message %s", err, err.Error()))
		return ""
	} else {
		// Wait for certificate to be available
		f.waitForCertCreation(*certificate.Id)
		return *certificate.Id
	}
}

func (f *Framework) GetCA() (string, certificatesmanagement.CertificateAuthorityLifecycleStateEnum) {
	if f.CertAuthorityOCID == "" {
		return "", ""
	}
	if wrResponse, err := f.certificateClient.GetCertificateAuthority(context.Background(), certificatesmanagement.GetCertificateAuthorityRequest{
		CertificateAuthorityId: common.String(f.CertAuthorityOCID),
	}); err != nil {
		return "", ""
	} else {
		return *wrResponse.Id, wrResponse.LifecycleState
	}
}

func (f *Framework) GetOrCreateAuthority() {
	req := certificatesmanagement.ListCertificateAuthoritiesRequest{
		CompartmentId:  common.String(f.Compartment1),
		LifecycleState: certificatesmanagement.ListCertificateAuthoritiesLifecycleStateActive,
	}
	if resp, err := f.certificateClient.ListCertificateAuthorities(context.Background(), req); err != nil {
		Logf("Error listing authorities: %v", err)
	} else if len(resp.Items) < 1 {
		// Create a certificate authority Id
		Logf("No Active certificate authorities found. Attempting to create one.")
		// Create Authority
		authReq := &certificatesmanagement.CreateCertificateAuthorityRequest{
			CreateCertificateAuthorityDetails: certificatesmanagement.CreateCertificateAuthorityDetails{
				CompartmentId: common.String(f.Compartment1),
				Name:          common.String("test-ca" + UniqueID()),
				KmsKeyId:      common.String(f.KMSKeyOCIDForCA),
				Description:   common.String("nil"),
				CertificateAuthorityConfig: certificatesmanagement.CreateRootCaByGeneratingInternallyConfigDetails{
					Subject: &certificatesmanagement.CertificateSubject{
						CommonName: common.String("Test-CA"),
					},
				},
			},
		}
		if authority, err := f.certificateClient.CreateCertificateAuthority(context.Background(), *authReq); err != nil {
			Logf("Error creating authority: %v", err)
		} else {
			// Wait for the certificate authority to come
			f.waitForCACreation(*authority.Id)
			f.CertAuthorityOCID = *authority.Id
		}
	} else {
		// Assign available certificate id
		Logf(fmt.Sprintf("Found existing authority %+v", resp.Items))
		f.CertAuthorityOCID = *(resp.Items[0].Id)
	}
}

func (f *Framework) DeleteCertificate() {
	req := certificatesmanagement.ScheduleCertificateDeletionRequest{
		CertificateId: &f.CertAuthorityOCID,
		ScheduleCertificateDeletionDetails: certificatesmanagement.ScheduleCertificateDeletionDetails{
			TimeOfDeletion: &common.SDKTime{Time: time.Now().UTC().Add(time.Duration(8) * 24 * time.Hour)},
		}}
	if _, err := f.certificateClient.ScheduleCertificateDeletion(context.Background(), req); err != nil {
		Logf("Error deleting certificate: %v with message %s", err, err.Error())
	}
}

func (f *Framework) waitForCACreation(id string) {
	// get the work request details
	timeout := 10 * time.Minute
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(Poll) {
		wrResponse, _ := f.certificateClient.GetCertificateAuthority(context.Background(), certificatesmanagement.GetCertificateAuthorityRequest{
			CertificateAuthorityId: common.String(id),
		})
		Expect(wrResponse.LifecycleState).NotTo(Equal(certificatesmanagement.CertificateAuthorityLifecycleStateFailed))
		Expect(wrResponse.LifecycleState).NotTo(Equal(certificatesmanagement.CertificateAuthorityLifecycleStateDeleted))
		if wrResponse.LifecycleState == certificatesmanagement.CertificateAuthorityLifecycleStateActive {
			return
		}
		Logf("Waiting for resource to come to active state. Current state: %s", string(wrResponse.LifecycleState))
	}
	Failf("Timeout waiting for Resource '%s' workRequest to SUCCEED\n", id)
}

func (f *Framework) waitForCertCreation(id string) {
	// get the work request details
	timeout := 10 * time.Minute
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(Poll) {
		wrResponse, _ := f.certificateClient.GetCertificate(context.Background(), certificatesmanagement.GetCertificateRequest{
			CertificateId: common.String(id),
		})
		Expect(wrResponse.LifecycleState).NotTo(Equal(certificatesmanagement.CertificateLifecycleStateFailed))
		Expect(wrResponse.LifecycleState).NotTo(Equal(certificatesmanagement.CertificateLifecycleStateDeleted))
		if wrResponse.LifecycleState == certificatesmanagement.CertificateLifecycleStateActive {
			return
		}
		Logf("Waiting for resource to come to active state status: %s", string(wrResponse.LifecycleState))
	}
	Failf("Timeout waiting for Resource '%s' workRequest to SUCCEED\n", id)
}

func (f *Framework) DeleteCertificateAuthority() {
	req := certificatesmanagement.ScheduleCertificateAuthorityDeletionRequest{
		CertificateAuthorityId: &f.CertAuthorityOCID,
		ScheduleCertificateAuthorityDeletionDetails: certificatesmanagement.ScheduleCertificateAuthorityDeletionDetails{
			TimeOfDeletion: &common.SDKTime{Time: time.Now().UTC().Add(time.Duration(8) * 24 * time.Hour)},
		},
	}
	if _, err := f.certificateClient.ScheduleCertificateAuthorityDeletion(context.Background(), req); err != nil {
		Logf("Error deleting certificate: %v", err)
	}
}

func (f *Framework) GetCertificateAssociations(certOCid string) ([]string, error) {
	// check cert OCID available
	if cert, err := f.certificateClient.GetCertificate(context.Background(), certificatesmanagement.GetCertificateRequest{
		CertificateId: common.String(certOCid),
	}); err != nil {
		Logf("Error reading certificate %s", err.Error())
		return make([]string, 0), err
	} else {
		if associations, e := f.certificateClient.ListAssociations(context.Background(), certificatesmanagement.ListAssociationsRequest{
			CertificatesResourceId: common.String(certOCid),
			CompartmentId:          cert.CompartmentId,
		}); e != nil {
			Logf("Error reading associations found in certificate %s", e.Error())
			return make([]string, 0), e
		} else if len(associations.Items) == 0 {
			Logf("No associations found in certificate %s", certOCid)
			return make([]string, 0), fmt.Errorf("no associations found for certificate '%s'", certOCid)
		} else {
			names := make([]string, len(associations.Items))
			for i, association := range associations.Items {
				names[i] = *association.Name
			}
			return names, nil
		}
	}
}
