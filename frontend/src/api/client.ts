import type {
  Session,
  EmployeeSummary,
  Profile,
  Recommendations,
  Overview,
  ImportResult,
} from './types';
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !(init.body instanceof FormData))
    headers.set('Content-Type', 'application/json');
  const response = await fetch(`/api${path}`, { ...init, headers, credentials: 'same-origin' });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    if (response.status === 401 && path !== '/session')
      window.dispatchEvent(new Event('session-expired'));
    throw new ApiError(
      response.status,
      body?.error?.message || `Ошибка сервера (${response.status})`,
    );
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
export const api = {
  session: () => request<Session>('/session'),
  login: (role: string, password: string) =>
    request<Session>('/session', { method: 'POST', body: JSON.stringify({ role, password }) }),
  logout: () => request<void>('/session', { method: 'DELETE' }),
  employees: () => request<{ items: EmployeeSummary[] }>('/employees'),
  profile: (id: string, signal?: AbortSignal) =>
    request<Profile>(`/employees/${encodeURIComponent(id)}`, { signal }),
  recommendations: (id: string, signal?: AbortSignal) =>
    request<Recommendations>(`/employees/${encodeURIComponent(id)}/recommendations`, { signal }),
  complete: (id: string, eventId: string) =>
    request<{ profile: Profile; changed: boolean }>(
      `/employees/${encodeURIComponent(id)}/completions`,
      { method: 'POST', body: JSON.stringify({ event_id: eventId }) },
    ),
  overview: (signal?: AbortSignal) => request<Overview>('/hr/overview', { signal }),
  import: (files: FormData) => request<ImportResult>('/hr/import', { method: 'POST', body: files }),
};
