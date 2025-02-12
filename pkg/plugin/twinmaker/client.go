package twinmaker

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iottwinmaker"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/grafana/grafana-aws-sdk/pkg/awsds"
	"github.com/grafana/grafana-iot-twinmaker-app/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/build"
)

// TwinMakerClient calls AWS services and returns the raw results
type TwinMakerClient interface {
	GetSessionToken(ctx context.Context, duration time.Duration, workspaceId string) (*sts.Credentials, error)
	ListWorkspaces(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListWorkspacesOutput, error)
	GetWorkspace(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetWorkspaceOutput, error)
	ListScenes(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListScenesOutput, error)
	ListEntities(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListEntitiesOutput, error)
	ListComponentTypes(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListComponentTypesOutput, error)
	GetComponentType(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetComponentTypeOutput, error)
	GetEntity(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetEntityOutput, error)

	// NOTE: only works with non-timeseries data
	GetPropertyValue(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetPropertyValueOutput, error)

	// NOTE: only works with timeseries data
	GetPropertyValueHistory(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetPropertyValueHistoryOutput, error)
}

type twinMakerClient struct {
	tokenRole string

	twinMakerService func() (*iottwinmaker.IoTTwinMaker, error)
	tokenService     func() (*sts.STS, error)
}

// NewTwinMakerClient provides a twinMakerClient for the session and associated calls
func NewTwinMakerClient(settings models.TwinMakerDataSourceSetting) (TwinMakerClient, error) {
	sessions := awsds.NewSessionCache()
	agent := userAgentString("grafana-iot-twinmaker-app")

	// STS client can not use scoped down role to generate tokens
	stssettings := settings.AWSDatasourceSettings
	stssettings.AssumeRoleARN = ""
	stssettings.Endpoint = "" // always standard

	twinMakerService := func() (*iottwinmaker.IoTTwinMaker, error) {
		sess, err := sessions.GetSession("", settings.AWSDatasourceSettings)
		if err != nil {
			return nil, err
		}

		svc := iottwinmaker.New(sess, aws.NewConfig())
		svc.Handlers.Send.PushFront(func(r *request.Request) {
			r.HTTPRequest.Header.Set("User-Agent", agent)

		})
		return svc, err
	}

	tokenService := func() (*sts.STS, error) {
		sess, err := sessions.GetSession("", stssettings)
		if err != nil {
			return nil, err
		}
		svc := sts.New(sess, aws.NewConfig())
		svc.Handlers.Send.PushFront(func(r *request.Request) {
			r.HTTPRequest.Header.Set("User-Agent", agent)
		})
		return svc, err
	}

	return &twinMakerClient{
		twinMakerService: twinMakerService,
		tokenService:     tokenService,
		tokenRole:        settings.AWSDatasourceSettings.AssumeRoleARN,
	}, nil
}

func (c *twinMakerClient) ListWorkspaces(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListWorkspacesOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	params := &iottwinmaker.ListWorkspacesInput{
		MaxResults: aws.Int64(200),
		NextToken:  aws.String(query.NextToken),
	}

	workspaces, err := client.ListWorkspacesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	cWorkspaces := workspaces
	for cWorkspaces.NextToken != nil {
		params.NextToken = cWorkspaces.NextToken

		cWorkspaces, err := client.ListWorkspacesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		workspaces.WorkspaceSummaries = append(workspaces.WorkspaceSummaries, cWorkspaces.WorkspaceSummaries...)
		workspaces.NextToken = cWorkspaces.NextToken
	}

	return workspaces, nil
}

func (c *twinMakerClient) ListScenes(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListScenesOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	params := &iottwinmaker.ListScenesInput{
		MaxResults: aws.Int64(200),
		//Mode:        aws.String("PUBLISHED"),
		NextToken:   aws.String(query.NextToken),
		WorkspaceId: &query.WorkspaceId,
	}

	scenes, err := client.ListScenesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	cScenes := scenes
	for cScenes.NextToken != nil {
		params.NextToken = cScenes.NextToken

		cScenes, err := client.ListScenesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		scenes.SceneSummaries = append(scenes.SceneSummaries, cScenes.SceneSummaries...)
		scenes.NextToken = cScenes.NextToken
	}

	return scenes, nil
}

func (c *twinMakerClient) ListEntities(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListEntitiesOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	params := &iottwinmaker.ListEntitiesInput{
		MaxResults:  aws.Int64(200),
		NextToken:   aws.String(query.NextToken),
		WorkspaceId: &query.WorkspaceId,
	}

	if query.ComponentTypeId != "" {
		params.Filters = make([]*iottwinmaker.ListEntitiesFilter, 1)
		params.Filters[0] = &iottwinmaker.ListEntitiesFilter{
			ComponentTypeId: &query.ComponentTypeId,
		}
	}

	entities, err := client.ListEntitiesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	cEntities := entities
	for cEntities.NextToken != nil {
		params.NextToken = cEntities.NextToken

		cEntities, err := client.ListEntitiesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		entities.EntitySummaries = append(entities.EntitySummaries, cEntities.EntitySummaries...)
		entities.NextToken = cEntities.NextToken
	}

	return entities, nil
}

func (c *twinMakerClient) ListComponentTypes(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.ListComponentTypesOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	params := &iottwinmaker.ListComponentTypesInput{
		MaxResults:  aws.Int64(200),
		NextToken:   aws.String(query.NextToken),
		WorkspaceId: &query.WorkspaceId,
	}

	if query.ComponentTypeId != "" {
		params.Filters = make([]*iottwinmaker.ListComponentTypesFilter, 1)
		params.Filters[0] = &iottwinmaker.ListComponentTypesFilter{
			ExtendsFrom: &query.ComponentTypeId,
		}
	}

	componentTypes, err := client.ListComponentTypesWithContext(ctx, params)
	if err != nil {
		return nil, err
	}

	cComponentTypes := componentTypes
	for cComponentTypes.NextToken != nil {
		params.NextToken = cComponentTypes.NextToken

		cComponentTypes, err := client.ListComponentTypesWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		componentTypes.ComponentTypeSummaries = append(componentTypes.ComponentTypeSummaries, cComponentTypes.ComponentTypeSummaries...)
		componentTypes.NextToken = cComponentTypes.NextToken
	}

	return componentTypes, nil
}

func (c *twinMakerClient) GetComponentType(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetComponentTypeOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	if query.ComponentTypeId == "" {
		return nil, fmt.Errorf("missing component type id")
	}

	params := &iottwinmaker.GetComponentTypeInput{
		WorkspaceId:     &query.WorkspaceId,
		ComponentTypeId: &query.ComponentTypeId,
	}

	return client.GetComponentTypeWithContext(ctx, params)
}

func (c *twinMakerClient) GetEntity(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetEntityOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	if query.EntityId == "" {
		return nil, fmt.Errorf("missing entity id")
	}

	params := &iottwinmaker.GetEntityInput{
		EntityId:    &query.EntityId,
		WorkspaceId: &query.WorkspaceId,
	}

	return client.GetEntityWithContext(ctx, params)
}

func (c *twinMakerClient) GetWorkspace(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetWorkspaceOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	params := &iottwinmaker.GetWorkspaceInput{
		WorkspaceId: &query.WorkspaceId,
	}

	return client.GetWorkspaceWithContext(ctx, params)
}

func (c *twinMakerClient) GetPropertyValue(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetPropertyValueOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	if query.EntityId == "" {
		return nil, fmt.Errorf("missing entity id")
	}
	if query.ComponentName == "" {
		return nil, fmt.Errorf("missing component name")
	}
	if query.Properties == nil || len(query.Properties) < 1 {
		return nil, fmt.Errorf("missing property")
	}

	params := &iottwinmaker.GetPropertyValueInput{
		EntityId:           &query.EntityId,
		ComponentName:      &query.ComponentName,
		SelectedProperties: query.Properties,
		WorkspaceId:        &query.WorkspaceId,
	}

	return client.GetPropertyValueWithContext(ctx, params)
}

func (c *twinMakerClient) GetPropertyValueHistory(ctx context.Context, query models.TwinMakerQuery) (*iottwinmaker.GetPropertyValueHistoryOutput, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	if query.EntityId == "" && query.ComponentTypeId == "" {
		return nil, fmt.Errorf("missing entity id & component type id - either one required")
	}

	params := &iottwinmaker.GetPropertyValueHistoryInput{
		EndDateTime:        &query.TimeRange.To,
		SelectedProperties: query.Properties,
		StartDateTime:      &query.TimeRange.From,
		WorkspaceId:        &query.WorkspaceId,
	}

	if query.NextToken != "" {
		params.NextToken = &query.NextToken
	}

	if query.Order != "" {
		params.SetOrderByTime(query.Order)
	}

	if c := query.ComponentTypeId; c != "" {
		if query.Properties == nil || len(query.Properties) < 1 {
			return nil, fmt.Errorf("missing property")
		}
		params.ComponentTypeId = &c
	} else {
		if query.ComponentName == "" {
			return nil, fmt.Errorf("missing component name")
		}
		if query.Properties == nil || len(query.Properties) < 1 {
			return nil, fmt.Errorf("missing property")
		}
		params.EntityId = &query.EntityId
		params.ComponentName = &query.ComponentName
	}

	if len(query.Filter) > 0 {
		var filter []*iottwinmaker.PropertyFilter
		for _, fq := range query.Filter {
			if fq.Name != "" && fq.Value != "" {
				if fq.Op == "" {
					fq.Op = "=" // matches the placeholder text in the frontend
				}
				filter = append(filter, fq.ToTwinMakerFilter())
			}
		}
		params.SetPropertyFilters(filter)
	}

	return client.GetPropertyValueHistoryWithContext(ctx, params)
}

func (c *twinMakerClient) GetSessionToken(ctx context.Context, duration time.Duration, workspaceId string) (*sts.Credentials, error) {
	client, err := c.twinMakerService()
	if err != nil {
		return nil, err
	}

	tokenService, err := c.tokenService()
	if err != nil {
		return nil, err
	}

	// always call AssumeRole with an inline session policy if a role is provided
	if c.tokenRole != "" {
		params := &iottwinmaker.GetWorkspaceInput{
			WorkspaceId: &workspaceId,
		}

		workspace, err := client.GetWorkspaceWithContext(ctx, params)
		if err != nil {
			return nil, err
		}

		policy, err := LoadPolicy(workspace)
		if err != nil {
			return nil, err
		}

		input := &sts.AssumeRoleInput{
			RoleArn:         &c.tokenRole,
			DurationSeconds: aws.Int64(int64(duration.Seconds())),
			RoleSessionName: aws.String("grafana"),
			Policy:          aws.String(policy),
		}

		out, err := tokenService.AssumeRoleWithContext(ctx, input)
		if err != nil {
			return nil, err
		}

		return out.Credentials, err
	}

	// if there is a sessionToken set in the default credentials within
	// the chain of authentication providers, it means they are temporary,
	// and hence the frontend needs to use them directly and we don't
	// need to call sts for temporary session tokens
	creds, err := client.Config.Credentials.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}

	if creds.SessionToken != "" {
		// force expire the creds here because of the expire window
		// logic in the frontend, where it assumes the expiry before
		// a certain time of the actual expiry
		client.Config.Credentials.Expire()

		creds, err := client.Config.Credentials.GetWithContext(ctx)
		if err != nil {
			return nil, err
		}

		// just force an expiry time here too since using the
		// Credentials.ExpiresAt() is not supported for some
		// Providers that might be being used
		expiryTime := time.Now().Add(stscreds.DefaultDuration)

		return &sts.Credentials{
			AccessKeyId:     &creds.AccessKeyID,
			SecretAccessKey: &creds.SecretAccessKey,
			SessionToken:    &creds.SessionToken,
			Expiration:      &expiryTime,
		}, err
	}

	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(int64(duration.Seconds())),
	}
	out, err := tokenService.GetSessionTokenWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	return out.Credentials, err
}

// TODO, move to https://github.com/grafana/grafana-plugin-sdk-go
func userAgentString(name string) string {
	buildInfo, err := build.GetBuildInfo()
	if err != nil {
		buildInfo.Version = "dev"
		buildInfo.Hash = "?"
	}

	if len(buildInfo.Hash) > 8 {
		buildInfo.Hash = buildInfo.Hash[0:8]
	}

	return fmt.Sprintf("%s/%s (%s; %s;) %s/%s-%s Grafana/%s",
		aws.SDKName,
		aws.SDKVersion,
		runtime.Version(),
		runtime.GOOS,
		name,
		buildInfo.Version,
		buildInfo.Hash,
		os.Getenv("GF_VERSION"))
}
