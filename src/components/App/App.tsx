import * as React from 'react';
import { AppRootProps } from '@grafana/data';
import { PluginPropsContext } from '../../utils/utils.plugin';
import { Status } from "../../pages/Status";

export class App extends React.PureComponent<AppRootProps> {
  render() {
    return (
      <PluginPropsContext.Provider value={this.props}>
        <Status />
      </PluginPropsContext.Provider>
    );
  }
}
