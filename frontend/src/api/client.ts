import type {
  Session,
  EmployeeSummary,
  Profile,
  Recommendations,
  Overview,
  ImportResult,
  Goal,
  Development,
  EnrollmentDetail,
  SubmissionDetail,
  Experience,
  Leaderboard,
  AssistantResponse,
  AssistantAction,
  Locale,
} from './types';
import { getLocale } from '../i18n';
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public details?: unknown,
  ) {
    super(message);
  }
}
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Accept-Language', getLocale());
  if (init.body && !(init.body instanceof FormData))
    headers.set('Content-Type', 'application/json');
  const response = await fetch(`/api${path}`, { ...init, headers, credentials: 'same-origin' });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    if (response.status === 401 && path !== '/session')
      window.dispatchEvent(new Event('session-expired'));
    const fallback = {
      ru: 'Не удалось выполнить запрос',
      kk: 'Сұрауды орындау мүмкін болмады',
      en: 'The request could not be completed',
    }[getLocale()];
    const translated: Record<string, [string, string, string]> = {
      not_found: ['Запись не найдена.', 'Жазба табылмады.', 'This record could not be found.'],
      forbidden: [
        'Недостаточно прав для этого действия.',
        'Бұл әрекетке рұқсат жеткіліксіз.',
        'You do not have permission for this action.',
      ],
      invalid_input: [
        'Проверьте введённые данные.',
        'Енгізілген деректерді тексеріңіз.',
        'Please check the information you entered.',
      ],
      conflict: [
        'Состояние изменилось. Обновите данные и повторите.',
        'Күй өзгерді. Деректерді жаңартып, қайталаңыз.',
        'The state changed. Refresh the data and try again.',
      ],
      not_eligible: [
        'Модуль пока недоступен. Проверьте условия доступа.',
        'Модуль әзірге қолжетімсіз. Кіру шарттарын тексеріңіз.',
        'This module is not available yet. Check its prerequisites.',
      ],
      review_required: [
        'Отправьте результат на проверку HR.',
        'Нәтижені HR тексеруіне жіберіңіз.',
        'Submit your result for HR review.',
      ],
      rate_limit: [
        'Слишком много попыток. Подождите минуту.',
        'Тым көп әрекет. Бір минут күтіңіз.',
        'Too many attempts. Please wait one minute.',
      ],
      unauthorized: [
        'Войдите в свой аккаунт.',
        'Аккаунтыңызға кіріңіз.',
        'Please sign in to your account.',
      ],
      invalid_credentials: [
        'Неверный логин или пароль.',
        'Логин немесе құпиясөз қате.',
        'Incorrect login or password.',
      ],
      invalid_upload: [
        'Проверьте файлы и допустимый размер загрузки.',
        'Файлдар мен рұқсат етілген өлшемді тексеріңіз.',
        'Check your files and the permitted upload size.',
      ],
      invalid_file_type: [
        'Допустимы только PDF, PNG и JPEG.',
        'Тек PDF, PNG және JPEG рұқсат етілген.',
        'Only PDF, PNG and JPEG files are allowed.',
      ],
      save_failed: [
        'Не удалось сохранить изменения. Попробуйте ещё раз.',
        'Өзгерістерді сақтау мүмкін болмады. Қайталап көріңіз.',
        'Changes could not be saved. Please try again.',
      ],
      stale_revision: [
        'Данные изменились в другом процессе. Обратитесь к команде для перезапуска сервера.',
        'Деректер басқа процесте өзгерді. Серверді қайта іске қосу үшін командаға хабарласыңыз.',
        'Data changed in another process. Ask the team to restart the server.',
      ],
      storage_unavailable: [
        'База данных временно недоступна.',
        'Дерекқор уақытша қолжетімсіз.',
        'The database is temporarily unavailable.',
      ],
      invalid_dataset: [
        'Пакет не прошёл проверку. Никакие данные не изменены. Подробности ниже.',
        'Пакет тексеруден өтпеді. Деректер өзгерген жоқ. Мәліметтер төменде.',
        'The import failed validation. No data was changed. Details are shown below.',
      ],
      invalid_json: [
        'Некорректный запрос. Обновите страницу.',
        'Сұрау қате. Бетті жаңартыңыз.',
        'Invalid request. Please refresh the page.',
      ],
      origin_denied: [
        'Этот источник запроса не разрешён.',
        'Бұл сұрау көзіне рұқсат жоқ.',
        'This request origin is not allowed.',
      ],
    };
    const translatedMessage = translated[body?.error?.code]?.[{ ru: 0, kk: 1, en: 2 }[getLocale()]];
    throw new ApiError(
      response.status,
      translatedMessage || body?.error?.message || `${fallback} (${response.status})`,
      body?.error?.details || body?.errors,
    );
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
const employee = (id: string) => `/employees/${encodeURIComponent(id)}`;
const json = (method: string, body: unknown) => ({ method, body: JSON.stringify(body) });
export const requestKey = () => crypto.randomUUID();
export const api = {
  session: () => request<Session>('/session'),
  login: (login: string, password: string) =>
    request<Session>('/session', json('POST', { login, password })),
  logout: () => request<void>('/session', { method: 'DELETE' }),
  preferences: (locale: Locale) => request<Session>('/me/preferences', json('PATCH', { locale })),
  employees: () => request<{ items: EmployeeSummary[] }>('/employees'),
  profile: (id: string, signal?: AbortSignal) => request<Profile>(employee(id), { signal }),
  recommendations: (id: string, signal?: AbortSignal) =>
    request<Recommendations>(`${employee(id)}/recommendations`, { signal }),
  goals: () => request<{ items: Goal[] }>('/catalog/goals'),
  goal: (id: string, goal: Goal | null) =>
    request<Profile>(`${employee(id)}/goal`, json('PATCH', { goal })),
  development: (id: string, signal?: AbortSignal) =>
    request<Development>(`${employee(id)}/development`, { signal }),
  start: (id: string, eventId: string) =>
    request<EnrollmentDetail>(
      `${employee(id)}/modules/${encodeURIComponent(eventId)}/start`,
      json('POST', {}),
    ),
  enrollment: (id: string) => request<EnrollmentDetail>(`/enrollments/${encodeURIComponent(id)}`),
  submit: (id: string, data: FormData) =>
    request<SubmissionDetail>(`/enrollments/${encodeURIComponent(id)}/submissions`, {
      method: 'POST',
      body: data,
    }),
  reviews: (params: URLSearchParams, signal?: AbortSignal) =>
    request<{ items: SubmissionDetail[]; pending_count: number }>(`/hr/submissions?${params}`, {
      signal,
    }),
  decide: (id: string, action: 'approve' | 'return', comment: string, key: string) =>
    request<SubmissionDetail>(
      `/hr/submissions/${encodeURIComponent(id)}/decision`,
      json('POST', { action, comment, request_key: key }),
    ),
  revoke: (id: string, comment: string, key: string) =>
    request<SubmissionDetail>(
      `/hr/approvals/${encodeURIComponent(id)}/revoke`,
      json('POST', { comment, request_key: key }),
    ),
  experience: (id: string) => request<Experience>(`${employee(id)}/experience`),
  leaderboard: (
    month = '',
    signal?: AbortSignal,
    filters: { department?: string; role?: string } = {},
  ) => {
    const query = new URLSearchParams();
    if (month) query.set('month', month);
    if (filters.department) query.set('department', filters.department);
    if (filters.role) query.set('role', filters.role);
    return request<Leaderboard>(`/leaderboard${query.size ? `?${query}` : ''}`, { signal });
  },
  assistant: (id: string, eventId: string, action: AssistantAction, locale: Locale) =>
    request<AssistantResponse>(
      `${employee(id)}/assistant`,
      json('POST', { event_id: eventId, action, locale }),
    ),
  overview: (signal?: AbortSignal) => request<Overview>('/hr/overview', { signal }),
  import: (files: FormData) => request<ImportResult>('/hr/import', { method: 'POST', body: files }),
};
