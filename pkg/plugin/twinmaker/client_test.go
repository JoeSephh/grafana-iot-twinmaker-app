package twinmaker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/grafana/grafana-aws-sdk/pkg/awsds"
	"github.com/grafana/grafana-iot-twinmaker-app/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestFetchAWSData(t *testing.T) {
	t.Run("get a sts token with inline policy enforced", func(t *testing.T) {
		t.Skip()

		c, err := NewTwinMakerClient(models.TwinMakerDataSourceSetting{
			// use credentials in ~/.aws/credentials
			AWSDatasourceSettings: awsds.AWSDatasourceSettings{
				AuthType:      awsds.AuthTypeDefault,
				AssumeRoleARN: "arn:aws:iam::166800769179:role/TwinMakerGrafanaWorkspaceDashboardRole",
				Region:        "us-east-1",
			},
		})
		require.NoError(t, err)

		WorkspaceId := "GrafanaWorkspace"
		token, err := c.GetSessionToken(context.Background(), time.Second*3600, WorkspaceId)
		require.NoError(t, err)
		require.NotEmpty(t, token)
	})

	t.Run("get a sts token with custom expiry when creds are temp", func(t *testing.T) {
		c, err := NewTwinMakerClient(models.TwinMakerDataSourceSetting{
			// use credentials in ~/.aws/credentials
			AWSDatasourceSettings: awsds.AWSDatasourceSettings{
				AuthType:     awsds.AuthTypeKeys,
				AccessKey:    "dummyAccessKeyId",
				SecretKey:    "dummySecretKeyId",
				SessionToken: "dummySessionToken", // this means creds are already temp
			},
		})
		require.NoError(t, err)

		WorkspaceId := "GrafanaWorkspace"
		token, err := c.GetSessionToken(context.Background(), time.Second*3600, WorkspaceId)
		require.NoError(t, err)
		require.NotEmpty(t, token)
		require.NotNil(t, token.Expiration)
	})

	t.Run("manually get an sts token when creds are permanent", func(t *testing.T) {
		t.Skip()

		c, err := NewTwinMakerClient(models.TwinMakerDataSourceSetting{
			// use credentials in ~/.aws/credentials
			AWSDatasourceSettings: awsds.AWSDatasourceSettings{
				AuthType:      awsds.AuthTypeDefault,
				AssumeRoleARN: "arn:aws:iam::166800769179:role/TwinMakerGrafanaWorkspaceDashboardRole",
				Region:        "us-east-1",
			},
		})
		require.NoError(t, err)

		WorkspaceId := "GrafanaWorkspace"
		token, err := c.GetSessionToken(context.Background(), time.Second*3600, WorkspaceId)
		require.NoError(t, err)

		writeTestData("get-token", token, t)
	})

	t.Run("manually query twinmaker", func(t *testing.T) {
		t.Skip()

		c, err := NewTwinMakerClient(models.TwinMakerDataSourceSetting{
			// use credentials in ~/.aws/credentials
			AWSDatasourceSettings: awsds.AWSDatasourceSettings{
				AuthType: awsds.AuthTypeDefault,
				Region:   "us-east-1",
				Endpoint: "https://gamma.us-east-1.twinmaker.iot.aws.dev",
			},
		})
		require.NoError(t, err)

		w, err := c.ListWorkspaces(context.Background(), models.TwinMakerQuery{})
		require.NoError(t, err)
		writeTestData("list-workspaces", w, t)

		s, err := c.ListScenes(context.Background(), models.TwinMakerQuery{
			WorkspaceId: "CookieFactory-11-16",
		})
		require.NoError(t, err)
		writeTestData("list-scenes", s, t)

		e, err := c.ListEntities(context.Background(), models.TwinMakerQuery{
			WorkspaceId: "CookieFactory-11-16",
		})
		require.NoError(t, err)
		writeTestData("list-entities", e, t)

		ct, err := c.ListComponentTypes(context.Background(), models.TwinMakerQuery{
			WorkspaceId: "CookieFactory-11-16",
		})
		require.NoError(t, err)
		writeTestData("list-component-types", ct, t)

		ci, err := c.GetComponentType(context.Background(), models.TwinMakerQuery{
			WorkspaceId:     "CookieFactory-11-16",
			ComponentTypeId: "com.example.cookiefactory.alarm",
		})
		require.NoError(t, err)
		writeTestData("get-component-type", ci, t)

		g, err := c.GetEntity(context.Background(), models.TwinMakerQuery{
			EntityId:    "Mixer_1_4b57cbee-c391-4de6-b882-622c633a697e",
			WorkspaceId: "CookieFactory-11-16",
		})
		require.NoError(t, err)
		writeTestData("get-entity", g, t)

		pv, err := c.GetPropertyValue(context.Background(), models.TwinMakerQuery{
			EntityId:      "Mixer_1_4b57cbee-c391-4de6-b882-622c633a697e",
			WorkspaceId:   "CookieFactory-11-16",
			Properties:    []*string{aws.String("alarm_key"), aws.String("telemetryAssetType")},
			ComponentName: "AlarmComponent",
		})
		require.NoError(t, err)
		writeTestData("get-property-value", pv, t)

		// List data type property
		pv, err = c.GetPropertyValue(context.Background(), models.TwinMakerQuery{
			EntityId:      "Factory_aa3d7d8b-6b94-44fe-ab02-6936bfcdade6",
			WorkspaceId:   "CookieFactory-11-16",
			Properties:    []*string{aws.String("bounds")},
			ComponentName: "Space",
		})
		require.NoError(t, err)
		writeTestData("get-property-value-list", pv, t)

		// Map data type property
		pv, err = c.GetPropertyValue(context.Background(), models.TwinMakerQuery{
			EntityId:      "b2f31ce2-0c71-4e6d-8c65-000d8781d676",
			WorkspaceId:   "CookieFactory-11-16",
			Properties:    []*string{aws.String("documents")},
			ComponentName: "DocumentComponent",
		})
		require.NoError(t, err)
		writeTestData("get-property-value-map", pv, t)

		// check the combination: entityId -> componentName -> propertyName(s)
		p, err := c.GetPropertyValueHistory(context.Background(), models.TwinMakerQuery{
			EntityId:    "Mixer_1_4b57cbee-c391-4de6-b882-622c633a697e",
			WorkspaceId: "CookieFactory-11-16",
			TimeRange: backend.TimeRange{
				From: time.Date(2021, 11, 1, 1, 0, 0, 0, time.UTC),
				To:   time.Date(2021, 11, 7, 23, 5, 0, 0, time.UTC),
			},
			Properties:    []*string{aws.String("alarm_status")},
			ComponentName: "AlarmComponent",
		})
		require.NoError(t, err)
		writeTestData("get-property-history-alarms", p, t)

		// check the combination: componentTypeId -> propertyName(s)
		p, err = c.GetPropertyValueHistory(context.Background(), models.TwinMakerQuery{
			WorkspaceId: "CookieFactory-11-16",
			TimeRange: backend.TimeRange{
				From: time.Date(2021, 11, 1, 1, 0, 0, 0, time.UTC),
				To:   time.Date(2021, 11, 7, 23, 5, 0, 0, time.UTC),
			},
			Properties:      []*string{aws.String("alarm_status")},
			ComponentTypeId: "com.example.cookiefactory.alarm",
		})
		require.NoError(t, err)
		writeTestData("get-property-history-alarms-w-id", p, t)
	})

}

// This will write the results to local json file
//nolint:golint,unused
func writeTestData(filename string, res interface{}, t *testing.T) {
	json, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Println("marshalling results failed", err.Error())
	}

	f, err := os.Create("./testdata/" + filename + ".json")
	if err != nil {
		fmt.Println("create file failed: ", filename)
	}

	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	_, err = f.Write(json)
	if err != nil {
		fmt.Println("write file failed: ", filename)
	}
}
