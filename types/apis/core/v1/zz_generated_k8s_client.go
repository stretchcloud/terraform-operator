package v1

import (
	"context"
	"sync"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"github.com/rancher/norman/objectclient/dynamic"
	"github.com/rancher/norman/restwatch"
	"k8s.io/client-go/rest"
)

type (
	contextKeyType        struct{}
	contextClientsKeyType struct{}
)

type Interface interface {
	RESTClient() rest.Interface
	controller.Starter

	ConfigMapsGetter
	SecretsGetter
	ServiceAccountsGetter
}

type Clients struct {
	ConfigMap      ConfigMapClient
	Secret         SecretClient
	ServiceAccount ServiceAccountClient
}

type Client struct {
	sync.Mutex
	restClient rest.Interface
	starters   []controller.Starter

	configMapControllers      map[string]ConfigMapController
	secretControllers         map[string]SecretController
	serviceAccountControllers map[string]ServiceAccountController
}

func Factory(ctx context.Context, config rest.Config) (context.Context, controller.Starter, error) {
	c, err := NewForConfig(config)
	if err != nil {
		return ctx, nil, err
	}

	cs := NewClientsFromInterface(c)

	ctx = context.WithValue(ctx, contextKeyType{}, c)
	ctx = context.WithValue(ctx, contextClientsKeyType{}, cs)
	return ctx, c, nil
}

func ClientsFrom(ctx context.Context) *Clients {
	return ctx.Value(contextClientsKeyType{}).(*Clients)
}

func From(ctx context.Context) Interface {
	return ctx.Value(contextKeyType{}).(Interface)
}

func NewClients(config rest.Config) (*Clients, error) {
	iface, err := NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewClientsFromInterface(iface), nil
}

func NewClientsFromInterface(iface Interface) *Clients {
	return &Clients{

		ConfigMap: &configMapClient2{
			iface: iface.ConfigMaps(""),
		},
		Secret: &secretClient2{
			iface: iface.Secrets(""),
		},
		ServiceAccount: &serviceAccountClient2{
			iface: iface.ServiceAccounts(""),
		},
	}
}

func NewForConfig(config rest.Config) (Interface, error) {
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = dynamic.NegotiatedSerializer
	}

	restClient, err := restwatch.UnversionedRESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &Client{
		restClient: restClient,

		configMapControllers:      map[string]ConfigMapController{},
		secretControllers:         map[string]SecretController{},
		serviceAccountControllers: map[string]ServiceAccountController{},
	}, nil
}

func (c *Client) RESTClient() rest.Interface {
	return c.restClient
}

func (c *Client) Sync(ctx context.Context) error {
	return controller.Sync(ctx, c.starters...)
}

func (c *Client) Start(ctx context.Context, threadiness int) error {
	return controller.Start(ctx, threadiness, c.starters...)
}

type ConfigMapsGetter interface {
	ConfigMaps(namespace string) ConfigMapInterface
}

func (c *Client) ConfigMaps(namespace string) ConfigMapInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &ConfigMapResource, ConfigMapGroupVersionKind, configMapFactory{})
	return &configMapClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type SecretsGetter interface {
	Secrets(namespace string) SecretInterface
}

func (c *Client) Secrets(namespace string) SecretInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &SecretResource, SecretGroupVersionKind, secretFactory{})
	return &secretClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}

type ServiceAccountsGetter interface {
	ServiceAccounts(namespace string) ServiceAccountInterface
}

func (c *Client) ServiceAccounts(namespace string) ServiceAccountInterface {
	objectClient := objectclient.NewObjectClient(namespace, c.restClient, &ServiceAccountResource, ServiceAccountGroupVersionKind, serviceAccountFactory{})
	return &serviceAccountClient{
		ns:           namespace,
		client:       c,
		objectClient: objectClient,
	}
}
