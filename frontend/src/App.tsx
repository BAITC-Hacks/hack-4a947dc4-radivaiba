import { useEffect, useState } from 'react';
import {
  ArrowUpRight,
  ChartNoAxesCombined,
  ChevronRight,
  CircleHelp,
  LogOut,
  Route,
  ShieldCheck,
} from 'lucide-react';
import { api, ApiError } from './api/client';
import type { EmployeeSummary, Session } from './api/types';
import { Brand, ErrorMessage, Loading } from './components/Shared';
import Login from './pages/Login';
import Employee from './pages/Employee';
import HR from './pages/HR';

const currentPath = () => window.location.hash.replace(/^#/, '') || '/';
export default function App() {
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState('');
  const [path, setPath] = useState(currentPath);
  const [employees, setEmployees] = useState<EmployeeSummary[]>([]);
  const [showHelp, setShowHelp] = useState(false);
  useEffect(() => {
    api
      .session()
      .then(setSession)
      .catch((err) => {
        if (!(err instanceof ApiError && err.status === 401)) setError(err.message);
      })
      .finally(() => setReady(true));
    const hash = () => setPath(currentPath());
    const expired = () => {
      setSession(null);
      setEmployees([]);
    };
    window.addEventListener('hashchange', hash);
    window.addEventListener('session-expired', expired);
    return () => {
      window.removeEventListener('hashchange', hash);
      window.removeEventListener('session-expired', expired);
    };
  }, []);
  async function loadEmployees() {
    try {
      setEmployees((await api.employees()).items);
    } catch (err) {
      setError((err as Error).message);
    }
  }
  useEffect(() => {
    if (session?.role === 'hr') loadEmployees();
    else setEmployees([]);
  }, [session]);
  const navigate = (next: string) => {
    window.location.hash = next;
    setPath(next);
    window.scrollTo({ top: 0 });
  };
  function login(value: Session) {
    setSession(value);
    setError('');
    navigate(value.role === 'hr' ? '/hr' : `/employees/${value.employee_id}`);
  }
  async function logout() {
    try {
      await api.logout();
      setSession(null);
      setEmployees([]);
      navigate('/');
    } catch (err) {
      setError((err as Error).message);
    }
  }
  if (!ready) return <Loading text="Открываем пространство развития…" />;
  if (!session)
    return (
      <>
        {error && (
          <div className="global-error">
            <ErrorMessage message={error} retry={() => window.location.reload()} />
          </div>
        )}
        <Login onLogin={login} />
      </>
    );
  const isHR = session.role === 'hr';
  const isOverview = isHR && !path.startsWith('/employees/');
  const selectedId = isHR
    ? decodeURIComponent(path.split('/')[2] || employees[0]?.employee_id || 'E0001')
    : session.employee_id;
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Brand />
        <div className="workspace-label">РАБОЧЕЕ ПРОСТРАНСТВО</div>
        <nav>
          {isHR && (
            <button className={isOverview ? 'active' : ''} onClick={() => navigate('/hr')}>
              <ChartNoAxesCombined size={19} />
              Обзор команды
            </button>
          )}
          <button
            className={!isOverview ? 'active' : ''}
            onClick={() => navigate(`/employees/${isHR ? selectedId : session.employee_id}`)}
          >
            <Route size={19} />
            {isHR ? 'Профиль сотрудника' : 'Моя траектория'}
          </button>
        </nav>
        <div className="sidebar-tip">
          <span className="tip-icon">
            <ArrowUpRight size={22} />
          </span>
          <h3>Рост — это путь</h3>
          <p>Сосредоточьтесь на следующем полезном шаге. Остальное придёт с опытом.</p>
        </div>
        <div className="sidebar-bottom">
          <button onClick={() => setShowHelp((v) => !v)}>
            <CircleHelp size={18} />
            Как это работает
          </button>
          <div className="local-label">
            <span />
            Локальное демо
          </div>
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="breadcrumb">
            Пространство развития
            <ChevronRight size={14} />
            <strong>{isOverview ? 'HR-обзор' : 'Траектория'}</strong>
          </div>
          <div className="topbar-actions">
            <span className="role-pill">
              <ShieldCheck size={15} />
              {isHR ? 'HR-команда' : 'Сотрудник'}
            </span>
            <button className="icon-button" title="Выйти" aria-label="Выйти" onClick={logout}>
              <LogOut size={18} />
            </button>
          </div>
        </header>
        <main className="main-content">
          {error && <ErrorMessage message={error} />}{' '}
          {showHelp && (
            <section className="help-panel panel">
              <h2>Как строится ваш путь</h2>
              <p>
                Мы сравниваем навыки с требованиями карьерной цели. Из доступных активностей
                выбираем те, которые закрывают разрывы, учитывая историю участия и время на
                обучение. AI помогает выбрать следующий шаг и объяснить его пользу.
              </p>
              <p>
                Прогресс не означает автоматическое повышение. В демо завершение активности
                моделируется на дату среза; фактическая регистрация на мероприятие не выполняется.
              </p>
              <button className="text-button" onClick={() => setShowHelp(false)}>
                Понятно
              </button>
            </section>
          )}
          {isHR && !isOverview && (
            <div className="employee-picker">
              <label htmlFor="employee-picker">Профиль сотрудника</label>
              <select
                id="employee-picker"
                value={selectedId}
                onChange={(e) => navigate(`/employees/${e.target.value}`)}
              >
                {employees.map((e) => (
                  <option key={e.employee_id} value={e.employee_id}>
                    {e.full_name} · {e.role} · {e.employee_id}
                  </option>
                ))}
              </select>
              <button className="text-button" onClick={() => navigate('/hr')}>
                К обзору команды
              </button>
            </div>
          )}
          {isOverview ? (
            <HR
              employees={employees}
              onEmployee={(id) => navigate(`/employees/${id}`)}
              onImport={loadEmployees}
            />
          ) : (
            <Employee key={selectedId} id={selectedId} />
          )}
          <footer className="page-footer">
            <span>
              Career Quest <span className="footer-dot">·</span> Ваше развитие имеет направление.
            </span>
            <span>HackAlem AI · 2026</span>
          </footer>
        </main>
      </div>
    </div>
  );
}
