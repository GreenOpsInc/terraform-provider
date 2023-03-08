package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"io"
	"net/http"
	"time"
)

type clientInfo struct {
	httpClient *http.Client
	address    string
	orgName    string
	token      string
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SERVICE_ADDRESS", ""),
			},
			"org": {
				Type:        schema.TypeString,
				Required:    false,
				DefaultFunc: schema.EnvDefaultFunc("ORG_NAME", "org"),
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SERVICE_TOKEN", ""),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"cluster": resourceItem(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	address := d.Get("address").(string)
	org := d.Get("org").(string)
	token := d.Get("token").(string)
	client := &http.Client{
		Timeout: time.Second * 180,
	}
	return clientInfo{
		httpClient: client,
		address:    address,
		orgName:    org,
		token:      token,
	}, nil
}

func resourceItem() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the cluster where the GreenOps agent should be installed",
				ForceNew:    true,
				//ValidateFunc: validateName,
			},
			"rotate": {
				Type:        schema.TypeBool,
				Required:    false,
				Description: "Set to true to rotate the apikey used by the agent",
				Default:     false,
			},
			"description": {
				Type:        schema.TypeString,
				Required:    false,
				Description: "Description or notes for the cluster",
			},
			"apikey": {
				Type:        schema.TypeString,
				Required:    false,
				Description: "API key that is defined when the GreenOps agent is created",
				Sensitive:   true,
			},
		},
		Create: resourceCreateItem,
		Read:   resourceReadItem,
		Update: resourceUpdateItem,
		Delete: resourceDeleteItem,
		Exists: resourceExistsItem,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceCreateItem(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(clientInfo)

	clusterName := d.Get("name").(string)
	request, err := http.NewRequest(
		"POST",
		apiClient.address+fmt.Sprintf("/api/cluster/%s/%s/apikeys/generate", apiClient.orgName, clusterName),
		bytes.NewBuffer([]byte{}),
	)
	resp, err := apiClient.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respContents, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	respContentsMap := make(map[string]string)
	json.Unmarshal(respContents, &respContentsMap)
	apiKey := respContentsMap["apiKey"]
	d.SetId(clusterName)
	err = d.Set("apikey", apiKey)
	if err != nil {
		return err
	}
	return nil
}

type GetApiKeysResponseItem struct {
	Name   string `json:"name"`
	ApiKey string `json:"apiKey"`
}

type GetApiKeysResponse = []GetApiKeysResponseItem

func resourceReadItem(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(clientInfo)

	clusterName := d.Get("name").(string)
	request, err := http.NewRequest(
		"GET",
		apiClient.address+fmt.Sprintf("/api/cluster/%s/apikeys/cluster", apiClient.orgName),
		bytes.NewBuffer([]byte{}),
	)
	resp, err := apiClient.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respContents, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	respContentsList := make(GetApiKeysResponse, 0)
	json.Unmarshal(respContents, &respContentsList)
	for idx := range respContentsList {
		if respContentsList[idx].Name == clusterName {
			d.SetId(clusterName)
			return nil
		}
	}
	d.SetId("")
	return nil
}

func resourceUpdateItem(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(clientInfo)

	rotateApikey := d.Get("rotate").(bool)
	clusterName := d.Get("name").(string)
	if rotateApikey {
		request, err := http.NewRequest(
			"POST",
			apiClient.address+fmt.Sprintf("/api/cluster/%s/%s/apikeys/rotate", apiClient.orgName, clusterName),
			bytes.NewBuffer([]byte{}),
		)
		resp, err := apiClient.httpClient.Do(request)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respContents, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		respContentsMap := make(map[string]string)
		json.Unmarshal(respContents, &respContentsMap)
		apiKey := respContentsMap["apiKey"]
		d.SetId(clusterName)
		err = d.Set("apikey", apiKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func resourceDeleteItem(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(clientInfo)

	clusterName := d.Get("name").(string)
	request, err := http.NewRequest(
		"DELETE",
		apiClient.address+fmt.Sprintf("/api/cluster/%s/%s/apikeys", apiClient.orgName, clusterName),
		bytes.NewBuffer([]byte{}),
	)
	resp, err := apiClient.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		d.SetId("")
		return nil
	} else {
		errorFromServer, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(errorFromServer))
	}
}

func resourceExistsItem(d *schema.ResourceData, m interface{}) (bool, error) {
	apiClient := m.(clientInfo)

	clusterName := d.Get("name").(string)
	request, err := http.NewRequest(
		"GET",
		apiClient.address+fmt.Sprintf("/api/cluster/%s/apikeys/cluster", apiClient.orgName),
		bytes.NewBuffer([]byte{}),
	)
	resp, err := apiClient.httpClient.Do(request)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	respContents, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	respContentsList := make(GetApiKeysResponse, 0)
	json.Unmarshal(respContents, &respContentsList)
	for idx := range respContentsList {
		if respContentsList[idx].Name == clusterName {
			return true, nil
		}
	}
	return false, nil
}
