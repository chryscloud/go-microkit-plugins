package dockerhub

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"unicode/utf8"

	mclog "github.com/chryscloud/go-microkit-plugins/log"
	"github.com/go-resty/resty/v2"
)

var (
	authURL     = "https://auth.docker.io/token"
	serviceURL  = "registry.docker.io"
	registryURL = "https://registry-1.docker.io"
)

// Options for digital ocean
type Options struct {
	Log      mclog.Logger
	Host     string
	username string
	password string
}

// Option a single option
type Option func(*Options)

// Log - recommended but optional
func Log(log mclog.Logger) Option {
	return func(args *Options) {
		args.Log = log
	}
}

// Host - remote host, options, default = registry-1.docker.io
func Host(remoteHost string) Option {
	return func(args *Options) {
		args.Host = remoteHost
	}
}

// Credentials - optionsl
func Credentials(username, password string) Option {
	return func(args *Options) {
		args.username = username
		args.password = password
	}
}

// Client - dockerhub abstraction
type Client struct {
	host       string
	log        mclog.Logger
	httpClient *resty.Client
	username   string
	password   string
	token      string
	mutex      *sync.Mutex
}

type authResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
	IssuedAt  string `json:"issued_at"`
}

type tagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// NewClient new DockerHub client to interactions with private or public docker hub (currently only Tags method supported)
func NewClient(opts ...Option) DockerHub {
	args := &Options{}
	for _, op := range opts {
		if op != nil {
			op(args)
		}
	}
	if args.Host == "" {
		args.Host = registryURL
	}
	cl := resty.New().SetHeader("Content-Type", "application/json").SetHeader("Docker-Distribution-Api-Version", "registry/2.0")
	cl.Debug = false

	outClient := &Client{
		host:       args.Host,
		log:        args.Log,
		httpClient: cl,
		mutex:      &sync.Mutex{},
	}
	if args.username != "" {
		outClient.username = args.username
	}
	if args.password != "" {
		outClient.password = args.password
	}
	return outClient
}

// Tags - returns the list of tags from the dockerhub repository
func (client *Client) Tags(repository string) ([]string, error) {
	// remove first slash if exists in repository
	if strings.HasPrefix(repository, "/") {
		_, i := utf8.DecodeRuneInString(repository)
		repository = repository[i:]
	}
	url := client.host + "/v2/" + repository + "/tags/list"

	var tagsResponse tagsResponse
	var tagsErr error
	var tagsGetResp *resty.Response

	var token authResponse
	if client.token == "" {
		t, err := client.retrieveAuthToken(repository)
		if err != nil {
			return nil, err
		}
		token = *t
		client.mutex.Lock()
		client.token = token.Token
		client.mutex.Unlock()
	}

	tagsGetResp, tagsErr = client.httpClient.R().SetHeader("Authorization", "Bearer "+client.token).SetResult(&tagsResponse).Get(url)
	if tagsGetResp.StatusCode() == http.StatusUnauthorized {
		t, err := client.retrieveAuthToken(repository)
		if err != nil {
			return nil, err
		}
		token = *t
		client.mutex.Lock()
		client.token = token.Token
		client.mutex.Unlock()

		tagsGetResp, tagsErr = client.httpClient.R().SetResult(&tagsResponse).SetHeader("Authorization", "Bearer "+client.token).Get(url)

	}
	if tagsErr != nil {
		if client.log != nil {
			client.log.Error("failed to retrieve tags", tagsErr, tagsGetResp)
		}
		return nil, tagsErr
	}
	if tagsGetResp.StatusCode() != http.StatusOK {
		if client.log != nil {
			client.log.Error("unexpected http code returned", tagsGetResp.StatusCode(), string(tagsGetResp.Body()))
		}
		return nil, errors.New("unexpected http code returned")
	}

	return tagsResponse.Tags, nil
}

func (client *Client) retrieveAuthToken(repository string) (*authResponse, error) {
	scope := getScope(repository)
	tokenURL := authURL + "?service=" + serviceURL + "&" + "scope=" + scope + "&offline_token=1&client_id=microkit-plugins-1.0"
	var authResponse authResponse
	request := client.httpClient.R()
	if client.username != "" && client.password != "" {
		request = request.SetBasicAuth(client.username, client.password)
	}
	request = request.SetResult(&authResponse)
	tokenResp, tokenErr := request.Get(tokenURL)
	if tokenErr != nil {
		if client.log != nil {
			client.log.Error("failed to get authentication token", tokenErr)
			return nil, errors.New("Unauthirized")
		}
	}
	if tokenResp.StatusCode() != http.StatusOK {
		if client.log != nil {
			client.log.Error("failed to retrieve auth token", tokenResp)
			return nil, errors.New("failed to retrieve auth token")
		}
	}
	return &authResponse, nil
}

// e.g. cocooncam/cc
func getScope(repository string) string {
	return "repository:" + repository + ":pull"
}

func slashFirstSlash(repository string) string {
	// remove first slash if exists in repository
	if strings.HasPrefix(repository, "/") {
		_, i := utf8.DecodeRuneInString(repository)
		repository = repository[i:]
	}
	return repository
}
