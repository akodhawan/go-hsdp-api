// Package ai provides support the HSDP AI services
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/go-querystring/query"
	autoconf "github.com/philips-software/go-hsdp-api/config"
	"github.com/philips-software/go-hsdp-api/iam"
	"github.com/philips-software/go-hsdp-api/internal"
)

const (
	userAgent  = "go-hsdp-api/ai/" + internal.LibraryVersion
	APIVersion = "1"
)

// OptionFunc is the function signature function for options
type OptionFunc func(*http.Request) error

// Config contains the configuration of a Client
type Config struct {
	Region         string
	Environment    string
	OrganizationID string `Validate:"required"`
	BaseURL        string
	Service        string `Validate:"required"`
	DebugLog       string
	Retry          int
}

// A Client manages communication with HSDP AI APIs
type Client struct {
	// HTTP Client used to communicate with IAM API
	iamClient *iam.Client
	config    *Config
	baseURL   *url.URL

	// User agent used when communicating with the HSDP Notification API
	UserAgent string

	debugFile *os.File
	validate  *validator.Validate

	ComputeTarget   *ComputeTargetService
	ComputeProvider *ComputeProviderService
}

// NewClient returns a new AI base Client
func NewClient(iamClient *iam.Client, config *Config) (*Client, error) {
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, err
	}
	doAutoconf(config)
	c := &Client{iamClient: iamClient, config: config, UserAgent: userAgent, validate: validator.New()}

	if err := c.SetBaseURL(config.BaseURL); err != nil {
		return nil, err
	}

	c.ComputeProvider = &ComputeProviderService{client: c, validate: validator.New()}
	c.ComputeTarget = &ComputeTargetService{client: c, validate: validator.New()}

	return c, nil
}

func doAutoconf(config *Config) {
	if config.Region != "" && config.Environment != "" {
		c, err := autoconf.New(
			autoconf.WithRegion(config.Region),
			autoconf.WithEnv(config.Environment))
		if err == nil {
			theService := c.Service(config.Service)
			if theService.URL != "" && config.BaseURL == "" {
				config.BaseURL = theService.URL
			}
		}
	}
}

// Close releases allocated resources of clients
func (c *Client) Close() {
	if c.debugFile != nil {
		_ = c.debugFile.Close()
		c.debugFile = nil
	}
}

// GetBaseURL returns the base URL as configured
func (c *Client) GetBaseURL() string {
	if c.baseURL == nil {
		return ""
	}
	return c.baseURL.String()
}

// SetBaseURL sets the base URL for API requests
func (c *Client) SetBaseURL(urlStr string) error {
	if urlStr == "" {
		return ErrBaseURLCannotBeEmpty
	}
	// Make sure the given URL end with a slash
	if !strings.HasSuffix(urlStr, "/") {
		urlStr += "/"
	}
	var err error
	c.baseURL, err = url.Parse(urlStr)
	return err
}

// GetEndpointURL returns the FHIR Store Endpoint URL as configured
func (c *Client) GetEndpointURL() string {
	return c.GetBaseURL() + path.Join("analyze", c.config.Service, c.config.OrganizationID)
}

// SetEndpointURL sets the endpoint URL for API requests to a custom endpoint. urlStr
// should always be specified with a trailing slash.
func (c *Client) SetEndpointURL(urlStr string) error {
	if urlStr == "" {
		return ErrBaseURLCannotBeEmpty
	}
	// Make sure the given URL ends with a slash
	if !strings.HasSuffix(urlStr, "/") {
		urlStr += "/"
	}
	var err error
	c.baseURL, err = url.Parse(urlStr)
	if err != nil {
		return err
	}
	parts := strings.Split(c.baseURL.Path, "/")
	if len(parts) == 0 {
		return ErrBaseURLCannotBeEmpty
	}
	if len(parts) < 5 {
		return ErrInvalidEndpointURL
	}
	c.config.OrganizationID = parts[len(parts)-2]
	c.baseURL.Path = "/"
	return nil
}

func (c *Client) NewAIRequest(method, requestPath string, opt interface{}, options ...OptionFunc) (*http.Request, error) {
	u := *c.baseURL
	// Set the encoded opaque data
	u.Opaque = path.Join(c.baseURL.Path, "analyze", c.config.Service, c.config.OrganizationID, requestPath)

	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		u.RawQuery = q.Encode()
	}

	req := &http.Request{
		Method:     method,
		URL:        &u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}
	if opt != nil {
		q, err := query.Values(opt)
		if err != nil {
			return nil, err
		}
		u.RawQuery = q.Encode()
	}

	if method == "POST" || method == "PUT" {
		bodyBytes, err := json.Marshal(opt)
		if err != nil {
			return nil, err
		}
		bodyReader := bytes.NewReader(bodyBytes)

		u.RawQuery = ""
		req.Body = ioutil.NopCloser(bodyReader)
		req.ContentLength = int64(bodyReader.Len())
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "Bearer "+c.iamClient.Token())
	req.Header.Set("API-Version", APIVersion)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for _, fn := range options {
		if fn == nil {
			continue
		}
		if err := fn(req); err != nil {
			return nil, err
		}
	}
	return req, nil
}

// Response is a HSDP IAM API response. This wraps the standard http.Response
// returned from HSDP IAM and provides convenient access to things like errors
type Response struct {
	*http.Response
}

// newResponse creates a new Response for the provided http.Response.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	return response
}

// TokenRefresh forces a refresh of the IAM access token
func (c *Client) TokenRefresh() error {
	if c.iamClient == nil {
		return fmt.Errorf("invalid IAM Client, cannot refresh token")
	}
	return c.iamClient.TokenRefresh()
}

// Do executes a http request. If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.iamClient.HttpClient().Do(req)
	if err != nil {
		return nil, err
	}

	response := newResponse(resp)

	err = internal.CheckResponse(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return response, err
	}

	if v != nil {
		defer resp.Body.Close() // Only close if we plan to read it
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}

	return response, err
}
