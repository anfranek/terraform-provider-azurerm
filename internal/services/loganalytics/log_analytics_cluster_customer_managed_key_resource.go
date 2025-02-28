package loganalytics

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-sdk/resource-manager/operationalinsights/2020-08-01/clusters"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	keyVaultParse "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/parse"
	keyVaultValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/keyvault/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/loganalytics/migration"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceLogAnalyticsClusterCustomerManagedKey() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceLogAnalyticsClusterCustomerManagedKeyCreate,
		Read:   resourceLogAnalyticsClusterCustomerManagedKeyRead,
		Update: resourceLogAnalyticsClusterCustomerManagedKeyUpdate,
		Delete: resourceLogAnalyticsClusterCustomerManagedKeyDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(6 * time.Hour),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(6 * time.Hour),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := clusters.ParseClusterID(id)
			return err
		}),

		StateUpgraders: pluginsdk.StateUpgrades(map[int]pluginsdk.StateUpgrade{
			0: migration.ClusterCustomerManagedKeyV0ToV1{},
		}),
		SchemaVersion: 1,

		Schema: map[string]*pluginsdk.Schema{
			"log_analytics_cluster_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: clusters.ValidateClusterID,
			},

			"key_vault_key_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: keyVaultValidate.NestedItemIdWithOptionalVersion,
			},
		},
	}
}

func resourceLogAnalyticsClusterCustomerManagedKeyCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.ClusterClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := clusters.ParseClusterID(d.Get("log_analytics_cluster_id").(string))
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return fmt.Errorf("%s was not found", *id)
		}

		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}
	if model := resp.Model; model != nil {
		if props := model.Properties; props != nil && props.KeyVaultProperties != nil {
			keyProps := *props.KeyVaultProperties
			if keyProps.KeyName != nil && *keyProps.KeyName != "" {
				return tf.ImportAsExistsError("azurerm_log_analytics_cluster_customer_managed_key", id.ID())
			}
		}
	}

	keyId, err := keyVaultParse.ParseOptionallyVersionedNestedItemID(d.Get("key_vault_key_id").(string))
	if err != nil {
		return fmt.Errorf("parsing Key Vault Key ID: %+v", err)
	}

	clusterPatch := clusters.ClusterPatch{
		Properties: &clusters.ClusterPatchProperties{
			KeyVaultProperties: &clusters.KeyVaultProperties{
				KeyVaultUri: utils.String(keyId.KeyVaultBaseUrl),
				KeyName:     utils.String(keyId.Name),
				KeyVersion:  utils.String(keyId.Version),
			},
		},
	}

	if _, err := client.Update(ctx, *id, clusterPatch); err != nil {
		return fmt.Errorf("updating Customer Managed Key for %s: %+v", *id, err)
	}

	updateWait, err := logAnalyticsClusterWaitForState(ctx, client, *id)
	if err != nil {
		return err
	}
	if _, err := updateWait.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for %s to finish adding Customer Managed Key: %+v", *id, err)
	}

	d.SetId(id.ID())
	return resourceLogAnalyticsClusterCustomerManagedKeyRead(d, meta)
}

func resourceLogAnalyticsClusterCustomerManagedKeyUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.ClusterClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := clusters.ParseClusterID(d.Id())
	if err != nil {
		return err
	}

	keyId, err := keyVaultParse.ParseOptionallyVersionedNestedItemID(d.Get("key_vault_key_id").(string))
	if err != nil {
		return fmt.Errorf("parsing Key Vault Key ID: %+v", err)
	}

	clusterPatch := clusters.ClusterPatch{
		Properties: &clusters.ClusterPatchProperties{
			KeyVaultProperties: &clusters.KeyVaultProperties{
				KeyVaultUri: utils.String(keyId.KeyVaultBaseUrl),
				KeyName:     utils.String(keyId.Name),
				KeyVersion:  utils.String(keyId.Version),
			},
		},
	}

	if _, err := client.Update(ctx, *id, clusterPatch); err != nil {
		return fmt.Errorf("updating Customer Managed Key for %s: %+v", *id, err)
	}

	updateWait, err := logAnalyticsClusterWaitForState(ctx, client, *id)
	if err != nil {
		return err
	}
	if _, err := updateWait.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for update of Customer Managed Key for %s: %+v", *id, err)
	}

	return resourceLogAnalyticsClusterCustomerManagedKeyRead(d, meta)
}

func resourceLogAnalyticsClusterCustomerManagedKeyRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.ClusterClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := clusters.ParseClusterID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] %s does not exist - removing from state", *id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	keyVaultKeyId := ""
	if model := resp.Model; model != nil {
		if props := model.Properties; props != nil {
			if kvProps := props.KeyVaultProperties; kvProps != nil {
				var keyVaultUri, keyName, keyVersion string
				if kvProps.KeyVaultUri != nil && *kvProps.KeyVaultUri != "" {
					keyVaultUri = *kvProps.KeyVaultUri
				} else {
					return fmt.Errorf("empty value returned for Key Vault URI")
				}
				if kvProps.KeyName != nil && *kvProps.KeyName != "" {
					keyName = *kvProps.KeyName
				} else {
					return fmt.Errorf("empty value returned for Key Vault Key Name")
				}
				if kvProps.KeyVersion != nil {
					keyVersion = *kvProps.KeyVersion
				}
				keyId, err := keyVaultParse.NewNestedItemID(keyVaultUri, "keys", keyName, keyVersion)
				if err != nil {
					return err
				}
				keyVaultKeyId = keyId.ID()
			}
		}
	}

	if keyVaultKeyId == "" {
		log.Printf("[DEBUG] %s has no Customer Managed Key - removing from state", *id)
		return nil
	}

	d.Set("log_analytics_cluster_id", d.Id())
	d.Set("key_vault_key_id", keyVaultKeyId)

	return nil
}

func resourceLogAnalyticsClusterCustomerManagedKeyDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.ClusterClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := clusters.ParseClusterID(d.Id())
	if err != nil {
		return err
	}

	clusterPatch := clusters.ClusterPatch{
		Properties: &clusters.ClusterPatchProperties{
			KeyVaultProperties: &clusters.KeyVaultProperties{
				KeyVaultUri: nil,
				KeyName:     nil,
				KeyVersion:  nil,
			},
		},
	}

	if _, err = client.Update(ctx, *id, clusterPatch); err != nil {
		return fmt.Errorf("removing Customer Managed Key from %s: %+v", *id, err)
	}

	deleteWait, err := logAnalyticsClusterWaitForState(ctx, client, *id)
	if err != nil {
		return err
	}
	if _, err := deleteWait.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for removal of Customer Managed Key from %s: %+v", *id, err)
	}

	return nil
}
