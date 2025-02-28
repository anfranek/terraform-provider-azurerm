package compute_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
)

type SharedImageDataSource struct{}

func TestAccDataSourceAzureRMSharedImage_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_shared_image", "test")
	r := SharedImageDataSource{}
	data.DataSourceTest(t, []acceptance.TestStep{
		{
			Config: r.basic(data, ""),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).Key("tags.%").HasValue("0"),
			),
		},
	})
}

func TestAccDataSourceAzureRMSharedImage_basic_hyperVGeneration_V2(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_shared_image", "test")
	r := SharedImageDataSource{}
	data.DataSourceTest(t, []acceptance.TestStep{
		{
			Config: r.basic(data, "V2"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).Key("tags.%").HasValue("0"),
				check.That(data.ResourceName).Key("hyper_v_generation").HasValue("V2"),
			),
		},
	})
}

func TestAccDataSourceAzureRMSharedImage_complete(t *testing.T) {
	data := acceptance.BuildTestData(t, "data.azurerm_shared_image", "test")
	r := SharedImageDataSource{}
	data.DataSourceTest(t, []acceptance.TestStep{
		{
			Config: r.complete(data, "V1"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).Key("tags.%").HasValue("0"),
				check.That(data.ResourceName).Key("hyper_v_generation").HasValue("V1"),
			),
		},
	})
}

func (SharedImageDataSource) basic(data acceptance.TestData, hyperVGen string) string {
	return fmt.Sprintf(`
%s

data "azurerm_shared_image" "test" {
  name                = azurerm_shared_image.test.name
  gallery_name        = azurerm_shared_image.test.gallery_name
  resource_group_name = azurerm_shared_image.test.resource_group_name
}
`, SharedImageResource{}.basicWithHyperVGen(data, hyperVGen))
}

func (SharedImageDataSource) complete(data acceptance.TestData, hyperVGen string) string {
	return fmt.Sprintf(`
%s

data "azurerm_shared_image" "test" {
  name                = azurerm_shared_image.test.name
  gallery_name        = azurerm_shared_image.test.gallery_name
  resource_group_name = azurerm_shared_image.test.resource_group_name
}
`, SharedImageResource{}.completeWithHyperVGen(data, hyperVGen))
}
