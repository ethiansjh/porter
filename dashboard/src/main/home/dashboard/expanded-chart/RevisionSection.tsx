import React, { Component } from 'react';
import styled from 'styled-components';
import loading from '../../../../assets/loading.gif';

import api from '../../../../shared/api';
import { Context } from '../../../../shared/Context';
import { ChartType, StorageType } from '../../../../shared/types';

type PropsType = {
  showRevisions: boolean,
  toggleShowRevisions: () => void,
  chart: ChartType,
  refreshChart: () => void,
  setRevisionPreview: (preview: ChartType) => void
};

type StateType = {
  revisions: ChartType[],
  rollbackRevision: number | null,
  loading: boolean,
  maxVersion: number
};

export default class RevisionSection extends Component<PropsType, StateType> {
  state = {
    revisions: [] as ChartType[],
    rollbackRevision: null as (number | null),
    loading: false,
    maxVersion: 0, // Track most recent version even when previewing old revisions
  }

  refreshHistory = () => {
    let { chart } = this.props;

    api.getRevisions('<token>', {
      namespace: chart.namespace,
      context: this.context.currentCluster,
      storage: StorageType.Secret
    }, { name: chart.name }, (err: any, res: any) => {
      if (err) {
        console.log(err)
      } else {
        res.data.sort((a: ChartType, b: ChartType) => { return -(a.version - b.version) });
        this.setState({ revisions: res.data, maxVersion: res.data[0].version });
      }
    });
  }

  componentDidMount() {
    this.refreshHistory();
  }

  // Handle update of values.yaml
  componentDidUpdate(prevProps: PropsType) {
    if (this.props.chart !== prevProps.chart) {
      this.refreshHistory();
    }
  }

  readableDate = (s: string) => {
    let ts = new Date(s);
    let date = ts.toLocaleDateString();
    let time = ts.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
    return `${time} on ${date}`;
  }

  handleRollback = () => {
    let { setCurrentError, currentCluster } = this.context;

    let revisionNumber = this.state.rollbackRevision;
    this.setState({ loading: true, rollbackRevision: null });

    api.rollbackChart('<token>', {
      namespace: this.props.chart.namespace,
      context: currentCluster,
      storage: StorageType.Secret,
      revision: revisionNumber
    }, {
      name: this.props.chart.name
    }, (err: any, res: any) => {
      if (err) {
        setCurrentError(err.response.data.errors[0]);
        this.setState({ loading: false });
      } else {
        this.setState({ loading: false });
        this.props.refreshChart();
        this.refreshHistory();
      }
    });
  }

  renderRevisionList = () => {
    return this.state.revisions.map((revision: ChartType, i: number) => {
      let isCurrent = revision.version === this.state.maxVersion;
      return (
        <Tr
          key={i}
          onClick={() => this.props.setRevisionPreview(revision)}
          selected={this.props.chart.version === revision.version}
        >
          <Td>{revision.version}</Td>
          <Td>{this.readableDate(revision.info.last_deployed)}</Td>
          <Td>{revision.info.status}</Td>
          <Td>
            <RollbackButton
              disabled={isCurrent}
              onClick={() => this.setState({ rollbackRevision: revision.version })}
            >
              {isCurrent ? 'Current' : 'Revert'}
            </RollbackButton>
          </Td>
        </Tr>
      );
    });
  }

  renderExpanded = () => {
    if (this.props.showRevisions) {
      return (
        <TableWrapper>
          <RevisionsTable>
            <tbody>
              <Tr disableHover={true}>
                <Th>Revision No.</Th>
                <Th>Timestamp</Th>
                <Th>Status</Th>
                <Th>Rollback</Th>
              </Tr>
              {this.renderRevisionList()}
            </tbody>
          </RevisionsTable>
        </TableWrapper>
      )
    }
  }

  renderConfirmOverlay = () => {
    if (this.state.rollbackRevision) {
      return (
        <ConfirmOverlay>
          {`Are you sure you want to revert to version ${this.state.rollbackRevision}?`}
          <ButtonRow>
            <ConfirmButton
              onClick={() => this.handleRollback()}
            >
              Yes
            </ConfirmButton>
            <ConfirmButton
              onClick={() => this.setState({ rollbackRevision: null })}
            >
              No
            </ConfirmButton>
          </ButtonRow>
        </ConfirmOverlay>
      );
    }
  }

  renderContents = () => {
    if (this.state.loading) {
      return (
        <LoadingPlaceholder>
          <StatusWrapper>
            <LoadingGif src={loading} /> Updating . . .
          </StatusWrapper>
        </LoadingPlaceholder>
      )
    }

    let isCurrent = this.props.chart.version === this.state.maxVersion || this.state.maxVersion === 0;
    return (
      <div>
        <RevisionHeader
          showRevisions={this.props.showRevisions}
          onClick={this.props.toggleShowRevisions}
        >
          {isCurrent ? `Current Revision` : `Previewing Revision (Not Deployed)`} - <Revision>No. {this.props.chart.version}</Revision>
          <i className="material-icons">expand_more</i>
        </RevisionHeader>

        <RevisionList>
          {this.renderExpanded()}
        </RevisionList>
      </div>
    );
  }

  render() {
    return (
      <StyledRevisionSection showRevisions={this.props.showRevisions}>
        {this.renderContents()}
        {this.renderConfirmOverlay()}
      </StyledRevisionSection>
    );
  }
}

RevisionSection.contextType = Context;

const TableWrapper = styled.div`
  padding-bottom: 20px;
`;

const LoadingPlaceholder = styled.div`
  height: 40px;
  display: flex;
  align-items: center;
  padding-left: 20px;
`;

const LoadingGif = styled.img`
  width: 15px;
  height: 15px;
  margin-right: 9px;
  margin-bottom: 0px;
`;

const StatusWrapper = styled.div`
  display: flex;
  align-items: center;
  font-family: 'Work Sans', sans-serif;
  font-size: 13px;
  color: #ffffff55;
  margin-right: 25px;
`;

const ConfirmOverlay = styled.div`
  position: absolute;
  top: 0px;
  opacity: 100%;
  left: 0px;
  width: 100%;
  height: 100%;
  z-index: 999;
  display: flex;
  padding-bottom: 30px;
  align-items: center;
  justify-content: center;
  font-family: 'Work Sans', sans-serif;
  font-size: 18px;
  font-weight: 500;
  color: white;
  flex-direction: column;
  background: rgb(0,0,0,0.73);
  opacity: 0;
  animation: lindEnter 0.2s;
  animation-fill-mode: forwards;

  @keyframes lindEnter {
    from { opacity: 0; }
    to   { opacity: 1; }
  }
`;

const ButtonRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 180px;
  margin-top: 30px;
`;

const ConfirmButton = styled.div`
  font-size: 18px;
  padding: 10px 15px;
  outline: none; 
  border: 1px solid white;
  border-radius: 10px; 
  text-align: center; 
  width: 80px;
  cursor: pointer;
  opacity: 0;
  font-family: 'Work Sans', sans-serif;
  font-size: 18px;
  font-weight: 500;
  animation: linEnter 0.3s 0.1s;
  animation-fill-mode: forwards;
  @keyframes linEnter {
    from { transform: translateY(20px); opacity: 0; }
    to   { transform: translateY(0px); opacity: 1; }
  }
  :hover {
    background: white;
    color: #232323;
  }
`;

const RevisionList = styled.div`
  overflow-y: auto;
  max-height: 215px;
`;

const RollbackButton = styled.div`
  cursor: ${(props: { disabled: boolean }) => props.disabled ? 'not-allowed' :'pointer'};
  display: flex;
  border-radius: 3px;
  align-items: center;
  justify-content: center;
  font-weight: 500;
  height: 21px;
  font-size: 13px;
  width: 70px;
  background: ${(props: { disabled: boolean }) => props.disabled ? '#aaaabbee' :'#616FEEcc'};
  :hover {
    background: ${(props: { disabled: boolean }) => props.disabled ? '' : '#405eddbb'};
  }
`;

const Tr = styled.tr`
  line-height: 1.8em;
  cursor: ${(props: { disableHover?: boolean, selected?: boolean }) => props.disableHover ? '' : 'pointer'};
  background: ${(props: { disableHover?: boolean, selected?: boolean  }) => props.selected ? '#ffffff11' : ''};
  :hover {
    background: ${(props: { disableHover?: boolean, selected?: boolean  }) => props.disableHover ? '' : '#ffffff22'};
  }
`;

const Td = styled.td`
  font-size: 13px;
  color: #ffffff;
  padding-left: 32px;
`;

const Th = styled.td`
  font-size: 13px;
  font-weight: 500;
  color: #aaaabb;
  padding-left: 32px;
`;

const RevisionsTable = styled.table`
  width: 100%;
  margin-top: 5px;
  padding-left: 32px;
  padding-bottom: 20px;
  min-width: 500px;
  border-collapse: collapse;
`;

const Revision = styled.div`
  color: #ffffff;
  margin-left: 5px;
`;

const RevisionHeader = styled.div`
  color: #ffffff66;
  display: flex;
  align-items: center;
  height: 40px;
  font-size: 14px;
  width: 100%;
  padding-left: 15px;
  cursor: pointer;
  background: ${(props: { showRevisions: boolean }) => props.showRevisions ? '#ffffff11' : ''};
  :hover {
    background: #ffffff18;
    > i {
      background: #ffffff22;
    }
  }

  > i {
    margin-left: 12px;
    font-size: 20px;
    cursor: pointer;
    border-radius: 20px;
    background: ${(props: { showRevisions: boolean }) => props.showRevisions ? '#ffffff18' : ''};
    transform: ${(props: { showRevisions: boolean }) => props.showRevisions ? 'rotate(180deg)' : ''};
  }
`;

const StyledRevisionSection = styled.div`
  width: 100%;
  max-height: ${(props: { showRevisions: boolean }) => props.showRevisions ? '255px' : '40px'};
  background: #ffffff11;
  margin: 25px 0px;
  overflow: hidden;
  border-radius: 5px;
  animation: ${(props: { showRevisions: boolean }) => props.showRevisions ? 'expandRevisions 0.3s' : ''};
  animation-timing-function: ease-out;
  @keyframes expandRevisions {
    from { max-height: 40px }
    to { max-height: 250px }
  }
`;