package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudflare/cloudflare-go"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudflarev1alpha1 "github.com/shersch/cloudflare-operator/api/v1alpha1"
)

type DNSRecordReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DNSRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling DNSRecord", "request", req.NamespacedName)

	var dnsRecord cloudflarev1alpha1.DNSRecord
	err := r.Get(ctx, req.NamespacedName, &dnsRecord)
	if err != nil {
		log.Error(err, "Failed to get DNSRecord", "request", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	api, err := r.setupCloudflareClient()
	if err != nil {
		log.Error(err, "Failed to setup Cloudflare client")
		return ctrl.Result{}, err
	}

	zoneID, err := api.ZoneIDByName(dnsRecord.Spec.Zone)
	if err != nil {
		log.Error(err, "Failed to fetch zone ID", "zone", dnsRecord.Spec.Zone)
		return ctrl.Result{}, err
	}

	resourceContainer := &cloudflare.ResourceContainer{Identifier: zoneID}
	log.Info("Resource container prepared", "zoneID", zoneID)

	if !dnsRecord.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(dnsRecord.ObjectMeta.Finalizers, "dnsrecord.finalizers.cloudflare.local.dev") {
			log.Info("Deleting external DNS record", "name", dnsRecord.Name)
			if err := r.deleteExternalDNSRecord(ctx, log, api, &dnsRecord, resourceContainer); err != nil {
				log.Error(err, "Failed to delete external DNS record", "name", dnsRecord.Name)
				return ctrl.Result{}, err
			}
			dnsRecord.ObjectMeta.Finalizers = removeString(dnsRecord.ObjectMeta.Finalizers, "dnsrecord.finalizers.cloudflare.local.dev")
			if err := r.Update(ctx, &dnsRecord); err != nil {
				log.Error(err, "Failed to update DNSRecord after removing finalizer", "name", dnsRecord.Name)
				return ctrl.Result{}, err
			}
			log.Info("Finalizer removed and DNSRecord updated after deletion", "name", dnsRecord.Name)
		}
		return ctrl.Result{}, nil
	}

	listParams := cloudflare.ListDNSRecordsParams{
		Type: dnsRecord.Spec.Type,
		Name: dnsRecord.Spec.Name + "." + dnsRecord.Spec.Zone,
	}
	existingRecords, _, err := api.ListDNSRecords(ctx, resourceContainer, listParams)
	if err != nil {
		return ctrl.Result{}, err
	}

	var existingRecord *cloudflare.DNSRecord
	for _, record := range existingRecords {
		if record.Name == dnsRecord.Spec.Name+"."+dnsRecord.Spec.Zone && record.Type == dnsRecord.Spec.Type {
			existingRecord = &record
			break
		}
	}

	if existingRecord != nil {
		var proxied bool
		if dnsRecord.Spec.Proxied != nil {
			proxied = *dnsRecord.Spec.Proxied
		}
		log.Info("Checking for changes in DNS record", "current", existingRecord, "desired", dnsRecord)
		if existingRecord.Content != dnsRecord.Spec.Content || existingRecord.TTL != dnsRecord.Spec.TTL || (existingRecord.Proxied != nil && *existingRecord.Proxied != proxied) {
			updateParams := cloudflare.UpdateDNSRecordParams{
				ID:      existingRecord.ID,
				Content: dnsRecord.Spec.Content,
				TTL:     dnsRecord.Spec.TTL,
				Proxied: &proxied,
			}
			log.Info("Updating DNS record", "updateParams", updateParams)
			_, err := api.UpdateDNSRecord(ctx, resourceContainer, updateParams)
			if err != nil {
				log.Error(err, "Failed to update DNS record", "updateParams", updateParams)
				return ctrl.Result{}, err
			}
			log.Info("DNS record updated successfully", "recordID", existingRecord.ID)
		} else {
			log.Info("No changes needed for DNS record", "recordID", existingRecord.ID)
		}
	} else {
		var proxied bool
		if dnsRecord.Spec.Proxied != nil {
			proxied = *dnsRecord.Spec.Proxied
		}
		createParams := cloudflare.CreateDNSRecordParams{
			Type:    dnsRecord.Spec.Type,
			Name:    dnsRecord.Spec.Name,
			Content: dnsRecord.Spec.Content,
			TTL:     dnsRecord.Spec.TTL,
			Proxied: &proxied,
		}
		log.Info("Creating new DNS record", "params", createParams)
		_, err := api.CreateDNSRecord(ctx, resourceContainer, createParams)
		if err != nil {
			log.Error(err, "Failed to create DNS record", "params", createParams)
			return ctrl.Result{}, err
		}
		log.Info("DNS record created successfully", "params", createParams)
	}

	if err := r.Status().Update(ctx, &dnsRecord); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func containsString(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func removeString(slice []string, str string) []string {
	result := []string{}
	for _, v := range slice {
		if v != str {
			result = append(result, v)
		}
	}
	return result
}

func (r *DNSRecordReconciler) deleteExternalDNSRecord(ctx context.Context, log logr.Logger, api *cloudflare.API, dnsRecord *cloudflarev1alpha1.DNSRecord, resourceContainer *cloudflare.ResourceContainer) error {
	log.Info("Deleting external DNS record", "dnsRecord", dnsRecord.Name)

	listParams := cloudflare.ListDNSRecordsParams{
		Type: dnsRecord.Spec.Type,
		Name: dnsRecord.Spec.Name + "." + dnsRecord.Spec.Zone,
	}
	existingRecords, _, err := api.ListDNSRecords(ctx, resourceContainer, listParams)
	if err != nil {
		log.Error(err, "Failed to list DNS records", "zone", dnsRecord.Spec.Zone)
		return fmt.Errorf("failed to list DNS records for deletion: %w", err)
	}

	for _, record := range existingRecords {
		if record.Name == dnsRecord.Spec.Name+"."+dnsRecord.Spec.Zone && record.Type == dnsRecord.Spec.Type {
			if err := api.DeleteDNSRecord(ctx, resourceContainer, record.ID); err != nil {
				log.Error(err, "Failed to delete DNS record", "recordID", record.ID)
				return err
			}
			log.Info("DNS record deleted", "recordID", record.ID)
			break
		}
	}
	return nil
}

func (r *DNSRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudflarev1alpha1.DNSRecord{}).
		Complete(r)
}

func (r *DNSRecordReconciler) setupCloudflareClient() (*cloudflare.API, error) {
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, err
	}
	return api, nil
}
