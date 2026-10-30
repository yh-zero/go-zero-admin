import { useAuthStore } from '../stores/auth';
import { queryClient } from './queryClient';
import { cancelPendingRequests } from './requestLifecycle';
export function clearSession() {
  cancelPendingRequests();
  void queryClient.cancelQueries();
  queryClient.clear();
  useAuthStore.getState().logout();
}
