import React from 'react';
import { render } from '@testing-library/react';
import { mockGrafanaUI, mockTwinMakerPanelProps, mockTimeRange } from '../tests/utils/__mocks__';

mockGrafanaUI();

import { ComponentName } from 'aws-iot-twinmaker-grafana-utils';
import { VideoPlayer } from './VideoPlayer';
import { VideoPlayerPropsFromParent } from './interfaces';
import { mockDisplayOptions } from './tests/common';
import { setTemplateSrv } from '@grafana/runtime';

setTemplateSrv({
  getVariables: () => [],
  replace: (v: string) => v,
});

describe('VideoPlayer', () => {
  it('should load VideoPlayer component when providing kvsStreamName', () => {
    const mockKvsStream = 'mockKvsStream';
    const mockEntityId = 'mockEntityId';
    const mockComponentName = 'mockComponentName';
    const mockWorkspaceId = 'MockWorkspaceId';
    const expectedComponentOptions = {
      kvsStreamName: mockKvsStream,
      startTime: mockTimeRange.from.toDate(),
      endTime: mockTimeRange.to.toDate(),
      componentName: mockComponentName,
      entityId: mockEntityId,
      workspaceId: mockWorkspaceId,
    };

    const options = {
      kvsStreamName: mockKvsStream,
      entityId: mockEntityId,
      componentName: mockComponentName,
    };

    const panelProps = mockTwinMakerPanelProps(mockDisplayOptions);
    const props: VideoPlayerPropsFromParent = {
      ...panelProps,
      options,
      componentName: mockComponentName,
      entityId: mockEntityId,
      kvsStreamName: mockKvsStream,
    };

    render(<VideoPlayer {...props} />);
    expect(panelProps.twinMakerUxSdk.createComponentForReact).toHaveBeenCalledWith(
      ComponentName.VideoPlayer,
      expectedComponentOptions
    );
  });
});
