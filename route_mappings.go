package cfclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"github.com/pkg/errors"
)

type RouteMappingRequest struct {
	AppGUID   string `json:"app_guid"`
	RouteGUID string `json:"route_guid"`
	AppPort   int    `json:"app_port"`
}

type RouteMappingResponse struct {
	Count         int                    `json:"total_results"`
	Pages         int                    `json:"total_pages"`
	NextUrl       string                 `json:"next_url"`
	Resources []RouteMappingResource `json:"resources"`
}

type RouteMapping struct {
	Guid      string `json:"guid"`
	AppPort   int    `json:"app_port"`
	AppGUID   string `json:"app_guid"`
	RouteGUID string `json:"route_guid"`
	AppUrl    string `json:"app_url"`
	RouteUrl  string `json:"route_url"`
	c         *Client
}

type RouteMappingResource struct {
	Meta   Meta         `json:"metadata"`
	Entity RouteMapping `json:"entity"`
}

func (c *Client) MappingAppAndRoute(req RouteMappingRequest) (*RouteMapping, error) {
	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(req)
	if err != nil {
		return nil, err
	}
	r := c.NewRequestWithBody("POST", "/v2/route_mappings", buf)
	resp, err := c.DoRequest(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("CF API returned with status code %d", resp.StatusCode)
	}
	return c.handleMappingResp(resp)
}

func (c *Client) ListRouteMappings() ([]*RouteMapping, error) {
	return c.ListRouteMappingsByQuery(nil)
}

func (c *Client) ListRouteMappingsByQuery(query url.Values) ([]*RouteMapping, error) {
	var routeMappings []*RouteMapping
	var routeMappingsResp RouteMappingResponse
	pages := 0

	requestUrl := "/v2/route_mappings?" + query.Encode()
	for {
		r := c.NewRequest("GET", requestUrl)
		resp, err := c.DoRequest(r)
		if err != nil {
			return nil, errors.Wrap(err, "Error requesting route mappings")
		}
		resBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading route mappings request:")
		}

		err = json.Unmarshal(resBody, &routeMappingsResp)
		if err != nil {
			return nil, errors.Wrap(err, "Error unmarshaling route mappings")
		}

		for _, routeMapping := range routeMappingsResp.Resources {
			routeMapping.Entity.Guid = routeMapping.Meta.Guid
			routeMapping.Entity.c = c
			routeMappings = append(routeMappings, &routeMapping.Entity)
		}
		requestUrl = routeMappingsResp.NextUrl
		if requestUrl == "" {
			break
		}
		pages++
		totalPages := routeMappingsResp.Pages
		if totalPages > 0 && pages >= totalPages {
			break
		}
	}
	return routeMappings, nil
}

func (c *Client) handleMappingResp(resp *http.Response) (*RouteMapping, error) {
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var mappingResource RouteMappingResource
	err = json.Unmarshal(body, &mappingResource)
	if err != nil {
		return nil, err
	}
	return c.mergeMappingResource(mappingResource), nil
}

func (c *Client) mergeMappingResource(mapping RouteMappingResource) *RouteMapping {
	mapping.Entity.Guid = mapping.Meta.Guid
	mapping.Entity.c = c
	return &mapping.Entity
}
