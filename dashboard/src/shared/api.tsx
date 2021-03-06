import axios from 'axios';
import { baseApi } from './baseApi';

import { StorageType } from './types';

/**
 * Generic api call format
 * @param {string} token - Bearer token.
 * @param {Object} params - Body params.
 * @param {Object} pathParams - Path params.
 * @param {(err: Object, res: Object) => void} callback - Callback function.
 */

const checkAuth = baseApi('GET', '/api/auth/check');

const registerUser = baseApi<{ 
  email: string, 
  password: string
}>('POST', '/api/users');

const logInUser = baseApi<{
  email: string,
  password: string
}>('POST', '/api/login');

const logOutUser = baseApi('POST', '/api/logout');

const getUser = baseApi<{}, { id: number }>('GET', pathParams => {
  return `/api/users/${pathParams.id}`;
});

const updateUser = baseApi<{
  rawKubeConfig?: string,
  allowedContexts?: string[]
}, { id: number }>('PUT', pathParams => {
  return `/api/users/${pathParams.id}`;
});

const getContexts = baseApi<{}, { id: number }>('GET', pathParams => {
  return `/api/users/${pathParams.id}/contexts`;
});

const getCharts = baseApi<{
  namespace: string,
  context: string,
  storage: StorageType,
  limit: number,
  skip: number,
  byDate: boolean,
  statusFilter: string[]
}>('GET', '/api/releases');

const getChart = baseApi<{
  namespace: string,
  context: string,
  storage: StorageType
}, { name: string, revision: number }>('GET', pathParams => {
  return `/api/releases/${pathParams.name}/${pathParams.revision}`;
});

const getChartComponents = baseApi<{
  namespace: string,
  context: string,
  storage: StorageType
}, { name: string, revision: number }>('GET', pathParams => {
  return `/api/releases/${pathParams.name}/${pathParams.revision}/components`;
});

const getNamespaces = baseApi<{
  context: string
}>('GET', '/api/k8s/namespaces');

const getRevisions = baseApi<{
  namespace: string,
  context: string,
  storage: StorageType
}, { name: string }>('GET', pathParams => {
  return `/api/releases/${pathParams.name}/history`;
});

const rollbackChart = baseApi<{
  namespace: string,
  context: string,
  storage: StorageType,
  revision: number
}, { name: string }>('POST', pathParams => {
  return `/api/releases/${pathParams.name}/rollback`;
});

const upgradeChartValues = baseApi<{
  namespace: string,
  context: string,
  storage: StorageType,
  values: string
}, { name: string }>('POST', pathParams => {
  return `/api/releases/${pathParams.name}/upgrade`;
});

// Bundle export to allow default api import (api.<method> is more readable)
export default {
  checkAuth,
  registerUser,
  logInUser,
  logOutUser,
  getUser,
  updateUser,
  getContexts,
  getCharts,
  getChart,
  getChartComponents,
  getNamespaces,
  getRevisions,
  rollbackChart,
  upgradeChartValues
}